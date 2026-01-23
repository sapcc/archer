// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"github.com/go-openapi/strfmt"
	"github.com/sapcc/archer/models"
)

type ServiceInjection struct {
	models.Endpoint
	PortId          strfmt.UUID
	Network         strfmt.UUID
	IpAddress       strfmt.IPv4
	ServicePorts    []int
	ServiceProtocol string
}
