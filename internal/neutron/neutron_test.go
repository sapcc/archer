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
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud"
	fake "github.com/gophercloud/gophercloud/openstack/networking/v2/common"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/fixture"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
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
const PostPortRequestFixture = `{"port":{"binding:host_id":"testhost","device_id":"9bf57c58-5d9f-418b-a879-44d83e194ad0","device_owner":"network:f5snat","fixed_ips":[{"subnet_id":"a0304c3a-4f08-4c43-88af-d796509c97d2"}],"name":"local-testdevicehost","network_id":"9bf57c58-5d9f-418b-a879-44d83e194ad0","tenant_id":"test-project-1"}}`
const PostPortResponseFixture = `
{
	"port": {
		"admin_state_up": true,
		"allowed_address_pairs": [],
		"binding:host_id": "testhost",
		"binding:profile": {},
		"binding:vif_details": {},
		"binding:vif_type": "unbound",
		"binding:vnic_type": "normal",
		"created_at": "2021-03-18T14:20:00Z",
		"data_plane_status": null,
		"description": "",
		"device_id": "9bf57c58-5d9f-418b-a879-44d83e194ad0",
		"device_owner": "network:f5snat",
		"fixed_ips": [
			{
				"ip_address": "123.123.123.123",
				"subnet_id": "a0304c3a-4f08-4c43-88af-d796509c97d2"
			}
		],
		"id": "9bf57c58-5d9f-418b-a879-44d83e194ad0",
		"ip_allocation": "immediate",
		"mac_address": "fa:16:3e:4c:2c:2c",
		"name": "local-testdevicehost",
		"network_id": "9bf57c58-5d9f-418b-a879-44d83e194ad0",
		"port_security_enabled": true,
		"project_id": "test-project-1",
		"qos_network_policy_id": null,
		"revision_number": 1,
		"security_groups": [],
		"status": "DOWN",
		"tags": []
	}
}
`

func TestNeutronClient_GetNetworkSegment(t *testing.T) {
	type fields struct {
		ServiceClient *gophercloud.ServiceClient
		cache         *expirable.LRU[string, map[string]*ports.Port]
	}
	type args struct {
		networkId string
	}

	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "TestGetNetworkSegment",
			fields: fields{
				ServiceClient: fake.ServiceClient(),
			},
			args: args{
				networkId: NetworkIDFixture,
			},
			want:    100,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NeutronClient{
				ServiceClient: tt.fields.ServiceClient,
				portCache:     tt.fields.cache,
			}
			got, err := n.GetNetworkSegment(tt.args.networkId)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNetworkSegment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetNetworkSegment() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNeutronClient_AllocateSNATNeutronPort(t *testing.T) {
	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	config.Global.Default.Host = "testhost"
	fixture.SetupHandler(t, "/v2.0/networks/"+NetworkIDFixture, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/ports", "POST",
		PostPortRequestFixture, PostPortResponseFixture, http.StatusCreated)

	n := &NeutronClient{
		ServiceClient: fake.ServiceClient(),
	}
	h := "testdevicehost"

	got, err := n.AllocateSNATPort(h, NetworkIDFixture)
	assert.Nil(t, err)
	assert.Equal(t, "local-testdevicehost", got.Name)
}
