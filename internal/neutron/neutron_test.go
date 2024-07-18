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

package neutron

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/gophercloud/gophercloud/v2/testhelper/fixture"
	"github.com/hashicorp/golang-lru/v2/expirable"
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
	var cache *expirable.LRU[string, map[string]*ports.Port]

	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(),
		portCache:     cache,
	}
	segID, err := n.GetNetworkSegment(NetworkIDFixture)
	assert.Nil(t, err)
	assert.Equal(t, 100, segID)
}

func TestNeutronClient_GetNetworkSegment404(t *testing.T) {
	var cache *expirable.LRU[string, map[string]*ports.Port]

	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	config.Global.Agent.PhysicalNetwork = "physnet1"

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(),
		portCache:     cache,
	}
	_, err := n.GetNetworkSegment(NetworkIDFixture)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, aErrors.ErrNoPhysNetFound))
	assert.ErrorContains(t, err, "no physical network found, network not found")
}

func TestNeutronClient_GetNetworkSegmentMissing(t *testing.T) {
	var cache *expirable.LRU[string, map[string]*ports.Port]

	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	config.Global.Agent.PhysicalNetwork = "physnet2"
	fixture.SetupHandler(t, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(),
		portCache:     cache,
	}
	_, err := n.GetNetworkSegment(NetworkIDFixture)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, aErrors.ErrNoPhysNetFound))
	assert.ErrorContains(t, err, "no physical network found, physnet 'physnet2' not found for network")
}

func TestNeutronClient_GetNetworkSegment500(t *testing.T) {
	var cache *expirable.LRU[string, map[string]*ports.Port]

	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", "internal server error", http.StatusInternalServerError)

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(),
		portCache:     cache,
	}
	_, err := n.GetNetworkSegment(NetworkIDFixture)
	assert.Error(t, err)
	assert.False(t, errors.Is(err, aErrors.ErrNoPhysNetFound))
	assert.True(t, gophercloud.ResponseCodeIs(err, http.StatusInternalServerError))
	assert.ErrorContains(t, err, "internal server error")
}
