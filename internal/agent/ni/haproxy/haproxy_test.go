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
	"github.com/sapcc/archer/v2/internal/config"
)

func setupHaproxyTempDir(t *testing.T) func() {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "haproxy-test-*")
	require.NoError(t, err)
	config.Global.Agent.TempDir = tmpDir
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
			configFile, err := os.Create(configPath)
			require.NoError(t, err)

			funcMap := template.FuncMap{
				"lower":              strings.ToLower,
				"formatHost":         formatHost,
				"getSocketPath":      func(serviceID string, port int) string { return "/tmp/test.sock" },
				"getStatsSocketPath": GetStatsSocketPath,
				"getPidFilePath":     GetPidFilePath,
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
	configFile, err := os.Create(configPath)
	require.NoError(t, err)

	funcMap := template.FuncMap{
		"lower":              strings.ToLower,
		"formatHost":         formatHost,
		"getSocketPath":      func(serviceID string, port int) string { return "/tmp/test.sock" },
		"getStatsSocketPath": GetStatsSocketPath,
		"getPidFilePath":     GetPidFilePath,
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
	configFile, err := os.Create(configPath)
	require.NoError(t, err)

	funcMap := template.FuncMap{
		"lower":              strings.ToLower,
		"formatHost":         formatHost,
		"getSocketPath":      func(serviceID string, port int) string { return "/tmp/test.sock" },
		"getStatsSocketPath": GetStatsSocketPath,
		"getPidFilePath":     GetPidFilePath,
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
