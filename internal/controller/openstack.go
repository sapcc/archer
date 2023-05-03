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

package controller

import (
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/models"
)

func (c *Controller) AllocateNeutronPort(target *models.EndpointTarget, projectID string) (*ports.Port, error) {
	if target.Port != nil {
		port, err := ports.Get(c.neutron, target.Port.String()).Extract()
		if err != nil {
			return nil, ErrPortNotFound
		}

		if port.ProjectID != projectID {
			return nil, ErrProjectMismatch
		}

		if len(port.FixedIPs) < 1 {
			return nil, ErrMissingIPAddress
		}

		return port, nil

	}
	// allocate neutron port

	if target.Network == nil {
		subnet, err := subnets.Get(c.neutron, target.Subnet.String()).Extract()
		if err != nil {
			return nil, err
		}

		networkID := strfmt.UUID(subnet.NetworkID)
		target.Network = &networkID
	}

	var fixedIPs []ports.FixedIPOpts
	if target.Subnet != nil {
		fixedIPs = append(fixedIPs, ports.FixedIPOpts{SubnetID: target.Subnet.String()})
	}

	port := portsbinding.CreateOptsExt{
		CreateOptsBuilder: ports.CreateOpts{
			Name:        "service-endpoint-pending",
			DeviceOwner: "network-injector", // TODO: scheduler host
			DeviceID:    "todo",
			NetworkID:   target.Network.String(),
			TenantID:    projectID,
			FixedIPs:    fixedIPs,
		},
		HostID: config.Global.Default.Host,
	}

	res, err := ports.Create(c.neutron, port).Extract()
	if err != nil {
		return nil, err
	}
	return res, nil
}
