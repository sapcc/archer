// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package ni

import (
	"context"
	"strconv"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
)

func (a *Agent) createService(tx pgx.Tx) error {
	log.Infof("Creating service record %s in database", config.Global.Agent.ServiceName)
	var servicePorts = []int{config.Global.Agent.ServicePort}
	for _, port := range config.Global.Agent.ServicePorts {
		if i, err := strconv.Atoi(port); err == nil {
			servicePorts = append(servicePorts, i)
		}
	}
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
			servicePorts,
			config.Global.Default.Host,
			[]string{},
			config.Global.Agent.ServiceProtocol,
		).
		Suffix("RETURNING *").
		ToSql()
	if err != nil {
		return err
	}

	return pgxscan.Get(context.Background(), tx, &a.service, sql, args...)
}

func (a *Agent) discoverService() error {
	sql, args, err := db.
		Select("*").
		From("service").
		Where("host = ?", config.Global.Default.Host).
		Where("availability_zone = ?", config.Global.Default.AvailabilityZone).
		Where("name = ?", config.Global.Agent.ServiceName).
		Where("provider = 'cp'").
		ToSql()
	if err != nil {
		return err
	}

	if err = pgx.BeginFunc(context.Background(), a.pool, func(tx pgx.Tx) error {
		err = pgxscan.Get(context.Background(), tx, &a.service, sql, args...)
		if err != nil {
			if pgxscan.NotFound(err) && config.Global.Agent.CreateService {
				if err = a.createService(tx); err != nil {
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

	log.Infof("Agent associated to service %s (%s)", a.service.Name, a.service.ID)
	return nil
}
