// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package neutron

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2"
	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/gophercloud/gophercloud/v2/testhelper/fixture"
	"github.com/stretchr/testify/assert"

	aErrors "github.com/sapcc/archer/v2/internal/errors"
	"github.com/sapcc/archer/v2/models"
)

const allocProjectID = "test-project-1"
const allocEndpointID = strfmt.UUID("11111111-2222-3333-4444-555555555555")
const allocPortID = strfmt.UUID("66666666-7777-8888-9999-aaaaaaaaaaaa")

// getPortFixture is the canonical /v2.0/ports/{id} response body used by GetPort tests.
const getPortFixture = `
{
  "port": {
    "id": "66666666-7777-8888-9999-aaaaaaaaaaaa",
    "name": "test-port",
    "network_id": "9bf57c58-5d9f-418b-a879-44d83e194ad0"
  }
}
`

// allocSubnetFixture returns a subnet whose ID matches SubnetIDFixture and whose network_id
// matches NetworkIDFixture, used for the target_subnet branch of AllocateNeutronEndpointPort.
const allocSubnetFixture = `
{
  "subnet": {
    "id": "a0304c3a-4f08-4c43-88af-d796509c97d2",
    "network_id": "9bf57c58-5d9f-418b-a879-44d83e194ad0",
    "tenant_id": "test-project-1",
    "cidr": "10.0.0.0/24"
  }
}
`

// allocNetworkNoSubnetsFixture is GetNetworkResponseFixture with the subnets array emptied,
// used to drive the ErrMissingSubnets branch of AllocateNeutronEndpointPort.
const allocNetworkNoSubnetsFixture = `
{
  "network": {
    "id": "9bf57c58-5d9f-418b-a879-44d83e194ad0",
    "name": "no-subnets",
    "subnets": [],
    "segments": []
  }
}
`

// portJSON returns a Neutron port object for a target_port lookup. projectID is set so the
// caller can drive the project-mismatch branch; fixedIPs controls the missing-IP branch.
func portJSON(t *testing.T, projectID string, fixedIPs []map[string]string) string {
	t.Helper()
	body := fmt.Sprintf(`{"port":{"id":"%s","project_id":"%s","network_id":"%s","fixed_ips":[`,
		allocPortID, projectID, NetworkIDFixture)
	for i, fip := range fixedIPs {
		if i > 0 {
			body += ","
		}
		body += fmt.Sprintf(`{"subnet_id":"%s","ip_address":"%s"}`, fip["subnet_id"], fip["ip_address"])
	}
	body += `]}}`
	return body
}

// portFromUUID is a convenience to get a *strfmt.UUID from a literal.
func portFromUUID(s string) *strfmt.UUID {
	return new(strfmt.UUID(s))
}

func TestToPortListQuery(t *testing.T) {
	networkID := "test-network-id"
	subnetID := "test-subnet-id"
	hostID := "test-host-id"
	deviceOwner := "test-device-owner"

	opts := PortListOptsExt{
		HostID: hostID,
		ListOptsBuilder: ports.ListOpts{
			NetworkID:   networkID,
			FixedIPs:    []ports.FixedIPOpts{{SubnetID: subnetID}},
			DeviceOwner: deviceOwner,
		},
	}
	res, err := opts.ToPortListQuery()
	assert.Nil(t, err, "ToPortListQuery failed: %s", err)
	params := url.Values{}
	params.Add("binding:host_id", hostID)
	params.Add("device_owner", deviceOwner)
	params.Add("fixed_ips", fmt.Sprintf("subnet_id=%s", subnetID))
	params.Add("network_id", networkID)
	expected := fmt.Sprintf("?%s", params.Encode())
	assert.Equal(t, expected, res)
}

