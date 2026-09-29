// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package restapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/archer/v2/internal/config"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func fireRequests(t *testing.T, handler http.Handler, req *http.Request, n int) (n200, n429 int) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := &http.Client{}
	for range n {
		clone := req.Clone(req.Context())
		clone.RequestURI = ""
		u, err := clone.URL.Parse(srv.URL + req.URL.Path)
		require.NoError(t, err)
		clone.URL = u
		res, err := client.Do(clone)
		require.NoError(t, err)
		_ = res.Body.Close()
		switch res.StatusCode {
		case http.StatusOK:
			n200++
		case http.StatusTooManyRequests:
			n429++
		}
	}
	return
}

func TestSetupMiddlewares_RateLimitDisabled(t *testing.T) {
	config.Global.ApiSettings.RateLimit = 0
	t.Cleanup(func() { config.Global.ApiSettings.RateLimit = 0 })

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	n200, n429 := fireRequests(t, setupMiddlewares(okHandler), req, 10)
	assert.Equal(t, 10, n200)
	assert.Equal(t, 0, n429)
}

func TestSetupMiddlewares_RateLimitBehindProxy(t *testing.T) {
	config.Global.ApiSettings.RateLimit = 2
	config.Global.ApiSettings.EnableProxyHeadersParsing = true
	t.Cleanup(func() {
		config.Global.ApiSettings.RateLimit = 0
		config.Global.ApiSettings.EnableProxyHeadersParsing = false
	})

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	req.Header.Set("X-Auth-Token", "test-token")
	req.Header.Set("X-Forwarded-For", "192.0.2.1")

	_, n429 := fireRequests(t, setupMiddlewares(okHandler), req, 10)
	// burst=2 lets the first two through, the rest must be rejected
	assert.Equal(t, 8, n429)
}

func TestSetupMiddlewares_RateLimitDirect(t *testing.T) {
	config.Global.ApiSettings.RateLimit = 2
	config.Global.ApiSettings.EnableProxyHeadersParsing = false
	t.Cleanup(func() { config.Global.ApiSettings.RateLimit = 0 })

	// No proxy: RemoteAddr is used as key; httptest sets it automatically.
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	req.Header.Set("X-Auth-Token", "test-token")

	_, n429 := fireRequests(t, setupMiddlewares(okHandler), req, 10)
	assert.Equal(t, 8, n429)
}
