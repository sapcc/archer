// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/config"
)

// GetNetworkDir returns the host path of a network's directory (HAProxy files + proxy sockets; also HAProxy's chroot root): /run/archer/<network>.
func GetNetworkDir(networkID string) string {
	return fmt.Sprintf("%s/%s", config.Global.Agent.RunDir, networkID)
}

// GetSocketPath returns the host path of a proxy socket: /run/archer/<network>/<port>.sock.
func GetSocketPath(networkID string, port int32) string {
	return fmt.Sprintf("%s/%d.sock", GetNetworkDir(networkID), port)
}

// restartBackoffForTest is the supervisor restart delay; a var so tests can shorten it.
var restartBackoffForTest = 2 * time.Second

// networkProxy holds the cancel function for a network's supervised socat processes.
type networkProxy struct {
	cancel context.CancelFunc
	ports  []int32
}

// StartProc runs the proxies for a network and blocks until they exit. The
// default spawns socat; tests inject a stub to avoid needing socat/root.
type StartProc func(ctx context.Context, networkID strfmt.UUID, upstreamIP string, ports []int32) error

// Manager supervises unprivileged socat processes (one per port) per network.
type Manager struct {
	mu        sync.RWMutex
	proxies   map[strfmt.UUID]*networkProxy
	parentCtx context.Context
	startProc StartProc
}

// NewManager creates a proxy manager. By default it spawns unprivileged socat
// processes; pass a StartProc (e.g. in tests) to override how proxies are run.
func NewManager(ctx context.Context, startProc ...StartProc) *Manager {
	m := &Manager{
		proxies:   make(map[strfmt.UUID]*networkProxy),
		parentCtx: ctx,
	}
	if len(startProc) > 0 && startProc[0] != nil {
		m.startProc = startProc[0]
	} else {
		m.startProc = m.spawnSocat
	}
	return m
}

// StartProxy starts the supervised socat processes for a network; idempotent (no-op if already running, mirroring HAProxy).
func (m *Manager) StartProxy(networkID strfmt.UUID, upstream string, ports []int32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.proxies[networkID]; ok {
		return
	}

	ctx, cancel := context.WithCancel(m.parentCtx)
	m.proxies[networkID] = &networkProxy{cancel: cancel, ports: ports}

	log.Infof("proxymanager: starting proxy for network %s, upstream=%s, ports=%v", networkID, upstream, ports)
	go m.supervise(ctx, networkID, upstream, ports)
}

// supervise runs the proxies and restarts them with bounded backoff on unexpected exit, until ctx is cancelled.
func (m *Manager) supervise(ctx context.Context, networkID strfmt.UUID, upstream string, ports []int32) {
	for {
		err := m.startProc(ctx, networkID, upstream, ports)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.WithError(err).Errorf("proxymanager: proxy for network %s exited, restarting", networkID)
		} else {
			log.Warnf("proxymanager: proxy for network %s exited unexpectedly, restarting", networkID)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(restartBackoffForTest):
		}
	}
}

// spawnSocat runs one socat process per port and blocks until any exits (or ctx is cancelled).
func (m *Manager) spawnSocat(ctx context.Context, networkID strfmt.UUID, upstream string, ports []int32) error {
	socatPath, err := exec.LookPath("socat")
	if err != nil {
		return fmt.Errorf("socat binary not found in PATH: %w", err)
	}

	// Cancel the shared context as soon as one socat exits, so the others are
	// torn down and the supervisor restarts the whole set together.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// socat verbosity: default -d (fatal/error/warning only). Agent --debug raises
	// to -d -d -d -d (adds notice/info/debug, incl. per-connection accept/fork).
	verbosity := []string{"-d"}
	if config.IsDebug() {
		verbosity = []string{"-d", "-d", "-d", "-d"}
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(ports))
	for _, port := range ports {
		// mode=0666 so HAProxy can connect without chown (which needs CAP_CHOWN); safe as the socket sits in the 0700-root per-network dir.
		// No chroot: with fork each child re-applies options after su-d drops root, so its chroot() would EPERM — and socat only forwards bytes.
		listen := fmt.Sprintf("UNIX-LISTEN:%s,fork,mode=0666,su-d=%s",
			GetSocketPath(networkID.String(), port), config.Global.Agent.RunUser)
		connect := fmt.Sprintf("TCP:%s:%d", upstream, port)

		args := append(append([]string{}, verbosity...), listen, connect)
		cmd := exec.CommandContext(ctx, socatPath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		log.Infof("proxymanager: exec %s", cmd.String())

		wg.Go(func() {
			defer cancel()
			errCh <- cmd.Run()
		})
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil && ctx.Err() == nil {
			return err
		}
	}
	return nil
}

// StopProxy stops the supervised socat processes for a network and removes its socket(s).
func (m *Manager) StopProxy(networkID strfmt.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if proxy, ok := m.proxies[networkID]; ok {
		log.Infof("proxymanager: stopping proxy for network %s", networkID)
		proxy.cancel()
		for _, p := range proxy.ports {
			if err := os.Remove(GetSocketPath(networkID.String(), p)); err != nil && !os.IsNotExist(err) {
				log.WithError(err).Warnf("proxymanager: failed to remove socket for network %s port %d", networkID, p)
			}
		}
		delete(m.proxies, networkID)
	}
}

// IsRunning reports whether a proxy is supervised for a network.
func (m *Manager) IsRunning(networkID strfmt.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.proxies[networkID]
	return ok
}

// StopAll stops all supervised socat processes.
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for networkID, proxy := range m.proxies {
		log.Debugf("proxymanager: stopping proxy for network %s", networkID)
		proxy.cancel()
	}
	m.proxies = make(map[strfmt.UUID]*networkProxy)
}
