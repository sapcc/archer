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

package agent

import (
	"fmt"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/sapcc/go-bits/logg"
	"net/url"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/dns"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/utils/openstack/clientconfig"

	"github.com/sapcc/archer/internal/config"
)

func (a *Agent) ConnectToNeutron() error {
	authInfo := clientconfig.AuthInfo(config.Global.ServiceAuth)
	providerClient, err := clientconfig.AuthenticatedClient(&clientconfig.ClientOpts{
		AuthInfo: &authInfo})
	if err != nil {
		return err
	}

	if a.neutron, err = openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{}); err != nil {
		return err
	}

	// Set timeout to 30 secs
	a.neutron.HTTPClient.Timeout = time.Second * 30
	return nil
}

func (a *Agent) GetNetworkSegment(networkId string) (int, error) {
	if segment, ok := a.cache.Get(networkId); ok {
		return segment, nil
	}

	var network provider.NetworkProviderExt
	r := networks.Get(a.neutron, networkId)
	if err := r.ExtractInto(&network); err != nil {
		return 0, err
	}

	for _, segment := range network.Segments {
		if segment.PhysicalNetwork == config.Global.F5Config.PhysicalNetwork {
			a.cache.Add(networkId, segment.SegmentationID)
			return segment.SegmentationID, nil
		}
	}

	return 0, fmt.Errorf("Could not find physical-network %s for network '%s'",
		config.Global.F5Config.PhysicalNetwork,
		networkId)
}

func (a *Agent) GetSNATPort(portId *strfmt.UUID) (*ports.Port, error) {
	portResult := ports.Get(a.neutron, portId.String())
	return portResult.Extract()
}

func (a *Agent) AllocateSNATPort(service *ExtendedService) (*ports.Port, error) {
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
	r := ports.Create(a.neutron, port)
	return r.Extract()
}

func (a *Agent) DeleteSNATPort(service *ExtendedService) error {
	return ports.Delete(a.neutron, service.SnatPortId.String()).Err
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
