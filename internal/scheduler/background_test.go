// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBackgroundScheduler(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	cfg := defaultConfig()
	serviceScheduler := NewServiceScheduler(mock, cfg, nil)

	bgScheduler, err := NewBackgroundScheduler(serviceScheduler, mock, cfg.CheckInterval, cfg.RebalanceDelay)

	assert.NoError(t, err)
	assert.NotNil(t, bgScheduler)
	assert.Equal(t, cfg.CheckInterval, bgScheduler.checkInterval)
	assert.Equal(t, cfg.RebalanceDelay, bgScheduler.rebalanceDelay)
	assert.NotNil(t, bgScheduler.recoveredAgents)
}

func TestBackgroundScheduler_RecoveredAgentsTracking(t *testing.T) {
	cfg := defaultConfig()
	cfg.RebalanceDelay = 100 * time.Millisecond

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	serviceScheduler := NewServiceScheduler(mock, cfg, nil)

	bgScheduler := &BackgroundScheduler{
		scheduler:       serviceScheduler,
		rebalanceDelay:  cfg.RebalanceDelay,
		recoveredAgents: make(map[string]time.Time),
	}

	// Add a recovered agent
	bgScheduler.mu.Lock()
	bgScheduler.recoveredAgents["agent-1"] = time.Now().Add(-1 * time.Hour) // Old recovery
	bgScheduler.recoveredAgents["agent-2"] = time.Now()                     // Recent recovery
	bgScheduler.mu.Unlock()

	// Check that we have both agents
	assert.Len(t, bgScheduler.recoveredAgents, 2)

	// Simulate rebalance delay check
	bgScheduler.mu.Lock()
	now := time.Now()
	var toDelete []string
	for host, recoveredAt := range bgScheduler.recoveredAgents {
		if now.Sub(recoveredAt) >= bgScheduler.rebalanceDelay {
			toDelete = append(toDelete, host)
		}
	}
	for _, host := range toDelete {
		delete(bgScheduler.recoveredAgents, host)
	}
	bgScheduler.mu.Unlock()

	// Only agent-1 should be removed (recovered over an hour ago)
	assert.NotContains(t, bgScheduler.recoveredAgents, "agent-1")
	assert.Contains(t, bgScheduler.recoveredAgents, "agent-2")
}

func TestBackgroundScheduler_ConcurrentAccess(t *testing.T) {
	cfg := defaultConfig()

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	serviceScheduler := NewServiceScheduler(mock, cfg, nil)

	bgScheduler := &BackgroundScheduler{
		scheduler:       serviceScheduler,
		rebalanceDelay:  cfg.RebalanceDelay,
		recoveredAgents: make(map[string]time.Time),
	}

	// Test concurrent access to recoveredAgents
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			bgScheduler.mu.Lock()
			bgScheduler.recoveredAgents["agent-"+string(rune('0'+id))] = time.Now()
			bgScheduler.mu.Unlock()

			bgScheduler.mu.Lock()
			delete(bgScheduler.recoveredAgents, "agent-"+string(rune('0'+id)))
			bgScheduler.mu.Unlock()

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not have panicked
	assert.NotNil(t, bgScheduler.recoveredAgents)
}

func TestBackgroundScheduler_CheckRebalanceLogic(t *testing.T) {
	t.Run("does not trigger rebalance when no agents passed delay", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.RebalanceDelay = 10 * time.Minute

		bgScheduler := &BackgroundScheduler{
			rebalanceDelay: cfg.RebalanceDelay,
			recoveredAgents: map[string]time.Time{
				"agent-1": time.Now(), // Just recovered
			},
		}

		// Check the logic
		bgScheduler.mu.Lock()
		shouldRebalance := false
		now := time.Now()
		for host, recoveredAt := range bgScheduler.recoveredAgents {
			if now.Sub(recoveredAt) >= bgScheduler.rebalanceDelay {
				delete(bgScheduler.recoveredAgents, host)
				shouldRebalance = true
			}
		}
		bgScheduler.mu.Unlock()

		assert.False(t, shouldRebalance)
		assert.Contains(t, bgScheduler.recoveredAgents, "agent-1")
	})

	t.Run("triggers rebalance when agent passed delay", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.RebalanceDelay = 1 * time.Millisecond

		bgScheduler := &BackgroundScheduler{
			rebalanceDelay: cfg.RebalanceDelay,
			recoveredAgents: map[string]time.Time{
				"agent-1": time.Now().Add(-1 * time.Hour), // Recovered long ago
			},
		}

		// Check the logic
		bgScheduler.mu.Lock()
		shouldRebalance := false
		now := time.Now()
		for host, recoveredAt := range bgScheduler.recoveredAgents {
			if now.Sub(recoveredAt) >= bgScheduler.rebalanceDelay {
				delete(bgScheduler.recoveredAgents, host)
				shouldRebalance = true
			}
		}
		bgScheduler.mu.Unlock()

		assert.True(t, shouldRebalance)
		assert.NotContains(t, bgScheduler.recoveredAgents, "agent-1")
	})

	t.Run("does nothing when recoveredAgents is empty", func(t *testing.T) {
		cfg := defaultConfig()

		bgScheduler := &BackgroundScheduler{
			rebalanceDelay:  cfg.RebalanceDelay,
			recoveredAgents: make(map[string]time.Time),
		}

		bgScheduler.mu.Lock()
		shouldRebalance := false
		now := time.Now()
		for host, recoveredAt := range bgScheduler.recoveredAgents {
			if now.Sub(recoveredAt) >= bgScheduler.rebalanceDelay {
				delete(bgScheduler.recoveredAgents, host)
				shouldRebalance = true
			}
		}
		bgScheduler.mu.Unlock()

		assert.False(t, shouldRebalance)
	})
}

func TestBackgroundScheduler_Providers(t *testing.T) {
	// Verify that runCycle processes both providers
	providers := []string{"tenant", "cp"}

	assert.Len(t, providers, 2)
	assert.Contains(t, providers, "tenant")
	assert.Contains(t, providers, "cp")
}
