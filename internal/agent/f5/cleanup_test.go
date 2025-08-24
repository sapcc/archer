// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"net/http"
	"testing"

	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/gophercloud/gophercloud/v2/testhelper/fixture"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/neutron"
	"github.com/sapcc/archer/models"
)

func TestAgent_TestGetUsedSegments(t *testing.T) {
	var err error
	var dbMock pgxmock.PgxPoolIface
	serviceNetwork := "b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b"
	someOtherNetwork := "35a3ca82-62af-4e0a-9472-92331500fb3a"
	anotherOneBitesTheDust := "ba578650-e29d-4c63-8847-59ebeedf4629"

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
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+serviceNetwork, "GET",
		"", GetServiceNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+someOtherNetwork, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+anotherOneBitesTheDust, "GET",
		"", GetNetworkResponseAnotherOneFixture, http.StatusOK)

	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	neutronClient.InitCache()

	// initialize agent
	a := &Agent{
		pool:    dbMock,
		neutron: &neutronClient,
	}

	var segementID pgtype.Int4
	_ = segementID.Scan(int64(123))
	var epNetworkID pgtype.UUID
	_ = epNetworkID.Scan(someOtherNetwork)

	sql := `SELECT s.network_id, ep.segment_id, ep.network FROM service s LEFT JOIN endpoint e ON s.id = e.service_id LEFT JOIN endpoint_port ep ON ep.endpoint_id = e.id WHERE s.host = $1 AND s.provider = $2`
	dbMock.
		ExpectQuery(sql).
		WithArgs("host-123", models.ServiceProviderTenant).
		WillReturnRows(pgxmock.NewRows([]string{"network_id", "segment_id", "network"}).
			AddRow(serviceNetwork, segementID, epNetworkID).
			AddRow(someOtherNetwork, nil, nil).
			AddRow(serviceNetwork, nil, anotherOneBitesTheDust))

	// run the test function
	var usedSegments map[int]string
	usedSegments, err = a.getUsedSegments()
	assert.Nil(t, err)
	assert.EqualValues(t, map[int]string{123: someOtherNetwork, 666: serviceNetwork, 999: anotherOneBitesTheDust}, usedSegments)
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
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "",
		GetPortListSelfIPsResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b", "GET",
		"", GetServiceNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/35a3ca82-62af-4e0a-9472-92331500fb3a", "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports/c0c0c0c0-c0c0-4c0c-8c0c-0c0c0c0c0c0c", "DELETE",
		"", "", http.StatusNoContent)

	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	neutronClient.InitCache()
	// initialize agent
	a := &Agent{neutron: &neutronClient}

	// run the test function
	usedSegments := map[int]string{
		123: "b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b",
	}
	assert.Nil(t, a.cleanOrphanedNeutronPorts(usedSegments))
}

func TestAgent_TestCleanupOrphanedTenants(t *testing.T) {
	f5DeviceMock := NewMockF5Device(t)
	f5DeviceMock.On("GetHostname").Return("host-123")
	// we don't have the selfip yet, let it create it
	f5DeviceMock.EXPECT().
		GetPartitions().
		Return([]string{
			"Common",
			"net-4f891be2-c32f-4356-81c4-056b6101463a",
			"something-manual-don't-touch-me",
			"net-delete-me",
		}, nil)

	expectAS3 := &as3.AS3{
		Persist: false,
		Class:   "AS3",
		Action:  "deploy",
		Declaration: as3.ADC{
			Class:         "ADC",
			SchemaVersion: "3.36.0",
			UpdateMode:    "selective",
			Id:            "urn:uuid:07649173-4AF7-48DF-963F-84000C70F0DD",
			Tenants: map[string]as3.Tenant{
				"net-delete-me": {
					Class:        "Tenant",
					Label:        "",
					Remark:       "",
					Applications: map[string]as3.Application(nil),
				},
			},
		},
	}

	f5DeviceMock.EXPECT().
		PostAS3(expectAS3, "net-delete-me").
		Return(nil)
	// initialize agent
	a := &Agent{
		devices: []F5Device{f5DeviceMock},
		active:  f5DeviceMock,
	}

	// run the test function
	usedSegments := map[int]string{
		123: "4f891be2-c32f-4356-81c4-056b6101463a",
		666: "3ac03bd0-477d-4aa9-85f9-c1a95ca3a962",
	}
	assert.Nil(t, a.cleanupOrphanedTenants(usedSegments))
}
