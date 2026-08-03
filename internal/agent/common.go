// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-co-op/gocron/v2"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/internal/db"
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

// ErrProcessBusy is returned by a Worker's ProcessServices/ProcessEndpoint when
// another run already holds the serialization lock, so the caller knows the work
// was skipped (not completed) and should be retried.
var ErrProcessBusy = errors.New("another process run is already in progress")

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

// Retried after a service notification because a migrated-away service leaves
// nothing for PendingSyncLoop to re-trigger. Vars so tests can shrink the delay.
var (
	processServicesRetryDelay = 10 * time.Second
	processServicesMaxRetries = 20
	processServicesBusyDelay  = 5 * time.Second
)

// Coalescer collapses a burst of ProcessServices enqueue signals into at most
// one queued run, guaranteeing that a signal arriving while a run executes
// triggers exactly one follow-up run. The zero value is ready to use.
type Coalescer struct {
	mu      sync.Mutex
	running bool // a run is queued or executing
	rerun   bool // a signal arrived during the current run
}

// requestEnqueue returns true if the caller should enqueue a new run, or false
// if one is already queued/running (recording that a re-run is needed).
func (c *Coalescer) requestEnqueue() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running {
		c.rerun = true
		return false
	}
	c.running = true
	return true
}

// reset clears the state after a failed enqueue so the next signal can retry.
func (c *Coalescer) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
	c.rerun = false
}

// wrap runs task, then re-enqueues exactly once if a signal arrived meanwhile.
func (c *Coalescer) wrap(ctx context.Context, scheduler gocron.Scheduler, task func() error) func() error {
	return func() error {
		err := task()
		c.mu.Lock()
		rerun := c.rerun
		c.rerun = false
		c.running = rerun // stay "running" iff we are about to re-enqueue
		c.mu.Unlock()
		if rerun && ctx.Err() == nil {
			if _, serr := scheduler.NewJob(
				gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
				gocron.NewTask(c.wrap(ctx, scheduler, task)),
				gocron.WithName("ProcessServices"),
				gocron.WithContext(ctx),
			); serr != nil {
				log.WithError(serr).Error("failed re-enqueueing coalesced ProcessServices job")
				c.reset()
			}
		}
		return err
	}
}

// processServicesTask returns a ProcessServices task that, on failure,
// reschedules itself after a short delay until it succeeds or the retry budget
// is exhausted.
func processServicesTask(ctx context.Context, scheduler gocron.Scheduler, w Worker) func() error {
	var attempt int
	var run func() error
	reschedule := func(delay time.Duration) {
		if _, serr := scheduler.NewJob(
			gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(time.Now().Add(delay))),
			gocron.NewTask(run),
			gocron.WithName("ProcessServices"),
			gocron.WithContext(ctx),
		); serr != nil {
			log.WithError(serr).Error("failed rescheduling ProcessServices job")
		}
	}
	run = func() error {
		err := w.ProcessServices(ctx)
		if err == nil {
			return nil
		}
		// Lost the advisory-lock race: reschedule promptly instead of waiting for the 120s PendingSyncLoop.
		if errors.Is(err, ErrProcessBusy) {
			if ctx.Err() == nil {
				reschedule(processServicesBusyDelay)
			}
			return nil
		}
		attempt++
		if attempt >= processServicesMaxRetries || ctx.Err() != nil {
			return err
		}
		log.WithError(err).Warnf("ProcessServices failed, retrying in %s (attempt %d/%d)",
			processServicesRetryDelay, attempt, processServicesMaxRetries)
		reschedule(processServicesRetryDelay)
		return err
	}
	return run
}

// ScheduleProcessServices enqueues a ProcessServices run that reschedules itself
// on failure or advisory-lock contention, so a busy-skipped run is re-driven
// promptly rather than at the next sync interval. The scheduler is taken from
// the Worker, so both the notification handler and PendingSyncLoop share it.
//
// Enqueues are coalesced when the Worker provides a *Coalescer via
// ProcessServicesCoalescer(): a burst of signals collapses to at most one queued
// run, with exactly one follow-up run if a signal arrives while one is executing.
func ScheduleProcessServices(ctx context.Context, w Worker) error {
	scheduler := w.GetScheduler()
	var c *Coalescer
	if cw, ok := w.(interface{ ProcessServicesCoalescer() *Coalescer }); ok {
		c = cw.ProcessServicesCoalescer()
	}
	if c != nil && !c.requestEnqueue() {
		return nil // a run is already queued/running; it will re-run for this signal
	}
	task := processServicesTask(ctx, scheduler, w)
	if c != nil {
		task = c.wrap(ctx, scheduler, task)
	}
	_, err := scheduler.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
		gocron.NewTask(task),
		gocron.WithName("ProcessServices"),
		gocron.WithContext(ctx),
	)
	if err != nil && c != nil {
		c.reset() // enqueue failed; allow the next signal to try again
	}
	return err
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
			if err := ScheduleProcessServices(ctx, w); err != nil {
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
