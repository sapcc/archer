// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	"github.com/go-openapi/strfmt"

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
