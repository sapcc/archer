// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/sapcc/archer/internal/config"
)

var (
	activeConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "archer_proxy_active_connections",
			Help: "Number of active connections per Unix socket proxy",
		},
		[]string{"service_id", "port"},
	)
	totalConnections = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "archer_proxy_connections_total",
			Help: "Total number of connections handled per Unix socket proxy",
		},
		[]string{"service_id", "port"},
	)
)

func init() {
	prometheus.MustRegister(activeConnections, totalConnections)
}

type unixProxy struct {
	ip        net.IP
	port      int
	serviceID string
	log       *log.Entry
}

func (up *unixProxy) proxy(unixConn *net.UnixConn, tcpConn *net.TCPConn) {
	portStr := fmt.Sprintf("%d", up.port)
	activeConnections.WithLabelValues(up.serviceID, portStr).Inc()
	totalConnections.WithLabelValues(up.serviceID, portStr).Inc()
	defer activeConnections.WithLabelValues(up.serviceID, portStr).Dec()

	defer func() { _ = unixConn.Close() }()
	defer func() { _ = tcpConn.Close() }()

	var wg sync.WaitGroup

	wg.Go(func() {
		if _, err := io.Copy(unixConn, tcpConn); err != nil && err != io.EOF && !errors.Is(err, unix.EPIPE) {
			up.log.WithError(err).Error("copy backend->frontend failed")
		}
		_ = unixConn.CloseWrite()
	})

	wg.Go(func() {
		if _, err := io.Copy(tcpConn, unixConn); err != nil && err != io.EOF {
			up.log.WithError(err).Error("copy frontend->backend failed")
		}
		_ = tcpConn.CloseWrite()
	})

	wg.Wait()
}

func (up *unixProxy) run(listener *net.UnixListener) error {
	if listener == nil {
		return errors.New("nil listener")
	}

	for {
		frontend, err := listener.AcceptUnix()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Op == "accept" {
				up.log.Debug("listener closed")
				break
			}
			up.log.WithError(err).Warn("accept failed")
			continue
		}

		backend, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: up.ip, Port: up.port})
		if err != nil {
			up.log.WithError(err).Warn("dial backend failed")
			_ = frontend.Close()
			continue
		}

		go up.proxy(frontend, backend)
	}
	return nil
}

func UnixListenersThread(ctx context.Context, serviceID strfmt.UUID, upstream string, ports []int32) {
	logger := log.WithFields(log.Fields{
		"component": "unixproxy",
		"service":   serviceID.String(),
		"upstream":  upstream,
	})

	ips, err := net.LookupIP(upstream)
	if err != nil {
		logger.WithError(err).Fatal("DNS lookup failed")
	}

	logger.WithField("ports", ports).Info("starting listeners")

	var wg sync.WaitGroup
	var listeners []*net.UnixListener

	for _, port := range ports {
		socketPath := GetSocketPath(serviceID.String(), int(port))

		if err := os.RemoveAll(socketPath); err != nil {
			logger.WithError(err).WithField("socket", socketPath).Fatal("failed to remove existing socket")
		}

		listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
		if err != nil {
			logger.WithError(err).WithField("socket", socketPath).Fatal("failed to create listener")
		}

		listeners = append(listeners, listener)

		proxyLogger := logger.WithFields(log.Fields{
			"port":   port,
			"socket": socketPath,
		})

		wg.Go(func() {
			p := unixProxy{
				ip:        ips[0],
				port:      int(port),
				serviceID: serviceID.String(),
				log:       proxyLogger,
			}
			if err := p.run(listener); err != nil {
				proxyLogger.WithError(err).Error("proxy stopped with error")
			}
		})
	}

	<-ctx.Done()
	logger.Info("shutting down")

	for _, listener := range listeners {
		if err := listener.Close(); err != nil {
			logger.WithError(err).Error("failed to close listener")
		}
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		wg.Wait()
		cancel()
	}()

	<-timeoutCtx.Done()
	if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
		logger.Error("shutdown timed out")
	} else {
		logger.Info("shutdown complete")
	}
}

func GetSocketPath(serviceID string, port int) string {
	// Use first 8 chars of service ID to keep socket path short (Unix limit ~108 chars)
	shortID := serviceID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	return fmt.Sprintf("%s/proxy-%s-%d.sock", config.Global.Agent.TempDir, shortID, port)
}
