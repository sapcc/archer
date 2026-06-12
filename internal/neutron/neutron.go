// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package neutron

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/mtu"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/provider"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/hashicorp/golang-lru/v2/expirable"

	aErrors "github.com/sapcc/archer/v2/internal/errors"
)

type NetworkMTU struct {
	networks.Network
	mtu.NetworkMTUExt
	provider.NetworkProviderExt
}

type NeutronClient struct {
	*gophercloud.ServiceClient
	networkCache *expirable.LRU[string, *NetworkMTU]       // networkID -> *networks.network, expires after 10 mins
	subnetCache  *expirable.LRU[string, *subnets.Subnet]   // subnetID -> *subnets.subnet, expires after 10 mins
	portCache    *expirable.LRU[string, []PortWithBinding] // listOpts query -> ports + binding ext, 10 min expiry
}

func (n *NeutronClient) InitCache() {
	// Initialize local cache
	n.networkCache = expirable.NewLRU[string, *NetworkMTU](32, nil, time.Minute*10)
	n.subnetCache = expirable.NewLRU[string, *subnets.Subnet](32, nil, time.Minute*10)
	n.portCache = expirable.NewLRU[string, []PortWithBinding](64, nil, time.Minute*10)
}

func (n *NeutronClient) ResetCache() {
	n.networkCache.Purge()
	n.subnetCache.Purge()
	n.portCache.Purge()
}

func (n *NeutronClient) RemoveFromCache(id string) {
	// Remove from network cache
	n.networkCache.Remove(id)

	// Remove from subnet cache
	n.subnetCache.Remove(id)
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
func (n *NeutronClient) GetNetworkSegment(ctx context.Context, networkID, physnet string) (int, error) {
	network, err := n.GetNetwork(ctx, networkID)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return 0, fmt.Errorf("%w, network not found, %s", aErrors.ErrNoPhysNetFound, err.Error())
		}
		return 0, err
	}

	for _, segment := range network.Segments {
		if segment.PhysicalNetwork == physnet {
			return segment.SegmentationID, nil
		}
	}

	return 0, fmt.Errorf("%w, physnet '%s' not found for network '%s'",
		aErrors.ErrNoPhysNetFound, physnet, networkID)
}

func (n *NeutronClient) GetSubnetSegment(ctx context.Context, subnetID, physnet string) (int, error) {
	subnet, err := n.GetSubnet(ctx, subnetID)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return 0, fmt.Errorf("%w, subnet not found, %s", aErrors.ErrNoPhysNetFound, err.Error())
		}
		return 0, err
	}

	return n.GetNetworkSegment(ctx, subnet.NetworkID, physnet)
}

// GetNetworkMTU returns the MTU of the network
// throws ErrNoPhysNetFound if the physical network is not found
func (n *NeutronClient) GetNetworkMTU(ctx context.Context, networkID string) (int, error) {
	network, err := n.GetNetwork(ctx, networkID)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return 0, fmt.Errorf("%w, network not found, %s", aErrors.ErrNoPhysNetFound, err.Error())
		}

		return 0, err
	}

	return network.MTU, nil
}

func (n *NeutronClient) GetNetwork(ctx context.Context, networkID string) (*NetworkMTU, error) {
	if network, ok := n.networkCache.Get(networkID); ok {
		return network, nil
	}

	var network NetworkMTU
	err := networks.Get(ctx, n.ServiceClient, networkID).ExtractInto(&network)
	if err != nil {
		return nil, err
	}
	n.networkCache.Add(networkID, &network)
	return &network, nil
}

func (n *NeutronClient) GetSubnet(ctx context.Context, subnetID string) (*subnets.Subnet, error) {
	if network, ok := n.subnetCache.Get(subnetID); ok {
		return network, nil
	}

	subnet, err := subnets.Get(ctx, n.ServiceClient, subnetID).Extract()
	if err != nil {
		return nil, err
	}
	n.subnetCache.Add(subnetID, subnet)
	return subnet, nil
}

func (n *NeutronClient) GetMask(ctx context.Context, subnetID string) (int, error) {
	subnet, err := n.GetSubnet(ctx, subnetID)
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
