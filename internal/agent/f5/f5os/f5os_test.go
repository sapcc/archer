// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package f5os

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// makeJWT builds a minimal unsigned JWT whose payload carries the given exp.
func makeJWT(exp int64) token {
	payload, _ := json.Marshal(map[string]any{"exp": exp})
	enc := base64.RawURLEncoding.EncodeToString
	return token(fmt.Sprintf("%s.%s.%s", enc([]byte(`{"alg":"none"}`)), enc(payload), enc([]byte("sig"))))
}

func TestToken_Valid(t *testing.T) {
	now := time.Now()
	// Far in the future: valid.
	assert.True(t, makeJWT(now.Add(time.Hour).Unix()).Valid())
	// Already expired: invalid.
	assert.False(t, makeJWT(now.Add(-time.Minute).Unix()).Valid())
	// Within the skew window (expires in 30s < 60s skew): treated as invalid so we re-auth early.
	assert.False(t, makeJWT(now.Add(30*time.Second).Unix()).Valid())
	// Malformed token: invalid.
	assert.False(t, token("not-a-jwt").Valid())
}

func TestF5os_apiCall(t *testing.T) {
	// generate a test server so we can capture and inspect the request
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Auth-Token") == "invalid-token" {
			res.WriteHeader(http.StatusUnauthorized)
			_, _ = res.Write([]byte("Unauthorized request"))
		} else {
			res.WriteHeader(http.StatusOK)
		}
	}))
	defer testServer.Close()

	f5 := F5OS{
		client:   testServer.Client(),
		user:     "test-user",
		password: "test-password",
		token:    "invalid-token",
	}
	req, err := http.NewRequest(http.MethodGet, testServer.URL, nil)
	assert.NoError(t, err, "Failed to create HTTP request")
	assert.NoError(t, f5.apiCall(req, nil), "API call should not fail with unauthorized status")
}
