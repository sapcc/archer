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
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/fixture"
	lru "github.com/hashicorp/golang-lru/v2"

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

func TestNeutronClient_GetNetworkSegment(t *testing.T) {
	type fields struct {
		ServiceClient *gophercloud.ServiceClient
		cache         *lru.Cache[string, int]
	}
	type args struct {
		networkId string
	}

	th.SetupPersistentPortHTTP(t, 8931)
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
				cache:         tt.fields.cache,
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
	th.TeardownHTTP()
}
