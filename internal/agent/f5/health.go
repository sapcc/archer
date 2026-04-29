// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-co-op/gocron/v2"
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

// PrometheusQueryResponse represents the response from Prometheus /api/v1/query
type PrometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"` // [timestamp, value]
		} `json:"result"`
	} `json:"data"`
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
		// monitorStatus can be: "up", "down", "unchecked", "checking", etc.
		status := entry.NestedStats.Entries.MonitorStatus.Description
		switch status {
		case "up":
			upCount++
		case "down":
			downCount++
			// other statuses (unchecked, checking) don't count as up or down
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
	// All members are in an unknown/checking state
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

// ScrapeServiceHealth queries the F5 device for all pools of a service and computes aggregate health.
func (a *Agent) ScrapeServiceHealth(ctx context.Context, service *models.Service) string {
	// Get the active BigIP device
	device, ok := a.active.(*bigip.BigIP)
	if !ok {
		log.WithField("service_id", service.ID).Debug("Active device is not a BigIP, cannot scrape health")
		return HealthStatusUnchecked
	}

	var portStatuses []string

	for _, port := range service.Ports {
		poolName := as3.GetServicePoolName(service.ID, port)
		// Format: ~Common~Shared~pool-{id}-{port}
		poolPath := fmt.Sprintf("~Common~Shared~%s", poolName)

		stats, err := device.GetPoolMemberStats(poolPath)
		if err != nil {
			log.WithFields(log.Fields{
				"service_id": service.ID,
				"port":       port,
				"pool":       poolName,
			}).WithError(err).Debug("Failed to get pool member stats")
			portStatuses = append(portStatuses, HealthStatusUnchecked)
			continue
		}

		poolHealth := ComputePoolHealthStatus(stats)
		portStatuses = append(portStatuses, poolHealth)

		log.WithFields(log.Fields{
			"service_id": service.ID,
			"port":       port,
			"pool":       poolName,
			"health":     poolHealth,
		}).Debug("Scraped pool health")
	}

	return ComputeServiceHealth(portStatuses)
}

// ScrapeServiceHealthPrometheus queries Prometheus for pool health status using SNMP metrics.
func (a *Agent) ScrapeServiceHealthPrometheus(ctx context.Context, service *models.Service) string {
	var portStatuses []string

	for _, port := range service.Ports {
		poolName := as3.GetServicePoolName(service.ID, port)
		// Format: /Common/Shared/pool-{id}-{port}
		poolPath := fmt.Sprintf("/Common/Shared/%s", poolName)

		health, err := a.queryPrometheusPoolHealth(ctx, poolPath)
		if err != nil {
			log.WithFields(log.Fields{
				"service_id": service.ID,
				"port":       port,
				"pool":       poolName,
			}).WithError(err).Debug("Failed to query Prometheus for pool health")
			portStatuses = append(portStatuses, HealthStatusUnchecked)
			continue
		}

		portStatuses = append(portStatuses, health)

		log.WithFields(log.Fields{
			"service_id": service.ID,
			"port":       port,
			"pool":       poolName,
			"health":     health,
		}).Debug("Scraped pool health from Prometheus")
	}

	return ComputeServiceHealth(portStatuses)
}

// queryPrometheusPoolHealth queries Prometheus for the health of a specific pool.
func (a *Agent) queryPrometheusPoolHealth(ctx context.Context, poolPath string) (string, error) {
	promURL := config.Global.Agent.HealthScrapePrometheus

	// Build the PromQL query
	query := fmt.Sprintf(`snmp_f5_ltmPoolMbrStatusAvailState{ltmPoolMbrStatusPoolName="%s"}`, poolPath)

	// Build request URL
	reqURL, err := url.Parse(promURL + "/api/v1/query")
	if err != nil {
		return "", fmt.Errorf("invalid prometheus URL: %w", err)
	}

	params := url.Values{}
	params.Set("query", query)
	reqURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("prometheus query failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("prometheus returned status %d", resp.StatusCode)
	}

	var promResp PrometheusQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return "", fmt.Errorf("failed to decode prometheus response: %w", err)
	}

	if promResp.Status != "success" {
		return "", fmt.Errorf("prometheus query status: %s", promResp.Status)
	}

	return computeHealthFromPrometheusResult(promResp), nil
}

// computeHealthFromPrometheusResult computes health status from Prometheus query results.
// It filters for active devices and uses the "worst wins" strategy across all members.
func computeHealthFromPrometheusResult(resp PrometheusQueryResponse) string {
	if len(resp.Data.Result) == 0 {
		return HealthStatusUnchecked
	}

	var memberStatuses []string

	for _, result := range resp.Data.Result {
		// Only consider results from active devices
		if status, ok := result.Metric["status"]; !ok || status != "active" {
			continue
		}

		// Parse the value (second element in the value array)
		if len(result.Value) < 2 {
			continue
		}

		valueStr, ok := result.Value[1].(string)
		if !ok {
			continue
		}

		var value int
		if _, err := fmt.Sscanf(valueStr, "%d", &value); err != nil {
			continue
		}

		memberStatuses = append(memberStatuses, f5AvailStateToHealthStatus(value))
	}

	if len(memberStatuses) == 0 {
		return HealthStatusUnchecked
	}

	return ComputeServiceHealth(memberStatuses)
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

// HealthScrapeLoop is the main loop that scrapes health status for all available services.
// It distributes individual service scrapes evenly within the scrape interval.
func (a *Agent) HealthScrapeLoop() error {
	ctx := context.Background()

	// Get all AVAILABLE services for this host
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

	// Calculate time offset between each service scrape to distribute load
	interval := config.Global.Agent.HealthScrapeInterval
	offset := interval / time.Duration(len(services))

	log.WithFields(log.Fields{
		"service_count": len(services),
		"interval":      interval,
		"offset":        offset,
	}).Debug("HealthScrapeLoop: scheduling service health scrapes")

	// Schedule individual scrapes with staggered start times
	// Use a base time with small buffer to ensure all scheduled times are in the future
	baseTime := time.Now().Add(100 * time.Millisecond)
	for i, service := range services {
		startDelay := offset * time.Duration(i)
		svc := service // capture for closure

		if _, err := a.scheduler.NewJob(
			gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(baseTime.Add(startDelay))),
			gocron.NewTask(a.scrapeAndUpdateServiceHealth, svc),
			gocron.WithName(fmt.Sprintf("HealthScrape-%s", svc.ID)),
		); err != nil {
			log.WithField("service_id", svc.ID).WithError(err).Error("Failed to schedule health scrape job")
		}
	}

	return nil
}

// scrapeAndUpdateServiceHealth scrapes health for a single service and updates the database.
func (a *Agent) scrapeAndUpdateServiceHealth(ctx context.Context, service *models.Service) {
	var health string

	// Use Prometheus if configured, otherwise use direct F5 API
	if config.Global.Agent.HealthScrapePrometheus != "" {
		health = a.ScrapeServiceHealthPrometheus(ctx, service)
	} else {
		health = a.ScrapeServiceHealth(ctx, service)
	}

	if err := a.UpdateServiceHealthStatus(ctx, service.ID, health); err != nil {
		log.WithFields(log.Fields{
			"service_id": service.ID,
			"health":     health,
		}).WithError(err).Error("Failed to update service health status")
		return
	}

	log.WithFields(log.Fields{
		"service_id": service.ID,
		"health":     health,
	}).Debug("Updated service health status")
}
