// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/v2/internal/config"
)

// newStubManager returns a Manager whose startProc blocks until ctx is cancelled
// and records how many times it was invoked per network, so supervision can be
// tested without spawning real socat processes.
func newStubManager(ctx context.Context) (*Manager, func(strfmt.UUID) int) {
	m := NewManager(ctx)

	var mu sync.Mutex
	runs := map[strfmt.UUID]int{}
	m.startProc = func(ctx context.Context, networkID strfmt.UUID, _ string, _ []int32) error {
		mu.Lock()
		runs[networkID]++
		mu.Unlock()
		<-ctx.Done()
		return nil
	}
	return m, func(n strfmt.UUID) int {
		mu.Lock()
		defer mu.Unlock()
		return runs[n]
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager(context.Background())
	assert.NotNil(t, m)
	assert.NotNil(t, m.proxies)
	assert.Equal(t, 0, len(m.proxies))
}

func TestManager_IsRunning_Empty(t *testing.T) {
	m := NewManager(context.Background())
	assert.False(t, m.IsRunning(strfmt.UUID("550e8400-e29b-41d4-a716-446655440000")))
}

func TestManager_StartStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, runs := newStubManager(ctx)
	networkID := strfmt.UUID("550e8400-e29b-41d4-a716-446655440000")

	m.StartProxy(networkID, "127.0.0.1", []int32{18080, 18443})
	assert.Eventually(t, func() bool { return runs(networkID) == 1 }, time.Second, 5*time.Millisecond)
	assert.True(t, m.IsRunning(networkID))

	m.StopProxy(networkID)
	assert.False(t, m.IsRunning(networkID))
}

func TestManager_StartProxy_Idempotent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, runs := newStubManager(ctx)
	networkID := strfmt.UUID("550e8400-e29b-41d4-a716-446655440000")

	m.StartProxy(networkID, "127.0.0.1", []int32{18080})
	assert.Eventually(t, func() bool { return runs(networkID) == 1 }, time.Second, 5*time.Millisecond)

	// Second StartProxy for the same network is a no-op (endpoints in a network
	// share one proxy set, mirroring HAProxy).
	m.StartProxy(networkID, "127.0.0.1", []int32{18443})
	time.Sleep(50 * time.Millisecond)

	m.mu.RLock()
	assert.Equal(t, 1, len(m.proxies))
	assert.Equal(t, []int32{18080}, m.proxies[networkID].ports, "second StartProxy should not replace the ports")
	m.mu.RUnlock()
	assert.Equal(t, 1, runs(networkID), "second StartProxy should not re-invoke startProc")
}

func TestManager_StopProxy_NotRunning(t *testing.T) {
	m := NewManager(context.Background())
	assert.NotPanics(t, func() {
		m.StopProxy(strfmt.UUID("550e8400-e29b-41d4-a716-446655440000"))
	})
}

func TestManager_StopAll(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m, _ := newStubManager(ctx)
	ids := []strfmt.UUID{
		"550e8400-e29b-41d4-a716-446655440001",
		"550e8400-e29b-41d4-a716-446655440002",
		"550e8400-e29b-41d4-a716-446655440003",
	}
	for _, id := range ids {
		m.StartProxy(id, "127.0.0.1", []int32{18080})
	}
	assert.Eventually(t, func() bool { return m.IsRunning(ids[0]) && m.IsRunning(ids[2]) }, time.Second, 5*time.Millisecond)

	m.StopAll()
	for _, id := range ids {
		assert.False(t, m.IsRunning(id))
	}
}

func TestManager_Supervise_Restarts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(ctx)
	var mu sync.Mutex
	var count int
	m.startProc = func(ctx context.Context, _ strfmt.UUID, _ string, _ []int32) error {
		mu.Lock()
		count++
		mu.Unlock()
		return assert.AnError // exit immediately to trigger restart
	}

	orig := restartBackoffForTest
	restartBackoffForTest = 5 * time.Millisecond
	defer func() { restartBackoffForTest = orig }()

	networkID := strfmt.UUID("550e8400-e29b-41d4-a716-446655440000")
	m.StartProxy(networkID, "127.0.0.1", []int32{18080})

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return count >= 3
	}, time.Second, 5*time.Millisecond, "supervisor should restart after unexpected exit")

	m.StopProxy(networkID)
}

func TestManager_InjectedStartProc(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// NewManager accepts a StartProc override (used by callers' tests to avoid
	// spawning socat).
	m := NewManager(ctx, func(ctx context.Context, _ strfmt.UUID, _ string, _ []int32) error {
		<-ctx.Done()
		return nil
	})
	networkID := strfmt.UUID("550e8400-e29b-41d4-a716-446655440000")
	m.StartProxy(networkID, "127.0.0.1", []int32{18080})
	assert.True(t, m.IsRunning(networkID))
	m.StopProxy(networkID)
	assert.False(t, m.IsRunning(networkID))
}

// TestGetSocketPaths pins the per-network path layout the HAProxy backend and
// socat listener rely on.
func TestGetSocketPaths(t *testing.T) {
	config.Global.Agent.RunDir = "/run/archer"
	net := "660e8400-e29b-41d4-a716-446655440000"
	assert.Equal(t, "/run/archer/"+net, GetNetworkDir(net))
	assert.Equal(t, "/run/archer/"+net+"/80.sock", GetSocketPath(net, 80))
}
