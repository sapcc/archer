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
	"github.com/sapcc/archer/internal/agent/as3"
	"net/url"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/dns"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/config"
)

func ConnectToNeutron(providerClient *gophercloud.ProviderClient) (*gophercloud.ServiceClient, error) {
	serviceClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	// Set timeout to 30 secs
	serviceClient.HTTPClient.Timeout = time.Second * 30
	return serviceClient, nil
}

func GetNetworkSegment(cache *lru.Cache[string, int], c *gophercloud.ServiceClient, networkId string) (int, error) {
	if segment, ok := cache.Get(networkId); ok {
		return segment, nil
	}

	var network provider.NetworkProviderExt
	r := networks.Get(c, networkId)
	if err := r.ExtractInto(&network); err != nil {
		return 0, err
	}

	for _, segment := range network.Segments {
		if segment.PhysicalNetwork == config.Global.Agent.PhysicalNetwork {
			cache.Add(networkId, segment.SegmentationID)
			return segment.SegmentationID, nil
		}
	}

	return 0, fmt.Errorf("Could not find physical-network %s for network '%s'",
		config.Global.Agent.PhysicalNetwork,
		networkId)
}

func GetSNATPort(c *gophercloud.ServiceClient, portId *strfmt.UUID) (*ports.Port, error) {
	if portId == nil {
		return nil, nil
	}
	portResult := ports.Get(c, portId.String())
	return portResult.Extract()
}

func AllocateSNATPort(c *gophercloud.ServiceClient, service *as3.ExtendedService) (*ports.Port, error) {
	logg.Debug("Creating SNATPool Neutron port for service '%s' in network '%s'",
		service.ID, service.NetworkID)
	port := dns.PortCreateOptsExt{
		CreateOptsBuilder: portsbinding.CreateOptsExt{
			CreateOptsBuilder: ports.CreateOpts{
				Name:        fmt.Sprintf("archer snatpool %s", service.ID),
				DeviceOwner: "network:archer",
				DeviceID:    service.ID.String(),
				NetworkID:   service.NetworkID.String(),
				TenantID:    string(service.ProjectID),
			},
			HostID: "TODO",
		},
		DNSName: "TODO",
	}
	r := ports.Create(c, port)
	return r.Extract()
}

func DeleteSNATPort(c *gophercloud.ServiceClient, portId *strfmt.UUID) error {
	return ports.Delete(c, portId.String()).Err
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
