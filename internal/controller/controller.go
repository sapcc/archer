// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"github.com/go-openapi/loads"

	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/internal/neutron"
)

type Controller struct {
	spec    *loads.Document
	pool    db.PgxIface
	neutron *neutron.NeutronClient
}

func NewController(pool db.PgxIface, spec *loads.Document, client *neutron.NeutronClient) *Controller {
	return &Controller{pool: pool, spec: spec, neutron: client}
}