// TestUpdatePortBinding verifies that UpdatePortBinding issues a PUT to /v2.0/ports/{id}
// with binding:host_id set in the request body.
func TestUpdatePortBinding(t *testing.T) {
	const portID = "abc-123"
	const newHost = "host-b"

	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	var seenBody atomic.Value
	fakeServer.Mux.HandleFunc("/v2.0/ports/"+portID, func(w http.ResponseWriter, r *http.Request) {
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		assert.Equal(t, http.MethodPut, r.Method)
		body, _ := io.ReadAll(r.Body)
		seenBody.Store(string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"port":{"id":%q,"binding:host_id":%q}}`, portID, newHost)
	})

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	err := n.UpdatePortBinding(context.Background(), portID, newHost)
	assert.NoError(t, err)
	assert.Contains(t, seenBody.Load().(string), `"binding:host_id":"host-b"`)
}

// TestNeutronClient_UpdatePortBinding_InvalidatesCache covers cache busting on rebind.
func TestNeutronClient_UpdatePortBinding_InvalidatesCache(t *testing.T) {
	const portID = "abc-456"

	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	var listCalls atomic.Int64
	fakeServer.Mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		listCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"ports":[]}`)
	})
	fakeServer.Mux.HandleFunc("/v2.0/ports/"+portID, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"port":{"id":%q}}`, portID)
	})

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	// Prime the cache.
	_, err := n.ListPorts(context.Background(), ports.ListOpts{}, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), listCalls.Load())

	// Rebind invalidates.
	assert.NoError(t, n.UpdatePortBinding(context.Background(), portID, "host-c"))

	// Next list refetches.
	_, err = n.ListPorts(context.Background(), ports.ListOpts{}, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), listCalls.Load())
}

// TestPortWithBinding_Extract verifies that ports.ExtractPortsInto populates both the embedded
// Port fields and the binding extension's HostID from a list response.
func TestPortWithBinding_Extract(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	listResponse := `{"ports":[
        {"id":"p1","name":"port-1","binding:host_id":"host-a"},
        {"id":"p2","name":"port-2","binding:host_id":"host-b"}
    ]}`
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "", listResponse, http.StatusOK)

	pages, err := ports.List(fake.ServiceClient(fakeServer), ports.ListOpts{}).AllPages(context.Background())
	assert.NoError(t, err)

	var got []PortWithBinding
	assert.NoError(t, ports.ExtractPortsInto(pages, &got))
	assert.Len(t, got, 2)
	assert.Equal(t, "p1", got[0].ID)
	assert.Equal(t, "host-a", got[0].HostID)
	assert.Equal(t, "p2", got[1].ID)
	assert.Equal(t, "host-b", got[1].HostID)
}

// TestListPorts_CachesResults verifies that consecutive ListPorts calls with the same filters
// hit the cache (only one HTTP GET), and that invalidatePortCache forces a refetch.
func TestListPorts_CachesResults(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	var calls atomic.Int64
	fakeServer.Mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"ports":[{"id":"p1","name":"port-1","binding:host_id":"host-a"}]}`)
	})

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	// First call: HTTP GET.
	got, err := n.ListPorts(context.Background(),
		ports.ListOpts{DeviceOwner: "network:f5snat"}, "host-a")
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, int64(1), calls.Load())

	// Second call: served from cache.
	_, err = n.ListPorts(context.Background(),
		ports.ListOpts{DeviceOwner: "network:f5snat"}, "host-a")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), calls.Load())

	// Different filter combo: cache miss, separate HTTP GET.
	_, err = n.ListPorts(context.Background(),
		ports.ListOpts{DeviceOwner: "network:f5snat"}, "host-b")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), calls.Load())

	// Invalidation forces a refetch even for the original combo.
	n.invalidatePortCache()
	_, err = n.ListPorts(context.Background(),
		ports.ListOpts{DeviceOwner: "network:f5snat"}, "host-a")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), calls.Load())
}

