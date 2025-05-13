// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package as3

import (
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"

	"github.com/sapcc/archer/models"
)

// ExtendedEndpoint is an endpoint with additional fields...
type ExtendedEndpoint struct {
	models.Endpoint
	Port             *ports.Port
	ServicePortNr    int32
	ServiceNetworkId strfmt.UUID
	SegmentId        *int
	ProxyProtocol    bool
	Owned            bool
}
