// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package haproxy

import (
	"os"
	"strings"
	"testing"
	"text/template"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/archer/v2/internal/agent/ni/models"
	"github.com/sapcc/archer/v2/internal/agent/ni/proxy"
	"github.com/sapcc/archer/v2/internal/config"
)

func setupHaproxyTempDir(t *testing.T) func() {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "haproxy-test-*")
	require.NoError(t, err)
	config.Global.Agent.RunDir = tmpDir
	config.Global.Agent.RunUser = "nobody"
	config.Global.Agent.RunGroup = "nogroup"
	return func() {
		_ = os.RemoveAll(tmpDir)
	}
}

func templateData(si *models.ServiceInjection, endpointID string) map[string]any {
	return map[string]any{
		"EndpointID":    endpointID,
		"Ports":         si.ServicePorts,
		"UpstreamHost":  si.ServiceIPAddress,
		"Protocol":      si.ServiceProtocol,
		"ProxyProtocol": si.ProxyProtocol,
		"ChrootDir":     config.Global.Agent.RunDir,
		"LogLevel":      "info",
		"RunUser":       "nobody",
		"RunGroup":      "nogroup",
	}
}

func newTestTemplate(t *testing.T) *template.Template {
	t.Helper()
	funcMap := template.FuncMap{
		"lower":               strings.ToLower,
		"formatHost":          formatHost,
		"getSocketPath":       func(serviceID string, port int) string { return "/tmp/test.sock" },
		"getChrootSocketPath": func(port int) string { return "/test.sock" },
		"getStatsSocketPath":  GetStatsSocketPath,
		"getPidFilePath":      GetPidFilePath,
	}
	tmpl, err := template.New("haproxy").Funcs(funcMap).Parse(configTemplate)
	require.NoError(t, err)
	return tmpl
}

func TestConfigTemplate_IPv6BracketRendering(t *testing.T) {
	cleanup := setupHaproxyTempDir(t)
	defer cleanup()

	tests := []struct {
		name     string
		ip       string
		protocol string
		wantHost string
	}{
		{"IPv4 HTTP", "10.0.0.1", "HTTP", "http-request replace-header Host .* 10.0.0.1"},
		{"IPv6 HTTP", "2001:db8::1", "HTTP", "http-request replace-header Host .* [2001:db8::1]"},
		{"IPv6 loopback HTTP", "::1", "HTTP", "http-request replace-header Host .* [::1]"},
		{"IPv4 TCP no host", "10.0.0.1", "TCP", ""},
		{"IPv6 TCP no host", "2001:db8::1", "TCP", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			si := &models.ServiceInjection{
				ServiceIPAddress: tt.ip,
				ServicePorts:     []int{80},
				ServiceProtocol:  tt.protocol,
				ServiceID:        strfmt.UUID("550e8400-e29b-41d4-a716-446655440000"),
				Network:          strfmt.UUID("660e8400-e29b-41d4-a716-446655440000"),
			}
			endpointID := "test-endpoint-id"

			configPath := GetConfigFilePath(endpointID)
			require.NoError(t, os.MkdirAll(proxy.GetNetworkDir(endpointID), 0o777))
			configFile, err := os.Create(configPath)
			require.NoError(t, err)

			tmpl := newTestTemplate(t)
			err = tmpl.Execute(configFile, templateData(si, endpointID))
			require.NoError(t, err)
			_ = configFile.Close()

			content, err := os.ReadFile(configPath)
			require.NoError(t, err)
			configStr := string(content)

			assert.Contains(t, configStr, "bind :::80 v4v6",
				"should bind dual-stack (IPv4+IPv6)")

			if tt.wantHost != "" {
				assert.Contains(t, configStr, tt.wantHost)
			} else {
				assert.NotContains(t, configStr, "http-request replace-header Host")
			}
		})
	}
}

func TestConfigTemplate_ProxyProtocolEnabled(t *testing.T) {
	cleanup := setupHaproxyTempDir(t)
	defer cleanup()

	endpointID := "3ad9b1f0-4e5a-44c3-ada6-71696925ae64"
	si := &models.ServiceInjection{
		ServiceIPAddress: "10.0.0.1",
		ServicePorts:     []int{80, 443},
		ServiceProtocol:  "TCP",
		ServiceID:        strfmt.UUID("550e8400-e29b-41d4-a716-446655440000"),
		Network:          strfmt.UUID("660e8400-e29b-41d4-a716-446655440000"),
		ProxyProtocol:    true,
	}

	configPath := GetConfigFilePath(endpointID)
	require.NoError(t, os.MkdirAll(proxy.GetNetworkDir(endpointID), 0o777))
	configFile, err := os.Create(configPath)
	require.NoError(t, err)

	tmpl := newTestTemplate(t)
	err = tmpl.Execute(configFile, templateData(si, endpointID))
	require.NoError(t, err)
	_ = configFile.Close()

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	configStr := string(content)

	assert.Contains(t, configStr, "send-proxy-v2")
	assert.Contains(t, configStr, "set-proxy-v2-tlv-fmt(0xEC) %[str("+endpointID+")]")
	assert.Equal(t, 2, strings.Count(configStr, "send-proxy-v2"),
		"should have send-proxy-v2 for each backend server line")
}

