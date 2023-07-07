// Copyright 2023 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agent

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/go-bits/jobloop"
	"github.com/sapcc/go-bits/logg"
	"strings"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
)

func RegisterAgent(pool *pgxpool.Pool, provider string) {
	sql, args := db.Insert("agents").
		Columns("host", "availability_zone", "provider").
		Values(config.Global.Default.Host, config.Global.Default.AvailabilityZone, provider).
		Suffix("ON CONFLICT (host) DO UPDATE SET").
		SuffixExpr(sq.Expr("availability_zone = ?,", config.Global.Default.AvailabilityZone)).
		Suffix("updated_at = now()").
		MustSql()

	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		panic(err)
	}
}

type job struct {
	model string
	id    strfmt.UUID
}

type JobChan chan job

func (j *JobChan) Enqueue(model string, id strfmt.UUID) error {
	job := job{model: model, id: id}
	select {
	case *j <- job:
		logg.Debug("enqueued job %v", job)
		return nil
	default:
		return fmt.Errorf("failed to enque %v", j)
	}
}

func (j *JobChan) Dequeue() (string, strfmt.UUID) {
	job := <-*j
	logg.Debug("dequeued job %v", job)
	return job.model, job.id
}

type Worker interface {
	PendingSyncLoop(context.Context, prometheus.Labels) error
	ProcessServices(context.Context) error
	ProcessEndpoint(context.Context, strfmt.UUID) error
	GetJobQueue() *JobChan
}

func CronJob(w Worker) jobloop.Job {
	jl := jobloop.CronJob{
		Metadata: jobloop.JobMetadata{
			ReadableName:    "pending_sync_loop",
			ConcurrencySafe: false,
			CounterOpts: prometheus.CounterOpts{
				Name: "archer_pending_sync_loop_total",
				Help: "Total number of pending sync loops",
			},
			CounterLabels: nil,
		},
		Interval: config.Global.Agent.PendingSyncInterval,
		Task:     w.PendingSyncLoop,
	}

	return jl.Setup(prometheus.DefaultRegisterer)
}

func WorkerThread(ctx context.Context, w Worker) {
	for job := range *w.GetJobQueue() {
		var err error
		logg.Debug("received message %v", job)

		switch job.model {
		case "service":
			if err = w.ProcessServices(ctx); err != nil {
				logg.Error(err.Error())
			}
		case "endpoint":
			if err = w.ProcessEndpoint(ctx, job.id); err != nil {
				logg.Error(err.Error())
			}
		}

		outcome := "success"
		if err != nil {
			outcome = "failure"
		}
		processJobCount.WithLabelValues(job.model, outcome).Inc()
	}
}

func DBNotificationThread(ctx context.Context, pool *pgxpool.Pool, jobQueue *JobChan) {
	// Acquire one Connection for listen events
	conn, err := pool.Acquire(ctx)
	if err != nil {
		logg.Fatal(err.Error())
	}

	sql := "LISTEN service; LISTEN endpoint;"
	if _, err := conn.Exec(ctx, sql); err != nil {
		logg.Fatal(err.Error())
	}

	logg.Info("DBNotificationThread: Listening to service and endpoint notifications")

	for {
		var id strfmt.UUID
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if !pgconn.Timeout(err) {
				logg.Fatal(err.Error())
			}
			continue
		}

		logg.Debug("Received notification, channel=%s, payload=%s", notification.Channel, notification.Payload)
		s := strings.SplitN(notification.Payload, ":", 2)
		if len(s) < 1 {
			logg.Error("Received invalid notification payload: %s", notification.Payload)
			continue
		}

		if s[0] != config.Global.Default.Host {
			continue
		}
		if len(s) > 1 {
			id = strfmt.UUID(s[1])
		}

		if err := jobQueue.Enqueue(notification.Channel, id); err != nil {
			logg.Error(err.Error())
		}
	}
}
