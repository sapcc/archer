// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package neutron

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/hashicorp/golang-lru/v2/expirable"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
	aErrors "github.com/sapcc/archer/internal/errors"
	"github.com/sapcc/archer/models"
)

type NetworkMTU struct {
	networks.Network
	mtu.NetworkMTUExt
	provider.NetworkProviderExt
}

type NeutronClient struct {
	*gophercloud.ServiceClient
	portCache    *expirable.LRU[string, map[string]*ports.Port] // networkID -> map[hostname]*ports.port, 10 min expiry
	networkCache *expirable.LRU[string, *NetworkMTU]            // networkID -> *networks.network, expires after 10 mins
	subnetCache  *expirable.LRU[string, *subnets.Subnet]        // subnetID -> *subnets.subnet, expires after 10 mins
}

func (n *NeutronClient) InitCache() {
	// Initialize local cache
	n.portCache = expirable.NewLRU[string, map[string]*ports.Port](32, nil, time.Minute*10)
	n.networkCache = expirable.NewLRU[string, *NetworkMTU](32, nil, time.Minute*10)
	n.subnetCache = expirable.NewLRU[string, *subnets.Subnet](32, nil, time.Minute*10)
}

func (n *NeutronClient) ResetCache() {
	n.portCache.Purge()
	n.networkCache.Purge()
	n.subnetCache.Purge()
}

func ConnectToNeutron(providerClient *gophercloud.ProviderClient) (*NeutronClient, error) {
	serviceClient, err := openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	// Set timeout to 30 secs
	serviceClient.HTTPClient.Timeout = time.Second * 30
	return &NeutronClient{ServiceClient: serviceClient}, nil
}

// GetNetworkSegment return the segmentation ID for the given network
// throws ErrNoPhysNetFound if the physical network is not found
func (n *NeutronClient) GetNetworkSegment(networkID string) (int, error) {
	network, err := n.GetNetwork(networkID)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return 0, fmt.Errorf("%w, network not found, %s", aErrors.ErrNoPhysNetFound, err.Error())
		}
		return 0, err
	}

	for _, segment := range network.Segments {
		if segment.PhysicalNetwork == config.Global.Agent.PhysicalNetwork {
			return segment.SegmentationID, nil
		}
	}

	return 0, fmt.Errorf("%w, physnet '%s' not found for network '%s'",
		aErrors.ErrNoPhysNetFound, config.Global.Agent.PhysicalNetwork, networkID)
}

func (n *NeutronClient) GetSubnetSegment(subnetID string) (int, error) {
	subnet, err := n.GetSubnet(subnetID)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return 0, fmt.Errorf("%w, subnet not found, %s", aErrors.ErrNoPhysNetFound, err.Error())
		}
		return 0, err
	}

	return n.GetNetworkSegment(subnet.NetworkID)
}

// GetNetworkMTU returns the MTU of the network
// throws ErrNoPhysNetFound if the physical network is not found
func (n *NeutronClient) GetNetworkMTU(networkID string) (int, error) {
	network, err := n.GetNetwork(networkID)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return 0, fmt.Errorf("%w, network not found, %s", aErrors.ErrNoPhysNetFound, err.Error())
		}

		return 0, err
	}

	return network.MTU, nil
}

func (n *NeutronClient) GetPort(portId string) (*ports.Port, error) {
	return ports.Get(context.Background(), n.ServiceClient, portId).Extract()
}

func (n *NeutronClient) DeletePort(portId string) error {
	return ports.Delete(context.Background(), n.ServiceClient, portId).ExtractErr()
}

type fixedIP struct {
	SubnetID string `json:"subnet_id"`
}

