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
	"net/url"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/errors"
	"github.com/sapcc/archer/models"
)

type NeutronClient struct {
	*gophercloud.ServiceClient
	cache *lru.Cache[string, int]
}

func ConnectToNeutron(providerClient *gophercloud.ProviderClient) (*NeutronClient, error) {
	serviceClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	// Set timeout to 30 secs
	serviceClient.HTTPClient.Timeout = time.Second * 30

	// Initialize local cache
	var cache *lru.Cache[string, int]
	if cache, err = lru.New[string, int](128); err != nil {
		logg.Fatal(err.Error())
	}
	return &NeutronClient{serviceClient, cache}, nil
}

func (n *NeutronClient) GetNetworkSegment(networkId string) (int, error) {
	if segment, ok := n.cache.Get(networkId); ok {
		return segment, nil
	}

	var network provider.NetworkProviderExt
	r := networks.Get(n.ServiceClient, networkId)
	if err := r.ExtractInto(&network); err != nil {
		return 0, err
	}

	for _, segment := range network.Segments {
		if segment.PhysicalNetwork == config.Global.Agent.PhysicalNetwork {
			n.cache.Add(networkId, segment.SegmentationID)
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
	projectID string, host string) (*ports.Port, error) {
	if target.Port != nil {
		port, err := ports.Get(n.ServiceClient, target.Port.String()).Extract()
		if err != nil {
			return nil, errors.ErrPortNotFound
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
		subnet, err := subnets.Get(n.ServiceClient, target.Subnet.String()).Extract()
		if err != nil {
			return nil, err
		}

		fixedIPs = append(fixedIPs, fixedIP{subnet.ID})
		networkID := strfmt.UUID(subnet.NetworkID)
		target.Network = &networkID
	} else {
		network, err := networks.Get(n.ServiceClient, target.Network.String()).Extract()
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

func (n *NeutronClient) AllocateSNATNeutronPort(service *models.Service, hostname string) (*ports.Port, error) {
	var fixedIPs []fixedIP
	network, err := networks.Get(n.ServiceClient, service.NetworkID.String()).Extract()
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
			Name:        fmt.Sprintf("local-%s", hostname),
			DeviceOwner: "network:f5snat",
			DeviceID:    service.ID.String(),
			NetworkID:   service.NetworkID.String(),
			TenantID:    string(service.ProjectID),
			FixedIPs:    fixedIPs,
		},
		HostID: *service.Host,
	}

	res, err := ports.Create(n.ServiceClient, port).Extract()
	if err != nil {
		return nil, err
	}
	return res, nil
}

type PortListOpts struct {
	IDs []string
}

func (opts PortListOpts) ToPortListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	params := q.Query()
	for _, id := range opts.IDs {
		params.Add("id", id)
	}
	q = &url.URL{RawQuery: params.Encode()}
	return q.String(), err
}
