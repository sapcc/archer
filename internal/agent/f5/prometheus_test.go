// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/archer/v2/internal/config"
)

// prometheusHandler returns an httptest.Server that serves a fixed Prometheus API response.
func prometheusHandler(t *testing.T, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(body))
	}))
}

// failoverResponse builds a Prometheus query response for snmp_f5_sysCmFailoverStatusStatus
// mirroring the real qa-de-1 payload shape (value is always "1"; state is in the label).
func failoverResponse(activeDevicename, standbyDevicename string) PrometheusQueryResponse {
	makeResult := func(devicename, state string) struct {
		Metric map[string]string `json:"metric"`
		Value  []any             `json:"value"`
	} {
		return struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"`
		}{
			Metric: map[string]string{
				"__name__":                          "snmp_f5_sysCmFailoverStatusStatus",
				"availability_zone":                 "qa-de-1",
				"cluster":                           "qa-de-1-lb017-cluster-01",
				"cluster_type":                      "cc-f5-vcmp",
				"device_type":                       "f5-vcmp",
				"devicename":                        devicename,
				"instance":                          "10.46.100.150",
				"job":                               "scrapeConfig/infra-monitoring/snmp-exporter-f5archer",
				"manufacturer":                      "f5",
				"model":                             "F5-VCMP",
				"module":                            "f5archer",
				"platform":                          "f5-tmos",
				"role":                              "loadbalancer",
				"status":                            "active", // netbox label, NOT the failover state
				"snmp_f5_sysCmFailoverStatusStatus": state,
			},
			Value: []any{1790945235.891, "1"},
		}
	}

	return PrometheusQueryResponse{
		Status: "success",
		Data: struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []any             `json:"value"`
			} `json:"result"`
		}{
			ResultType: "vector",
			Result: []struct {
				Metric map[string]string `json:"metric"`
				Value  []any             `json:"value"`
			}{
				makeResult(activeDevicename, "ACTIVE"),
				makeResult(standbyDevicename, "STANDBY"),
			},
		},
	}
}

// poolHealthResponse builds a raw Prometheus response payload for snmp_f5_ltmPoolMbrStatusAvailState
// mirroring the real qa-de-1 payload shape. Returns an any so prometheusHandler can JSON-encode it
// without hitting Go's anonymous-struct assignability restrictions.
func poolHealthResponse(poolName string, members []struct{ node, state string }) any {
	type entry struct {
		Metric map[string]string `json:"metric"`
		Value  []any             `json:"value"`
	}
	type data struct {
		ResultType string  `json:"resultType"`
		Result     []entry `json:"result"`
	}
	type response struct {
		Status string `json:"status"`
		Data   data   `json:"data"`
	}

	var entries []entry
	for _, m := range members {
		entries = append(entries, entry{
			Metric: map[string]string{
				"__name__":                 "snmp_f5_ltmPoolMbrStatusAvailState",
				"availability_zone":        "qa-de-1",
				"cluster":                  "qa-de-1-lb011-cluster-01",
				"device":                   "lb999z-test",
				"device_type":              "f5-vcmp",
				"devicename":               "qa-de-1-lb999z-test",
				"instance":                 "10.46.100.233",
				"job":                      "scrapeConfig/infra-monitoring/snmp-exporter-f5archer",
				"ltmPoolMbrStatusNodeName": m.node,
				"ltmPoolMbrStatusPoolName": poolName,
				"ltmPoolMbrStatusPort":     "80",
				"manufacturer":             "f5",
				"model":                    "F5-VCMP",
				"module":                   "f5archer",
				"platform":                 "f5-tmos",
				"role":                     "loadbalancer",
				"status":                   "active",
			},
			Value: []any{1790945239.292, m.state},
		})
	}
	return response{Status: "success", Data: data{ResultType: "vector", Result: entries}}
}

// --- syncActiveFromPrometheus tests ---