func (n *NeutronClient) AllocateNeutronEndpointPort(target *models.EndpointTarget, endpoint *models.Endpoint,
	projectID string, host string, client *gophercloud.ServiceClient) (*ports.Port, error) {

	if target.Port != nil {
		port, err := ports.Get(context.Background(), client, target.Port.String()).Extract()
		if err != nil {
			return nil, err
		}

		if port.ProjectID != projectID {
			return nil, aErrors.ErrProjectMismatch
		}

		if len(port.FixedIPs) < 1 {
			return nil, aErrors.ErrMissingIPAddress
		}

		return port, nil
	}

	var fixedIPs []fixedIP
	if target.Network == nil {
		subnet, err := subnets.Get(context.Background(), client, target.Subnet.String()).Extract()
		if err != nil {
			return nil, err
		}

		fixedIPs = append(fixedIPs, fixedIP{subnet.ID})
		networkID := strfmt.UUID(subnet.NetworkID)
		target.Network = &networkID
	} else {
		network, err := n.GetNetwork(target.Network.String())
		if err != nil {
			return nil, err
		}
		if len(network.Subnets) == 0 {
			return nil, aErrors.ErrMissingSubnets
		}

		fixedIPs = append(fixedIPs, fixedIP{network.Subnets[0]})
	}

	// allocate neutron port
	port := portsbinding.CreateOptsExt{
		CreateOptsBuilder: ports.CreateOpts{
			Name:        fmt.Sprintf("endpoint-%s", endpoint.ID),
			DeviceOwner: "network:archer",
			DeviceID:    endpoint.ID.String(),
			NetworkID:   target.Network.String(),
			TenantID:    projectID,
			FixedIPs:    fixedIPs,
		},
		HostID: host,
	}

	res, err := ports.Create(context.Background(), n.ServiceClient, port).Extract()
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (n NeutronClient) ClearCache(networkID string) {
	n.portCache.Remove(networkID)
}

// TODO: Remove after a while
func (n *NeutronClient) FetchSNATPorts(networkID string) (map[string]*ports.Port, error) {
	if p, ok := n.portCache.Get(networkID); ok {
		return p, nil
	}

	portMap := make(map[string]*ports.Port)
	opts := PortListOptsExt{
		ListOptsBuilder: ports.ListOpts{
			NetworkID:   networkID,
			DeviceOwner: "network:f5snat",
		},
		HostID: config.Global.Default.Host,
	}
	pages, err := ports.List(n.ServiceClient, opts).AllPages(context.Background())
	if err != nil {
		return nil, err
	}
	snatPorts, err := ports.ExtractPorts(pages)
	if err != nil {
		return nil, err
	}

	for _, port := range snatPorts {
		hostname := strings.TrimPrefix(port.Name, "local-")
		portMap[hostname] = &port
	}
	n.portCache.Add(networkID, portMap)
	return portMap, nil
}

func (n *NeutronClient) GetNetwork(networkID string) (*NetworkMTU, error) {
	if network, ok := n.networkCache.Get(networkID); ok {
		return network, nil
	}

	var network NetworkMTU
	err := networks.Get(context.Background(), n.ServiceClient, networkID).ExtractInto(&network)
	if err != nil {
		return nil, err
	}
	n.networkCache.Add(networkID, &network)
	return &network, nil
}

func (n *NeutronClient) GetSubnet(subnetID string) (*subnets.Subnet, error) {
	if network, ok := n.subnetCache.Get(subnetID); ok {
		return network, nil
	}

	subnet, err := subnets.Get(context.Background(), n.ServiceClient, subnetID).Extract()
	if err != nil {
		return nil, err
	}
	n.subnetCache.Add(subnetID, subnet)
	return subnet, nil
}

func (n *NeutronClient) FetchSelfIPPorts() (map[string][]*ports.Port, error) {
	portMap := make(map[string][]*ports.Port)
	opts := PortListOptsExt{
		ListOptsBuilder: ports.ListOpts{
			DeviceOwner: "network:f5selfip",
		},
		HostID: config.Global.Default.Host,
	}
	pages, err := ports.List(n.ServiceClient, opts).AllPages(context.Background())
	if err != nil {
		return nil, err
	}
	selfIPPorts, err := ports.ExtractPorts(pages)
	if err != nil {
		return nil, err
	}

	for _, port := range selfIPPorts {
		portMap[port.NetworkID] = append(portMap[port.NetworkID], &port)
	}
	return portMap, nil
}

// EnsureNeutronSelfIPs ensures that a SelfIPs exists for the given deviceID and subnetID
func (n *NeutronClient) EnsureNeutronSelfIPs(deviceIDs []string, subnetID string, dryRun bool) (map[string]*ports.Port, error) {
	log.WithFields(log.Fields{
		"subnet":  subnetID,
		"devices": deviceIDs,
		"dry_run": dryRun,
	}).Debug("EnsureNeutronSelfIPs")
	subnet, err := n.GetSubnet(subnetID)
	if err != nil {
		return nil, err
	}

	var pages pagination.Page
	opts := PortListOptsExt{
		ListOptsBuilder: ports.ListOpts{
			NetworkID:   subnet.NetworkID,
			FixedIPs:    []ports.FixedIPOpts{{SubnetID: subnetID}},
			DeviceOwner: "network:f5selfip",
		},
		HostID: config.Global.Default.Host,
	}
	pages, err = ports.List(n.ServiceClient, opts).AllPages(context.Background())
	if err != nil {
		return nil, err
	}

	neutronPorts, err := ports.ExtractPorts(pages)
	if err != nil {
		return nil, err
	}

	selfIPs := make(map[string]*ports.Port, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		for _, neutronPort := range neutronPorts {
			if neutronPort.DeviceOwner != "network:f5selfip" {
				continue
			}

			if neutronPort.Name == fmt.Sprintf("local-%s", deviceID) {
				neutronPort := neutronPort
				selfIPs[deviceID] = &neutronPort
			}
		}

		if _, ok := selfIPs[deviceID]; !ok && !dryRun {
			// allocate neutron port
			log.WithFields(log.Fields{
				"network": subnet.NetworkID,
				"subnet":  subnetID,
				"device":  deviceID,
			}).Info("EnsureNeutronSelfIP: Allocating new SelfIP")
			port := portsbinding.CreateOptsExt{
				CreateOptsBuilder: ports.CreateOpts{
					Name:        fmt.Sprintf("local-%s", deviceID),
					Description: fmt.Sprintf("Archer SelfIP for device %s", deviceID),
					DeviceOwner: "network:f5selfip",
					DeviceID:    subnetID,
					NetworkID:   subnet.NetworkID,
					TenantID:    subnet.TenantID,
					FixedIPs:    []fixedIP{{SubnetID: subnetID}},
				},
				HostID: config.Global.Default.Host,
			}

			selfIPs[deviceID], err = ports.Create(context.Background(), n.ServiceClient, port).Extract()
			if err != nil {
				return nil, err
			}
		}
	}
	return selfIPs, nil
}

func (n *NeutronClient) GetMask(subnetID string) (int, error) {
	subnet, err := n.GetSubnet(subnetID)
	if err != nil {
		return 0, err
	}
	_, ipNet, err := net.ParseCIDR(subnet.CIDR)
	if err != nil {
		return 0, err
	}
	mask, _ := ipNet.Mask.Size()
	return mask, nil
}
