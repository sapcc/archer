// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package as3

import (
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"

	"github.com/sapcc/archer/models"
)

// ExtendedService is a service with additional fields for snat ports etc.
type ExtendedService struct {
	models.Service
	NeutronPorts map[string]*ports.Port // SelfIPs / SNAT IPs
	SubnetID     string
	SegmentId    int
	MTU          int
}
