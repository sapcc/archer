// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package haproxy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"text/template"

	"github.com/bcicen/go-haproxy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sapcc/archer/internal/agent/ni/models"
	"github.com/sapcc/archer/internal/agent/ni/proxy"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
)

var configTemplate = `
global
    log         stdout format raw local0
    stats       socket "{{getStatsSocketPath .Network}}" mode 600 level admin
    stats       timeout 2m
    maxconn     1024
    pidfile     "{{getPidFilePath .Network}}"
    #user        haproxy
    #group       haproxy
    daemon

defaults
    log global
    mode http
    option httplog
    option dontlognull
    option http-server-close
    option forwardfor
    retries                 3
    timeout http-request    30s
    timeout connect         30s
    timeout client          32s
    timeout server          32s
    timeout http-keep-alive 30s

{{- $protocol := .Protocol }}
{{- $upstream := .UpstreamHost }}

{{ range .UpstreamPorts }}
frontend fronted_{{ . }}
    bind *:{{ . }}
    mode {{ lower $protocol }}
    default_backend backend_{{ . }}

backend backend_{{ . }}
    mode {{ lower $protocol }}
	{{- if eq $protocol "HTTP" }}
    http-request replace-header Host .* {{ $upstream }}
	{{- end }}
    server upstream {{ . | getSocketPath }}

{{ end }}
`

type haProxyInstance struct {
	cmd    *exec.Cmd
	config *os.File
	client *haproxy.HAProxyClient
	pid    int
}

type HAProxyController struct {
	instances map[string]*haProxyInstance
}

var (
	totalBytesOut = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "haproxy_total_bytes_out",
		Help: "Total Bytes out",
	}, []string{"network"})
	currConns = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "haproxy_curr_conns",
		Help: "Current number of connections",
	}, []string{"network"})
	metricScrape = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "haproxy_scraped",
		Help: "Counter of haproxy metric scrapes",
	}, []string{"network"})
)

func NewHAProxyController() *HAProxyController {
	return &HAProxyController{
		make(map[string]*haProxyInstance),
	}
}

func (h *HAProxyController) CollectStats() {
	for networkID, instance := range h.instances {
		info, err := instance.client.Info()
		if err != nil {
			log.Debugf("Failed fetching stats for instance '%s'", networkID)
		}
		totalBytesOut.WithLabelValues(networkID).Set(float64(info.TotalBytesOut))
		currConns.WithLabelValues(networkID).Set(float64(info.CurrConns))
		metricScrape.WithLabelValues(networkID).Inc()
	}
}

func (h *HAProxyController) IsRunning(networkID string) bool {
	_, ok := h.instances[networkID]
	if !ok {
		return false
	}

	// read pid and check if process exists
	pid, err := readPidFile(GetPidFilePath(networkID))
	if err != nil {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Debugf("Failed to find process: %s", err)
		return false
	}

	return process.Signal(syscall.Signal(0)) == nil
}

func (h *HAProxyController) AddInstance(si *models.ServiceInjection) error {
	// create config
	filename := GetConfigFilePath(si.Network.String())
	configFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer func() { _ = configFile.Close() }()
	log.Debugf("Created HAProxy config file '%s'", configFile.Name())

	funcMap := template.FuncMap{
		"lower":              strings.ToLower,
		"getSocketPath":      proxy.GetSocketPath,
		"getStatsSocketPath": GetStatsSocketPath,
		"getPidFilePath":     GetPidFilePath,
	}

	// template config
	t, err := template.New("haproxy").Funcs(funcMap).Parse(configTemplate)
	if err != nil {
		return err
	}
	data := map[string]any{
		"UpstreamHost":  config.Global.Agent.ServiceUpstreamHost,
		"UpstreamPorts": si.ServicePorts,
		"Network":       si.Network.String(),
		"Protocol":      si.ServiceProtocol,
	}
	if err = t.Execute(configFile, data); err != nil {
		return err
	}

	outfile, err := os.Create(GetLogFilePath(si.Network.String()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = outfile.Close() }()

	// run haproxy
	cmd := exec.Command("haproxy", "-f", configFile.Name())
	cmd.Stdout = outfile
	cmd.Stderr = outfile
	if err = cmd.Run(); err != nil {
		return err
	}

	// read pid
	pid, err := readPidFile(GetPidFilePath(si.Network.String()))
	if err != nil {
		return err
	}

	// init haproxy stats client
	haProxyClient := haproxy.HAProxyClient{
		Addr: fmt.Sprintf("unix://%s", GetStatsSocketPath(si.Network.String())),
	}
	info, err := haProxyClient.Info()
	if err != nil {
		return err
	}
	log.Printf("Running %s version %s PID %d for %s", info.Name, info.Version, pid, si.Network)

	instance := haProxyInstance{
		cmd:    cmd,
		config: configFile,
		client: &haProxyClient,
		pid:    pid,
	}

	h.instances[si.Network.String()] = &instance
	return nil
}

func (h *HAProxyController) RemoveInstance(networkID string) error {
	instance, ok := h.instances[networkID]
	if !ok {
		return fmt.Errorf("instance '%s' not found", networkID)
	}

	// Terminate haproxy
	if err := syscall.Kill(instance.pid, syscall.SIGTERM); err != nil {
		return err
	}

	// Remove config and pidfile
	TryRemoveFile(instance.config.Name())
	TryRemoveFile(GetPidFilePath(networkID))

	delete(h.instances, networkID)
	return nil
}

func (h *HAProxyController) Run(ctx context.Context) {
	<-ctx.Done()
	log.Debug("Shutting down HAProxy instances...")
	for networkID := range h.instances {
		if err := h.RemoveInstance(networkID); err != nil {
			log.Errorf("Failed to remove instance '%s': %s", networkID, err)
		}
	}
}

func Dump(file string) {
	d, err := os.Open(file)
	if err != nil {
		log.Errorf("Failed to opening file(path=%s): %s", file, err)
	}
	defer func() { _ = d.Close() }()
	scanner := bufio.NewScanner(d)
	log.Infof("###### Dumping file '%s'", file)
	for scanner.Scan() {
		log.Error(scanner.Text())
	}
}

func readPidFile(pidFile string) (int, error) {
	if pidFile == "" {
		return 0, fmt.Errorf("no pidfile")
	}

	d, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(string(bytes.TrimSpace(d)))
	if err != nil {
		return 0, fmt.Errorf("failed converting pid %s: %s", pidFile, err)
	}

	return pid, nil
}

func TryRemoveFile(file string) {
	if err := os.Remove(file); err != nil {
		log.WithError(err).Warnf("Failed to remove file '%s'", file)
	}
}

func GetStatsSocketPath(networkID string) string {
	return fmt.Sprintf("%s/haproxy-stats-%s.sock", config.Global.Agent.TempDir, networkID)
}

func GetPidFilePath(networkID string) string {
	return fmt.Sprintf("%s/haproxy-%s.pid", config.Global.Agent.TempDir, networkID)
}

func GetLogFilePath(networkID string) string {
	return fmt.Sprintf("%s/haproxy-%s.log", config.Global.Agent.TempDir, networkID)
}

func GetConfigFilePath(networkID string) string {
	return fmt.Sprintf("%s/haproxy-%s.conf", config.Global.Agent.TempDir, networkID)
}
