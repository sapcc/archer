// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package notifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestProviderClient(server *httptest.Server) *gophercloud.ProviderClient {
	return &gophercloud.ProviderClient{
		TokenID:    "test-token-123",
		HTTPClient: *server.Client(),
	}
}

func TestCampfireClient_SendEmail(t *testing.T) {
	var receivedReq CampfireRequest
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("X-Auth-Token")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedReq)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	providerClient := newTestProviderClient(server)
	client := NewCampfireClient(server.URL+"/v1/send-email?from=archer", providerClient)
	err := client.SendEmail(context.Background(), &CampfireRequest{
		ProjectID: "project-123",
		Subject:   "Test Subject",
		MimeType:  "text/plain",
		MailText:  "Hello",
	})

	require.NoError(t, err)
	assert.Equal(t, "test-token-123", receivedAuth)
	assert.Equal(t, "project-123", receivedReq.ProjectID)
	assert.Equal(t, "Test Subject", receivedReq.Subject)
	assert.Equal(t, "text/plain", receivedReq.MimeType)
	assert.Equal(t, "Hello", receivedReq.MailText)
}

func TestCampfireClient_SendEmail_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("no recipients"))
	}))
	defer server.Close()

	providerClient := newTestProviderClient(server)
	client := NewCampfireClient(server.URL+"/v1/send-email?from=archer", providerClient)
	err := client.SendEmail(context.Background(), &CampfireRequest{
		ProjectID: "project-123",
		Subject:   "Test",
		MimeType:  "text/plain",
		MailText:  "Hello",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "418")
}
