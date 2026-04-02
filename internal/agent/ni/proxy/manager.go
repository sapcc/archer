// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"sync"

	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"
)

// serviceProxy holds the context and cancel function for a service's proxy thread.
type serviceProxy struct {
	cancel context.CancelFunc
	ports  []int32
}

// Manager manages Unix proxy threads for multiple services.
// Each service gets its own set of Unix socket proxies.
type Manager struct {
	mu        sync.RWMutex
	proxies   map[strfmt.UUID]*serviceProxy // serviceID -> proxy info
	parentCtx context.Context
}

// NewManager creates a new proxy manager.
func NewManager(ctx context.Context) *Manager {
	return &Manager{
		proxies:   make(map[strfmt.UUID]*serviceProxy),
		parentCtx: ctx,
	}
}

// StartProxy starts a Unix proxy thread for a service.
// If a proxy is already running for this service, it will be stopped first.
func (m *Manager) StartProxy(serviceID strfmt.UUID, upstream string, ports []int32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop existing proxy if any
	if existing, ok := m.proxies[serviceID]; ok {
		log.Debugf("proxymanager: stopping existing proxy for service %s", serviceID)
		existing.cancel()
		delete(m.proxies, serviceID)
	}

	// Create a new context for this service's proxy
	ctx, cancel := context.WithCancel(m.parentCtx)

	m.proxies[serviceID] = &serviceProxy{
		cancel: cancel,
		ports:  ports,
	}

	log.Infof("proxymanager: starting proxy for service %s, upstream=%s, ports=%v", serviceID, upstream, ports)
	go UnixListenersThread(ctx, upstream, ports)
}

// StopProxy stops the Unix proxy thread for a service.
func (m *Manager) StopProxy(serviceID strfmt.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if proxy, ok := m.proxies[serviceID]; ok {
		log.Infof("proxymanager: stopping proxy for service %s", serviceID)
		proxy.cancel()
		delete(m.proxies, serviceID)
	}
}

// IsRunning checks if a proxy is running for a service.
func (m *Manager) IsRunning(serviceID strfmt.UUID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.proxies[serviceID]
	return ok
}

// StopAll stops all running proxy threads.
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for serviceID, proxy := range m.proxies {
		log.Debugf("proxymanager: stopping proxy for service %s", serviceID)
		proxy.cancel()
	}
	m.proxies = make(map[strfmt.UUID]*serviceProxy)
}
