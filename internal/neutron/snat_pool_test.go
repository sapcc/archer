// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package neutron

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/go-openapi/strfmt"
	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/gophercloud/gophercloud/v2/testhelper/fixture"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/v2/internal/config"
)

const SubnetIDFixture = "a0304c3a-4f08-4c43-88af-d796509c97d2"
const ServiceIDFixture = strfmt.UUID("4d2bb22e-5e6a-44df-9b4f-9d6b7c2c2222")

const GetSubnetSnatFixture = `
{
  "subnet": {
    "id": "a0304c3a-4f08-4c43-88af-d796509c97d2",
    "network_id": "9bf57c58-5d9f-418b-a879-44d83e194ad0",
    "tenant_id": "test-project-1",
    "cidr": "10.0.0.0/24"
  }
}
`

func snatPortJSON(serviceID strfmt.UUID, idx int) string {
	return fmt.Sprintf(`{
        "id": "%s-%d",
        "name": "snat-%s-%d",
        "device_owner": "network:f5snat",
        "device_id": "%s",
        "network_id": "9bf57c58-5d9f-418b-a879-44d83e194ad0",
        "fixed_ips": [{"subnet_id": "a0304c3a-4f08-4c43-88af-d796509c97d2", "ip_address": "10.0.0.%d"}]
    }`, serviceID, idx, serviceID, idx, serviceID, 10+idx)
}

// snatTestServer wires up a minimal Neutron stub that supports listing/creating/deleting
// SNAT ports keyed by name. Each test gets a fresh in-memory map seeded with whatever the
// caller wants to be "already there".
func snatTestServer(t *testing.T, seed map[string]string) (*NeutronClient, func(), *atomic.Int64) {
	t.Helper()
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)

	state := make(map[string]string, len(seed))
	for k, v := range seed {
		state[k] = v
	}
	var listCalls atomic.Int64

	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+SubnetIDFixture, "GET",
		"", GetSubnetSnatFixture, http.StatusOK)

	fakeServer.Mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			listCalls.Add(1)
			items := make([]json.RawMessage, 0, len(state))
			for _, body := range state {
				items = append(items, json.RawMessage(body))
			}
			out, _ := json.Marshal(map[string]any{"ports": items})
			_, _ = w.Write(out)
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			var wrapped struct {
				Port map[string]any `json:"port"`
			}
			if err := json.Unmarshal(body, &wrapped); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Mirror Neutron: each fixed_ips entry must carry subnet_id or ip_address using
			// the wire-format keys. Catches client-side serialization bugs (e.g. structs
			// without JSON tags) that would otherwise pass tests but fail in production.
			if fips, ok := wrapped.Port["fixed_ips"].([]any); ok {
				for _, fip := range fips {
					m, _ := fip.(map[string]any)
					if m["subnet_id"] == nil && m["ip_address"] == nil {
						http.Error(w, `{"NeutronError":{"type":"InvalidInput","message":"IP allocation requires subnet_id or ip_address."}}`,
							http.StatusBadRequest)
						return
					}
				}
			}
			name, _ := wrapped.Port["name"].(string)
			id := fmt.Sprintf("created-%s", name)
			wrapped.Port["id"] = id
			wrapped.Port["network_id"] = "9bf57c58-5d9f-418b-a879-44d83e194ad0"
			wrapped.Port["fixed_ips"] = []map[string]any{{
				"subnet_id":  SubnetIDFixture,
				"ip_address": fmt.Sprintf("10.0.0.%d", 100+len(state)),
			}}
			out, _ := json.Marshal(wrapped.Port)
			state[name] = string(out)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"port":` + string(out) + `}`))
		default:
			http.Error(w, "unsupported", http.StatusMethodNotAllowed)
		}
	})

	fakeServer.Mux.HandleFunc("/v2.0/ports/", func(w http.ResponseWriter, r *http.Request) {
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		switch r.Method {
		case http.MethodPut:
			// Rebind via binding:host_id update — return the stored port unchanged.
			id := r.URL.Path[len("/v2.0/ports/"):]
			for _, body := range state {
				var p struct {
					ID string `json:"id"`
				}
				_ = json.Unmarshal([]byte(body), &p)
				if p.ID == id {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"port":` + body + `}`))
					return
				}
			}
			http.Error(w, "not found", http.StatusNotFound)
		case http.MethodDelete:
			id := r.URL.Path[len("/v2.0/ports/"):]
			for name, body := range state {
				var p struct {
					ID string `json:"id"`
				}
				_ = json.Unmarshal([]byte(body), &p)
				if p.ID == id {
					delete(state, name)
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			http.Error(w, "not found", http.StatusNotFound)
		default:
			http.Error(w, "unsupported", http.StatusMethodNotAllowed)
		}
	})

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()
	return n, fakeServer.Teardown, &listCalls
}

