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
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"

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
		log.WithField("job", job).Debug("Job enqueued")
		return nil
	default:
		return fmt.Errorf("failed to enque %v", j)
	}
}

func (j *JobChan) Dequeue() (string, strfmt.UUID) {
	job := <-*j
	log.WithField("job", job).Debug("Job dequeued")
	return job.model, job.id
}

type Worker interface {
	ProcessServices(context.Context) error
	ProcessEndpoint(context.Context, strfmt.UUID) error
	GetJobQueue() *JobChan
}

func WorkerThread(ctx context.Context, w Worker) {
	for job := range *w.GetJobQueue() {
		var err error
		log.WithField("job", job).Debug("Message received")

		switch job.model {
		case "service":
			if err = w.ProcessServices(ctx); err != nil {
				log.Error(err.Error())
			}
		case "endpoint":
			if err = w.ProcessEndpoint(ctx, job.id); err != nil {
				log.Error(err.Error())
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
		log.Fatal(err.Error())
	}

	sql := "LISTEN service; LISTEN endpoint;"
	if _, err := conn.Exec(ctx, sql); err != nil {
		log.Fatal(err.Error())
	}

	log.Info("DBNotificationThread: Listening to service and endpoint notifications")

	for {
		var id strfmt.UUID
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if !pgconn.Timeout(err) {
				log.Fatal(err.Error())
			}
			continue
		}

		log.Debugf("Received notification, channel=%s, payload=%s", notification.Channel, notification.Payload)
		s := strings.SplitN(notification.Payload, ":", 2)
		if len(s) < 1 {
			log.Errorf("Received invalid notification payload: %s", notification.Payload)
			continue
		}

		if s[0] != config.Global.Default.Host {
			continue
		}
		if len(s) > 1 {
			id = strfmt.UUID(s[1])
		}

		if err := jobQueue.Enqueue(notification.Channel, id); err != nil {
			log.Error(err.Error())
		}
	}
}
