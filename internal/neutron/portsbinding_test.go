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

package neutron

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/stretchr/testify/assert"
)

func TestToPortListQuery(t *testing.T) {
	networkID := "test-network-id"
	subnetID := "test-subnet-id"
	hostID := "test-host-id"
	deviceOwner := "test-device-owner"

	opts := PortListOptsExt{
		HostID: hostID,
		ListOptsBuilder: ports.ListOpts{
			NetworkID:   networkID,
			FixedIPs:    []ports.FixedIPOpts{{SubnetID: subnetID}},
			DeviceOwner: deviceOwner,
		},
	}
	res, err := opts.ToPortListQuery()
	assert.Nil(t, err, "ToPortListQuery failed: %s", err)
	params := url.Values{}
	params.Add("binding:host_id", hostID)
	params.Add("device_owner", deviceOwner)
	params.Add("fixed_ips", fmt.Sprintf("subnet_id=%s", subnetID))
	params.Add("network_id", networkID)
	expected := fmt.Sprintf("?%s", params.Encode())
	assert.Equal(t, expected, res)
}