// TestListPorts_NoCacheWhenUninitialized verifies ListPorts works (without caching) when
// InitCache has never been called — important for tests that don't bother with cache setup.
func TestListPorts_NoCacheWhenUninitialized(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	var calls atomic.Int64
	fakeServer.Mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"ports":[]}`)
	})

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	// no InitCache

	for i := 0; i < 3; i++ {
		_, err := n.ListPorts(context.Background(), ports.ListOpts{}, "")
		assert.NoError(t, err)
	}
	assert.Equal(t, int64(3), calls.Load())
}

// --------------------------------------------------------------------------
// GetPort
// --------------------------------------------------------------------------

func TestNeutronClient_GetPort_Success(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports/"+allocPortID.String(), "GET",
		"", getPortFixture, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	got, err := n.GetPort(t.Context(), allocPortID.String())
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, allocPortID.String(), got.ID)
}

func TestNeutronClient_GetPort_NotFound(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports/"+allocPortID.String(), "GET",
		"", `{"NeutronError":{"type":"PortNotFound"}}`, http.StatusNotFound)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	_, err := n.GetPort(t.Context(), allocPortID.String())
	assert.Error(t, err)
	assert.True(t, gophercloud.ResponseCodeIs(err, http.StatusNotFound))
}

// --------------------------------------------------------------------------
// PortListOpts.ToPortListQuery (multi-id variant; the Ext variant is covered above)
// --------------------------------------------------------------------------

func TestPortListOpts_ToPortListQuery(t *testing.T) {
	opts := PortListOpts{IDs: []string{"id-1", "id-2", "id-3"}}
	got, err := opts.ToPortListQuery()
	assert.NoError(t, err)

	parsed, err := url.ParseQuery(got[1:]) // strip leading '?'
	assert.NoError(t, err)
	assert.Equal(t, []string{"id-1", "id-2", "id-3"}, parsed["id"])

	empty, err := PortListOpts{}.ToPortListQuery()
	assert.NoError(t, err)
	assert.Equal(t, "", empty)
}

// --------------------------------------------------------------------------
// AllocateNeutronEndpointPort
// --------------------------------------------------------------------------

// TestAllocateNeutronEndpointPort_TargetPort_Success: target.Port set, GET returns a port that
// belongs to the caller's project and has at least one fixed IP — function returns it as-is.
func TestAllocateNeutronEndpointPort_TargetPort_Success(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	resp := portJSON(t, allocProjectID, []map[string]string{
		{"subnet_id": SubnetIDFixture, "ip_address": "10.0.0.5"},
	})
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports/"+allocPortID.String(), "GET",
		"", resp, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	target := &models.EndpointTarget{Port: portFromUUID(allocPortID.String())}
	endpoint := &models.Endpoint{ID: allocEndpointID}

	got, err := n.AllocateNeutronEndpointPort(t.Context(), target, endpoint,
		allocProjectID, "host-a", n.ServiceClient)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, allocPortID.String(), got.ID)
	assert.Equal(t, allocProjectID, got.ProjectID)
}

// TestAllocateNeutronEndpointPort_TargetPort_ProjectMismatch: GET returns a port owned by a
// different project — must surface ErrProjectMismatch.
func TestAllocateNeutronEndpointPort_TargetPort_ProjectMismatch(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	resp := portJSON(t, "other-project", []map[string]string{
		{"subnet_id": SubnetIDFixture, "ip_address": "10.0.0.5"},
	})
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports/"+allocPortID.String(), "GET",
		"", resp, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	got, err := n.AllocateNeutronEndpointPort(t.Context(),
		&models.EndpointTarget{Port: portFromUUID(allocPortID.String())},
		&models.Endpoint{ID: allocEndpointID},
		allocProjectID, "host-a", n.ServiceClient)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, aErrors.ErrProjectMismatch)
}

// TestAllocateNeutronEndpointPort_TargetPort_MissingIP: GET returns a port with no fixed_ips —
// must surface ErrMissingIPAddress.
func TestAllocateNeutronEndpointPort_TargetPort_MissingIP(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	resp := portJSON(t, allocProjectID, nil)
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports/"+allocPortID.String(), "GET",
		"", resp, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	got, err := n.AllocateNeutronEndpointPort(t.Context(),
		&models.EndpointTarget{Port: portFromUUID(allocPortID.String())},
		&models.Endpoint{ID: allocEndpointID},
		allocProjectID, "host-a", n.ServiceClient)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, aErrors.ErrMissingIPAddress)
}

// TestAllocateNeutronEndpointPort_TargetPort_GetError: GET returns 404 — must propagate the
// gophercloud error verbatim (not wrapped in an aErrors sentinel).
func TestAllocateNeutronEndpointPort_TargetPort_GetError(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	fixture.SetupHandler(t, fakeServer, "/v2.0/ports/"+allocPortID.String(), "GET",
		"", `{"NeutronError":{"type":"PortNotFound"}}`, http.StatusNotFound)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	got, err := n.AllocateNeutronEndpointPort(t.Context(),
		&models.EndpointTarget{Port: portFromUUID(allocPortID.String())},
		&models.Endpoint{ID: allocEndpointID},
		allocProjectID, "host-a", n.ServiceClient)
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.False(t, errors.Is(err, aErrors.ErrProjectMismatch))
}

// TestAllocateNeutronEndpointPort_TargetSubnet: target.Subnet set with Network nil — function
// fetches the subnet, derives the network ID from it, then POSTs to /v2.0/ports.
func TestAllocateNeutronEndpointPort_TargetSubnet(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+SubnetIDFixture, "GET",
		"", allocSubnetFixture, http.StatusOK)

	var createCalls atomic.Int64
	fakeServer.Mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		createCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"port":{"id":"created-1","network_id":%q,"fixed_ips":[{"subnet_id":%q,"ip_address":"10.0.0.10"}]}}`,
			NetworkIDFixture, SubnetIDFixture)
	})

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	target := &models.EndpointTarget{Subnet: portFromUUID(SubnetIDFixture)}
	endpoint := &models.Endpoint{ID: allocEndpointID}

	got, err := n.AllocateNeutronEndpointPort(t.Context(), target, endpoint,
		allocProjectID, "host-a", n.ServiceClient)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "created-1", got.ID)
	assert.Equal(t, int64(1), createCalls.Load())
	// Function backfills target.Network from the subnet's network_id.
	assert.NotNil(t, target.Network)
	assert.Equal(t, NetworkIDFixture, target.Network.String())
}

