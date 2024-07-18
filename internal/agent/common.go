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
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-co-op/gocron/v2"
	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgx/v5/pgconn"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
)

func RegisterAgent(pool db.PgxIface, provider string) {
	var az *string
	if config.Global.Default.AvailabilityZone != "" {
		az = &config.Global.Default.AvailabilityZone
	}
	sql, args := db.Insert("agents").
		Columns("host", "availability_zone", "provider").
		Values(config.Global.Default.Host, az, provider).
		Suffix("ON CONFLICT (host) DO UPDATE SET").
		SuffixExpr(sq.Expr("availability_zone = ?,", az)).
		Suffix("updated_at = now()").
		MustSql()

	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		panic(err)
	}
}

type Worker interface {
	ProcessServices(context.Context) error
	ProcessEndpoint(context.Context, strfmt.UUID) error
	GetPool() db.PgxIface
	GetScheduler() gocron.Scheduler
}

func NewScheduler() gocron.Scheduler {
	scheduler, err := gocron.NewScheduler(
		gocron.WithLimitConcurrentJobs(1, gocron.LimitModeReschedule),
		gocron.WithLogger(NewGoCronLogger()),
		gocron.WithMonitor(NewPrometheusMonitor()),
	)
	if err != nil {
		log.Fatal(err)
	}
	return scheduler
}

func DBNotificationThread(ctx context.Context, w Worker) {
	// Acquire one Connection for listen events
	conn, err := w.GetPool().Acquire(ctx)
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

		scheduler := w.GetScheduler()
		switch notification.Channel {
		case "service":
			if _, err := scheduler.NewJob(
				gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
				gocron.NewTask(w.ProcessServices, ctx),
				gocron.WithName("ProcessServices"),
			); nil != err {
				log.WithError(err).Error("failed enqueueing ProcessServices job")
			}
		case "endpoint":
			if id == "" {
				log.Error("Received endpoint notification without ID")
				continue
			}
			if _, err := scheduler.NewJob(
				gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
				gocron.NewTask(w.ProcessEndpoint, ctx, id),
				gocron.WithName("ProcessEndpoint"),
				gocron.WithTags(id.String()),
			); nil != err {
				log.WithError(err).WithField("id", id).Error("failed enqueueing ProcessEndpoint job")
			}
		}
	}
}
