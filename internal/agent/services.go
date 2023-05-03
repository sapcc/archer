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

package agent

import (
	"context"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/agent/as3"
	"github.com/sapcc/archer/internal/agent/neutron"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/models"
)

func processSNATPort(a *Agent, ctx context.Context, tx pgx.Tx, service *as3.ExtendedService) error {
	var err error

	if service.Status == "PENDING_DELETE" {
		// Handle service in pending deletion, cleanup SNAT port
		if service.SnatPortId != nil {
			if err = neutron.DeleteSNATPort(a.neutron, service.SnatPortId); err != nil {
				logg.Error("Failed deleting SNAT Port: %s", err.Error())
			}
			if _, err = tx.Exec(ctx, `DELETE FROM service_snat_port WHERE port_id = $1`,
				service.SnatPort.ID); err != nil {
				return err
			}
		} else {
			logg.Other("WARNING",
				"Service pending for deletion '%s' has no SNAT port allocated", service.ID)
		}

		return nil
	}

	// Ensure SNAT port
	if service.SnatPortId != nil {
		service.SnatPort, err = neutron.GetSNATPort(a.neutron, service.SnatPortId)
		if err != nil {
			return err
		}
	} else {
		service.SnatPort, err = neutron.AllocateSNATPort(a.neutron, service)
		if err != nil {
			return err
		}
		// set allocated flag, for deletion during rollback
		service.TXAllocated = true

		if _, err = tx.Exec(ctx, `INSERT INTO service_snat_port(service_id, port_id) VALUES ($1, $2)`,
			service.ID, service.SnatPort.ID); err != nil {
			return err
		}
	}

	// Fetch segment ID from neutron
	service.SegmentId, err = neutron.GetNetworkSegment(a.cache, a.neutron, service.NetworkID.String())
	if err != nil {
		return err
	}
	if err := as3.EnsureVLAN(a.bigip, service.SegmentId); err != nil {
		return err
	}
	if err := as3.EnsureRouteDomain(a.bigip, service.SegmentId); err != nil {
		return err
	}

	return nil
}

func doProcessing(a *Agent, ctx context.Context, tx pgx.Tx, services []*as3.ExtendedService) error {
	// Ensure SNAT neutron ports and segment ids
	for _, service := range services {
		if err := processSNATPort(a, ctx, tx, service); err != nil {
			return err
		}
	}

	data := as3.GetAS3Declaration(map[string]as3.Tenant{
		"Common": as3.GetServiceTenants(services),
	})

	if err := as3.PostBigIP(a.bigip, &data, "Common"); err != nil {
		return err
	}

	// Successfully updated the tenant
	for _, service := range services {
		if service.Status == models.ServiceStatusPENDINGDELETE {
			if _, err := tx.Exec(ctx, `DELETE FROM service WHERE id = $1 AND status = 'PENDING_DELETE';`,
				service.ID); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(ctx, `UPDATE service SET status = 'AVAILABLE', updated_at = NOW() WHERE id = $1;`,
				service.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Agent) ProcessServices(ctx context.Context) error {
	var services []*as3.ExtendedService

	// We need to fetch all services of this host since the AS3 tenant is shared
	if err := pgxscan.Select(ctx, a.pool, &services,
		`SELECT id, status, enabled, network_id, proxy_protocol, port, ip_addresses, sap.port_id AS snat_port_id
              FROM service 
                  LEFT JOIN service_snat_port sap ON service.id = sap.service_id 
              WHERE host = $1`,
		config.Global.Default.Host,
	); err != nil {
		return err
	}

	if err := pgx.BeginFunc(context.Background(), a.pool, func(tx pgx.Tx) error {
		return doProcessing(a, context.Background(), tx, services)
	}); err != nil {
		// delete ports that have been created in this transaction
		for _, service := range services {
			if service.TXAllocated {
				logg.Error("Orphaned neutron SNAT port due rollback, deleting '%s'", service.SnatPortId)
				if err := neutron.DeleteSNATPort(a.neutron, service.SnatPortId); err != nil {
					logg.Error(err.Error())
				}
			}
		}

		return err
	}

	return nil
}
