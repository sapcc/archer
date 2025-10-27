// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package ni

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
)

func createService(tx pgx.Tx, serviceID *strfmt.UUID) error {
	log.Infof("Creating service record %s in database", config.Global.Agent.ServiceName)
	sql, args, err := db.Insert("service").
		Columns("description",
			"network_id",
			"status",
			"visibility",
			"provider",
			"proxy_protocol",
			"name",
			"require_approval",
			"availability_zone",
			"project_id",
			"ports",
			"host",
			"ip_addresses",
			"protocol").
		Values("Created by Network Injection agent",
			"00000000-0000-0000-0000-000000000000",
			"AVAILABLE",
			"public",
			"cp",
			"false",
			config.Global.Agent.ServiceName,
			config.Global.Agent.ServiceRequireApproval,
			config.Global.Default.AvailabilityZone,
			config.Global.ServiceAuth.ProjectID,
			[]int{config.Global.Agent.ServicePort},
			config.Global.Default.Host,
			[]string{},
			config.Global.Agent.ServiceProtocol,
		).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return err
	}

	return pgxscan.Get(context.Background(), tx, serviceID, sql, args...)
}

func (a *Agent) discoverService() error {
	sql, args, err := db.Select("id").
		From("service").
		Where("host = ?", config.Global.Default.Host).
		Where("availability_zone = ?", config.Global.Default.AvailabilityZone).
		Where("name = ?", config.Global.Agent.ServiceName).
		Where("ports[1] = ?", config.Global.Agent.ServicePort).
		Where("provider = 'cp'").
		ToSql()
	if err != nil {
		return err
	}

	if err = pgx.BeginFunc(context.Background(), a.pool, func(tx pgx.Tx) error {
		err = pgxscan.Get(context.Background(), tx, &a.serviceID, sql, args...)
		if err != nil {
			if pgxscan.NotFound(err) && config.Global.Agent.CreateService {
				if err = createService(tx, &a.serviceID); err != nil {
					return err
				}
			} else {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	log.Infof("Agent associated to service %s", a.serviceID)
	return nil
}