func TestSyncActiveFromPrometheus_ActiveDeviceMatchedByInstanceLabel(t *testing.T) {
	devA := NewMockF5Device(t)
	devA.EXPECT().GetHostname().Return("qa-de-1-lb017a-archer.cc.qa-de-1.cloud.sap")
	devB := NewMockF5Device(t)
	devB.EXPECT().GetHostname().Maybe().Return("qa-de-1-lb017b-archer.cc.qa-de-1.cloud.sap")

	srv := prometheusHandler(t, failoverResponse("qa-de-1-lb017a-archer", "qa-de-1-lb017b-archer"))
	defer srv.Close()

	a := &Agent{devices: []F5Device{devA, devB}}
	got := a.syncActiveFromPrometheus(context.Background(), srv.URL)

	assert.Equal(t, devA, got)
}

func TestSyncActiveFromPrometheus_ReturnsNewActiveAfterFailover(t *testing.T) {
	devA := NewMockF5Device(t)
	devA.EXPECT().GetHostname().Maybe().Return("qa-de-1-lb017a-archer.cc.qa-de-1.cloud.sap")
	devB := NewMockF5Device(t)
	devB.EXPECT().GetHostname().Return("qa-de-1-lb017b-archer.cc.qa-de-1.cloud.sap")

	// B is now active after failover
	srv := prometheusHandler(t, failoverResponse("qa-de-1-lb017b-archer", "qa-de-1-lb017a-archer"))
	defer srv.Close()

	a := &Agent{devices: []F5Device{devA, devB}, active: devA}
	got := a.syncActiveFromPrometheus(context.Background(), srv.URL)

	assert.Equal(t, devB, got)
}

func TestSyncActiveFromPrometheus_ReturnsNilWhenActiveInstanceNotInDeviceList(t *testing.T) {
	devA := NewMockF5Device(t)
	devA.EXPECT().GetHostname().Return("qa-de-1-lb017a-archer.cc.qa-de-1.cloud.sap")

	// Prometheus returns a devicename not matching any configured device
	srv := prometheusHandler(t, failoverResponse("qa-de-1-lb099a-archer", "qa-de-1-lb099b-archer"))
	defer srv.Close()

	a := &Agent{devices: []F5Device{devA}}
	got := a.syncActiveFromPrometheus(context.Background(), srv.URL)

	assert.Nil(t, got)
}

func TestSyncActiveFromPrometheus_ReturnsNilOnEmptyResult(t *testing.T) {
	srv := prometheusHandler(t, PrometheusQueryResponse{
		Status: "success",
		Data: struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []any             `json:"value"`
			} `json:"result"`
		}{ResultType: "vector"},
	})
	defer srv.Close()

	a := &Agent{}
	got := a.syncActiveFromPrometheus(context.Background(), srv.URL)

	assert.Nil(t, got)
}

func TestSyncActiveFromPrometheus_ReturnsNilWhenPrometheusUnreachable(t *testing.T) {
	// Use a closed server to simulate Prometheus being unavailable.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	a := &Agent{}
	got := a.syncActiveFromPrometheus(context.Background(), srv.URL)

	assert.Nil(t, got)
}

// --- syncActiveFromDevices tests ---

func TestSyncActiveFromDevices_ReturnsFirstActiveDevice(t *testing.T) {
	devA := NewMockF5Device(t)
	devA.EXPECT().GetFailoverState().Return("standby")
	devB := NewMockF5Device(t)
	devB.EXPECT().GetFailoverState().Return("active")

	a := &Agent{devices: []F5Device{devA, devB}}
	got := a.syncActiveFromDevices()

	assert.Equal(t, devB, got)
}

func TestSyncActiveFromDevices_ReturnsNilWhenAllDevicesStandby(t *testing.T) {
	devA := NewMockF5Device(t)
	devA.EXPECT().GetFailoverState().Return("standby")
	devB := NewMockF5Device(t)
	devB.EXPECT().GetFailoverState().Return("standby")

	a := &Agent{devices: []F5Device{devA, devB}}
	got := a.syncActiveFromDevices()

	assert.Nil(t, got)
}

// --- SyncActiveDevice integration tests (Prometheus path with device poll fallback) ---

