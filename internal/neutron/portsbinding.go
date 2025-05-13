// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package neutron

import (
	"fmt"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
)

// PortListOptsExt adds the PortBinding options to the base port ListOpts.
type PortListOptsExt struct {
	ports.ListOptsBuilder

	// The ID of the host where the port is allocated
	HostID string
}

// ToPortListQuery adds the PortBinding options to the base port list options.
func (opts PortListOptsExt) ToPortListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts.ListOptsBuilder)
	if err != nil {
		return "", err
	}

	params := q.Query()

	// From ListOpts.FixedIPs
	for _, _fixedIP := range opts.ListOptsBuilder.(ports.ListOpts).FixedIPs {
		if _fixedIP.IPAddress != "" {
			params.Add("fixed_ips", fmt.Sprintf("ip_address=%s", _fixedIP.IPAddress))
		}
		if _fixedIP.IPAddressSubstr != "" {
			params.Add("fixed_ips", fmt.Sprintf("ip_address_substr=%s", _fixedIP.IPAddressSubstr))
		}
		if _fixedIP.SubnetID != "" {
			params.Add("fixed_ips", fmt.Sprintf("subnet_id=%s", _fixedIP.SubnetID))
		}
	}

	if opts.HostID != "" {
		params.Add("binding:host_id", opts.HostID)
	}

	q = &url.URL{RawQuery: params.Encode()}
	return q.String(), err
}
