// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	runtimeclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/archer/client"
)

func TestResolveServiceID_ValidUUID(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	result, err := ResolveServiceID(uuid)
	require.NoError(t, err)
	assert.Equal(t, strfmt.UUID(uuid), result)
}

func TestResolveServiceID_ByName(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/service", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"items": [
				{"id": "550e8400-e29b-41d4-a716-446655440000", "name": "my-service"},
				{"id": "550e8400-e29b-41d4-a716-446655440001", "name": "other-service"}
			]
		}`))
	}))
	defer server.Close()

	// Setup client
	rt := runtimeclient.New(server.Listener.Addr().String(), "/v1", []string{"http"})
	ArcherClient = client.New(rt, strfmt.Default)

	result, err := ResolveServiceID("my-service")
	require.NoError(t, err)
	assert.Equal(t, strfmt.UUID("550e8400-e29b-41d4-a716-446655440000"), result)
}

func TestResolveServiceID_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items": []}`))
	}))
	defer server.Close()

	rt := runtimeclient.New(server.Listener.Addr().String(), "/v1", []string{"http"})
	ArcherClient = client.New(rt, strfmt.Default)

	_, err := ResolveServiceID("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service not found: nonexistent")
}

func TestResolveServiceID_MultipleMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"items": [
				{"id": "550e8400-e29b-41d4-a716-446655440000", "name": "duplicate-name"},
				{"id": "550e8400-e29b-41d4-a716-446655440001", "name": "duplicate-name"}
			]
		}`))
	}))
	defer server.Close()

	rt := runtimeclient.New(server.Listener.Addr().String(), "/v1", []string{"http"})
	ArcherClient = client.New(rt, strfmt.Default)

	_, err := ResolveServiceID("duplicate-name")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple services found with name 'duplicate-name'")
}

func TestResolveEndpointID_ValidUUID(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	result, err := ResolveEndpointID(uuid)
	require.NoError(t, err)
	assert.Equal(t, strfmt.UUID(uuid), result)
}

func TestResolveEndpointID_ByName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/endpoint", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"items": [
				{"id": "550e8400-e29b-41d4-a716-446655440000", "name": "my-endpoint", "service_id": "00000000-0000-0000-0000-000000000001"},
				{"id": "550e8400-e29b-41d4-a716-446655440001", "name": "other-endpoint", "service_id": "00000000-0000-0000-0000-000000000001"}
			]
		}`))
	}))
	defer server.Close()

	rt := runtimeclient.New(server.Listener.Addr().String(), "/v1", []string{"http"})
	ArcherClient = client.New(rt, strfmt.Default)

	result, err := ResolveEndpointID("my-endpoint")
	require.NoError(t, err)
	assert.Equal(t, strfmt.UUID("550e8400-e29b-41d4-a716-446655440000"), result)
}

func TestResolveEndpointID_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items": []}`))
	}))
	defer server.Close()

	rt := runtimeclient.New(server.Listener.Addr().String(), "/v1", []string{"http"})
	ArcherClient = client.New(rt, strfmt.Default)

	_, err := ResolveEndpointID("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint not found: nonexistent")
}

func TestResolveEndpointID_MultipleMatches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"items": [
				{"id": "550e8400-e29b-41d4-a716-446655440000", "name": "duplicate-name", "service_id": "00000000-0000-0000-0000-000000000001"},
				{"id": "550e8400-e29b-41d4-a716-446655440001", "name": "duplicate-name", "service_id": "00000000-0000-0000-0000-000000000001"}
			]
		}`))
	}))
	defer server.Close()

	rt := runtimeclient.New(server.Listener.Addr().String(), "/v1", []string{"http"})
	ArcherClient = client.New(rt, strfmt.Default)

	_, err := ResolveEndpointID("duplicate-name")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple endpoints found with name 'duplicate-name'")
}

func TestResolveNetworkID_ValidUUID(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	result, err := ResolveNetworkID(uuid)
	require.NoError(t, err)
	assert.Equal(t, strfmt.UUID(uuid), result)
}

func TestResolveNetworkID_NoProvider(t *testing.T) {
	// Ensure Provider is nil
	Provider = nil

	_, err := ResolveNetworkID("my-network")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network name lookup requires authentication")
}
