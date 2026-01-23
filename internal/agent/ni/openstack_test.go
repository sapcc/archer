// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ni

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2"
	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/gophercloud/gophercloud/v2/testhelper/fixture"
	"github.com/sapcc/archer/internal/agent/ni/haproxy"
	"github.com/sapcc/archer/internal/agent/ni/models"
	"github.com/sapcc/archer/internal/config"
	"github.com/stretchr/testify/assert"
)

// setupTestServer creates a test HTTP server for mocking OpenStack API
func setupTestServer(handler http.Handler) (*httptest.Server, *gophercloud.ServiceClient) {
	server := httptest.NewServer(handler)
	client := &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{
			TokenID: "test-token",
		},
		Endpoint: server.URL + "/",
	}
	return server, client
}

func TestAgent_SetupOpenStack(t *testing.T) {
	// Setup mock HTTP handler for auth
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		// read and decode http body
		readall, _ := io.ReadAll(r.Body)
		assert.JSONEq(t, `{
			"auth": {
				"identity": {
					"methods": ["password"],
					"password": {
						"user": {
							"name": "test-username",
							"domain": { "name": "test-domain-name" },
							"password": "test-password"
						}
					}
				},
				"scope": {
					"project": {
						"name": "test-project-name",
						"domain": { "name": "test-project-domain-name" }
					}
				}
			}
		}`, string(readall))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Subject-Token", "test-token")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"token": {
				"catalog": [{
					"type": "network",
					"endpoints": [{
						"interface": "public",
						"url": "http://` + r.Host + `/network/v2.0"
					},{
						"interface": "internal",
						"url": "http://` + r.Host + `/network/v2.0"
					},{
						"interface": "admin",
						"url": "http://` + r.Host + `/network/v2.0"
					}]
				}]
			}
		}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	config.Global.ServiceAuth.AuthURL = ts.URL + "/v3"
	config.Global.ServiceAuth.Password = "test-password"
	config.Global.ServiceAuth.Username = "test-username"
	config.Global.ServiceAuth.UserDomainName = "test-domain-name"
	config.Global.ServiceAuth.DomainName = "test-domain-name"
	config.Global.ServiceAuth.ProjectName = "test-project-name"
	config.Global.ServiceAuth.ProjectDomainName = "test-project-domain-name"

	a := Agent{}
	for _, ep := range []string{"public", "internal", "admin"} {
		config.Global.Default.EndpointType = ep
		assert.NoError(t, a.SetupOpenStack())
	}
}

func TestAgent_EnableInjection_PortNotFound(t *testing.T) {
	portID := "550e8400-e29b-41d4-a716-446655440000"

	// Setup mock HTTP handler to return 404 for port get
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && (r.URL.Path == "/ports/"+portID || r.URL.Path == "/v2.0/ports/"+portID) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprintf(w, `{"NeutronError": {"type": "PortNotFound", "message": "Port not found", "detail": ""}}`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}

	server, client := setupTestServer(http.HandlerFunc(handler))
	defer server.Close()

	agent := &Agent{
		neutron: client,
	}

	networkID := strfmt.UUID("660e8400-e29b-41d4-a716-446655440000")
	ipAddress := strfmt.IPv4("192.168.1.100")

	si := &models.ServiceInjection{
		PortId:          strfmt.UUID(portID),
		Network:         networkID,
		IpAddress:       ipAddress,
		ServicePorts:    []int{80},
		ServiceProtocol: "tcp",
	}

	err := agent.EnableInjection(si)
	assert.Error(t, err)
}

func TestAgent_EnableInjection_Success(t *testing.T) {
	fakeServer := th.SetupHTTP()
	defer fakeServer.Teardown()

	portFixture := `{
		"port": {
			"id": "550e8400-e29b-41d4-a716-446655440000",
			"network_id": "660e8400-e29b-41d4-a716-446655440000",
			"tenant_id": "26a7980765d0414dbc1fc1f88cdb7e6e"
        }
}`

	a := &Agent{
		neutron: fake.ServiceClient(fakeServer),
		haproxy: haproxy.NewFakeHaproxy(),
	}
	si := &models.ServiceInjection{
		PortId:    strfmt.UUID("550e8400-e29b-41d4-a716-446655440000"),
		Network:   strfmt.UUID("660e8400-e29b-41d4-a716-446655440000"),
		IpAddress: strfmt.IPv4("1.2.3.4"),
	}
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports/"+si.PortId.String(), "GET",
		"", portFixture, http.StatusOK)

	assert.NoError(t, a.EnableInjection(si))

	a.haproxy.(*haproxy.FakeHaproxy).AddInstanceReturnError = fmt.Errorf("haproxy error")
	assert.NoError(t, a.EnableInjection(si))

	a.haproxy.(*haproxy.FakeHaproxy).Running = true
	assert.NoError(t, a.EnableInjection(si))
}

