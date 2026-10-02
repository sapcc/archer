// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/agent/f5/as3"
	"github.com/sapcc/archer/v2/internal/agent/f5/bigip"
	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/internal/db"
	"github.com/sapcc/archer/v2/models"
)

// Health status constants matching the API enum
const (
	HealthStatusOnline    = "ONLINE"
	HealthStatusDegraded  = "DEGRADED"
	HealthStatusOffline   = "OFFLINE"
	HealthStatusUnchecked = "UNCHECKED"
)

// F5 SNMP ltmPoolMbrStatusAvailState values
const (
	f5AvailStateNone   = iota // 0 = error
	f5AvailStateGreen         // 1 = available
	f5AvailStateYellow        // 2 = degraded
	f5AvailStateRed           // 3 = unavailable
	f5AvailStateBlue          // 4 = unknown
)

// healthStatusPriority defines the "worst wins" priority (higher = worse)
var healthStatusPriority = map[string]int{
	HealthStatusOnline:    0,
	HealthStatusUnchecked: 1,
	HealthStatusDegraded:  2,
	HealthStatusOffline:   3,
}

// ComputePoolHealthStatus computes the aggregate health status from pool member stats.
// Returns ONLINE if all members are up, OFFLINE if all are down, DEGRADED if mixed,
// and UNCHECKED if no members or status unknown.
func ComputePoolHealthStatus(stats *bigip.PoolMemberStatsResponse) string {
	if stats == nil || len(stats.Entries) == 0 {
		return HealthStatusUnchecked
	}

	upCount := 0
	downCount := 0
	totalCount := 0

	for _, entry := range stats.Entries {
		totalCount++
		status := entry.NestedStats.Entries.MonitorStatus.Description
		switch status {
		case "up":
			upCount++
		case "down":
			downCount++
		}
	}

	if totalCount == 0 {
		return HealthStatusUnchecked
	}

	if upCount == totalCount {
		return HealthStatusOnline
	}
	if downCount == totalCount {
		return HealthStatusOffline
	}
	if upCount > 0 && downCount > 0 {
		return HealthStatusDegraded
	}
	return HealthStatusUnchecked
}

// ComputeServiceHealth aggregates health across all port statuses using "worst wins" strategy.
// Priority: OFFLINE > DEGRADED > UNCHECKED > ONLINE
func ComputeServiceHealth(portStatuses []string) string {
	if len(portStatuses) == 0 {
		return HealthStatusUnchecked
	}

	worst := HealthStatusOnline
	worstPriority := healthStatusPriority[worst]

	for _, status := range portStatuses {
		priority, ok := healthStatusPriority[status]
		if !ok {
			priority = healthStatusPriority[HealthStatusUnchecked]
		}
		if priority > worstPriority {
			worst = status
			worstPriority = priority
		}
	}

	return worst
}

// f5AvailStateToHealthStatus converts F5 SNMP ltmPoolMbrStatusAvailState to health status.
func f5AvailStateToHealthStatus(state int) string {
	switch state {
	case f5AvailStateGreen:
		return HealthStatusOnline
	case f5AvailStateYellow:
		return HealthStatusDegraded
	case f5AvailStateRed:
		return HealthStatusOffline
	case f5AvailStateNone, f5AvailStateBlue:
		return HealthStatusUnchecked
	default:
		return HealthStatusUnchecked
	}
}

// UpdateServiceHealthStatus updates the health_status column for a service in the database.
func (a *Agent) UpdateServiceHealthStatus(ctx context.Context, serviceID strfmt.UUID, status string) error {
	sql, args := db.Update("service").
		Set("health_status", status).
		Where("id = ?", serviceID).
		MustSql()
	_, err := a.pool.Exec(ctx, sql, args...)
	return err
}

// HealthScrapeLoop scrapes health status for all available services each interval.
// With Prometheus configured, all services are updated in a single bulk query.
func (a *Agent) HealthScrapeLoop() error {
	ctx := context.Background()

	sql, args := db.Select("id", "ports").
		From("service").
		Where("host = ?", config.Global.Default.Host).
		Where("provider = ?", models.ServiceProviderTenant).
		Where("status = ?", models.ServiceStatusAVAILABLE).
		MustSql()

	var services []*models.Service
	err := pgxscan.Select(ctx, a.pool, &services, sql, args...)
	if err != nil {
		return fmt.Errorf("HealthScrapeLoop: failed to fetch services: %w", err)
	}

	if len(services) == 0 {
		log.Debug("HealthScrapeLoop: no services to scrape")
		return nil
	}

	if config.Global.Agent.HealthScrapePrometheus == "" {
		return a.scrapeAllServicesDirect(ctx, services)
	}
	if err := a.scrapeAllServicesPrometheus(ctx, services); err != nil {
		log.WithError(err).Warning("HealthScrapeLoop: Prometheus scrape failed, falling back to direct device scraping")
		return a.scrapeAllServicesDirect(ctx, services)
	}
	return nil
}

// poolAvailStateToHealthStatus maps F5 pool availability state strings to health status.
func poolAvailStateToHealthStatus(state string) string {
	switch state {
	case "available":
		return HealthStatusOnline
	case "degraded":
		return HealthStatusDegraded
	case "offline":
		return HealthStatusOffline
	default:
		return HealthStatusUnchecked
	}
}

// scrapeAllServicesDirect fetches all pool stats in one F5 API call and updates all services.
func (a *Agent) scrapeAllServicesDirect(ctx context.Context, services []*models.Service) error {
	device, ok := a.getActive().(*bigip.BigIP)
	if !ok {
		return fmt.Errorf("scrapeAllServicesDirect: active device is not a BigIP")
	}

	allStats, err := device.GetAllPoolStats()
	if err != nil {
		return err
	}

	// Build pool path → availability state from the bulk response.
	// Response keys are full URLs: https://localhost/mgmt/tm/ltm/pool/~Common~Shared~pool-.../stats
	poolState := make(map[string]string, len(allStats.Entries))
	for key, entry := range allStats.Entries {
		// Extract the pool path between "pool/" and "/stats"
		start := len("https://localhost/mgmt/tm/ltm/pool/")
		end := len(key) - len("/stats")
		if end <= start {
			continue
		}
		poolPath := key[start:end]
		poolState[poolPath] = entry.NestedStats.Entries.AvailabilityState.Description
	}

	for _, svc := range services {
		var portStatuses []string
		for _, port := range svc.Ports {
			poolName := as3.GetServicePoolName(svc.ID, port)
			poolPath := fmt.Sprintf("~Common~Shared~%s", poolName)
			state, ok := poolState[poolPath]
			if !ok {
				portStatuses = append(portStatuses, HealthStatusUnchecked)
				continue
			}
			portStatuses = append(portStatuses, poolAvailStateToHealthStatus(state))
		}

		health := ComputeServiceHealth(portStatuses)
		log.WithFields(log.Fields{
			"service_id": svc.ID,
			"health":     health,
		}).Debug("HealthScrapeLoop: service health (direct)")
		if err := a.UpdateServiceHealthStatus(ctx, svc.ID, health); err != nil {
			log.WithFields(log.Fields{
				"service_id": svc.ID,
				"health":     health,
			}).WithError(err).Error("Failed to update service health status")
		}
	}

	log.WithField("service_count", len(services)).Debug("HealthScrapeLoop: bulk direct scrape complete")
	return nil
}
