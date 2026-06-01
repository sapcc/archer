// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInetAddress_ScanNetipPrefix(t *testing.T) {
	tests := []struct {
		name     string
		prefix   netip.Prefix
		expected InetAddress
	}{
		{"IPv4 host", netip.MustParsePrefix("1.2.3.4/32"), "1.2.3.4"},
		{"IPv6 host", netip.MustParsePrefix("2001:db8::1/128"), "2001:db8::1"},
		{"IPv4 network", netip.MustParsePrefix("10.0.0.0/24"), "10.0.0.0"},
		{"zero prefix", netip.Prefix{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var addr InetAddress
			err := addr.ScanNetipPrefix(tt.prefix)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, addr)
		})
	}
}

func TestInetAddress_NetipPrefixValue(t *testing.T) {
	tests := []struct {
		name     string
		addr     InetAddress
		expected netip.Prefix
		wantErr  bool
	}{
		{"IPv4", "1.2.3.4", netip.MustParsePrefix("1.2.3.4/32"), false},
		{"IPv6", "2001:db8::1", netip.MustParsePrefix("2001:db8::1/128"), false},
		{"empty", "", netip.Prefix{}, false},
		{"invalid", "not-an-ip", netip.Prefix{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.addr.NetipPrefixValue()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestInetAddress_Validate(t *testing.T) {
	tests := []struct {
		name    string
		addr    InetAddress
		wantErr bool
	}{
		{"valid IPv4", "1.2.3.4", false},
		{"valid IPv6", "2001:db8::1", false},
		{"valid IPv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"valid loopback", "::1", false},
		{"empty", "", true},
		{"invalid garbage", "not-an-ip", true},
		{"CIDR notation", "1.2.3.4/32", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.addr.Validate(nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
