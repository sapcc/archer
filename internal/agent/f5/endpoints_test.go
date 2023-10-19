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
	"context"
	"net/http"
	"testing"

	"github.com/go-openapi/strfmt"
	fake "github.com/gophercloud/gophercloud/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/fixture"
	"github.com/pashagolub/pgxmock/v3"

	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/neutron"
)

const GetNetworkResponseFixture = `
{
    "network": {
        "id": "35a3ca82-62af-4e0a-9472-92331500fb3a",
        "subnets": ["e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e"],
		"project_id": "test-project-1",
		"segments": [
			{
				"provider:physical_network": "physnet1",
				"provider:network_type": "vlan",
				"provider:segmentation_id": 123
			}
		]
    }
}
`

const GetServiceNetworkResponseFixture = `
{
    "network": {
        "id": "b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b",
		"segments": [
			{
				"provider:physical_network": "physnet1",
				"provider:network_type": "vlan",
				"provider:segmentation_id": 666
			}
		]
    }
}
`

const GetNetworkListResponseFixture = `
{
	"ports": [
		{
			"id": "c0c0c0c0-c0c0-4c0c-8c0c-0c0c0c0c0c0c",
			"network_id": "35a3ca82-62af-4e0a-9472-92331500fb3a",
			"project_id": "test-project-1"
		}
	]
}
`

const PostBigIPFixture = `{
  "persist": false,
  "class": "AS3",
  "action": "deploy",
  "declaration": {
    "class": "ADC",
    "id": "urn:uuid:07649173-4AF7-48DF-963F-84000C70F0DD",
    "net-35a3ca82-62af-4e0a-9472-92331500fb3a": {
      "class": "Tenant",
      "si-endpoints": {
        "class": "Application",
        "endpoint-95dbe813-62f9-47f1-90ba-09f2dadcaefa": {
          "label": "endpoint-95dbe813-62f9-47f1-90ba-09f2dadcaefa",
          "class": "Service_L4",
          "allowVlans": [
            "/Common/vlan-123"
          ],
          "iRules": [],
          "mirroring": "L4",
          "persistenceMethods": [],
          "pool": {
            "bigip": "/Common/Shared/pool-a0a0a0a0-a0a0-4a0a-8a0a-0a0a0a0a0a0a"
          },
          "profileL4": {
            "bigip": "/Common/cc_fastL4_profile"
          },
          "snat": {
            "bigip": "/Common/Shared/snatpool-a0a0a0a0-a0a0-4a0a-8a0a-0a0a0a0a0a0a"
          },
          "virtualAddresses": null,
          "translateServerPort": true,
          "virtualPort": 80
        },
        "template": "generic"
      }
    },
    "schemaVersion": "3.36.0",
    "updateMode": "selective"
  }
}`

func TestAgent_ProcessEndpoint(t *testing.T) {
	endpoint := strfmt.UUID("95dbe813-62f9-47f1-90ba-09f2dadcaefa")
	port := strfmt.UUID("c0c0c0c0-c0c0-4c0c-8c0c-0c0c0c0c0c0c")
	network := strfmt.UUID("35a3ca82-62af-4e0a-9472-92331500fb3a")
	subnet := strfmt.UUID("e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e")
	service := strfmt.UUID("a0a0a0a0-a0a0-4a0a-8a0a-0a0a0a0a0a0a")
	serviceNetwork := strfmt.UUID("b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b")

	th.SetupPersistentPortHTTP(t, 8931)
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, "/v2.0/networks/"+network.String(), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/networks/"+serviceNetwork.String(), "GET",
		"", GetServiceNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, "/v2.0/ports", "GET", "",
		GetNetworkListResponseFixture, http.StatusOK)

	ctx := context.Background()
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	bigiphost := as3.NewMockBigIPIface(t)
	bigiphost.EXPECT().
		PostAs3Bigip(PostBigIPFixture, "net-35a3ca82-62af-4e0a-9472-92331500fb3a").
		Return(nil, "", "")

	config.Global.Default.Host = "host-123"
	a := &Agent{
		jobQueue: nil,
		pool:     dbMock,
		neutron:  &neutron.NeutronClient{ServiceClient: fake.ServiceClient()},
		bigips:   []*as3.BigIP{},
		vcmps:    []*as3.BigIP{},
		bigip:    &as3.BigIP{Host: "dummybigiphost", BigIPIface: bigiphost},
	}
	dbMock.
		ExpectBegin()
	dbMock.ExpectQuery("SELECT network, owned FROM endpoint_port WHERE endpoint_id = $1").
		WithArgs(endpoint).
		WillReturnRows(pgxmock.NewRows([]string{"network", "owned"}).AddRow(network, true))
	dbMock.ExpectQuery("SELECT endpoint.*, service.port AS service_port_nr, service.proxy_protocol, service.network_id AS service_network_id, endpoint_port.segment_id, endpoint_port.port_id AS \"target.port\", endpoint_port.network AS \"target.network\", endpoint_port.subnet AS \"target.subnet\" FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE network = $1 AND service.host = $2 AND service.provider = 'tenant' FOR UPDATE of endpoint").
		WithArgs(network, config.Global.Default.Host).
		WillReturnRows(pgxmock.
			NewRows([]string{"id", "service_id", "name", "service_port_nr", "proxy_protocol", "service_network_id", "segment_id", "target.port", "target.network", "target.subnet"}).
			AddRow(endpoint, service, "test-service", int32(80), false, serviceNetwork, nil, &port, &network, &subnet))
	dbMock.ExpectExec("UPDATE endpoint_port SET segment_id = $1 WHERE endpoint_id = $2").
		WithArgs(123, endpoint).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	dbMock.ExpectExec("UPDATE endpoint SET status = 'AVAILABLE', updated_at = NOW() WHERE id = $1;").
		WithArgs(endpoint).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	dbMock.ExpectCommit()

	if err := a.ProcessEndpoint(ctx, endpoint); err != nil {
		t.Errorf("Agent.ProcessEndpoint() error = %v", err)
	}
	if err := dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
