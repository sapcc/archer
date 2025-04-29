// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"strings"
	"time"

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
		gocron.WithLimitConcurrentJobs(1, gocron.LimitModeWait),
		gocron.WithLogger(NewGoCronLogger()),
		gocron.WithMonitor(NewPrometheusMonitor()),
		gocron.WithMonitorStatus(&DebugMonitor{}),
		gocron.WithStopTimeout(time.Second*30),
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
