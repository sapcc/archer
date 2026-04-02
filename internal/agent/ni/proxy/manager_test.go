// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/archer/internal/config"
)

func setupTempDir(t *testing.T) func() {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "proxy-test-*")
	require.NoError(t, err)
	config.Global.Agent.TempDir = tmpDir
	return func() {
		_ = os.RemoveAll(tmpDir)
	}
}

func TestNewManager(t *testing.T) {
	ctx := context.Background()
	m := NewManager(ctx)

	assert.NotNil(t, m)
	assert.NotNil(t, m.proxies)
	assert.Equal(t, 0, len(m.proxies))
}

func TestManager_IsRunning_Empty(t *testing.T) {
	ctx := context.Background()
	m := NewManager(ctx)

	serviceID := strfmt.UUID("550e8400-e29b-41d4-a716-446655440000")
	assert.False(t, m.IsRunning(serviceID))
}

func TestManager_StartProxy(t *testing.T) {
	cleanup := setupTempDir(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(ctx)
	serviceID := strfmt.UUID("550e8400-e29b-41d4-a716-446655440000")
	ports := []int32{18080, 18443}

	m.StartProxy(serviceID, "127.0.0.1", ports)

	// Give the goroutine time to start
	time.Sleep(50 * time.Millisecond)

	assert.True(t, m.IsRunning(serviceID))

	// Verify socket files were created
	assert.FileExists(t, GetSocketPath(18080))
	assert.FileExists(t, GetSocketPath(18443))
}

func TestManager_StartProxy_ReplacesExisting(t *testing.T) {
	cleanup := setupTempDir(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(ctx)
	serviceID := strfmt.UUID("550e8400-e29b-41d4-a716-446655440000")

	// Start first proxy
	m.StartProxy(serviceID, "127.0.0.1", []int32{18080})
	time.Sleep(50 * time.Millisecond)
	assert.True(t, m.IsRunning(serviceID))

	// Start second proxy with same service ID - should replace
	m.StartProxy(serviceID, "127.0.0.1", []int32{18443})
	time.Sleep(50 * time.Millisecond)
	assert.True(t, m.IsRunning(serviceID))

	// Should still only have one proxy
	m.mu.RLock()
	assert.Equal(t, 1, len(m.proxies))
	assert.Equal(t, []int32{18443}, m.proxies[serviceID].ports)
	m.mu.RUnlock()
}

func TestManager_StopProxy(t *testing.T) {
	cleanup := setupTempDir(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(ctx)
	serviceID := strfmt.UUID("550e8400-e29b-41d4-a716-446655440000")

	m.StartProxy(serviceID, "127.0.0.1", []int32{18080})
	time.Sleep(50 * time.Millisecond)
	assert.True(t, m.IsRunning(serviceID))

	m.StopProxy(serviceID)
	assert.False(t, m.IsRunning(serviceID))
}

func TestManager_StopProxy_NotRunning(t *testing.T) {
	ctx := context.Background()
	m := NewManager(ctx)

	serviceID := strfmt.UUID("550e8400-e29b-41d4-a716-446655440000")

	// Should not panic when stopping a non-existent proxy
	assert.NotPanics(t, func() {
		m.StopProxy(serviceID)
	})
	assert.False(t, m.IsRunning(serviceID))
}

func TestManager_StopAll(t *testing.T) {
	cleanup := setupTempDir(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(ctx)
	service1 := strfmt.UUID("550e8400-e29b-41d4-a716-446655440001")
	service2 := strfmt.UUID("550e8400-e29b-41d4-a716-446655440002")
	service3 := strfmt.UUID("550e8400-e29b-41d4-a716-446655440003")

	m.StartProxy(service1, "127.0.0.1", []int32{18080})
	m.StartProxy(service2, "127.0.0.1", []int32{18081})
	m.StartProxy(service3, "127.0.0.1", []int32{18082})
	time.Sleep(50 * time.Millisecond)

	assert.True(t, m.IsRunning(service1))
	assert.True(t, m.IsRunning(service2))
	assert.True(t, m.IsRunning(service3))

	m.StopAll()

	assert.False(t, m.IsRunning(service1))
	assert.False(t, m.IsRunning(service2))
	assert.False(t, m.IsRunning(service3))
}

func TestManager_StopAll_Empty(t *testing.T) {
	ctx := context.Background()
	m := NewManager(ctx)

	// Should not panic when stopping all with no proxies
	assert.NotPanics(t, func() {
		m.StopAll()
	})
}

func TestManager_ConcurrentAccess(t *testing.T) {
	cleanup := setupTempDir(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewManager(ctx)

	// Run concurrent operations
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			serviceID := strfmt.UUID("550e8400-e29b-41d4-a716-44665544000" + string(rune('0'+id)))
			m.StartProxy(serviceID, "127.0.0.1", []int32{int32(19000 + id)})
			time.Sleep(10 * time.Millisecond)
			m.IsRunning(serviceID)
			m.StopProxy(serviceID)
		}(i)
	}

	// Wait for all goroutines
	wg.Wait()

	// Should not have panicked and all proxies should be stopped
	m.mu.RLock()
	assert.Equal(t, 0, len(m.proxies))
	m.mu.RUnlock()
}

func TestManager_UnixProxyConnection(t *testing.T) {
	cleanup := setupTempDir(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a TCP echo server
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = tcpListener.Close() }()

	tcpPort := tcpListener.Addr().(*net.TCPAddr).Port

	// Echo server goroutine
	go func() {
		for {
			conn, err := tcpListener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				buf := make([]byte, 1024)
				for {
					n, err := c.Read(buf)
					if err != nil {
						return
					}
					_, _ = c.Write(buf[:n])
				}
			}(conn)
		}
	}()

	// Start the proxy manager
	m := NewManager(ctx)
	serviceID := strfmt.UUID("550e8400-e29b-41d4-a716-446655440000")
	m.StartProxy(serviceID, "127.0.0.1", []int32{int32(tcpPort)})

	// Give the proxy time to start listening
	time.Sleep(50 * time.Millisecond)

	// Connect to the Unix socket
	socketPath := GetSocketPath(tcpPort)
	unixConn, err := net.Dial("unix", socketPath)
	require.NoError(t, err)
	defer func() { _ = unixConn.Close() }()

	// Send test data through the proxy
	testData := []byte("hello proxy")
	_, err = unixConn.Write(testData)
	require.NoError(t, err)

	// Read the echoed response
	buf := make([]byte, 1024)
	err = unixConn.SetReadDeadline(time.Now().Add(time.Second))
	require.NoError(t, err)
	n, err := unixConn.Read(buf)
	require.NoError(t, err)

	assert.Equal(t, testData, buf[:n])

	// Cleanup
	m.StopProxy(serviceID)
	assert.False(t, m.IsRunning(serviceID))
}
