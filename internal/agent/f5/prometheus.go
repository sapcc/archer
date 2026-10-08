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
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/agent/f5/as3"
	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/models"
)

// PrometheusQueryResponse represents the response from Prometheus /api/v1/query.
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

// queryPrometheus executes a PromQL instant query against promURL and returns the parsed response.
func queryPrometheus(ctx context.Context, promURL, query string) (*PrometheusQueryResponse, error) {
	reqURL, err := url.Parse(promURL + "/api/v1/query")
	if err != nil {
		return nil, fmt.Errorf("invalid prometheus URL: %w", err)
	}

	params := url.Values{}
	params.Set("query", query)
	reqURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus returned status %d", resp.StatusCode)
	}

	var promResp PrometheusQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return nil, fmt.Errorf("failed to decode prometheus response: %w", err)
	}

	if promResp.Status != "success" {
		return nil, fmt.Errorf("prometheus query status: %s", promResp.Status)
	}

	return &promResp, nil
}

// getActive returns the current active F5 device.
func (a *Agent) getActive() F5Device {
	a.activeMu.RLock()
	defer a.activeMu.RUnlock()
	return a.active
}

// SyncActiveDevice re-detects which device is active and updates a.active if it changed.
// When Prometheus is configured it queries snmp_f5_sysCmFailoverStatusStatus; if that fails
// or is unconfigured, it falls back to polling GetFailoverState() directly on each device.
func (a *Agent) SyncActiveDevice() {
	ctx := context.Background()

	var newActive F5Device
	if config.Global.Agent.HealthScrapePrometheus != "" {
		newActive = a.syncActiveFromPrometheus(ctx, config.Global.Agent.HealthScrapePrometheus)
	}
	if newActive == nil {
		newActive = a.syncActiveFromDevices()
	}
	if newActive == nil {
		log.Warning("SyncActiveDevice: no active device found, keeping current active device")
		return
	}

	a.activeMu.Lock()
	if a.active != newActive {
		log.WithFields(log.Fields{
			"previous": a.active.GetHostname(),
			"new":      newActive.GetHostname(),
		}).Warning("F5 failover detected, switching active device")
		a.active = newActive
	}
	a.activeMu.Unlock()
}

// syncActiveFromPrometheus queries snmp_f5_sysCmFailoverStatusStatus (f5archer job) and returns the matching device.
func (a *Agent) syncActiveFromPrometheus(ctx context.Context, promURL string) F5Device {
	resp, err := queryPrometheus(ctx, promURL, "snmp_f5_sysCmFailoverStatusStatus")
	if err != nil {
		log.WithError(err).Warning("SyncActiveDevice: Prometheus query failed, falling back to direct poll")
		return nil
	}

	for _, result := range resp.Data.Result {
		if result.Metric["snmp_f5_sysCmFailoverStatusStatus"] != "ACTIVE" {
			continue
		}
		for _, dev := range a.devices {
			if strings.HasPrefix(dev.GetHostname(), result.Metric["devicename"]) {
				return dev
			}
		}
	}

	log.Warning("SyncActiveDevice: no ACTIVE device from Prometheus matched configured devices, falling back to direct poll")
	return nil
}

// syncActiveFromDevices polls each device directly for its failover state.
func (a *Agent) syncActiveFromDevices() F5Device {
	for _, dev := range a.devices {
		if dev.GetFailoverState() == "active" {
			return dev
		}
	}
	return nil
}

// activeDeviceName returns the short hostname (Prometheus devicename label) for a device.
// F5 device URLs use FQDNs (e.g. "qa-de-1-lb017a-archer.cc.qa-de-1.cloud.sap") but the
// Prometheus devicename label only contains the first segment ("qa-de-1-lb017a-archer").
func activeDeviceName(dev F5Device) string {
	hostname := dev.GetHostname()
	short, _, _ := strings.Cut(hostname, ".")
	return short
}

// queryPrometheusPoolHealth queries Prometheus for the health of a specific pool on the active device.
func (a *Agent) queryPrometheusPoolHealth(ctx context.Context, poolPath string) (string, error) {
	query := fmt.Sprintf(`snmp_f5_ltmPoolMbrStatusAvailState{devicename="%s",ltmPoolMbrStatusPoolName="%s"}`,
		activeDeviceName(a.getActive()), poolPath)
	promResp, err := queryPrometheus(ctx, config.Global.Agent.HealthScrapePrometheus, query)
	if err != nil {
		return "", err
	}
	return computeHealthFromPrometheusResult(*promResp), nil
}

// scrapeAllServicesPrometheus fetches all pool member health in one Prometheus query and updates all services.
func (a *Agent) scrapeAllServicesPrometheus(ctx context.Context, services []*models.Service) error {
	devName := activeDeviceName(a.getActive())
	query := fmt.Sprintf(
		`snmp_f5_ltmPoolMbrStatusAvailState{devicename="%s",ltmPoolMbrStatusPoolName=~"/Common/Shared/pool-.*"}`,
		devName,
	)
	promResp, err := queryPrometheus(ctx, config.Global.Agent.HealthScrapePrometheus, query)
	if err != nil {
		return fmt.Errorf("scrapeAllServicesPrometheus: %w", err)
	}

	// Build pool path → member avail states from the single response.
	poolStates := make(map[string][]int)
	for _, result := range promResp.Data.Result {
		poolName := result.Metric["ltmPoolMbrStatusPoolName"]
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
		poolStates[poolName] = append(poolStates[poolName], value)
	}

	var missingFromPrometheus []*models.Service
	for _, svc := range services {
		var portStatuses []string
		allMissing := true
		for _, port := range svc.Ports {
			poolName := as3.GetServicePoolName(svc.ID, port)
			poolPath := fmt.Sprintf("/Common/Shared/%s", poolName)
			members, ok := poolStates[poolPath]
			if !ok {
				portStatuses = append(portStatuses, HealthStatusUnchecked)
				continue
			}
			allMissing = false
			var memberStatuses []string
			for _, state := range members {
				memberStatuses = append(memberStatuses, f5AvailStateToHealthStatus(state))
			}
			portStatuses = append(portStatuses, ComputeServiceHealth(memberStatuses))
		}

		if allMissing {
			log.WithField("service_id", svc.ID).Warning("HealthScrapeLoop: no Prometheus data for service, falling back to direct scrape")
			missingFromPrometheus = append(missingFromPrometheus, svc)
			continue
		}

		health := ComputeServiceHealth(portStatuses)
		log.WithFields(log.Fields{
			"service_id": svc.ID,
			"health":     health,
		}).Debug("HealthScrapeLoop: service health (prometheus)")
		if err := a.UpdateServiceHealthStatus(ctx, svc.ID, health); err != nil {
			log.WithFields(log.Fields{
				"service_id": svc.ID,
				"health":     health,
			}).WithError(err).Error("Failed to update service health status")
		}
	}

	if len(missingFromPrometheus) > 0 {
		if err := a.scrapeAllServicesDirect(ctx, missingFromPrometheus); err != nil {
			return fmt.Errorf("scrapeAllServicesPrometheus: direct fallback failed: %w", err)
		}
	}

	log.WithField("service_count", len(services)).Debug("HealthScrapeLoop: bulk Prometheus scrape complete")
	return nil
}

// computeHealthFromPrometheusResult computes health status from Prometheus pool member results
// using the "worst wins" strategy.
func computeHealthFromPrometheusResult(resp PrometheusQueryResponse) string {
	if len(resp.Data.Result) == 0 {
		return HealthStatusUnchecked
	}

	var memberStatuses []string

	for _, result := range resp.Data.Result {
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
