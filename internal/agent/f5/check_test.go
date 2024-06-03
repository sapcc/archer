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
	"context"
	"net/http"
	"testing"

	fake "github.com/gophercloud/gophercloud/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/fixture"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/neutron"
)

func TestCheckCleanupL2(t *testing.T) {
	networkID := "e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e"
	config.Global.Default.Host = "host-5678"

	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	dbMock.ExpectBegin()
	dbMock.ExpectExec("SELECT 1 FROM service WHERE network_id = $1 AND host = $2 AND provider = 'tenant'").
		WithArgs(networkID, config.Global.Default.Host).
		WillReturnResult(pgxmock.NewResult("SELECT 1", 0))
	dbMock.ExpectExec("SELECT 1 FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint_port.network = $1 AND service.host = $2 AND service.provider = 'tenant'").
		WithArgs(networkID, config.Global.Default.Host).
		WillReturnResult(pgxmock.NewResult("SELECT 1", 0))

	ctx := context.TODO()
	tx, _ := dbMock.Begin(ctx)
	err, ret := checkCleanupL2(ctx, tx, networkID, false, false)
	assert.Nil(t, err, "checkCleanupL2() should not return an error")
	assert.True(t, ret, "checkCleanupL2() should return true")
	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCheckCleanupSelfIPs(t *testing.T) {
	networkID := "35a3ca82-62af-4e0a-9472-92331500fb3a"
	subnetID := "a2dfade2-4437-48c4-86d5-43ff204bd3a5"
	config.Global.Agent.PhysicalNetwork = "physnet1"
	config.Global.Default.Host = "host-2511"

	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	fixture.SetupHandler(t, "/v2.0/networks/"+networkID, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("SELECT 1 FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint_port.subnet = $1 AND service.host = $2 AND service.provider = 'tenant'").
		WithArgs(subnetID, config.Global.Default.Host).
		WillReturnResult(pgxmock.NewResult("SELECT 1", 0))

	ctx := context.TODO()
	tx, _ := dbMock.Begin(ctx)
	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient()}
	neutronClient.InitCache()
	a := &Agent{
		neutron: &neutronClient,
	}
	err, ret := a.checkCleanupSelfIPs(ctx, tx, networkID, subnetID,
		false, false)
	assert.Nil(t, err, "checkCleanupL2() should not return an error")
	assert.True(t, ret, "checkCleanupL2() should return true")
	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCheckCleanupSelfIPs_negative(t *testing.T) {
	networkID := "35a3ca82-62af-4e0a-9472-92331500fb3a"
	subnetID := "e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e"
	config.Global.Agent.PhysicalNetwork = "physnet1"
	config.Global.Default.Host = "host-2511"

	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	fixture.SetupHandler(t, "/v2.0/networks/"+networkID, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("SELECT 1 FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint_port.subnet = $1 AND service.host = $2 AND service.provider = 'tenant'").
		WithArgs(subnetID, config.Global.Default.Host).
		WillReturnResult(pgxmock.NewResult("SELECT 1", 0))
	dbMock.ExpectExec("SELECT 1 FROM service WHERE network_id = $1 AND host = $2 AND provider = 'tenant'").
		WithArgs(networkID, config.Global.Default.Host).
		WillReturnResult(pgxmock.NewResult("SELECT 1", 1))

	ctx := context.TODO()
	tx, _ := dbMock.Begin(ctx)
	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient()}
	neutronClient.InitCache()
	a := &Agent{
		neutron: &neutronClient,
	}
	err, ret := a.checkCleanupSelfIPs(ctx, tx, networkID, subnetID,
		false, false)
	assert.Nil(t, err, "checkCleanupL2() should not return an error")
	assert.False(t, ret, "checkCleanupL2() should return true")
	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
