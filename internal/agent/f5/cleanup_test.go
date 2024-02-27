// Copyright 2024 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package f5

import (
	"net/http"
	"testing"

	fake "github.com/gophercloud/gophercloud/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/fixture"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/neutron"
)

func TestAgent_TestGetUsedSegments(t *testing.T) {
	var err error
	var dbMock pgxmock.PgxPoolIface
	serviceNetwork := "b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b"
	someOtherNetwork := "35a3ca82-62af-4e0a-9472-92331500fb3a"

	// prepare sql mock
	dbMock, err = pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	// prepare global config
	config.Global.Default.Host = "host-123"
	config.Global.Agent.PhysicalNetwork = "physnet1"

	// setup neutron "server
	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, "/v2.0/networks/"+serviceNetwork, "GET",
		"", GetServiceNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/networks/"+someOtherNetwork, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)

	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient()}
	neutronClient.InitCache()

	// initialize agent
	a := &Agent{
		pool:    dbMock,
		neutron: &neutronClient,
	}

	sql := `SELECT s.network_id, COALESCE(ep.segment_id, 0) FROM service s LEFT JOIN endpoint e ON s.id = e.service_id LEFT JOIN endpoint_port ep ON ep.endpoint_id = e.id WHERE s.host = $1 AND s.provider = 'tenant'`
	dbMock.
		ExpectQuery(sql).
		WithArgs("host-123").
		WillReturnRows(pgxmock.NewRows([]string{"network_id", "segment_id"}).
			AddRow(serviceNetwork, 123).
			AddRow(someOtherNetwork, 0))

	// run the test function
	var usedSegments map[int]struct{}
	usedSegments, err = a.getUsedSegments()
	assert.EqualValues(t, map[int]struct{}{123: {}, 666: {}}, usedSegments)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestAgent_TestCleanOrphanedNeutronPorts(t *testing.T) {
	const GetPortListSelfIPsResponseFixture = `
{
	"ports": [
		{
            "name": "This port should be deleted",
			"id": "c0c0c0c0-c0c0-4c0c-8c0c-0c0c0c0c0c0c",
			"network_id": "b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b"
		},
		{
			"id": "5a8ad669-4ffe-4133-b9f9-6de62cd654a4",
			"network_id": "35a3ca82-62af-4e0a-9472-92331500fb3a"
		}
	]
}
`

	// prepare global config
	config.Global.Default.Host = "host-123"
	config.Global.Agent.PhysicalNetwork = "physnet1"

	// setup neutron "server
	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	fixture.SetupHandler(t, "/v2.0/ports", "GET", "",
		GetPortListSelfIPsResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/networks/b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b", "GET",
		"", GetServiceNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/networks/35a3ca82-62af-4e0a-9472-92331500fb3a", "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/ports/c0c0c0c0-c0c0-4c0c-8c0c-0c0c0c0c0c0c", "DELETE",
		"", "", http.StatusNoContent)

	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient()}
	neutronClient.InitCache()
	// initialize agent
	a := &Agent{neutron: &neutronClient}

	// run the test function
	usedSegments := map[int]struct{}{
		123: {},
	}
	assert.Nil(t, a.cleanOrphanedNeutronPorts(usedSegments))
}
