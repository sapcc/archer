// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"
)

func (c *Controller) notifyService(host string) {
	if _, err := c.pool.Exec(context.Background(), "SELECT pg_notify('service', $1)", host); err != nil {
		log.Error(err.Error())
	}
}

func (c *Controller) notifyEndpoint(host string, id strfmt.UUID) {
	payload := fmt.Sprintf("%s:%s", host, id)
	if _, err := c.pool.Exec(context.Background(), "SELECT pg_notify('endpoint', $1)", payload); err != nil {
		log.Error(err.Error())
	}
}
