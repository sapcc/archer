// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package f5os

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