func TestConfigTemplate_ProxyProtocolDisabled(t *testing.T) {
	cleanup := setupHaproxyTempDir(t)
	defer cleanup()

	endpointID := "test-endpoint-disabled"
	si := &models.ServiceInjection{
		ServiceIPAddress: "10.0.0.1",
		ServicePorts:     []int{80},
		ServiceProtocol:  "TCP",
		ServiceID:        strfmt.UUID("550e8400-e29b-41d4-a716-446655440000"),
		Network:          strfmt.UUID("660e8400-e29b-41d4-a716-446655440000"),
		ProxyProtocol:    false,
	}

	configPath := GetConfigFilePath(endpointID)
	require.NoError(t, os.MkdirAll(proxy.GetNetworkDir(endpointID), 0o777))
	configFile, err := os.Create(configPath)
	require.NoError(t, err)

	tmpl := newTestTemplate(t)
	err = tmpl.Execute(configFile, templateData(si, endpointID))
	require.NoError(t, err)
	_ = configFile.Close()

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	configStr := string(content)

	assert.NotContains(t, configStr, "send-proxy-v2")
	assert.NotContains(t, configStr, "set-proxy-v2-tlv-fmt")
}

func TestConfigTemplate_Hardening(t *testing.T) {
	cleanup := setupHaproxyTempDir(t)
	defer cleanup()

	endpointID := "384678f1-0ca4-4ce1-bc66-047053f629dc"
	si := &models.ServiceInjection{
		ServiceIPAddress: "10.0.0.1",
		ServicePorts:     []int{80},
		ServiceProtocol:  "TCP",
		ServiceID:        strfmt.UUID("550e8400-e29b-41d4-a716-446655440000"),
		Network:          strfmt.UUID("660e8400-e29b-41d4-a716-446655440000"),
	}

	funcMap := template.FuncMap{
		"lower":               strings.ToLower,
		"formatHost":          formatHost,
		"getSocketPath":       proxy.GetSocketPath,
		"getChrootSocketPath": func(port int) string { return "/80.sock" },
		"getStatsSocketPath":  GetStatsSocketPath,
		"getPidFilePath":      GetPidFilePath,
	}

	tmpl, err := template.New("haproxy").Funcs(funcMap).Parse(configTemplate)
	require.NoError(t, err)

	data := map[string]any{
		"EndpointID":    endpointID,
		"Ports":         si.ServicePorts,
		"UpstreamHost":  si.ServiceIPAddress,
		"Protocol":      si.ServiceProtocol,
		"ProxyProtocol": false,
		"ChrootDir":     proxy.GetNetworkDir(endpointID),
		"LogLevel":      "info",
		"RunUser":       "nobody",
		"RunGroup":      "nogroup",
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, data)
	require.NoError(t, err)
	configStr := buf.String()

	assert.Contains(t, configStr, "user        nobody", "should drop privileges to nobody user")
	assert.Contains(t, configStr, "group       nogroup", "should drop privileges to nogroup group")
	assert.Contains(t, configStr, `chroot      "`+proxy.GetNetworkDir(endpointID)+`"`,
		"should chroot into the per-endpoint dir")

	assert.Contains(t, configStr, "server upstream /80.sock")
	assert.NotContains(t, configStr, "server upstream "+config.Global.Agent.RunDir,
		"backend server line must not use the host-absolute path")
}

func TestAddInstanceConfigFilePermissions(t *testing.T) {
	cleanup := setupHaproxyTempDir(t)
	defer cleanup()

	endpointID := "384678f1-0ca4-4ce1-bc66-047053f629dc"
	require.NoError(t, os.MkdirAll(proxy.GetNetworkDir(endpointID), 0o777))
	path := GetConfigFilePath(endpointID)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(),
		"config file must be created with 0600 permissions")
}

func TestHaproxyLogLevel(t *testing.T) {
	orig := config.Global.Default.Debug
	defer func() { config.Global.Default.Debug = orig }()

	config.Global.Default.Debug = false
	assert.Equal(t, "info", haproxyLogLevel(), "default agent verbosity maps to info")

	config.Global.Default.Debug = true
	assert.Equal(t, "debug", haproxyLogLevel(), "agent --debug maps to debug")
}
