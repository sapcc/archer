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
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/neutron"
)

func TestAgent_EnsureSelfIPs_Create(t *testing.T) {
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	// prepare global config
	config.Global.Default.Host = "host-1234"
	config.Global.Agent.PhysicalNetwork = "physnet1"

	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	fixture.SetupHandler(t, "/v2.0/subnets/e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e", "GET", "",
		GetSubnetResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/networks/35a3ca82-62af-4e0a-9472-92331500fb3a", "GET", "",
		GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/ports", "GET", "",
		GetPortListResponseFixture, http.StatusOK)

	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient()}
	neutronClient.InitCache()

	f5DeviceHost := NewMockF5Device(t)
	f5DeviceHost.On("GetHostname").Return("dummybigiphost")
	f5DeviceHost.EXPECT().
		EnsureBigIPSelfIP(
			"selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4",
			"42.42.42.42%123/8",
			123,
		).Return(nil)

	a := &Agent{
		pool:    dbMock,
		neutron: &neutronClient,
		devices: []F5Device{f5DeviceHost},
		hosts:   []F5Device{},
		active:  f5DeviceHost,
	}

	subnetID := "e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e"
	assert.Nil(t, a.EnsureSelfIPs(subnetID, false), "EnsureSelfIPs() should not return an error")
}

func TestAgent_EnsureSelfIPs_NoOp(t *testing.T) {
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	// prepare global config
	config.Global.Default.Host = "host-1234"
	config.Global.Agent.PhysicalNetwork = "physnet1"

	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	fixture.SetupHandler(t, "/v2.0/subnets/e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e", "GET", "",
		GetSubnetResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/networks/35a3ca82-62af-4e0a-9472-92331500fb3a", "GET", "",
		GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/ports", "GET", "",
		GetPortListResponseFixture, http.StatusOK)

	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient()}
	neutronClient.InitCache()

	f5DeviceHost := NewMockF5Device(t)
	f5DeviceHost.EXPECT().
		GetHostname().
		Return("dummybigiphost")
	// we don't have the selfip yet, let it create it
	f5DeviceHost.EXPECT().
		EnsureBigIPSelfIP(
			"selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4",
			"42.42.42.42%123/8",
			123,
		).Return(nil)

	a := &Agent{
		pool:    dbMock,
		neutron: &neutronClient,
		devices: []F5Device{f5DeviceHost},
		hosts:   []F5Device{},
		active:  f5DeviceHost,
	}

	subnetID := "e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e"
	assert.Nil(t, a.EnsureSelfIPs(subnetID, false), "EnsureSelfIPs() should not return an error")
}

func TestAgent_CleanupSelfIPs(t *testing.T) {
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	fixture.SetupHandler(t, "/v2.0/subnets/e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e", "GET", "",
		GetSubnetResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/ports", "GET", "",
		GetPortListResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/ports/5a8ad669-4ffe-4133-b9f9-6de62cd654a4", "DELETE", "",
		"", http.StatusAccepted)

	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient()}
	neutronClient.InitCache()

	f5DeviceHost := NewMockF5Device(t)
	f5DeviceHost.On("GetHostname").Return("dummybigiphost")
	f5DeviceHost.EXPECT().
		DeleteSelfIP("selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4").
		Return(nil)

	a := &Agent{
		pool:    dbMock,
		neutron: &neutronClient,
		devices: []F5Device{f5DeviceHost},
		hosts:   []F5Device{},
		active:  f5DeviceHost,
	}

	// Port should be deleted
	subnetID := "e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e"
	assert.Nil(t, a.CleanupSelfIPs(subnetID), "CleanupSelfIPs() should not return an error")
}