func TestSyncActiveDevice_FallsBackToDevicePollWhenPrometheusUnavailable(t *testing.T) {
	devA := NewMockF5Device(t)
	devA.EXPECT().GetHostname().Maybe().Return("10.246.245.214")
	devA.EXPECT().GetFailoverState().Return("active")
	devB := NewMockF5Device(t)
	devB.EXPECT().GetFailoverState().Maybe().Return("standby")

	// Closed server simulates Prometheus being down.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	config.Global.Agent.HealthScrapePrometheus = srv.URL
	t.Cleanup(func() { config.Global.Agent.HealthScrapePrometheus = "" })

	a := &Agent{devices: []F5Device{devA, devB}, active: devA}
	a.SyncActiveDevice()

	// active must remain devA (found via direct device poll)
	assert.Equal(t, devA, a.active)
}

func TestSyncActiveDevice_FallsBackToDevicePollWhenPrometheusUnconfigured(t *testing.T) {
	devA := NewMockF5Device(t)
	devA.EXPECT().GetFailoverState().Return("active")
	devB := NewMockF5Device(t)
	devB.EXPECT().GetFailoverState().Maybe().Return("standby")

	config.Global.Agent.HealthScrapePrometheus = ""

	a := &Agent{devices: []F5Device{devA, devB}, active: devA}
	a.SyncActiveDevice()

	assert.Equal(t, devA, a.active)
}

func TestSyncActiveDevice_UpdatesActiveDeviceOnFailoverDetectedViaPrometheus(t *testing.T) {
	devA := NewMockF5Device(t)
	devA.EXPECT().GetHostname().Return("10.246.245.214").Maybe()
	devB := NewMockF5Device(t)
	devB.EXPECT().GetHostname().Return("10.246.245.215")

	srv := prometheusHandler(t, failoverResponse("10.246.245.215", "10.246.245.214"))
	defer srv.Close()

	config.Global.Agent.HealthScrapePrometheus = srv.URL
	t.Cleanup(func() { config.Global.Agent.HealthScrapePrometheus = "" })

	a := &Agent{devices: []F5Device{devA, devB}, active: devA}
	a.SyncActiveDevice()

	assert.Equal(t, devB, a.active)
}

// --- queryPrometheusPoolHealth tests ---

func TestQueryPrometheusPoolHealth_AllMembersOnline(t *testing.T) {
	poolName := "/Common/Shared/pool-3796d782-7e03-4078-8473-1a07711a14de-80"
	srv := prometheusHandler(t, poolHealthResponse(poolName, []struct{ node, state string }{
		{"/Common/Shared/1.1.1.1%3019", "1"},
		{"/Common/Shared/1.1.1.2%3019", "1"},
	}))
	defer srv.Close()

	config.Global.Agent.HealthScrapePrometheus = srv.URL
	t.Cleanup(func() { config.Global.Agent.HealthScrapePrometheus = "" })

	dev := NewMockF5Device(t)
	dev.EXPECT().GetHostname().Return("qa-de-1-lb999z-test.cc.qa-de-1.example.test")
	a := &Agent{active: dev}
	got, err := a.queryPrometheusPoolHealth(context.Background(), poolName)

	require.NoError(t, err)
	assert.Equal(t, HealthStatusOnline, got)
}

func TestQueryPrometheusPoolHealth_OneMemberDownYieldsOffline(t *testing.T) {
	poolName := "/Common/Shared/pool-0cf14ffc-e6cf-4fbb-8f79-84b15229ef02-80"
	srv := prometheusHandler(t, poolHealthResponse(poolName, []struct{ node, state string }{
		{"/Common/Shared/10.180.0.215%3317", "1"}, // green
		{"/Common/Shared/10.180.0.216%3317", "3"}, // red = OFFLINE
	}))
	defer srv.Close()

	config.Global.Agent.HealthScrapePrometheus = srv.URL
	t.Cleanup(func() { config.Global.Agent.HealthScrapePrometheus = "" })

	dev := NewMockF5Device(t)
	dev.EXPECT().GetHostname().Return("qa-de-1-lb999z-test.cc.qa-de-1.example.test")
	a := &Agent{active: dev}
	got, err := a.queryPrometheusPoolHealth(context.Background(), poolName)

	require.NoError(t, err)
	assert.Equal(t, HealthStatusOffline, got)
}