func TestEnsureServiceSnatPorts_CreateFromEmpty(t *testing.T) {
	n, teardown, _ := snatTestServer(t, nil)
	defer teardown()

	got, err := n.EnsureServiceSnatPorts(context.Background(), ServiceIDFixture, SubnetIDFixture, 3, false)
	assert.NoError(t, err)
	assert.Len(t, got, 3)
}

func TestEnsureServiceSnatPorts_ScaleUp(t *testing.T) {
	seed := map[string]string{
		fmt.Sprintf("snat-%s-0", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 0),
		fmt.Sprintf("snat-%s-1", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 1),
	}
	n, teardown, _ := snatTestServer(t, seed)
	defer teardown()

	got, err := n.EnsureServiceSnatPorts(context.Background(), ServiceIDFixture, SubnetIDFixture, 4, false)
	assert.NoError(t, err)
	assert.Len(t, got, 4)
}

func TestEnsureServiceSnatPorts_ScaleDown(t *testing.T) {
	seed := map[string]string{
		fmt.Sprintf("snat-%s-0", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 0),
		fmt.Sprintf("snat-%s-1", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 1),
		fmt.Sprintf("snat-%s-2", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 2),
	}
	n, teardown, _ := snatTestServer(t, seed)
	defer teardown()

	got, err := n.EnsureServiceSnatPorts(context.Background(), ServiceIDFixture, SubnetIDFixture, 1, false)
	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestEnsureServiceSnatPorts_NoOp(t *testing.T) {
	seed := map[string]string{
		fmt.Sprintf("snat-%s-0", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 0),
		fmt.Sprintf("snat-%s-1", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 1),
	}
	n, teardown, _ := snatTestServer(t, seed)
	defer teardown()

	got, err := n.EnsureServiceSnatPorts(context.Background(), ServiceIDFixture, SubnetIDFixture, 2, false)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestEnsureServiceSnatPorts_DeleteAll(t *testing.T) {
	seed := map[string]string{
		fmt.Sprintf("snat-%s-0", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 0),
		fmt.Sprintf("snat-%s-1", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 1),
	}
	n, teardown, _ := snatTestServer(t, seed)
	defer teardown()

	got, err := n.EnsureServiceSnatPorts(context.Background(), ServiceIDFixture, "", 0, false)
	assert.NoError(t, err)
	assert.Len(t, got, 0)
}

func TestEnsureServiceSnatPorts_NoOpDryRun(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()

	listResponse := fmt.Sprintf(`{"ports":[%s,%s]}`,
		snatPortJSON(ServiceIDFixture, 0),
		snatPortJSON(ServiceIDFixture, 1))
	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+SubnetIDFixture, "GET",
		"", GetSubnetSnatFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "", listResponse, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	got, err := n.EnsureServiceSnatPorts(context.Background(), ServiceIDFixture, SubnetIDFixture, 2, true)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestCleanupServiceSnatPorts(t *testing.T) {
	seed := map[string]string{
		fmt.Sprintf("snat-%s-0", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 0),
		fmt.Sprintf("snat-%s-1", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 1),
	}
	n, teardown, _ := snatTestServer(t, seed)
	defer teardown()

	assert.NoError(t, n.CleanupServiceSnatPorts(context.Background(), ServiceIDFixture))

	got, err := n.EnsureServiceSnatPorts(context.Background(), ServiceIDFixture, SubnetIDFixture, 0, true)
	assert.NoError(t, err)
	assert.Len(t, got, 0)
}

func TestCleanupServiceSnatPorts_Empty(t *testing.T) {
	n, teardown, _ := snatTestServer(t, nil)
	defer teardown()

	assert.NoError(t, n.CleanupServiceSnatPorts(context.Background(), ServiceIDFixture))
}

// TestEnsureServiceSnatPorts_Migration verifies that ports left over from a previous host
// (different binding:host_id) are rebound to the current host instead of being duplicated.
func TestEnsureServiceSnatPorts_Migration(t *testing.T) {
	// Seed two ports as if they had been allocated by host-a; the agent now runs as host-b.
	seed := map[string]string{
		fmt.Sprintf("snat-%s-0", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 0),
		fmt.Sprintf("snat-%s-1", ServiceIDFixture): snatPortJSON(ServiceIDFixture, 1),
	}
	n, teardown, _ := snatTestServer(t, seed)
	defer teardown()
	config.Global.Default.Host = "host-b"

	got, err := n.EnsureServiceSnatPorts(context.Background(), ServiceIDFixture, SubnetIDFixture, 2, false)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	// IDs match the seeded ports — no new ones were created.
	assert.Equal(t, fmt.Sprintf("%s-0", ServiceIDFixture), got["snat-0"].ID)
	assert.Equal(t, fmt.Sprintf("%s-1", ServiceIDFixture), got["snat-1"].ID)
}

// TestEnsureServiceSnatPorts_SkipsRebindWhenHostMatches verifies that ports already bound to
// this agent's host are returned untouched.
func TestEnsureServiceSnatPorts_SkipsRebindWhenHostMatches(t *testing.T) {
	const host = "host-a"
	config.Global.Default.Host = host
	bound := func(idx int) string {
		return fmt.Sprintf(`{
            "id": "%s-%d",
            "name": "snat-%s-%d",
            "device_owner": "network:f5snat",
            "device_id": "%s",
            "network_id": "9bf57c58-5d9f-418b-a879-44d83e194ad0",
            "binding:host_id": %q,
            "fixed_ips": [{"subnet_id": "a0304c3a-4f08-4c43-88af-d796509c97d2", "ip_address": "10.0.0.%d"}]
        }`, ServiceIDFixture, idx, ServiceIDFixture, idx, ServiceIDFixture, host, 10+idx)
	}
	seed := map[string]string{
		fmt.Sprintf("snat-%s-0", ServiceIDFixture): bound(0),
		fmt.Sprintf("snat-%s-1", ServiceIDFixture): bound(1),
	}
	n, teardown, _ := snatTestServer(t, seed)
	defer teardown()

	got, err := n.EnsureServiceSnatPorts(context.Background(), ServiceIDFixture, SubnetIDFixture, 2, false)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

// TestFetchSnatPorts_Flat verifies SNAT ports are returned as a flat slice
// (the orphan signal is per-service, not per-network).
func TestFetchSnatPorts_Flat(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Default.Host = "host-a"

	listResponse := fmt.Sprintf(`{"ports":[
        {"id": "p0", "name": "snat-svc-A-0", "device_owner": "network:f5snat",
         "device_id": "svc-A", "network_id": "net-1",
         "fixed_ips": [{"subnet_id": %q, "ip_address": "10.0.0.10"}]},
        {"id": "p1", "name": "snat-svc-B-0", "device_owner": "network:f5snat",
         "device_id": "svc-B", "network_id": "net-2",
         "fixed_ips": [{"subnet_id": %q, "ip_address": "10.0.0.11"}]}
    ]}`, SubnetIDFixture, SubnetIDFixture)
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "", listResponse, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	got, err := n.FetchSnatPorts(context.Background())
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	deviceIDs := []string{got[0].DeviceID, got[1].DeviceID}
	assert.Contains(t, deviceIDs, "svc-A")
	assert.Contains(t, deviceIDs, "svc-B")
}

func TestFetchSnatPorts_Empty(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "", `{"ports":[]}`, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	got, err := n.FetchSnatPorts(context.Background())
	assert.NoError(t, err)
	assert.Len(t, got, 0)
}
