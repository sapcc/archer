// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"fmt"

	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"
)

// NotifyService sends a NOTIFY to the service channel for the specified host.
func NotifyService(pool PgxIface, host string) {
	if _, err := pool.Exec(context.Background(), "SELECT pg_notify('service', $1)", host); err != nil {
		log.Error(err.Error())
	}
}

// NotifyEndpoint sends a NOTIFY to the endpoint channel for the specified host and endpoint ID.
func NotifyEndpoint(pool PgxIface, host string, id strfmt.UUID) {
	payload := fmt.Sprintf("%s:%s", host, id)
	if _, err := pool.Exec(context.Background(), "SELECT pg_notify('endpoint', $1)", payload); err != nil {
		log.Error(err.Error())
	}
}
