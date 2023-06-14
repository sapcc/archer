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
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/agent/neutron"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/models"
)

func (a *Agent) populateEndpointPorts(segmentId int, endpoints []*as3.ExtendedEndpoint) error {
	// Fetch ports from neutron
	var opts neutron.PortListOpts
	for _, endpoint := range endpoints {
		opts.IDs = append(opts.IDs, endpoint.Target.Port.String())
	}

	var pages pagination.Page
	pages, err := ports.List(a.neutron, opts).AllPages()
	if err != nil {
		return err
	}
	endpointPorts, err := ports.ExtractPorts(pages)
	if err != nil {
		return err
	}
	for _, port := range endpointPorts {
		for _, endpoint := range endpoints {
			if endpoint.Target.Port.String() == port.ID {
				endpoint.Port = &port
				endpoint.SegmentId = segmentId
			}
		}
	}

	return nil
}

func (a *Agent) ProcessEndpoint(ctx context.Context, endpointID strfmt.UUID) error {
	var endpoints []*as3.ExtendedEndpoint
	var networkID strfmt.UUID

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
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
	if err = tx.QueryRow(ctx, sql, args...).Scan(&networkID); err != nil {
		return err
	}

	sql, args = db.Select("endpoint.*",
		"service.port AS service_port_nr",
		"service.proxy_protocol",
		`endpoint_port.port_id AS "target.port"`,
		`endpoint_port.network AS "target.network"`,
		`endpoint_port.subnet AS "target.subnet"`).
		From("endpoint").
		InnerJoin("service ON service_id = service.id").
		Join("endpoint_port ON endpoint_id = endpoint.id").
		Where("network = ?", networkID).
		MustSql()
	if err = pgxscan.Select(ctx, tx, &endpoints, sql, args...); err != nil {
		return err
	}

	deleteAll := true
	for _, endpoint := range endpoints {
		if endpoint.Status != models.EndpointStatusPENDINGDELETE {
			deleteAll = false
		}
	}

	if !deleteAll {
		// Fetch segment ID from neutron
		segmentId, err := neutron.GetNetworkSegment(a.cache, a.neutron, networkID.String())
		if err != nil {
			return err
		}

		// Ensure VLAN and Route Domain
		if err := as3.EnsureVLAN(a.bigip, segmentId, ""); err != nil {
			return err
		}
		if err := as3.EnsureRouteDomain(a.bigip, segmentId); err != nil {
			return err
		}
		if err := a.populateEndpointPorts(segmentId, endpoints); err != nil {
			return err
		}
	}

	tenantName := as3.GetEndpointTenantName(networkID)
	data := as3.GetAS3Declaration(map[string]as3.Tenant{
		tenantName: as3.GetEndpointTenants(endpoints),
	})

	if err := as3.PostBigIP(a.bigip, &data, tenantName); err != nil {
		return err
	}

	for _, endpoint := range endpoints {
		if endpoint.Status == models.EndpointStatusPENDINGDELETE {
			// TODO: check if archer owns the port
			if err := neutron.DeletePort(a.neutron, endpoint.Target.Port); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `DELETE FROM endpoint_port WHERE endpoint_id = $1`,
				endpoint.ID); err != nil {
				return err
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
