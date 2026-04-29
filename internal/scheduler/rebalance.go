// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/db"
)

// AgentLoad represents an agent and its current service count.
type AgentLoad struct {
	Host         string
	ServiceCount int
}

// ShouldRebalance checks if the imbalance between agents exceeds the threshold.
// Returns true if rebalancing is needed.
func (s *ServiceScheduler) ShouldRebalance(ctx context.Context, provider string, az *string) (bool, error) {
	agents, err := s.getAgentLoads(ctx, provider, az)
	if err != nil {
		return false, err
	}

	if len(agents) < 2 {
		return false, nil // Nothing to rebalance with < 2 agents
	}

	// Find min and max service counts
	minCount := agents[0].ServiceCount
	maxCount := agents[0].ServiceCount
	for _, agent := range agents {
		if agent.ServiceCount < minCount {
			minCount = agent.ServiceCount
		}
		if agent.ServiceCount > maxCount {
			maxCount = agent.ServiceCount
		}
	}

	if maxCount == 0 {
		return false, nil // No services to rebalance
	}

	// Calculate imbalance ratio: (max - min) / max
	imbalance := float64(maxCount-minCount) / float64(maxCount)

	log.WithFields(log.Fields{
		"provider":  provider,
		"az":        az,
		"min":       minCount,
		"max":       maxCount,
		"imbalance": imbalance,
		"threshold": s.config.RebalanceThreshold,
	}).Debug("Checking rebalance threshold")

	return imbalance > s.config.RebalanceThreshold, nil
}

// RebalanceServices redistributes services across agents when imbalance exceeds threshold.
func (s *ServiceScheduler) RebalanceServices(ctx context.Context, provider string, az *string) error {
	shouldRebalance, err := s.ShouldRebalance(ctx, provider, az)
	if err != nil {
		return err
	}

	if !shouldRebalance {
		return nil
	}

	log.WithFields(log.Fields{
		"provider": provider,
		"az":       az,
	}).Info("Rebalancing services")

	agents, err := s.getAgentLoads(ctx, provider, az)
	if err != nil {
		return err
	}

	if len(agents) < 2 {
		return nil
	}

	// Calculate target (average)
	totalServices := 0
	for _, agent := range agents {
		totalServices += agent.ServiceCount
	}
	target := totalServices / len(agents)

	// Find overloaded and underloaded agents
	var overloaded, underloaded []AgentLoad
	for _, agent := range agents {
		if agent.ServiceCount > target+1 {
			overloaded = append(overloaded, agent)
		} else if agent.ServiceCount < target {
			underloaded = append(underloaded, agent)
		}
	}

	migrations := 0
	for _, over := range overloaded {
		if migrations >= s.config.RebalanceMaxMigrations {
			break
		}

		excess := over.ServiceCount - target
		for i := 0; i < excess && migrations < s.config.RebalanceMaxMigrations; i++ {
			if len(underloaded) == 0 {
				break
			}

			// Pick the most underloaded agent
			underIdx := 0
			for j, u := range underloaded {
				if u.ServiceCount < underloaded[underIdx].ServiceCount {
					underIdx = j
				}
			}

			// Find a service to migrate
			serviceID, err := s.getRandomServiceFromHost(ctx, over.Host, provider)
			if err != nil {
				log.WithError(err).WithField("host", over.Host).Warning("Failed to get service for migration")
				break
			}

			if err := s.MigrateService(ctx, serviceID, over.Host, underloaded[underIdx].Host); err != nil {
				log.WithError(err).Warning("Failed to migrate service during rebalance")
				continue
			}

			// Update counts
			underloaded[underIdx].ServiceCount++
			if underloaded[underIdx].ServiceCount >= target {
				// Remove from underloaded list
				underloaded = append(underloaded[:underIdx], underloaded[underIdx+1:]...)
			}

			migrations++
		}
	}

	log.WithFields(log.Fields{
		"provider":   provider,
		"az":         az,
		"migrations": migrations,
	}).Info("Rebalance complete")

	return nil
}

func (s *ServiceScheduler) getAgentLoads(ctx context.Context, provider string, az *string) ([]AgentLoad, error) {
	sql, args := db.Select("agents.host", "COUNT(service.id) AS service_count").
		From("agents").
		LeftJoin("service ON service.host = agents.host AND service.provider = agents.provider").
		Where("agents.enabled = true").
		Where("agents.provider = ?", provider).
		Where("agents.availability_zone = ?", az).
		Where("agents.heartbeat_at > NOW() - INTERVAL '1 second' * ?", int(s.config.StaleTimeout.Seconds())).
		GroupBy("agents.host").
		OrderBy("service_count DESC").
		MustSql()

	var agents []AgentLoad
	if err := pgxscan.Select(ctx, s.pool, &agents, sql, args...); err != nil {
		return nil, err
	}

	return agents, nil
}

func (s *ServiceScheduler) getRandomServiceFromHost(ctx context.Context, host, provider string) (strfmt.UUID, error) {
	sql, args := db.Select("id").
		From("service").
		Where("host = ?", host).
		Where("provider = ?", provider).
		Where("status = 'AVAILABLE'").
		Limit(1).
		MustSql()

	var serviceID strfmt.UUID
	if err := s.pool.QueryRow(ctx, sql, args...).Scan(&serviceID); err != nil {
		return "", err
	}

	return serviceID, nil
}
