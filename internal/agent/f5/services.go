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

package f5

import (
	"context"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/agent/neutron"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/models"
	"golang.org/x/sync/errgroup"
)

func processVCMP(a *Agent, service *as3.ExtendedService) error {
	for _, vcmp := range a.vcmps {
		if err := as3.EnsureVLAN(vcmp, service.SegmentId, a.iface); err != nil {
			return err
		}

		if err := as3.EnsureGuestVlan(vcmp, service.SegmentId); err != nil {
			return err
		}
	}
	return nil
}

func processSNATPort(a *Agent, service *as3.ExtendedService) error {
	var err error

	if err = as3.EnsureVLAN(a.bigip, service.SegmentId, ""); err != nil {
		return err
	}
	if err = as3.EnsureRouteDomain(a.bigip, service.SegmentId); err != nil {
		return err
	}

	return nil
}

func (a *Agent) ProcessServices(ctx context.Context) error {
	var services []*as3.ExtendedService

	// We need to fetch all services of this host since the AS3 tenant is shared
	sql, args := db.Select("id", "status", "enabled", "network_id", "proxy_protocol", "port", "ip_addresses",
		"sap.port_id AS snat_port_id").
		From("service").
		LeftJoin("service_snat_port sap ON service.id = sap.service_id").
		Where("host = ?", config.Global.Default.Host).
		Where("provider = 'tenant'").
		Suffix("FOR UPDATE OF service").
		MustSql()
	if err := pgx.BeginFunc(context.Background(), a.pool, func(tx pgx.Tx) error {
		if err := pgxscan.Select(ctx, tx, &services, sql, args...); err != nil {
			return err
		}

		g, _ := errgroup.WithContext(ctx)
		var err error

		for _, service := range services {
			// Fetch SNAT port
			g.Go(func() error {
				service.SnatPort, err = neutron.GetSNATPort(a.neutron, service.SnatPortId)
				return err
			})

			// Fetch segment ID from neutron
			g.Go(func() error {
				service.SegmentId, err = neutron.GetNetworkSegment(a.cache, a.neutron, service.NetworkID.String())
				return err
			})

			if err = g.Wait(); err != nil {
				return err
			}

			service := service
			// Ensure VCMP segment port configuration, parallelize
			g.Go(func() error {
				return processVCMP(a, service)
			})

			// Ensure SNAT neutron ports and segment ids on VCMP guests
			g.Go(func() error {
				return processSNATPort(a, service)
			})

			if err = g.Wait(); err != nil {
				return err
			}
		}

		// Post final declaration
		data := as3.GetAS3Declaration(map[string]as3.Tenant{
			"Common": as3.GetServiceTenants(services),
		})

		if err := as3.PostBigIP(a.bigip, &data, "Common"); err != nil {
			return err
		}

		// Successfully updated the tenant
		for _, service := range services {
			if service.Status == models.ServiceStatusPENDINGDELETE {
				if err = neutron.DeletePort(a.neutron, service.SnatPortId); err != nil {
					return err
				}

				if _, err = tx.Exec(ctx, `DELETE FROM service WHERE id = $1 AND status = 'PENDING_DELETE';`,
					service.ID); err != nil {
					return err
				}
			} else {
				if _, err = tx.Exec(ctx, `UPDATE service SET status = 'AVAILABLE', updated_at = NOW() WHERE id = $1;`,
					service.ID); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
