// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"github.com/go-openapi/strfmt"
	"github.com/sapcc/archer/models"
)

// ServiceInjection contains all data needed to set up network injection for an endpoint.
// It combines endpoint data with service configuration for HAProxy and netlink setup.
type ServiceInjection struct {
	models.Endpoint              // Embedded endpoint with ID, status, etc.
	PortId           strfmt.UUID // Neutron port ID for the endpoint
	Network          strfmt.UUID // Network ID where the endpoint resides
	ServiceID        strfmt.UUID // ID of the service this endpoint belongs to
	ServicePorts     []int       // Ports exposed by the service
	ServiceProtocol  string      // Protocol type (HTTP or TCP)
	ServiceIPAddress strfmt.IPv4 // First IP address of the service (upstream target)
}
