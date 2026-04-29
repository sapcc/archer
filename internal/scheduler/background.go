// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-co-op/gocron/v2"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/db"
)

// BackgroundScheduler runs periodic rescheduling and rebalancing jobs using gocron.
// It uses PostgreSQL advisory locks for distributed leader election to ensure
// only one instance runs jobs across multiple API server instances.
type BackgroundScheduler struct {
	scheduler       *ServiceScheduler
	gocronScheduler gocron.Scheduler
	elector         *PostgresElector
	checkInterval   time.Duration
	rebalanceDelay  time.Duration
	recoveredAgents map[string]time.Time // host -> recoveredAt
	mu              sync.Mutex
}

// NewBackgroundScheduler creates a new background scheduler with distributed leader election.
func NewBackgroundScheduler(serviceScheduler *ServiceScheduler, pool db.PgxIface, checkInterval, rebalanceDelay time.Duration) (*BackgroundScheduler, error) {
	elector := NewPostgresElector(pool)

	gocronSched, err := gocron.NewScheduler(
		gocron.WithDistributedElector(elector),
		gocron.WithLimitConcurrentJobs(1, gocron.LimitModeWait),
		gocron.WithStopTimeout(time.Second*30),
	)
	if err != nil {
		return nil, err
	}

	return &BackgroundScheduler{
		scheduler:       serviceScheduler,
		gocronScheduler: gocronSched,
		elector:         elector,
		checkInterval:   checkInterval,
		rebalanceDelay:  rebalanceDelay,
		recoveredAgents: make(map[string]time.Time),
	}, nil
}

// Start starts the background scheduler and registers all periodic jobs.
func (b *BackgroundScheduler) Start(ctx context.Context) error {
	// Start the elector to acquire dedicated connection
	if err := b.elector.Start(ctx); err != nil {
		return err
	}

	// Register the main scheduling cycle job
	_, err := b.gocronScheduler.NewJob(
		gocron.DurationJob(b.checkInterval),
		gocron.NewTask(func() {
			b.runCycle(ctx)
		}),
		gocron.WithName("SchedulerCycle"),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	if err != nil {
		return err
	}

	log.Infof("Background scheduler started with check interval %v (distributed mode)", b.checkInterval)
	b.gocronScheduler.Start()
	return nil
}

// Stop stops the background scheduler gracefully.
func (b *BackgroundScheduler) Stop() error {
	b.elector.Close()
	return b.gocronScheduler.Shutdown()
}

func (b *BackgroundScheduler) runCycle(ctx context.Context) {
	// Process only cp providers for now
	for _, provider := range []string{"cp"} {
		// 1. Check for and handle stale agents
		if err := b.scheduler.RescheduleStaleAgentServices(ctx, provider); err != nil {
			log.WithError(err).WithField("provider", provider).Error("Failed to reschedule stale agent services")
		}

		// 2. Check for recovered agents
		b.checkRecoveredAgents(ctx, provider)

		// 3. Rebalance if needed (only for agents that have been recovered for rebalanceDelay)
		b.checkRebalance(ctx, provider)
	}
}

func (b *BackgroundScheduler) checkRecoveredAgents(ctx context.Context, provider string) {
	// Find agents that were previously disabled but are now sending heartbeats
	sql, args := db.Select("host", "availability_zone").
		From("agents").
		Where("provider = ?", provider).
		Where("enabled = false").
		Where("heartbeat_at > NOW() - INTERVAL '1 second' * ?", int(b.scheduler.config.StaleTimeout.Seconds())).
		MustSql()

	type disabledAgent struct {
		Host             string
		AvailabilityZone *string
	}
	var agents []disabledAgent
	if err := pgxscan.Select(ctx, b.scheduler.pool, &agents, sql, args...); err != nil {
		log.WithError(err).Error("Failed to query recovered agents")
		return
	}

	for _, agent := range agents {
		log.WithField("host", agent.Host).Info("Agent recovered, re-enabling")

		// Re-enable the agent
		sql, args = db.Update("agents").
			Set("enabled", true).
			Where("host = ?", agent.Host).
			MustSql()

		if _, err := b.scheduler.pool.Exec(ctx, sql, args...); err != nil {
			log.WithError(err).WithField("host", agent.Host).Error("Failed to re-enable recovered agent")
			continue
		}

		// Track recovery time for rebalance delay
		b.mu.Lock()
		b.recoveredAgents[agent.Host] = time.Now()
		b.mu.Unlock()
	}
}

func (b *BackgroundScheduler) checkRebalance(ctx context.Context, provider string) {
	// Get all availability zones for this provider
	azs, err := b.getAvailabilityZones(ctx, provider)
	if err != nil {
		log.WithError(err).Error("Failed to get availability zones")
		return
	}

	// Check if any recovered agents have passed the rebalance delay
	b.mu.Lock()
	shouldRebalance := false
	now := time.Now()
	for host, recoveredAt := range b.recoveredAgents {
		if now.Sub(recoveredAt) >= b.rebalanceDelay {
			log.WithField("host", host).Info("Recovered agent passed rebalance delay, triggering rebalance")
			delete(b.recoveredAgents, host)
			shouldRebalance = true
		}
	}
	b.mu.Unlock()

	if !shouldRebalance {
		return
	}

	// Rebalance each AZ
	for _, az := range azs {
		azPtr := az
		if az == "" {
			azPtr = ""
		}
		if err := b.scheduler.RebalanceServices(ctx, provider, &azPtr); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"provider": provider,
				"az":       az,
			}).Error("Failed to rebalance services")
		}
	}
}

func (b *BackgroundScheduler) getAvailabilityZones(ctx context.Context, provider string) ([]string, error) {
	sql, args := db.Select("DISTINCT availability_zone").
		From("agents").
		Where("provider = ?", provider).
		Where("enabled = true").
		MustSql()

	rows, err := b.scheduler.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var azs []string
	for rows.Next() {
		var az *string
		if err := rows.Scan(&az); err != nil {
			return nil, err
		}
		if az != nil {
			azs = append(azs, *az)
		} else {
			azs = append(azs, "")
		}
	}

	return azs, rows.Err()
}
