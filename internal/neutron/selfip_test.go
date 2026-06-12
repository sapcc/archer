// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package neutron

import (
	"fmt"
	"net/http"
	"testing"

	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/gophercloud/gophercloud/v2/testhelper/fixture"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/v2/internal/config"
)

func selfIPPortJSON(deviceID, networkID string, idx int) string {
	return fmt.Sprintf(`{
        "id": "selfip-%s",
        "name": "local-%s",
        "device_owner": "network:f5selfip",
        "device_id": "a0304c3a-4f08-4c43-88af-d796509c97d2",
        "network_id": "%s",
        "fixed_ips": [{"subnet_id": "a0304c3a-4f08-4c43-88af-d796509c97d2", "ip_address": "10.0.0.%d"}]
    }`, deviceID, deviceID, networkID, 50+idx)
}

func TestEnsureNeutronSelfIPs_DryRun_NoExisting(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Default.Host = "host-a"

	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+SubnetIDFixture, "GET",
		"", GetSubnetSnatFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "", `{"ports":[]}`, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	// dryRun=true must not create anything; with no existing matches the result is empty.
	got, err := n.EnsureNeutronSelfIPs(t.Context(), []string{"device-1", "device-2"}, SubnetIDFixture, true)
	assert.NoError(t, err)
	assert.Len(t, got, 0)
}

func TestEnsureNeutronSelfIPs_DryRun_AllExisting(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Default.Host = "host-a"

	listResponse := fmt.Sprintf(`{"ports":[%s,%s]}`,
		selfIPPortJSON("device-1", "9bf57c58-5d9f-418b-a879-44d83e194ad0", 0),
		selfIPPortJSON("device-2", "9bf57c58-5d9f-418b-a879-44d83e194ad0", 1))
	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+SubnetIDFixture, "GET",
		"", GetSubnetSnatFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "", listResponse, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	got, err := n.EnsureNeutronSelfIPs(t.Context(), []string{"device-1", "device-2"}, SubnetIDFixture, true)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Contains(t, got, "device-1")
	assert.Contains(t, got, "device-2")
}

func TestEnsureNeutronSelfIPs_CreatesMissing(t *testing.T) {
	// Reuse the snat stub: it implements list/create/delete generically by name, so SelfIP
	// reconcile works against it without modification.
	n, teardown, _ := snatTestServer(t, nil)
	defer teardown()
	config.Global.Default.Host = "host-a"

	got, err := n.EnsureNeutronSelfIPs(t.Context(), []string{"device-1", "device-2"}, SubnetIDFixture, false)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Contains(t, got, "device-1")
	assert.Contains(t, got, "device-2")
}

func TestFetchSelfIPPorts_GroupsByNetwork(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Default.Host = "host-a"

	listResponse := fmt.Sprintf(`{"ports":[%s,%s,%s]}`,
		selfIPPortJSON("device-1", "net-1", 0),
		selfIPPortJSON("device-2", "net-1", 1),
		selfIPPortJSON("device-3", "net-2", 2))
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "", listResponse, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	got, err := n.FetchSelfIPPorts(t.Context())
	assert.NoError(t, err)
	assert.Len(t, got["net-1"], 2)
	assert.Len(t, got["net-2"], 1)
}

func TestFetchSelfIPPorts_Empty(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "", `{"ports":[]}`, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()

	got, err := n.FetchSelfIPPorts(t.Context())
	assert.NoError(t, err)
	assert.Len(t, got, 0)
}
