// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package neutron

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/gophercloud/gophercloud/v2/testhelper/fixture"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	aErrors "github.com/sapcc/archer/internal/errors"
)

const NetworkIDFixture = "9bf57c58-5d9f-418b-a879-44d83e194ad0"
const GetNetworkResponseFixture = `
{
    "network": {
        "id": "9bf57c58-5d9f-418b-a879-44d83e194ad0",
        "subnets": ["a0304c3a-4f08-4c43-88af-d796509c97d2"],
		"project_id": "test-project-1",
		"segments": [
			{
				"provider:physical_network": "physnet1",
				"provider:network_type": "vlan",
				"provider:segmentation_id": 100
			}
		]
    }
}
`

func TestNeutronClient_GetNetworkSegment(t *testing.T) {
	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(),
	}
	n.InitCache()
	segID, err := n.GetNetworkSegment(NetworkIDFixture, config.Global.Agent.PhysicalNetwork)
	assert.Nil(t, err)
	assert.Equal(t, 100, segID)
}

func TestNeutronClient_GetNetworkSegment404(t *testing.T) {
	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	config.Global.Agent.PhysicalNetwork = "physnet1"

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(),
	}
	n.InitCache()
	_, err := n.GetNetworkSegment(NetworkIDFixture, config.Global.Agent.PhysicalNetwork)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, aErrors.ErrNoPhysNetFound))
	assert.ErrorContains(t, err, "no physical network found, network not found")
}

func TestNeutronClient_GetNetworkSegmentMising(t *testing.T) {
	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	config.Global.Agent.PhysicalNetwork = "physnet2"
	fixture.SetupHandler(t, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(),
	}
	n.InitCache()
	_, err := n.GetNetworkSegment(NetworkIDFixture, config.Global.Agent.PhysicalNetwork)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, aErrors.ErrNoPhysNetFound))
	assert.ErrorContains(t, err, "no physical network found, physnet 'physnet2' not found for network")
}

func TestNeutronClient_GetNetworkSegment500(t *testing.T) {
	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", "internal server error", http.StatusInternalServerError)

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(),
	}
	n.InitCache()
	_, err := n.GetNetworkSegment(NetworkIDFixture, config.Global.Agent.PhysicalNetwork)
	assert.Error(t, err)
	assert.False(t, errors.Is(err, aErrors.ErrNoPhysNetFound))
	assert.True(t, gophercloud.ResponseCodeIs(err, http.StatusInternalServerError))
	assert.ErrorContains(t, err, "internal server error")
}