func TestQueryPrometheusPoolHealth_AllMembersDownYieldsOffline(t *testing.T) {
	poolName := "/Common/Shared/pool-0e650048-86ff-45e6-ae4b-c9148e3f1c49-80"
	srv := prometheusHandler(t, poolHealthResponse(poolName, []struct{ node, state string }{
		{"/Common/Shared/10.180.0.215%3317", "3"},
		{"/Common/Shared/10.180.0.216%3317", "3"},
	}))
	defer srv.Close()

	config.Global.Agent.HealthScrapePrometheus = srv.URL
	t.Cleanup(func() { config.Global.Agent.HealthScrapePrometheus = "" })

	dev := NewMockF5Device(t)
	dev.EXPECT().GetHostname().Return("qa-de-1-lb999z-test.cc.qa-de-1.example.test")
	a := &Agent{active: dev}
	got, err := a.queryPrometheusPoolHealth(context.Background(), poolName)

	require.NoError(t, err)
	assert.Equal(t, HealthStatusOffline, got)
}

func TestQueryPrometheusPoolHealth_EmptyPoolYieldsUnchecked(t *testing.T) {
	poolName := "/Common/Shared/pool-00000000-0000-0000-0000-000000000000-80"
	srv := prometheusHandler(t, poolHealthResponse(poolName, nil))
	defer srv.Close()

	config.Global.Agent.HealthScrapePrometheus = srv.URL
	t.Cleanup(func() { config.Global.Agent.HealthScrapePrometheus = "" })

	dev := NewMockF5Device(t)
	dev.EXPECT().GetHostname().Return("qa-de-1-lb999z-test.cc.qa-de-1.example.test")
	a := &Agent{active: dev}
	got, err := a.queryPrometheusPoolHealth(context.Background(), poolName)

	require.NoError(t, err)
	assert.Equal(t, HealthStatusUnchecked, got)
}

// --- computeHealthFromPrometheusResult unit tests ---

func TestComputeHealthFromPrometheusResult(t *testing.T) {
	tests := []struct {
		name     string
		resp     PrometheusQueryResponse
		expected string
	}{
		{
			name:     "empty results returns UNCHECKED",
			resp:     makePromResponse(nil),
			expected: HealthStatusUnchecked,
		},
		{
			name:     "single green member returns ONLINE",
			resp:     makePromResponse([]promMember{{"1"}}),
			expected: HealthStatusOnline,
		},
		{
			name:     "single red member returns OFFLINE",
			resp:     makePromResponse([]promMember{{"3"}}),
			expected: HealthStatusOffline,
		},
		{
			name:     "green and red members yields OFFLINE (worst wins)",
			resp:     makePromResponse([]promMember{{"1"}, {"3"}}),
			expected: HealthStatusOffline,
		},
		{
			name:     "all green members yields ONLINE",
			resp:     makePromResponse([]promMember{{"1"}, {"1"}}),
			expected: HealthStatusOnline,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, computeHealthFromPrometheusResult(tt.resp))
		})
	}
}

type promMember struct{ value string }

func makePromResponse(members []promMember) PrometheusQueryResponse {
	var results []struct {
		Metric map[string]string `json:"metric"`
		Value  []any             `json:"value"`
	}
	for _, m := range members {
		results = append(results, struct {
			Metric map[string]string `json:"metric"`
			Value  []any             `json:"value"`
		}{
			Metric: map[string]string{},
			Value:  []any{1234567890.0, m.value},
		})
	}
	return PrometheusQueryResponse{
		Status: "success",
		Data: struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []any             `json:"value"`
			} `json:"result"`
		}{ResultType: "vector", Result: results},
	}
}