func TestAgent_DisableInjection_PortNotFound(t *testing.T) {
	portID := "550e8400-e29b-41d4-a716-446655440000"

	// Setup mock HTTP handler to return 404 for port get
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && (r.URL.Path == "/ports/"+portID || r.URL.Path == "/v2.0/ports/"+portID) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprintf(w, `{"NeutronError": {"type": "PortNotFound", "message": "Port not found", "detail": ""}}`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}

	server, client := setupTestServer(http.HandlerFunc(handler))
	defer server.Close()

	agent := &Agent{
		neutron: client,
	}

	networkID := strfmt.UUID("660e8400-e29b-41d4-a716-446655440000")
	ipAddress := strfmt.IPv4("192.168.1.100")

	si := &models.ServiceInjection{
		PortId:          strfmt.UUID(portID),
		Network:         networkID,
		IpAddress:       ipAddress,
		ServicePorts:    []int{80},
		ServiceProtocol: "tcp",
	}

	err := agent.DisableInjection(si)
	assert.Error(t, err)
}

func TestAgent_DisableInjection(t *testing.T) {
	fakeServer := th.SetupHTTP()
	defer fakeServer.Teardown()

	portFixture := `{
		"port": {
			"id": "550e8400-e29b-41d4-a716-446655440000",
			"network_id": "660e8400-e29b-41d4-a716-446655440000",
			"tenant_id": "26a7980765d0414dbc1fc1f88cdb7e6e"
        }
}`

	a := &Agent{
		neutron: fake.ServiceClient(fakeServer),
		haproxy: haproxy.NewFakeHaproxy(),
	}
	si := &models.ServiceInjection{
		PortId:    strfmt.UUID("550e8400-e29b-41d4-a716-446655440000"),
		Network:   strfmt.UUID("660e8400-e29b-41d4-a716-446655440000"),
		IpAddress: strfmt.IPv4("1.2.3.4"),
	}
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports/"+si.PortId.String(), "GET",
		"", portFixture, http.StatusOK)

	assert.NoError(t, a.DisableInjection(si))

	a.haproxy.(*haproxy.FakeHaproxy).Running = true
	assert.NoError(t, a.DisableInjection(si))
}

func TestAgent_CollectStats(t *testing.T) {
	agent := &Agent{
		haproxy: haproxy.NewHAProxyController(),
	}

	// Should not panic when collecting stats
	assert.NotPanics(t, func() {
		agent.CollectStats()
	})
}

func TestAgent_EnableInjection_GetPortSuccess(t *testing.T) {
	portID := "550e8400-e29b-41d4-a716-446655440000"
	networkID := "660e8400-e29b-41d4-a716-446655440000"

	// Setup mock HTTP handler to return a valid port
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Match both /ports/xxx and /v2.0/ports/xxx patterns
		if r.Method == "GET" && (r.URL.Path == "/ports/"+portID || r.URL.Path == "/v2.0/ports/"+portID) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{
				"port": {
					"id": "%s",
					"network_id": "%s",
					"tenant_id": "26a7980765d0414dbc1fc1f88cdb7e6e",
					"mac_address": "fa:16:3e:c9:cb:f0",
					"fixed_ips": [
						{
							"subnet_id": "a0304c3a-4f08-4c43-88af-d796509c97d2",
							"ip_address": "192.168.1.100"
						}
					],
					"status": "ACTIVE",
					"admin_state_up": true,
					"device_id": "network-injector",
					"device_owner": "network:injector"
				}
			}`, portID, networkID)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}

	server, client := setupTestServer(http.HandlerFunc(handler))
	defer server.Close()

	agent := &Agent{
		neutron: client,
	}

	// Test that port can be retrieved successfully
	port, err := ports.Get(context.Background(), agent.neutron, portID).Extract()
	assert.NoError(t, err)
	assert.NotNil(t, port)
	assert.Equal(t, portID, port.ID)
	assert.Equal(t, networkID, port.NetworkID)
	if len(port.FixedIPs) > 0 {
		assert.Equal(t, "192.168.1.100", port.FixedIPs[0].IPAddress)
	}
}

func TestAgent_DisableInjection_GetPortSuccess(t *testing.T) {
	portID := "550e8400-e29b-41d4-a716-446655440000"
	networkID := "660e8400-e29b-41d4-a716-446655440000"

	// Setup mock HTTP handler to return a valid port
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Match both /ports/xxx and /v2.0/ports/xxx patterns
		if r.Method == "GET" && (r.URL.Path == "/ports/"+portID || r.URL.Path == "/v2.0/ports/"+portID) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{
				"port": {
					"id": "%s",
					"network_id": "%s",
					"tenant_id": "26a7980765d0414dbc1fc1f88cdb7e6e",
					"mac_address": "fa:16:3e:c9:cb:f0",
					"fixed_ips": [
						{
							"subnet_id": "a0304c3a-4f08-4c43-88af-d796509c97d2",
							"ip_address": "192.168.1.100"
						}
					],
					"status": "ACTIVE",
					"admin_state_up": true,
					"device_id": "network-injector",
					"device_owner": "network:injector"
				}
			}`, portID, networkID)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}

	server, client := setupTestServer(http.HandlerFunc(handler))
	defer server.Close()

	agent := &Agent{
		neutron: client,
	}

	// Test validation - port can be retrieved
	port, err := ports.Get(context.Background(), agent.neutron, portID).Extract()
	assert.NoError(t, err)
	assert.NotNil(t, port)
	assert.Equal(t, portID, port.ID)
}
