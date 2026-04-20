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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
)

func RegisterAgent(pool db.PgxIface, provider string) {
	var az *string
	var physnet *string
	if config.Global.Default.AvailabilityZone != "" {
		az = &config.Global.Default.AvailabilityZone
	}
	if config.Global.Agent.PhysicalNetwork != "" {
		physnet = &config.Global.Agent.PhysicalNetwork
	}
	sql, args := db.Insert("agents").
		Columns("host", "availability_zone", "provider", "physnet").
		Values(config.Global.Default.Host, az, provider, physnet).
		Suffix("ON CONFLICT (host) DO UPDATE SET").
		SuffixExpr(sq.Expr("availability_zone = ?,", az)).
		SuffixExpr(sq.Expr("physnet = ?,", physnet)).
		Suffix("updated_at = now(),").
		Suffix("heartbeat_at = now(),").
		Suffix("enabled = true").
		MustSql()

	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		panic(err)
	}
}

// UpdateHeartbeat updates the agent's heartbeat timestamp in the database.
// This should be called periodically to indicate the agent is still alive.
func UpdateHeartbeat(pool db.PgxIface) {
	sql, args := db.Update("agents").
		Set("heartbeat_at", sq.Expr("NOW()")).
		Where("host = ?", config.Global.Default.Host).
		MustSql()

	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		log.WithError(err).Error("Failed to update heartbeat")
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
		gocron.WithGlobalJobOptions(
			gocron.WithEventListeners(
				gocron.BeforeJobRuns(func(jobID uuid.UUID, jobName string) {
					log.Debugf("Job STARTING: name=%s, id=%s", jobName, jobID)
				}),
				gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
					log.Debugf("Job FINISHED: name=%s, id=%s", jobName, jobID)
				}),
				gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
					log.Errorf("Job FAILED: name=%s, job_id=%s, error=%v", jobName, jobID, err)
				}),
			),
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	return scheduler
}

func DBNotificationThread(ctx context.Context, w Worker) {
	const reconnectionDelay = time.Minute / 2

	for {
		// Check if context is canceled before acquiring connection
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Acquire a connection for listen events
		conn, err := w.GetPool().Acquire(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // Context cancelled, graceful shutdown
			}
			log.WithError(err).Error("DBNotificationThread: Failed to acquire connection")
			time.Sleep(reconnectionDelay)
			continue
		}

		sql := "LISTEN service; LISTEN endpoint;"
		if _, err = conn.Exec(ctx, sql); err != nil {
			conn.Release()
			if ctx.Err() != nil {
				return // Context cancelled, graceful shutdown
			}
			log.WithError(err).Error("DBNotificationThread: Failed to setup LISTEN")
			time.Sleep(reconnectionDelay)
			continue
		}

		log.Infof("DBNotificationThread: Listening to service and endpoint notifications, reconnection delay %v",
			reconnectionDelay)

		// Process notifications until connection error
		if err = processNotifications(ctx, conn, w); err != nil {
			conn.Release()
			if ctx.Err() != nil {
				return // Context cancelled, graceful shutdown
			}
			log.WithError(err).Warn("DBNotificationThread: Connection lost, reconnecting...")
			time.Sleep(reconnectionDelay)
			continue
		}

		conn.Release()
		return // Context canceled
	}
}

func processNotifications(ctx context.Context, conn *pgxpool.Conn, w Worker) error {
	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // Context cancelled, graceful shutdown
			}
			if pgconn.Timeout(err) {
				continue // Timeout is normal, just retry
			}
			return err // Connection error, need to reconnect
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

		var id strfmt.UUID
		if len(s) > 1 {
			id = strfmt.UUID(s[1])
		}

		scheduler := w.GetScheduler()
		switch notification.Channel {
		case "service":
			if _, err := scheduler.NewJob(
				gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
				gocron.NewTask(w.ProcessServices),
				gocron.WithName("ProcessServices"),
				gocron.WithContext(ctx),
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
				gocron.NewTask(w.ProcessEndpoint, id),
				gocron.WithName("ProcessEndpoint"),
				gocron.WithTags(id.String()),
				gocron.WithContext(ctx),
			); nil != err {
				log.WithError(err).WithField("endpoint_id", id).Error("failed enqueueing ProcessEndpoint job")
			}
		}
	}
}
