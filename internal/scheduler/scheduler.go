// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/db"
	"github.com/sapcc/archer/v2/models"
)

// Config holds configuration for the service scheduler.
type Config struct {
	StaleTimeout           time.Duration
	CheckInterval          time.Duration
	RebalanceDelay         time.Duration
	RebalanceThreshold     float64
	RebalanceMaxMigrations int
}

// ServiceScheduler handles service scheduling, rescheduling, and rebalancing.
type ServiceScheduler struct {
	pool   db.PgxIface
	config Config
	notify func(host string) // callback to notify agents
}

// NewServiceScheduler creates a new service scheduler.
func NewServiceScheduler(pool db.PgxIface, cfg Config, notifyFunc func(host string)) *ServiceScheduler {
	return &ServiceScheduler{
		pool:   pool,
		config: cfg,
		notify: notifyFunc,
	}
}

// AgentInfo contains information about an agent and its load.
type AgentInfo struct {
	Host             string
	AvailabilityZone *string
	ServiceCount     int
	HeartbeatAt      time.Time
}

// FindLeastLoadedAgent returns the least-loaded healthy agent for a provider/AZ.
// excludeHost can be set to exclude a specific host (e.g., during migration).
func (s *ServiceScheduler) FindLeastLoadedAgent(ctx context.Context, provider string, az *string, excludeHost string) (string, error) {
	q := db.Select("agents.host", "COUNT(service.id) AS usage").
		From("agents").
		LeftJoin("service ON service.host = agents.host").
		Where(sq.And{
			sq.Eq{"agents.enabled": true},
			sq.Eq{"agents.provider": provider},
			sq.Eq{"agents.availability_zone": az},
			sq.Expr("agents.heartbeat_at > NOW() - INTERVAL '1 second' * ?", int(s.config.StaleTimeout.Seconds())),
		}).
		GroupBy("agents.host").
		OrderBy("usage ASC", "agents.heartbeat_at DESC").
		Limit(1)

	if excludeHost != "" {
		q = q.Where(sq.NotEq{"agents.host": excludeHost})
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return "", err
	}

	var host string
	var usage int
	if err = s.pool.QueryRow(ctx, sql, args...).Scan(&host, &usage); err != nil {
		return "", err
	}

	return host, nil
}

// GetStaleAgents returns agents that haven't sent a heartbeat within the stale timeout.
func (s *ServiceScheduler) GetStaleAgents(ctx context.Context, provider string) ([]AgentInfo, error) {
	sql, args := db.Select("host", "availability_zone").
		From("agents").
		Where("provider = ?", provider).
		Where("enabled = true").
		Where("heartbeat_at < NOW() - INTERVAL '1 second' * ?", int(s.config.StaleTimeout.Seconds())).
		MustSql()

	var agents []AgentInfo
	if err := pgxscan.Select(ctx, s.pool, &agents, sql, args...); err != nil {
		return nil, err
	}

	return agents, nil
}

// RescheduleStaleAgentServices migrates services from stale agents to healthy ones.
// Only disables agents after all their services have been successfully migrated.
func (s *ServiceScheduler) RescheduleStaleAgentServices(ctx context.Context, provider string) error {
	staleAgents, err := s.GetStaleAgents(ctx, provider)
	if err != nil {
		return err
	}

	for _, agent := range staleAgents {
		log.WithField("host", agent.Host).Warning("Agent is stale, rescheduling services")

		allMigrated, err := s.rescheduleAgentServices(ctx, provider, agent)
		if err != nil {
			log.WithError(err).WithField("host", agent.Host).Error("Failed to reschedule services from stale agent")
			continue
		}

		// Only disable the agent if all services were successfully migrated
		if !allMigrated {
			log.WithField("host", agent.Host).Warning("Not all services migrated, keeping agent enabled for retry")
			continue
		}

		// Disable the stale agent
		sql, args := db.Update("agents").
			Set("enabled", false).
			Where("host = ?", agent.Host).
			MustSql()

		if _, err := s.pool.Exec(ctx, sql, args...); err != nil {
			log.WithError(err).WithField("host", agent.Host).Error("Failed to disable stale agent")
		}
	}

	return nil
}

// rescheduleAgentServices migrates all services from a stale agent to healthy ones.
// Returns true if all services were successfully migrated, false otherwise.
func (s *ServiceScheduler) rescheduleAgentServices(ctx context.Context, provider string, agent AgentInfo) (bool, error) {
	// Find services on this agent
	sql, args := db.Select("id").
		From("service").
		Where("host = ?", agent.Host).
		Where("provider = ?", provider).
		MustSql()

	var serviceIDs []strfmt.UUID
	if err := pgxscan.Select(ctx, s.pool, &serviceIDs, sql, args...); err != nil {
		return false, err
	}

	if len(serviceIDs) == 0 {
		return true, nil // No services to migrate
	}

	failedCount := 0
	for _, serviceID := range serviceIDs {
		if err := s.MigrateService(ctx, serviceID, agent.Host, ""); err != nil {
			log.WithError(err).WithField("service", serviceID).Warning("Failed to reschedule service")
			failedCount++
			continue
		}
	}

	return failedCount == 0, nil
}

// MigrateService moves a service from one agent to another.
// If targetHost is empty, the least-loaded agent is selected.
func (s *ServiceScheduler) MigrateService(ctx context.Context, serviceID strfmt.UUID, currentHost, targetHost string) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// Get service details
		var provider string
		var az *string
		sql, args := db.Select("provider", "availability_zone").
			From("service").
			Where("id = ?", serviceID).
			Suffix("FOR UPDATE").
			MustSql()

		if err := tx.QueryRow(ctx, sql, args...).Scan(&provider, &az); err != nil {
			return err
		}

		// Determine target host
		var newHost string
		var err error
		if targetHost != "" {
			newHost = targetHost
			// Validate target host exists and is healthy
			sql, args = db.Select("1").
				From("agents").
				Where("host = ?", targetHost).
				Where("enabled = true").
				Where("provider = ?", provider).
				Where("availability_zone = ?", az).
				Where("heartbeat_at > NOW() - INTERVAL '1 second' * ?", int(s.config.StaleTimeout.Seconds())).
				MustSql()

			var exists int
			if err = tx.QueryRow(ctx, sql, args...).Scan(&exists); err != nil {
				return err
			}
		} else {
			// Find least-loaded agent
			newHost, err = s.FindLeastLoadedAgent(ctx, provider, az, currentHost)
			if err != nil {
				return err
			}
		}

		if newHost == currentHost {
			return nil // Already on target host
		}

		log.WithFields(log.Fields{
			"service": serviceID,
			"from":    currentHost,
			"to":      newHost,
		}).Info("Migrating service")

		// Update service host
		sql, args = db.Update("service").
			Set("host", newHost).
			Set("status", models.ServiceStatusPENDINGUPDATE).
			Set("updated_at", sq.Expr("NOW()")).
			Where("id = ?", serviceID).
			MustSql()

		if _, err = tx.Exec(ctx, sql, args...); err != nil {
			return err
		}

		// Update all AVAILABLE endpoints to PENDING_UPDATE
		sql, args = db.Update("endpoint").
			Set("status", models.EndpointStatusPENDINGUPDATE).
			Set("updated_at", sq.Expr("NOW()")).
			Where("service_id = ?", serviceID).
			Where("status = ?", models.EndpointStatusAVAILABLE).
			MustSql()

		if _, err = tx.Exec(ctx, sql, args...); err != nil {
			return err
		}

		// Notify both agents
		if s.notify != nil {
			s.notify(currentHost)
			s.notify(newHost)
		}

		return nil
	})
}
