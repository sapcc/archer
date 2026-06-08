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

	"github.com/sapcc/archer/v2/internal/config"
	aErrors "github.com/sapcc/archer/v2/internal/errors"
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
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(fakeServer),
	}
	n.InitCache()
	segID, err := n.GetNetworkSegment(t.Context(), NetworkIDFixture, config.Global.Agent.PhysicalNetwork)
	assert.Nil(t, err)
	assert.Equal(t, 100, segID)
}

func TestNeutronClient_GetNetworkSegment404(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(fakeServer),
	}
	n.InitCache()
	_, err := n.GetNetworkSegment(t.Context(), NetworkIDFixture, config.Global.Agent.PhysicalNetwork)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, aErrors.ErrNoPhysNetFound))
	assert.ErrorContains(t, err, "no physical network found, network not found")
}

func TestNeutronClient_GetNetworkSegmentMissing(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet2"
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(fakeServer),
	}
	n.InitCache()
	_, err := n.GetNetworkSegment(t.Context(), NetworkIDFixture, config.Global.Agent.PhysicalNetwork)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, aErrors.ErrNoPhysNetFound))
	assert.ErrorContains(t, err, "no physical network found, physnet 'physnet2' not found for network")
}

func TestNeutronClient_GetNetworkSegment500(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", "internal server error", http.StatusInternalServerError)

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(fakeServer),
	}
	n.InitCache()
	_, err := n.GetNetworkSegment(t.Context(), NetworkIDFixture, config.Global.Agent.PhysicalNetwork)
	assert.Error(t, err)
	assert.False(t, errors.Is(err, aErrors.ErrNoPhysNetFound))
	assert.True(t, gophercloud.ResponseCodeIs(err, http.StatusInternalServerError))
	assert.ErrorContains(t, err, "internal server error")
}

// --------------------------------------------------------------------------
// GetSubnetSegment / GetNetworkMTU / GetMask
// --------------------------------------------------------------------------

// getNetworkWithMTUFixture mirrors GetNetworkResponseFixture but adds an mtu field so the
// NetworkMTUExt extractor populates NetworkMTU.MTU.
const getNetworkWithMTUFixture = `
{
  "network": {
    "id": "9bf57c58-5d9f-418b-a879-44d83e194ad0",
    "subnets": ["a0304c3a-4f08-4c43-88af-d796509c97d2"],
    "mtu": 1450,
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

// getSubnetForMaskFixture is a minimal subnet response with a CIDR — drives GetMask's happy
// path. Reused by GetSubnetSegment_Success (also needs network_id).
const getSubnetForMaskFixture = `
{
  "subnet": {
    "id": "a0304c3a-4f08-4c43-88af-d796509c97d2",
    "network_id": "9bf57c58-5d9f-418b-a879-44d83e194ad0",
    "cidr": "10.0.0.0/24"
  }
}
`

// getSubnetBadCIDRFixture has an unparsable CIDR so net.ParseCIDR returns an error and
// GetMask propagates it.
const getSubnetBadCIDRFixture = `
{
  "subnet": {
    "id": "a0304c3a-4f08-4c43-88af-d796509c97d2",
    "network_id": "9bf57c58-5d9f-418b-a879-44d83e194ad0",
    "cidr": "not-a-cidr"
  }
}
`

func TestNeutronClient_GetMask_Success(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+SubnetIDFixture, "GET",
		"", getSubnetForMaskFixture, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()
	mask, err := n.GetMask(t.Context(), SubnetIDFixture)
	assert.NoError(t, err)
	assert.Equal(t, 24, mask)
}

func TestNeutronClient_GetMask_BadCIDR(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+SubnetIDFixture, "GET",
		"", getSubnetBadCIDRFixture, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()
	_, err := n.GetMask(t.Context(), SubnetIDFixture)
	assert.Error(t, err)
}

func TestNeutronClient_GetNetworkMTU_Success(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", getNetworkWithMTUFixture, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()
	mtu, err := n.GetNetworkMTU(t.Context(), NetworkIDFixture)
	assert.NoError(t, err)
	assert.Equal(t, 1450, mtu)
}

func TestNeutronClient_GetNetworkMTU_NotFound(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", `{"NeutronError":{"type":"NetworkNotFound"}}`, http.StatusNotFound)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()
	_, err := n.GetNetworkMTU(t.Context(), NetworkIDFixture)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, aErrors.ErrNoPhysNetFound))
}

func TestNeutronClient_GetSubnetSegment_Success(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"

	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+SubnetIDFixture, "GET",
		"", getSubnetForMaskFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()
	segID, err := n.GetSubnetSegment(t.Context(), SubnetIDFixture, config.Global.Agent.PhysicalNetwork)
	assert.NoError(t, err)
	assert.Equal(t, 100, segID)
}

func TestNeutronClient_GetSubnetSegment_NotFound(t *testing.T) {
	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"

	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+SubnetIDFixture, "GET",
		"", `{"NeutronError":{"type":"SubnetNotFound"}}`, http.StatusNotFound)

	n := &NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	n.InitCache()
	_, err := n.GetSubnetSegment(t.Context(), SubnetIDFixture, config.Global.Agent.PhysicalNetwork)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, aErrors.ErrNoPhysNetFound))
	assert.ErrorContains(t, err, "subnet not found")
}
