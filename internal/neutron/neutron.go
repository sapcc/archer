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
	"fmt"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/errors"
	"github.com/sapcc/archer/models"
)

type NeutronClient struct {
	*gophercloud.ServiceClient
	portCache *expirable.LRU[string, map[string]*ports.Port]
}

func ConnectToNeutron(providerClient *gophercloud.ProviderClient) (*NeutronClient, error) {
	serviceClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	// Set timeout to 30 secs
	serviceClient.HTTPClient.Timeout = time.Second * 30

	// Initialize local cache
	cache := expirable.NewLRU[string, map[string]*ports.Port](32, nil, time.Minute*10)
	return &NeutronClient{serviceClient, cache}, nil
}

func (n *NeutronClient) GetNetworkSegment(networkId string) (int, error) {
	var network provider.NetworkProviderExt
	r := networks.Get(n.ServiceClient, networkId)
	if err := r.ExtractInto(&network); err != nil {
		return 0, err
	}

	for _, segment := range network.Segments {
		if segment.PhysicalNetwork == config.Global.Agent.PhysicalNetwork {
			return segment.SegmentationID, nil
		}
	}

	return 0, fmt.Errorf("could not find physical-network %s for network '%s'",
		config.Global.Agent.PhysicalNetwork, networkId)
}

func (n *NeutronClient) GetPort(portId string) (*ports.Port, error) {
	return ports.Get(n.ServiceClient, portId).Extract()
}

func (n *NeutronClient) DeletePort(portId string) error {
	return ports.Delete(n.ServiceClient, portId).ExtractErr()
}

type fixedIP struct {
	SubnetID string `json:"subnet_id"`
}

func (n *NeutronClient) AllocateNeutronEndpointPort(target *models.EndpointTarget, endpoint *models.Endpoint,
	projectID string, host string, client *gophercloud.ServiceClient) (*ports.Port, error) {
	if target.Port != nil {
		port, err := ports.Get(client, target.Port.String()).Extract()
		if err != nil {
			return nil, err
		}

		if port.ProjectID != projectID {
			return nil, errors.ErrProjectMismatch
		}

		if len(port.FixedIPs) < 1 {
			return nil, errors.ErrMissingIPAddress
		}

		return port, nil
	}

	var fixedIPs []fixedIP
	if target.Network == nil {
		subnet, err := subnets.Get(client, target.Subnet.String()).Extract()
		if err != nil {
			return nil, err
		}

		fixedIPs = append(fixedIPs, fixedIP{subnet.ID})
		networkID := strfmt.UUID(subnet.NetworkID)
		target.Network = &networkID
	} else {
		network, err := networks.Get(client, target.Network.String()).Extract()
		if err != nil {
			return nil, err
		}
		if len(network.Subnets) == 0 {
			return nil, errors.ErrMissingSubnets
		}

		fixedIPs = append(fixedIPs, fixedIP{network.Subnets[0]})
	}

	// allocate neutron port
	port := portsbinding.CreateOptsExt{
		CreateOptsBuilder: ports.CreateOpts{
			Name:        fmt.Sprintf("endpoint-%s", endpoint.ServiceID),
			DeviceOwner: "network:archer",
			DeviceID:    endpoint.ID.String(),
			NetworkID:   target.Network.String(),
			TenantID:    projectID,
			FixedIPs:    fixedIPs,
		},
		HostID: host,
	}

	res, err := ports.Create(n.ServiceClient, port).Extract()
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (n NeutronClient) ClearCache(networkId string) {
	n.portCache.Remove(networkId)
}

func (n *NeutronClient) FetchSNATPorts(networkId string) (map[string]*ports.Port, error) {
	if p, ok := n.portCache.Get(networkId); ok {
		return p, nil
	}

	portMap := make(map[string]*ports.Port)
	opts := PortListOptsExt{
		ListOptsBuilder: ports.ListOpts{
			NetworkID:   networkId,
			DeviceOwner: "network:f5snat",
		},
		HostID: config.Global.Default.Host,
	}
	pages, err := ports.List(n.ServiceClient, opts).AllPages()
	if err != nil {
		return nil, err
	}
	snatPorts, err := ports.ExtractPorts(pages)
	if err != nil {
		return nil, err
	}

	for _, port := range snatPorts {
		port := port
		hostname := strings.TrimPrefix(port.Name, "local-")
		portMap[hostname] = &port
	}
	n.portCache.Add(networkId, portMap)
	return portMap, nil
}

func (n *NeutronClient) AllocateSNATPort(deviceId string, networkId string) (*ports.Port, error) {
	var fixedIPs []fixedIP
	network, err := networks.Get(n.ServiceClient, networkId).Extract()
	if err != nil {
		return nil, err
	}
	if len(network.Subnets) == 0 {
		return nil, errors.ErrMissingSubnets
	}
	fixedIPs = append(fixedIPs, fixedIP{network.Subnets[0]})

	// allocate neutron port
	port := portsbinding.CreateOptsExt{
		CreateOptsBuilder: ports.CreateOpts{
			Name:        fmt.Sprintf("local-%s", deviceId),
			DeviceOwner: "network:f5snat",
			DeviceID:    networkId,
			NetworkID:   networkId,
			TenantID:    network.ProjectID,
			FixedIPs:    fixedIPs,
		},
		HostID: config.Global.Default.Host,
	}

	res, err := ports.Create(n.ServiceClient, port).Extract()
	if err != nil {
		return nil, err
	}
	return res, nil
}
