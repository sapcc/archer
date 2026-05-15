// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"github.com/go-openapi/loads"

	"github.com/sapcc/archer/v2/internal/db"
	"github.com/sapcc/archer/v2/internal/neutron"
	"github.com/sapcc/archer/v2/internal/notifier"
)

type Controller struct {
	spec     *loads.Document
	pool     db.PgxIface
	neutron  *neutron.NeutronClient
	notifier *notifier.Notifier
}

func NewController(pool db.PgxIface, spec *loads.Document, client *neutron.NeutronClient, n *notifier.Notifier) *Controller {
	return &Controller{pool: pool, spec: spec, neutron: client, notifier: n}
}
