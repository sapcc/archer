// Copyright 2023 SAP SE
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

	"github.com/f5devcentral/go-bigip"
	fake "github.com/gophercloud/gophercloud/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/fixture"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/agent/f5/as3"
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

	bigIPMock := as3.NewMockBigIPIface(t)
	// we don't have the selfip yet, let it create it
	bigIPMock.EXPECT().
		SelfIP("selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4").
		Return(nil, nil)
	bigIPMock.EXPECT().
		CreateSelfIP(&bigip.SelfIP{
			Name:    "selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4",
			Address: "42.42.42.42%123/8",
			Vlan:    "/Common/vlan-123",
		}).
		Return(nil)

	a := &Agent{
		jobQueue: nil,
		pool:     dbMock,
		neutron:  &neutronClient,
		bigips:   []*as3.BigIP{{Host: "dummybigiphost", BigIPIface: bigIPMock}},
		vcmps:    []*as3.BigIP{},
		bigip:    &as3.BigIP{Host: "dummybigiphost", BigIPIface: bigIPMock},
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

	bigIPMock := as3.NewMockBigIPIface(t)
	// we don't have the selfip yet, let it create it
	bigIPMock.EXPECT().
		SelfIP("selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4").
		Return(&bigip.SelfIP{
			Name:    "selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4",
			Address: "42.42.42.42%1234/8",
			Vlan:    "/Common/vlan-1234",
		}, nil)

	a := &Agent{
		jobQueue: nil,
		pool:     dbMock,
		neutron:  &neutronClient,
		bigips:   []*as3.BigIP{{Host: "dummybigiphost", BigIPIface: bigIPMock}},
		vcmps:    []*as3.BigIP{},
		bigip:    &as3.BigIP{Host: "dummybigiphost", BigIPIface: bigIPMock},
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

	bigIPMock := as3.NewMockBigIPIface(t)
	// we don't have the selfip yet, let it create it
	bigIPMock.EXPECT().
		SelfIP("selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4").
		Return(&bigip.SelfIP{
			Name:    "selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4",
			Address: "42.42.42.42%1234/8",
			Vlan:    "/Common/vlan-1234",
		}, nil)
	bigIPMock.EXPECT().
		DeleteSelfIP("selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4").
		Return(nil)

	a := &Agent{
		jobQueue: nil,
		pool:     dbMock,
		neutron:  &neutronClient,
		bigips:   []*as3.BigIP{{Host: "dummybigiphost", BigIPIface: bigIPMock}},
		vcmps:    []*as3.BigIP{},
		bigip:    &as3.BigIP{Host: "dummybigiphost", BigIPIface: bigIPMock},
	}

	// Port should be deleted
	subnetID := "e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e"
	assert.Nil(t, a.CleanupSelfIPs(subnetID), "CleanupSelfIPs() should not return an error")
}
