// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"time"

	"github.com/go-openapi/loads"

	"github.com/sapcc/archer/v2/internal/db"
	"github.com/sapcc/archer/v2/internal/neutron"
	"github.com/sapcc/archer/v2/internal/notifier"
)

// defaultLockTimeout bounds how long a FOR UPDATE handler transaction waits for
// a row lock before failing with a lock_not_available error. It must stay well
// below the HTTP request deadline so a contended lock surfaces as a clean,
// retryable 503 instead of hanging until the request is cancelled.
const defaultLockTimeout = 5 * time.Second

type Controller struct {
	spec     *loads.Document
	pool     db.PgxIface
	neutron  *neutron.NeutronClient
	notifier *notifier.Notifier
	// lockTimeout is applied to FOR UPDATE handler transactions. Overridable in
	// tests; defaults to defaultLockTimeout.
	lockTimeout time.Duration
}

func NewController(pool db.PgxIface, spec *loads.Document, client *neutron.NeutronClient, n *notifier.Notifier) *Controller {
	return &Controller{pool: pool, spec: spec, neutron: client, notifier: n, lockTimeout: defaultLockTimeout}
}