// TestAllocateNeutronEndpointPort_TargetNetwork: target.Network set — function looks up the
// network, picks the first subnet, then POSTs to /v2.0/ports.
func TestAllocateNeutronEndpointPort_TargetNetwork(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)

	var createCalls atomic.Int64
	fakeServer.Mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		createCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"port":{"id":"created-2","network_id":%q,"fixed_ips":[]}}`,
			NetworkIDFixture)
	})

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	target := &models.EndpointTarget{Network: portFromUUID(NetworkIDFixture)}
	endpoint := &models.Endpoint{ID: allocEndpointID}

	got, err := n.AllocateNeutronEndpointPort(t.Context(), target, endpoint,
		allocProjectID, "host-a", n.ServiceClient)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "created-2", got.ID)
	assert.Equal(t, int64(1), createCalls.Load())
}

// TestAllocateNeutronEndpointPort_TargetNetwork_NoSubnets: target.Network resolves to a network
// with an empty subnets array — must surface ErrMissingSubnets without issuing a port create.
func TestAllocateNeutronEndpointPort_TargetNetwork_NoSubnets(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", allocNetworkNoSubnetsFixture, http.StatusOK)

	var createCalls atomic.Int64
	fakeServer.Mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		createCalls.Add(1)
		http.Error(w, "should not be called", http.StatusInternalServerError)
	})

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	got, err := n.AllocateNeutronEndpointPort(t.Context(),
		&models.EndpointTarget{Network: portFromUUID(NetworkIDFixture)},
		&models.Endpoint{ID: allocEndpointID},
		allocProjectID, "host-a", n.ServiceClient)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, aErrors.ErrMissingSubnets)
	assert.Equal(t, int64(0), createCalls.Load())
}

// TestAllocateNeutronEndpointPort_TargetNetwork_GetError: GET /networks/{id} fails (404) —
// must propagate the gophercloud error.
func TestAllocateNeutronEndpointPort_TargetNetwork_GetError(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", `{"NeutronError":{"type":"NetworkNotFound"}}`, http.StatusNotFound)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	got, err := n.AllocateNeutronEndpointPort(t.Context(),
		&models.EndpointTarget{Network: portFromUUID(NetworkIDFixture)},
		&models.Endpoint{ID: allocEndpointID},
		allocProjectID, "host-a", n.ServiceClient)
	assert.Nil(t, got)
	assert.Error(t, err)
	assert.False(t, errors.Is(err, aErrors.ErrMissingSubnets))
}

// TestAllocateNeutronEndpointPort_CreateError: subnets.Get succeeds but ports.Create returns
// 409 — must propagate the create error.
func TestAllocateNeutronEndpointPort_CreateError(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+SubnetIDFixture, "GET",
		"", allocSubnetFixture, http.StatusOK)

	fakeServer.Mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		http.Error(w, `{"NeutronError":{"type":"OverQuota"}}`, http.StatusConflict)
	})

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	got, err := n.AllocateNeutronEndpointPort(t.Context(),
		&models.EndpointTarget{Subnet: portFromUUID(SubnetIDFixture)},
		&models.Endpoint{ID: allocEndpointID},
		allocProjectID, "host-a", n.ServiceClient)
	assert.Nil(t, got)
	assert.Error(t, err)
}
