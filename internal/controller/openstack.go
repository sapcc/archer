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
	"fmt"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"

	"github.com/sapcc/archer/models"
)

type fixedIP struct {
	SubnetID string `json:"subnet_id"`
}

func (c *Controller) AllocateNeutronEndpointPort(target *models.EndpointTarget, endpoint *models.Endpoint, projectID string, host string) (*ports.Port, error) {
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

	var fixedIPs []fixedIP
	if target.Network == nil {
		subnet, err := subnets.Get(c.neutron, target.Subnet.String()).Extract()
		if err != nil {
			return nil, err
		}

		fixedIPs = append(fixedIPs, fixedIP{subnet.ID})
		networkID := strfmt.UUID(subnet.NetworkID)
		target.Network = &networkID
	} else {
		network, err := networks.Get(c.neutron, target.Network.String()).Extract()
		if err != nil {
			return nil, err
		}
		if len(network.Subnets) == 0 {
			return nil, ErrMissingSubnets
		}

		fixedIPs = append(fixedIPs, fixedIP{network.Subnets[0]})
	}

	// allocate neutron port
	port := portsbinding.CreateOptsExt{
		CreateOptsBuilder: ports.CreateOpts{
			Name:        fmt.Sprintf("endpoint-%s", endpoint.ServiceName),
			DeviceOwner: "network:archer",
			DeviceID:    endpoint.ID.String(),
			NetworkID:   target.Network.String(),
			TenantID:    projectID,
			FixedIPs:    fixedIPs,
		},
		HostID: host,
	}

	res, err := ports.Create(c.neutron, port).Extract()
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Controller) AllocateSNATNeutronPort(service *models.Service) (*ports.Port, error) {
	var fixedIPs []fixedIP
	network, err := networks.Get(c.neutron, service.NetworkID.String()).Extract()
	if err != nil {
		return nil, err
	}
	if len(network.Subnets) == 0 {
		return nil, ErrMissingSubnets
	}
	fixedIPs = append(fixedIPs, fixedIP{network.Subnets[0]})

	// allocate neutron port
	port := portsbinding.CreateOptsExt{
		CreateOptsBuilder: ports.CreateOpts{
			Name:        fmt.Sprintf("endpoint-service-snat-%s", service.Name),
			DeviceOwner: "network:f5snat",
			DeviceID:    service.ID.String(),
			NetworkID:   service.NetworkID.String(),
			TenantID:    string(service.ProjectID),
			FixedIPs:    fixedIPs,
		},
		HostID: *service.Host,
	}

	res, err := ports.Create(c.neutron, port).Extract()
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Controller) DeallocateNeutronPort(portID string) error {
	return ports.Delete(c.neutron, portID).ExtractErr()
}
