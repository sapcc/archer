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

	"github.com/sapcc/archer/internal/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

type unixProxy struct {
	ip   net.IP
	port int
}

func (up *unixProxy) proxy(unixConn *net.UnixConn, tcpConn *net.TCPConn) {
	defer func() { _ = unixConn.Close() }()
	defer func() { _ = tcpConn.Close() }()

	var wg sync.WaitGroup

	// Goroutine to copy data from conn2 to conn1
	wg.Go(func() {
		defer wg.Done()
		if _, err := io.Copy(unixConn, tcpConn); err != nil && err != io.EOF && !errors.Is(err, unix.EPIPE) {
			log.Errorf("unixproxy: io.Copy(unixConn, tcpConn): %v\n", err)
		}
		// Signal to the other side that no more data is coming from this direction.
		_ = unixConn.CloseWrite()
	})

	// Goroutine to copy data from conn1 to conn2
	wg.Go(func() {
		defer wg.Done()
		if _, err := io.Copy(tcpConn, unixConn); err != nil && err != io.EOF {
			log.Errorf("unixproxy: io.Copy(tcpConn, unixConn): %v\n", err)
		}
		// Signal to the other side that no more data is coming from this direction.
		_ = tcpConn.CloseWrite()
	})

	// Wait for both copy operations to complete
	wg.Wait()
}

func (l *unixProxy) run(listener *net.UnixListener) error {
	if listener == nil {
		return errors.New("nil listener")
	}

	var err error
	for {
		var frontend *net.UnixConn
		var backend *net.TCPConn

		frontend, err = listener.AcceptUnix()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Op == "accept" {
				log.Debugf("unixproxy: Listener closed, exiting accept loop.")
				break
			}
			log.Warnf("unixproxy: accept error for %s: %v\n", listener.Addr(), err)
			continue
		}
		defer func() { _ = frontend.Close() }()

		// Dial
		backend, err = net.DialTCP("tcp", nil, &net.TCPAddr{IP: l.ip, Port: l.port})
		if err != nil {
			log.Warnf("unixproxy: Dial error for %s:%d: %v\n", l.ip, l.port, err)
			continue
		}
		defer func() { _ = backend.Close() }()

		// blocks until connection terminated
		l.proxy(frontend, backend)
	}
	return nil
}

func UnixListenersThread(ctx context.Context, upstream string, Ports []int32) {
	var wg sync.WaitGroup
	var listeners []*net.UnixListener

	ips, err := net.LookupIP(upstream) // Resolves IPv4 and IPv6 addresses
	if err != nil {
		log.Fatalf("unixproxy: LookupIP error: %v\n", err)
	}
	for _, port := range Ports {
		socketPath := GetSocketPath(int(port))

		// Ensure the socket file is removed when the program exits
		// or if it already exists from a previous run.
		if err := os.RemoveAll(socketPath); err != nil {
			log.Fatalf("unixproxy: removing existing socket: %v", err)
		}

		// Listen on the Unix socket
		listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
		if err != nil {
			log.Fatalf("unixproxy: listening on unix socket: %v", err)
		}

		log.Debugf("unixproxy: listening on unix socket: %s", listener.Addr())

		listeners = append(listeners, listener)
		wg.Go(func() {
			p := unixProxy{ips[0], int(port)}
			log.Debugf("unixproxy: forwarding to: %s:%d", p.ip, p.port)
			if err := p.run(listener); err != nil {
				log.Errorf("unixproxy: error running proxy on %s: %v\n", listener.Addr(), err)
			}
		})
	}

	// Wait for shutdown signal
	<-ctx.Done()
	log.Info("unixproxy: Shutting down all Unix listeners")

	for _, listener := range listeners {
		if err := listener.Close(); err != nil {
			log.Errorf("error closing unix socket listener: %v\n", err)
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
		log.Errorf("unixproxy: timed out waiting for all unix sockets to close")
	} else {
		log.Infof("unixproxy: all unix sockets closed")
	}
}

func GetSocketPath(port int) string {
	return fmt.Sprintf("%s/proxy-%d.sock", config.Global.Agent.TempDir, port)
}
