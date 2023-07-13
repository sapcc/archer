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
	"fmt"
	"github.com/sapcc/archer/internal/config"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/sapcc/go-bits/logg"
	"golang.org/x/sync/errgroup"

	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/internal/neutron"
	"github.com/sapcc/archer/models"
)

func (a *Agent) populateEndpointPorts(endpoints []*as3.ExtendedEndpoint) error {
	// Fetch ports from neutron
	var opts neutron.PortListOpts
	for _, endpoint := range endpoints {
		opts.IDs = append(opts.IDs, endpoint.Target.Port.String())
	}

	var pages pagination.Page
	pages, err := ports.List(a.neutron.ServiceClient, opts).AllPages()
	if err != nil {
		return err
	}
	endpointPorts, err := ports.ExtractPorts(pages)
	if err != nil {
		return err
	}

	if len(endpointPorts) == 0 {
		return fmt.Errorf("no neutron ports found for endpoint(s) %s", opts.IDs)
	}
	for _, port := range endpointPorts {
		for _, endpoint := range endpoints {
			if endpoint.Target.Port.String() == port.ID {
				endpoint.Port = &port
			}
		}
	}

	return nil
}

func (a *Agent) ProcessEndpoint(ctx context.Context, endpointID strfmt.UUID) error {
	var endpoints []*as3.ExtendedEndpoint
	var networkID strfmt.UUID
	var tx pgx.Tx

	{
		var err error
		if tx, err = a.pool.Begin(ctx); err != nil {
			return err
		}
	}

	defer func(tx pgx.Tx, ctx context.Context) {
		// Rollback is safe to call even if the tx is already closed, so if
		// the tx commits successfully, this is a no-op
		_ = tx.Rollback(ctx)
	}(tx, ctx)

	sql, args := db.Select("network").
		From("endpoint_port").
		Where("endpoint_id = ?", endpointID).
		MustSql()
	if err := tx.QueryRow(ctx, sql, args...).Scan(&networkID); err != nil {
		return err
	}

	sql, args = db.Select("endpoint.*",
		"service.port AS service_port_nr",
		"service.proxy_protocol",
		"service.network_id AS service_network_id",
		`endpoint_port.port_id AS "target.port"`,
		`endpoint_port.network AS "target.network"`,
		`endpoint_port.subnet AS "target.subnet"`).
		From("endpoint").
		InnerJoin("service ON endpoint.service_id = service.id").
		Join("endpoint_port ON endpoint_id = endpoint.id").
		Where("network = ?", networkID).
		Where("service.host = ?", config.Global.Default.Host).
		Where("service.provider = 'tenant'").
		Suffix("FOR UPDATE of endpoint").
		MustSql()
	if err := pgxscan.Select(ctx, tx, &endpoints, sql, args...); err != nil {
		return err
	}

	deleteAll := true
	for _, endpoint := range endpoints {
		if endpoint.Status != models.EndpointStatusPENDINGDELETE {
			deleteAll = false
		}
	}

	var endpointSegmentID, serviceSegmentID int
	g, _ := errgroup.WithContext(ctx)

	g.Go(func() (err error) {
		// Fetch segment ID from neutron
		endpointSegmentID, err = a.neutron.GetNetworkSegment(networkID.String())
		return
	})
	g.Go(func() (err error) {
		serviceSegmentID, err = a.neutron.GetNetworkSegment(endpoints[0].ServiceNetworkId.String())
		return
	})
	g.Go(func() error {
		if err := a.populateEndpointPorts(endpoints); err != nil && deleteAll {
			// ignore missing ports if all endpoints are about to be deleted, print error instead
			logg.Error(err.Error())
		}
		return nil
	})

	// Wait for populating endpoints struct
	if err := g.Wait(); err != nil {
		return err
	}
	for _, ep := range endpoints {
		ep.SegmentId = endpointSegmentID
	}

	/* ==================================================
	   Layer 2 VCMP + Guest configuration
	   ================================================== */
	if !deleteAll {
		// VCMP configuration
		if err := a.EnsureL2(ctx, endpointSegmentID, &serviceSegmentID); err != nil {
			return err
		}
	}

	/* ==================================================
	   Post AS3 Declaration to active BigIP
	   ================================================== */
	tenantName := as3.GetEndpointTenantName(networkID)
	data := as3.GetAS3Declaration(map[string]as3.Tenant{
		tenantName: as3.GetEndpointTenants(endpoints),
	})

	if err := a.bigip.PostBigIP(&data, tenantName); err != nil {
		return err
	}

	/* ==================================================
	   Layer 2 VCMP + Guest cleanup
	   ================================================== */
	if deleteAll {
		// Ensure L2 configuration is no longer needed
		var skipCleanup bool
		// check if other service uses the same segment
		ct, err := tx.Exec(ctx, "SELECT 1 FROM service WHERE network_id = $1 AND status != 'PENDING_DELETE'",
			networkID)
		if err != nil {
			return err
		}
		if ct.RowsAffected() > 0 {
			skipCleanup = true
		}

		// Check if there are still endpoints using the same segment
		ct, err = tx.Exec(ctx, "SELECT 1 FROM endpoint_port WHERE network = $1 AND endpoint_id != $2",
			networkID, endpointID)
		if err != nil {
			return err
		}
		if ct.RowsAffected() > 0 {
			skipCleanup = true
		}

		if !skipCleanup {
			if err := a.CleanupL2(ctx, endpointSegmentID); err != nil {
				logg.Error("CleanupL2(vlan=%d): %s", endpointSegmentID, err.Error())
			}
		} else {
			logg.Info("Skipping CleanupL2(vlan=%d) since it is still in use", endpointSegmentID)
		}
	}

	for _, endpoint := range endpoints {
		if endpoint.Status == models.EndpointStatusPENDINGDELETE {
			// TODO: check if archer owns the port
			if endpoint.Target.Port != nil {
				if err := a.neutron.DeletePort(endpoint.Target.Port.String()); err != nil {
					if _, ok := err.(gophercloud.ErrDefault404); !ok {
						return err
					} else {
						logg.Error("Port '%s' already deleted: %s", endpoint.Target.Port.String(), err.Error())
					}
				}
			}
			if _, err := tx.Exec(ctx, `DELETE FROM endpoint WHERE id = $1 AND status = 'PENDING_DELETE';`,
				endpoint.ID); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(ctx, `UPDATE endpoint SET status = 'AVAILABLE', updated_at = NOW() WHERE id = $1;`,
				endpoint.ID); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	logg.Info("ProcessEndpoint successful")
	return nil
}
