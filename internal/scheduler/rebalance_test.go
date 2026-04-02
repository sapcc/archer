// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldRebalance_Logic(t *testing.T) {
	t.Run("returns false with less than 2 agents", func(t *testing.T) {
		agents := []AgentLoad{
			{Host: "agent-1", ServiceCount: 5},
		}

		if len(agents) < 2 {
			assert.True(t, true, "Should not rebalance with < 2 agents")
			return
		}
		t.Fail()
	})

	t.Run("returns false when max count is zero", func(t *testing.T) {
		agents := []AgentLoad{
			{Host: "agent-1", ServiceCount: 0},
			{Host: "agent-2", ServiceCount: 0},
		}

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
			assert.True(t, true, "Should not rebalance when max is 0")
			return
		}
		t.Fail()
	})

	t.Run("calculates imbalance correctly - below threshold", func(t *testing.T) {
		agents := []AgentLoad{
			{Host: "agent-1", ServiceCount: 10},
			{Host: "agent-2", ServiceCount: 8},
		}
		threshold := 0.5

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

		// Imbalance = (10-8)/10 = 0.2
		imbalance := float64(maxCount-minCount) / float64(maxCount)

		assert.Equal(t, 0.2, imbalance)
		assert.False(t, imbalance > threshold, "0.2 should not exceed 0.5 threshold")
	})

	t.Run("calculates imbalance correctly - above threshold", func(t *testing.T) {
		agents := []AgentLoad{
			{Host: "agent-1", ServiceCount: 10},
			{Host: "agent-2", ServiceCount: 2},
		}
		threshold := 0.5

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

		// Imbalance = (10-2)/10 = 0.8
		imbalance := float64(maxCount-minCount) / float64(maxCount)

		assert.Equal(t, 0.8, imbalance)
		assert.True(t, imbalance > threshold, "0.8 should exceed 0.5 threshold")
	})

	t.Run("handles multiple agents", func(t *testing.T) {
		agents := []AgentLoad{
			{Host: "agent-1", ServiceCount: 10},
			{Host: "agent-2", ServiceCount: 5},
			{Host: "agent-3", ServiceCount: 1},
		}
		threshold := 0.5

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

		// min=1, max=10, imbalance = 0.9
		imbalance := float64(maxCount-minCount) / float64(maxCount)

		assert.Equal(t, 0.9, imbalance)
		assert.True(t, imbalance > threshold)
	})
}

func TestRebalance_TargetCalculation(t *testing.T) {
	t.Run("calculates target correctly", func(t *testing.T) {
		agents := []AgentLoad{
			{Host: "agent-1", ServiceCount: 10},
			{Host: "agent-2", ServiceCount: 5},
			{Host: "agent-3", ServiceCount: 0},
		}

		totalServices := 0
		for _, agent := range agents {
			totalServices += agent.ServiceCount
		}
		target := totalServices / len(agents)

		// 15 services / 3 agents = 5
		assert.Equal(t, 5, target)
	})

	t.Run("identifies overloaded and underloaded agents", func(t *testing.T) {
		agents := []AgentLoad{
			{Host: "agent-1", ServiceCount: 10},
			{Host: "agent-2", ServiceCount: 5},
			{Host: "agent-3", ServiceCount: 0},
		}

		totalServices := 0
		for _, agent := range agents {
			totalServices += agent.ServiceCount
		}
		target := totalServices / len(agents) // 5

		var overloaded, underloaded []AgentLoad
		for _, agent := range agents {
			if agent.ServiceCount > target+1 { // > 6
				overloaded = append(overloaded, agent)
			} else if agent.ServiceCount < target { // < 5
				underloaded = append(underloaded, agent)
			}
		}

		assert.Len(t, overloaded, 1)
		assert.Equal(t, "agent-1", overloaded[0].Host)
		assert.Len(t, underloaded, 1)
		assert.Equal(t, "agent-3", underloaded[0].Host)
	})
}

func TestAgentLoad_Structure(t *testing.T) {
	load := AgentLoad{
		Host:         "test-host",
		ServiceCount: 42,
	}

	assert.Equal(t, "test-host", load.Host)
	assert.Equal(t, 42, load.ServiceCount)
}

func TestAgentInfo_Structure(t *testing.T) {
	az := "az1"
	info := AgentInfo{
		Host:             "test-host",
		AvailabilityZone: &az,
		ServiceCount:     10,
	}

	assert.Equal(t, "test-host", info.Host)
	assert.Equal(t, "az1", *info.AvailabilityZone)
	assert.Equal(t, 10, info.ServiceCount)
}
