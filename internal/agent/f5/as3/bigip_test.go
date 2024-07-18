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

package as3

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/models"
)

func TestGetEndpointTenants(t *testing.T) {

	config.Global.Agent.L4Profile = "test-l4-profile"
	config.Global.Agent.TCPProfile = "test-tcp-profile"
	endpoints := []*ExtendedEndpoint{
		{
			Endpoint: models.Endpoint{
				CreatedAt:   time.Time{},
				Description: "test-description",
				ID:          "3ad9b1f0-4e5a-44c3-ada6-71696925ae64",
				IPAddress:   "1.2.3.4",
				Name:        "test-name",
				ProjectID:   "test-project",
				ServiceID:   strfmt.UUID("4e50bf87-e597-41f2-9ce0-83d3e24dedf3"),
				Target:      models.EndpointTarget{},
				UpdatedAt:   time.Time{},
			},
			ProxyProtocol: false,
			Port: &ports.Port{
				FixedIPs: []ports.IP{{IPAddress: "1.2.3.4"}},
			},
			SegmentId: swag.Int(1),
		},
	}
	tenant := GetEndpointTenants(endpoints)
	assert.NotNil(t, tenant)
	json, err := tenant.MarshalJSON()
	assert.Nil(t, err)

	expectedJSON := `{"class":"Tenant","si-endpoints":{"class":"Application","endpoint-3ad9b1f0-4e5a-44c3-ada6-71696925ae64":{"label":"endpoint-3ad9b1f0-4e5a-44c3-ada6-71696925ae64","class":"Service_L4","allowVlans":["/Common/vlan-1"],"iRules":[],"mirroring":"L4","persistenceMethods":[],"pool":{"bigip":"/Common/Shared/pool-4e50bf87-e597-41f2-9ce0-83d3e24dedf3"},"profileL4":{"bigip":"test-l4-profile"},"profileTCP":{"bigip":"test-tcp-profile"},"snat":{"bigip":"/Common/Shared/snatpool-4e50bf87-e597-41f2-9ce0-83d3e24dedf3"},"virtualAddresses":["1.2.3.4%1"],"translateServerPort":true,"virtualPort":0},"template":"generic"}}`
	assert.JSONEq(t, expectedJSON, string(json), "Tenant JSON should be equal")

}
