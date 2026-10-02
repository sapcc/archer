// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/v2/internal/agent/f5/bigip"
)

func TestComputePoolHealthStatus(t *testing.T) {
	tests := []struct {
		name     string
		stats    *bigip.PoolMemberStatsResponse
		expected string
	}{
		{
			name:     "nil stats returns UNCHECKED",
			stats:    nil,
			expected: HealthStatusUnchecked,
		},
		{
			name: "empty entries returns UNCHECKED",
			stats: &bigip.PoolMemberStatsResponse{Entries: map[string]struct {
				NestedStats struct {
					Entries bigip.PoolMemberStatsEntry `json:"entries"`
				} `json:"nestedStats"`
			}{}},
			expected: HealthStatusUnchecked,
		},
		{
			name: "all members up returns ONLINE",
			stats: &bigip.PoolMemberStatsResponse{
				Entries: map[string]struct {
					NestedStats struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					} `json:"nestedStats"`
				}{
					"member1": {NestedStats: struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					}{Entries: bigip.PoolMemberStatsEntry{MonitorStatus: struct {
						Description string `json:"description"`
					}{Description: "up"}}}},
					"member2": {NestedStats: struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					}{Entries: bigip.PoolMemberStatsEntry{MonitorStatus: struct {
						Description string `json:"description"`
					}{Description: "up"}}}},
				},
			},
			expected: HealthStatusOnline,
		},
		{
			name: "all members down returns OFFLINE",
			stats: &bigip.PoolMemberStatsResponse{
				Entries: map[string]struct {
					NestedStats struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					} `json:"nestedStats"`
				}{
					"member1": {NestedStats: struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					}{Entries: bigip.PoolMemberStatsEntry{MonitorStatus: struct {
						Description string `json:"description"`
					}{Description: "down"}}}},
					"member2": {NestedStats: struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					}{Entries: bigip.PoolMemberStatsEntry{MonitorStatus: struct {
						Description string `json:"description"`
					}{Description: "down"}}}},
				},
			},
			expected: HealthStatusOffline,
		},
		{
			name: "mixed status returns DEGRADED",
			stats: &bigip.PoolMemberStatsResponse{
				Entries: map[string]struct {
					NestedStats struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					} `json:"nestedStats"`
				}{
					"member1": {NestedStats: struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					}{Entries: bigip.PoolMemberStatsEntry{MonitorStatus: struct {
						Description string `json:"description"`
					}{Description: "up"}}}},
					"member2": {NestedStats: struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					}{Entries: bigip.PoolMemberStatsEntry{MonitorStatus: struct {
						Description string `json:"description"`
					}{Description: "down"}}}},
				},
			},
			expected: HealthStatusDegraded,
		},
		{
			name: "all members unchecked returns UNCHECKED",
			stats: &bigip.PoolMemberStatsResponse{
				Entries: map[string]struct {
					NestedStats struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					} `json:"nestedStats"`
				}{
					"member1": {NestedStats: struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					}{Entries: bigip.PoolMemberStatsEntry{MonitorStatus: struct {
						Description string `json:"description"`
					}{Description: "unchecked"}}}},
				},
			},
			expected: HealthStatusUnchecked,
		},
		{
			name: "single member up returns ONLINE",
			stats: &bigip.PoolMemberStatsResponse{
				Entries: map[string]struct {
					NestedStats struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					} `json:"nestedStats"`
				}{
					"member1": {NestedStats: struct {
						Entries bigip.PoolMemberStatsEntry `json:"entries"`
					}{Entries: bigip.PoolMemberStatsEntry{MonitorStatus: struct {
						Description string `json:"description"`
					}{Description: "up"}}}},
				},
			},
			expected: HealthStatusOnline,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputePoolHealthStatus(tt.stats)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComputeServiceHealth(t *testing.T) {
	tests := []struct {
		name         string
		portStatuses []string
		expected     string
	}{
		{
			name:         "empty port statuses returns UNCHECKED",
			portStatuses: []string{},
			expected:     HealthStatusUnchecked,
		},
		{
			name:         "single ONLINE returns ONLINE",
			portStatuses: []string{HealthStatusOnline},
			expected:     HealthStatusOnline,
		},
		{
			name:         "single OFFLINE returns OFFLINE",
			portStatuses: []string{HealthStatusOffline},
			expected:     HealthStatusOffline,
		},
		{
			name:         "ONLINE and OFFLINE returns OFFLINE (worst wins)",
			portStatuses: []string{HealthStatusOnline, HealthStatusOffline},
			expected:     HealthStatusOffline,
		},
		{
			name:         "ONLINE and DEGRADED returns DEGRADED (worst wins)",
			portStatuses: []string{HealthStatusOnline, HealthStatusDegraded},
			expected:     HealthStatusDegraded,
		},
		{
			name:         "DEGRADED and OFFLINE returns OFFLINE (worst wins)",
			portStatuses: []string{HealthStatusDegraded, HealthStatusOffline},
			expected:     HealthStatusOffline,
		},
		{
			name:         "ONLINE and UNCHECKED returns UNCHECKED (worst wins)",
			portStatuses: []string{HealthStatusOnline, HealthStatusUnchecked},
			expected:     HealthStatusUnchecked,
		},
		{
			name:         "UNCHECKED and OFFLINE returns OFFLINE (worst wins)",
			portStatuses: []string{HealthStatusUnchecked, HealthStatusOffline},
			expected:     HealthStatusOffline,
		},
		{
			name:         "all ONLINE returns ONLINE",
			portStatuses: []string{HealthStatusOnline, HealthStatusOnline, HealthStatusOnline},
			expected:     HealthStatusOnline,
		},
		{
			name:         "all OFFLINE returns OFFLINE",
			portStatuses: []string{HealthStatusOffline, HealthStatusOffline},
			expected:     HealthStatusOffline,
		},
		{
			name:         "multiple mixed returns worst (OFFLINE)",
			portStatuses: []string{HealthStatusOnline, HealthStatusDegraded, HealthStatusOffline, HealthStatusUnchecked},
			expected:     HealthStatusOffline,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeServiceHealth(tt.portStatuses)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHealthStatusPriority(t *testing.T) {
	// Verify the priority order: OFFLINE > DEGRADED > UNCHECKED > ONLINE
	assert.Greater(t, healthStatusPriority[HealthStatusOffline], healthStatusPriority[HealthStatusDegraded])
	assert.Greater(t, healthStatusPriority[HealthStatusDegraded], healthStatusPriority[HealthStatusUnchecked])
	assert.Greater(t, healthStatusPriority[HealthStatusUnchecked], healthStatusPriority[HealthStatusOnline])
}

func TestF5AvailStateToHealthStatus(t *testing.T) {
	tests := []struct {
		name     string
		state    int
		expected string
	}{
		{"green (1) returns ONLINE", f5AvailStateGreen, HealthStatusOnline},
		{"yellow (2) returns DEGRADED", f5AvailStateYellow, HealthStatusDegraded},
		{"red (3) returns OFFLINE", f5AvailStateRed, HealthStatusOffline},
		{"none (0) returns UNCHECKED", f5AvailStateNone, HealthStatusUnchecked},
		{"blue (4) returns UNCHECKED", f5AvailStateBlue, HealthStatusUnchecked},
		{"unknown value returns UNCHECKED", 99, HealthStatusUnchecked},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f5AvailStateToHealthStatus(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}
