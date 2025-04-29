// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package neutron

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
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
