// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"

	"github.com/sapcc/archer/client/endpoint"
	"github.com/sapcc/archer/client/service"
)

// ResolveServiceID resolves a service name or ID to a UUID.
// If the input is a valid UUID, it returns it directly.
// If it's a name, it looks up the service and returns its ID.
// Returns an error if no service is found or multiple services have the same name.
func ResolveServiceID(nameOrID string) (strfmt.UUID, error) {
	// Check if it's already a valid UUID
	if strfmt.IsUUID(nameOrID) {
		return strfmt.UUID(nameOrID), nil
	}

	// Look up by name
	params := service.NewGetServiceParams()
	resp, err := ArcherClient.Service.GetService(params, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list services: %w", err)
	}

	var matches []strfmt.UUID
	for _, svc := range resp.GetPayload().Items {
		if svc.Name == nameOrID {
			matches = append(matches, svc.ID)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("service not found: %s", nameOrID)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple services found with name '%s', use ID instead", nameOrID)
	}
}

// ResolveEndpointID resolves an endpoint name or ID to a UUID.
// If the input is a valid UUID, it returns it directly.
// If it's a name, it looks up the endpoint and returns its ID.
// Returns an error if no endpoint is found or multiple endpoints have the same name.
func ResolveEndpointID(nameOrID string) (strfmt.UUID, error) {
	// Check if it's already a valid UUID
	if strfmt.IsUUID(nameOrID) {
		return strfmt.UUID(nameOrID), nil
	}

	// Look up by name
	params := endpoint.NewGetEndpointParams()
	resp, err := ArcherClient.Endpoint.GetEndpoint(params, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list endpoints: %w", err)
	}

	var matches []strfmt.UUID
	for _, ep := range resp.GetPayload().Items {
		if ep.Name == nameOrID {
			matches = append(matches, ep.ID)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("endpoint not found: %s", nameOrID)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple endpoints found with name '%s', use ID instead", nameOrID)
	}
}

// ResolveNetworkID resolves a network name or ID to a UUID.
// If the input is a valid UUID, it returns it directly.
// If it's a name, it queries Neutron and returns the network ID.
// Returns an error if no network is found or multiple networks have the same name.
func ResolveNetworkID(nameOrID string) (strfmt.UUID, error) {
	// Check if it's already a valid UUID
	if strfmt.IsUUID(nameOrID) {
		return strfmt.UUID(nameOrID), nil
	}

	// Need Provider to be initialized for Neutron access
	if Provider == nil {
		return "", fmt.Errorf("network name lookup requires authentication (not available with --os-token + --os-endpoint)")
	}

	// Create Neutron client
	networkClient, err := openstack.NewNetworkV2(Provider, gophercloud.EndpointOpts{})
	if err != nil {
		return "", fmt.Errorf("failed to create network client: %w", err)
	}

	// List networks filtered by name
	listOpts := networks.ListOpts{
		Name: nameOrID,
	}
	allPages, err := networks.List(networkClient, listOpts).AllPages(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to list networks: %w", err)
	}

	allNetworks, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return "", fmt.Errorf("failed to extract networks: %w", err)
	}

	switch len(allNetworks) {
	case 0:
		return "", fmt.Errorf("network not found: %s", nameOrID)
	case 1:
		return strfmt.UUID(allNetworks[0].ID), nil
	default:
		return "", fmt.Errorf("multiple networks found with name '%s', use ID instead", nameOrID)
	}
}
