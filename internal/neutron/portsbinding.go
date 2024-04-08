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
	"net/url"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
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
		params.Add("fixed_ips", _fixedIP.String())
	}

	if opts.HostID != "" {
		params.Add("binding:host_id", opts.HostID)
	}

	q = &url.URL{RawQuery: params.Encode()}
	return q.String(), err
}
