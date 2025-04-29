// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package ni

import (
	"bytes"
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
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
)

type haProxyInstance struct {
	cmd    *exec.Cmd
	config *os.File
	client *haproxy.HAProxyClient
	pid    int
}

type HAProxyController struct {
	instances map[string]*haProxyInstance
	tempdir   string
}

var configTemplate = `
global
    log         stdout format raw local0
    stats       socket "{{.TempDir}}haproxy-stats-{{.Network}}.sock"
    maxconn     1024
    pidfile     "{{.TempDir}}haproxy-{{.Network}}.pid"
    user haproxy
    group haproxy
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
    default_backend         upstream

frontend downstream
    bind *:80
	mode {{lower .Protocol}}

backend upstream
    mode {{lower .Protocol}}
{{- if eq .Protocol "HTTP" }}
    http-request replace-header Host .* {{.UpstreamHost}}
{{- end }}
    server upstream {{.ProxyPath}}
`

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
	tempdir := os.TempDir()
	if tempdir[len(tempdir)-1:] != "/" {
		tempdir = tempdir + "/"
	}

	return &HAProxyController{
		instances: make(map[string]*haProxyInstance),
		tempdir:   tempdir,
	}
}

func (h *HAProxyController) collectStats() {
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

func (h *HAProxyController) isRunning(networkID string) bool {
	_, ok := h.instances[networkID]
	if !ok {
		return false
	}

	// read pid and check if process exists
	pid, err := readPidFile(fmt.Sprintf("%shaproxy-%s.pid", h.tempdir, networkID))
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

func (h *HAProxyController) addInstance(networkID string, protocol string) (*haProxyInstance, error) {
	// create config
	filename := fmt.Sprintf("%shaproxy-%s.conf", h.tempdir, networkID)
	configFile, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	defer func() { _ = configFile.Close() }()
	log.Debugf("Created HAProxy config file '%s'", configFile.Name())

	funcMap := template.FuncMap{
		"lower": strings.ToLower,
	}

	// template config
	t, err := template.New("haproxy").Funcs(funcMap).Parse(configTemplate)
	if err != nil {
		tryRemoveFile(configFile.Name())
		return nil, err
	}
	data := map[string]string{
		"TempDir":      h.tempdir,
		"ProxyPath":    config.Global.Agent.ServiceProxyPath,
		"UpstreamHost": config.Global.Agent.ServiceUpstreamHost,
		"Network":      networkID,
		"Protocol":     protocol,
	}
	if err := t.Execute(configFile, data); err != nil {
		tryRemoveFile(configFile.Name())
		return nil, err
	}

	outfile, err := os.Create(getLogFilePath(h.tempdir, networkID))
	if err != nil {
		panic(err)
	}
	defer func() { _ = outfile.Close() }()

	// run haproxy
	cmd := exec.Command("haproxy", "-f", configFile.Name())
	cmd.Stdout = outfile
	cmd.Stderr = outfile
	if err := cmd.Run(); err != nil {
		tryRemoveFile(configFile.Name())
		return nil, err
	}

	// read pid
	pid, err := readPidFile(fmt.Sprintf("%shaproxy-%s.pid", h.tempdir, networkID))
	if err != nil {
		tryRemoveFile(configFile.Name())
		return nil, err
	}

	// init haproxy stats client
	haProxyClient := haproxy.HAProxyClient{
		Addr: fmt.Sprintf("unix://%shaproxy-stats-%s.sock", h.tempdir, networkID),
	}
	info, err := haProxyClient.Info()
	if err != nil {
		tryRemoveFile(configFile.Name())
		return nil, err
	}
	log.Printf("Running %s version %s PID %d for %s\n", info.Name, info.Version, pid, networkID)

	instance := haProxyInstance{
		cmd:    cmd,
		config: configFile,
		client: &haProxyClient,
		pid:    pid,
	}

	h.instances[networkID] = &instance
	return &instance, nil
}

func (h *HAProxyController) removeInstance(networkID string) error {
	instance, ok := h.instances[networkID]
	if !ok {
		return fmt.Errorf("instance '%s' not found", networkID)
	}

	// Terminate haproxy
	if err := syscall.Kill(instance.pid, syscall.SIGTERM); err != nil {
		return err
	}

	// Remove config and pidfile
	tryRemoveFile(instance.config.Name())
	tryRemoveFile(fmt.Sprintf("%shaproxy-%s.pid", h.tempdir, networkID))

	delete(h.instances, networkID)
	return nil
}

func (h *HAProxyController) dumpLog(networkID string) {
	logFile := getLogFilePath(h.tempdir, networkID)
	d, err := os.ReadFile(logFile)
	if err != nil {
		log.Errorf("Failed to reading log file(path=%s): %s", logFile, err)
	}
	log.Print(string(d))
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

func tryRemoveFile(file string) {
	if err := os.Remove(file); err != nil {
		log.Print(err)
	}
}

func getLogFilePath(tempdir string, networkID string) string {
	return fmt.Sprintf("%shaproxy-%s.log", tempdir, networkID)
}
