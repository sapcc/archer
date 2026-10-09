// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package haproxy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"text/template"

	"github.com/bcicen/go-haproxy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/agent/ni/models"
	"github.com/sapcc/archer/v2/internal/agent/ni/proxy"
	"github.com/sapcc/archer/v2/internal/config"
)

var configTemplate = `
global
    log         stdout format raw local0 {{.LogLevel}}
    stats       socket "{{getStatsSocketPath .EndpointID}}" mode 600 level admin
    stats       timeout 2m
    maxconn     1024
    pidfile     "{{getPidFilePath .EndpointID}}"
    chroot      "{{.ChrootDir}}"
    user        {{.RunUser}}
    group       {{.RunGroup}}
    daemon

defaults
    log global
    option dontlognull
    retries                 3
    timeout connect         30s
    timeout client          32s
    timeout server          32s
    timeout tunnel          1h

{{ range .Ports }}
frontend fronted_{{ . }}
    bind :::{{ . }} v4v6
    mode {{ lower $.Protocol }}
{{- if eq $.Protocol "HTTP" }}
    option httplog
    option forwardfor
{{- end }}
    default_backend backend_{{ . }}

backend backend_{{ . }}
    mode {{ lower $.Protocol }}
{{- if eq $.Protocol "HTTP" }}
    option http-server-close
    timeout http-request    30s
    timeout http-keep-alive 30s
    http-request replace-header Host .* {{ formatHost $.UpstreamHost }}
{{- end }}
    server upstream {{ getChrootSocketPath . }}{{- if $.ProxyProtocol }} send-proxy-v2 set-proxy-v2-tlv-fmt(0xEC) %[str({{ $.EndpointID }})]{{- end }}

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
	}, []string{"endpoint"})
	currConns = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "haproxy_curr_conns",
		Help: "Current number of connections",
	}, []string{"endpoint"})
	metricScrape = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "haproxy_scraped",
		Help: "Counter of haproxy metric scrapes",
	}, []string{"endpoint"})
)

// formatHost returns the IP in HTTP Host header format.
// IPv6 addresses are wrapped in brackets per RFC 2732.
func formatHost(ip string) string {
	if parsed := net.ParseIP(ip); parsed != nil && parsed.To4() == nil {
		return "[" + ip + "]"
	}
	return ip
}

// haproxyLogLevel maps the agent's verbosity to HAProxy's max syslog level (--debug -> debug, else info).
func haproxyLogLevel() string {
	if config.IsDebug() {
		return "debug"
	}
	return "info"
}

func NewHAProxyController() *HAProxyController {
	return &HAProxyController{
		make(map[string]*haProxyInstance),
	}
}

func (h *HAProxyController) CollectStats() {
	for endpointID, instance := range h.instances {
		info, err := instance.client.Info()
		if err != nil {
			log.Debugf("Failed fetching stats for instance '%s'", endpointID)
		}
		totalBytesOut.WithLabelValues(endpointID).Set(float64(info.TotalBytesOut))
		currConns.WithLabelValues(endpointID).Set(float64(info.CurrConns))
		metricScrape.WithLabelValues(endpointID).Inc()
	}
}

func (h *HAProxyController) IsRunning(endpointID string) bool {
	_, ok := h.instances[endpointID]
	if !ok {
		return false
	}

	pid, err := readPidFile(GetPidFilePath(endpointID))
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
	endpointID := si.ID.String()
	if h.IsRunning(endpointID) {
		return nil
	}

	configFile, err := h.writeConfig(si)
	if err != nil {
		return err
	}
	defer func() { _ = configFile.Close() }()

	haproxyPath, err := exec.LookPath("haproxy")
	if err != nil {
		return fmt.Errorf("haproxy binary not found in PATH: %w", err)
	}
	cmd := exec.Command(haproxyPath, "-f", configFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Infof("exec %s", cmd.String())
	if err = cmd.Run(); err != nil {
		return err
	}

	pid, err := readPidFile(GetPidFilePath(endpointID))
	if err != nil {
		return err
	}

	haProxyClient := haproxy.HAProxyClient{
		Addr: fmt.Sprintf("unix://%s", GetStatsSocketPath(endpointID)),
	}
	info, err := haProxyClient.Info()
	if err != nil {
		return err
	}
	log.Printf("Running %s version %s PID %d for endpoint %s", info.Name, info.Version, pid, endpointID)

	h.instances[endpointID] = &haProxyInstance{
		cmd:    cmd,
		config: configFile,
		client: &haProxyClient,
		pid:    pid,
	}
	return nil
}

func (h *HAProxyController) writeConfig(si *models.ServiceInjection) (*os.File, error) {
	endpointID := si.ID.String()
	filename := GetConfigFilePath(endpointID)
	configFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	log.Debugf("Writing HAProxy config file '%s'", configFile.Name())

	funcMap := template.FuncMap{
		"lower":               strings.ToLower,
		"formatHost":          formatHost,
		"getChrootSocketPath": func(port int) string { return fmt.Sprintf("/%d.sock", port) },
		"getStatsSocketPath":  GetStatsSocketPath,
		"getPidFilePath":      GetPidFilePath,
	}
	t, err := template.New("haproxy").Funcs(funcMap).Parse(configTemplate)
	if err != nil {
		_ = configFile.Close()
		return nil, err
	}

	data := map[string]any{
		"EndpointID":    endpointID,
		"Ports":         si.ServicePorts,
		"UpstreamHost":  si.ServiceIPAddress,
		"Protocol":      si.ServiceProtocol,
		"ProxyProtocol": si.ProxyProtocol,
		"ChrootDir":     proxy.GetNetworkDir(endpointID),
		"LogLevel":      haproxyLogLevel(),
		"RunUser":       config.Global.Agent.RunUser,
		"RunGroup":      config.Global.Agent.RunGroup,
	}
	if err = t.Execute(configFile, data); err != nil {
		_ = configFile.Close()
		return nil, err
	}
	return configFile, nil
}

func (h *HAProxyController) RemoveInstance(endpointID string) error {
	instance, ok := h.instances[endpointID]
	if !ok {
		return fmt.Errorf("instance '%s' not found", endpointID)
	}

	if err := syscall.Kill(instance.pid, syscall.SIGTERM); err != nil {
		return err
	}

	TryRemoveFile(instance.config.Name())
	TryRemoveFile(GetPidFilePath(endpointID))

	delete(h.instances, endpointID)
	return nil
}

func (h *HAProxyController) Run(ctx context.Context) {
	<-ctx.Done()
	log.Debug("Shutting down HAProxy instances...")
	for endpointID := range h.instances {
		if err := h.RemoveInstance(endpointID); err != nil {
			log.Errorf("Failed to remove instance '%s': %s", endpointID, err)
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

func GetStatsSocketPath(endpointID string) string {
	return fmt.Sprintf("%s/haproxy-stats.sock", proxy.GetNetworkDir(endpointID))
}

func GetPidFilePath(endpointID string) string {
	return fmt.Sprintf("%s/haproxy.pid", proxy.GetNetworkDir(endpointID))
}

func GetConfigFilePath(endpointID string) string {
	return fmt.Sprintf("%s/haproxy.conf", proxy.GetNetworkDir(endpointID))
}
