// Copyright 2023 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
			"port",
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
			config.Global.Agent.ServicePort,
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
		Where("port = ?", config.Global.Agent.ServicePort).
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
