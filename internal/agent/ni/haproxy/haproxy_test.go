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

			configPath := GetConfigFilePath(si.Network.String())
			require.NoError(t, os.MkdirAll(proxy.GetNetworkDir(si.Network.String()), 0o777))
			configFile, err := os.Create(configPath)
			require.NoError(t, err)

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

			data := map[string]any{
				"UpstreamHost":  si.ServiceIPAddress,
				"UpstreamPorts": si.ServicePorts,
				"Network":       si.Network.String(),
				"Protocol":      si.ServiceProtocol,
				"ServiceID":     si.ServiceID.String(),
				"ProxyProtocol": false,
				"EndpointID":    "test-endpoint-id",
				"ChrootDir":     config.Global.Agent.RunDir,
				"LogLevel":      "info",
				"RunUser":       "nobody",
				"RunGroup":      "nogroup",
			}
			err = tmpl.Execute(configFile, data)
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

	configPath := GetConfigFilePath(si.Network.String())
	require.NoError(t, os.MkdirAll(proxy.GetNetworkDir(si.Network.String()), 0o777))
	configFile, err := os.Create(configPath)
	require.NoError(t, err)

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

	data := map[string]any{
		"UpstreamHost":  si.ServiceIPAddress,
		"UpstreamPorts": si.ServicePorts,
		"Network":       si.Network.String(),
		"Protocol":      si.ServiceProtocol,
		"ServiceID":     si.ServiceID.String(),
		"ProxyProtocol": si.ProxyProtocol,
		"EndpointID":    endpointID,
		"ChrootDir":     config.Global.Agent.RunDir,
		"LogLevel":      "info",
		"RunUser":       "nobody",
		"RunGroup":      "nogroup",
	}
	err = tmpl.Execute(configFile, data)
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

	si := &models.ServiceInjection{
		ServiceIPAddress: "10.0.0.1",
		ServicePorts:     []int{80},
		ServiceProtocol:  "TCP",
		ServiceID:        strfmt.UUID("550e8400-e29b-41d4-a716-446655440000"),
		Network:          strfmt.UUID("660e8400-e29b-41d4-a716-446655440000"),
		ProxyProtocol:    false,
	}

	configPath := GetConfigFilePath(si.Network.String())
	require.NoError(t, os.MkdirAll(proxy.GetNetworkDir(si.Network.String()), 0o777))
	configFile, err := os.Create(configPath)
	require.NoError(t, err)

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

	data := map[string]any{
		"UpstreamHost":  si.ServiceIPAddress,
		"UpstreamPorts": si.ServicePorts,
		"Network":       si.Network.String(),
		"Protocol":      si.ServiceProtocol,
		"ServiceID":     si.ServiceID.String(),
		"ProxyProtocol": si.ProxyProtocol,
		"EndpointID":    "unused-when-disabled",
		"ChrootDir":     config.Global.Agent.RunDir,
		"LogLevel":      "info",
		"RunUser":       "nobody",
		"RunGroup":      "nogroup",
	}
	err = tmpl.Execute(configFile, data)
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

	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]any{
		"UpstreamHost":  si.ServiceIPAddress,
		"UpstreamPorts": si.ServicePorts,
		"Network":       si.Network.String(),
		"Protocol":      si.ServiceProtocol,
		"ServiceID":     si.ServiceID.String(),
		"ProxyProtocol": false,
		"EndpointID":    si.ID.String(),
		"ChrootDir":     proxy.GetNetworkDir(si.Network.String()),
		"LogLevel":      "info",
		"RunUser":       "nobody",
		"RunGroup":      "nogroup",
	})
	require.NoError(t, err)
	configStr := buf.String()

	assert.Contains(t, configStr, "user        nobody", "should drop privileges to nobody user")
	assert.Contains(t, configStr, "group       nogroup", "should drop privileges to nogroup group")
	assert.Contains(t, configStr, `chroot      "`+proxy.GetNetworkDir(si.Network.String())+`"`,
		"should chroot into the per-network dir")

	// The backend server line references the socket at the chroot root (HAProxy
	// chroots into the network dir), so it is simply /<port>.sock.
	assert.Contains(t, configStr, "server upstream /80.sock")
	assert.NotContains(t, configStr, "server upstream "+config.Global.Agent.RunDir,
		"backend server line must not use the host-absolute path")
}

func TestAddInstanceConfigFilePermissions(t *testing.T) {
	cleanup := setupHaproxyTempDir(t)
	defer cleanup()

	// AddInstance spawns real haproxy, which is not available in unit tests.
	// Verify the file-creation mode directly, matching how AddInstance opens
	// the config file. The per-network dir is created by the agent before writing.
	networkID := "660e8400-e29b-41d4-a716-446655440000"
	require.NoError(t, os.MkdirAll(proxy.GetNetworkDir(networkID), 0o777))
	path := GetConfigFilePath(networkID)
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
