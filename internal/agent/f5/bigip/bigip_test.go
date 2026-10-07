// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package bigip

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gobigip "github.com/f5devcentral/go-bigip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestBigIP wires a BigIP instance against a test HTTP server.
func newTestBigIP(srv *httptest.Server) *BigIP {
	b := &gobigip.BigIP{
		Host:      srv.URL,
		Token:     "test-token",
		Transport: srv.Client().Transport.(*http.Transport),
		ConfigOptions: &gobigip.ConfigOptions{
			APICallRetries: 1,
		},
	}
	return (*BigIP)(b)
}

func TestEnsureVLAN_AlreadyExists(t *testing.T) {
	createCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/mgmt/tm/net/vlan":
			// Return an empty VLAN list — simulates no VLAN found yet.
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(gobigip.Vlans{})
		case r.Method == http.MethodPost && r.URL.Path == "/mgmt/tm/net/vlan":
			createCalled = true
			// Simulate F5 returning "already exists" error (concurrent creation).
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":    409,
				"message": "01020066:3: The requested VLAN (/Common/vlan-2933) already exists in partition Common.",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	b := newTestBigIP(srv)
	err := b.EnsureVLAN(2933, 1500)
	require.NoError(t, err, "already-exists error from CreateVlan must be treated as success")
	assert.True(t, createCalled, "CreateVlan should have been attempted")
}

func TestEnsureVLAN_CreatesWhenAbsent(t *testing.T) {
	createCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/mgmt/tm/net/vlan":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(gobigip.Vlans{})
		case r.Method == http.MethodPost && r.URL.Path == "/mgmt/tm/net/vlan":
			createCalled = true
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	b := newTestBigIP(srv)
	err := b.EnsureVLAN(2933, 1500)
	require.NoError(t, err)
	assert.True(t, createCalled)
}

func TestEnsureVLAN_SkipsCreateWhenPresent(t *testing.T) {
	createCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/mgmt/tm/net/vlan":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(gobigip.Vlans{
				Vlans: []gobigip.Vlan{{Name: "vlan-2933", Tag: 2933, MTU: 1500}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/mgmt/tm/net/vlan":
			createCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	b := newTestBigIP(srv)
	err := b.EnsureVLAN(2933, 1500)
	require.NoError(t, err)
	assert.False(t, createCalled, "CreateVlan must not be called when VLAN already exists in the list")
}
