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
	"errors"
	"fmt"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
	aErrors "github.com/sapcc/archer/internal/errors"
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
	pages, err := ports.List(a.neutron.ServiceClient, opts).AllPages(context.Background())
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
				n := port
				endpoint.Port = &n
			}
		}
	}

	return nil
}

// refreshSegments ensures that the segment_id is set for the given endpoint
func refreshSegments(ctx context.Context, pool db.PgxIface, endpoints []*as3.ExtendedEndpoint, n *neutron.NeutronClient) {
	for _, endpoint := range endpoints {
		logger := log.WithFields(log.Fields{"port_id": endpoint.Target.Port, "endpoint": endpoint.ID})
		if endpoint.SegmentId == nil {
			// Sync endpoint segment to database - we want this because in case of port has been deleted meanwhile,
			// we loose the segment-id and therefor the ability to delete the l2 configuration
			var err error
			var segmentId int
			segmentId, err = n.GetNetworkSegment(endpoint.Target.Network.String())
			if err != nil {
				logger.WithError(err).Warning("ProcessEndpoint: Could not find valid segment")
				continue
			}
			endpoint.SegmentId = &segmentId

			log.Infof("ProcessEndpoint: Updating segment_id to %d", segmentId)
			sql, args := db.Update("endpoint_port").
				Set("segment_id", segmentId).
				Where("endpoint_id = ?", endpoint.ID).
				MustSql()
			if _, err = pool.Exec(ctx, sql, args...); err != nil {
				logger.WithError(err).Warning("ProcessEndpoint: Could not update segment_id")
			}
		}
	}
}

func checkAllPendingDelete(endpoints []*as3.ExtendedEndpoint, subnetID string) bool {
	for _, endpoint := range endpoints {
		if endpoint.Target.Subnet.String() == subnetID &&
			(endpoint.Status != models.EndpointStatusPENDINGDELETE && endpoint.Status != models.EndpointStatusPENDINGREJECTED) {
			// if any endpoint with same subnet is not in PENDING_DELETE/PENDING_REJECTED, skip cleanup
			return false
		}
	}

	return true
}

func (a *Agent) ProcessEndpoint(ctx context.Context, endpointID strfmt.UUID) error {
	var endpoints []*as3.ExtendedEndpoint
	var networkID strfmt.UUID
	var subnetID string
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

	sql, args := db.Select("network", "subnet").
		From("endpoint_port").
		Where("endpoint_id = ?", endpointID).
		MustSql()
	if err := tx.QueryRow(ctx, sql, args...).Scan(&networkID, &subnetID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.WithField("id", endpointID).Warning("Endpoint not found")
			return nil
		}
		return err
	}

	// Sync endpoint segment cache, is a no-op if already cached
	sql, args = db.Select("endpoint.*",
		"service.port AS service_port_nr",
		"service.proxy_protocol",
		"service.network_id AS service_network_id",
		"endpoint_port.segment_id",
		`endpoint_port.port_id AS "target.port"`,
		`endpoint_port.network AS "target.network"`,
		`endpoint_port.subnet AS "target.subnet"`,
		`endpoint_port.owned`).
		From("endpoint").
		InnerJoin("service ON endpoint.service_id = service.id").
		Join("endpoint_port ON endpoint_id = endpoint.id").
		Where(sq.NotEq{"endpoint.status": []models.EndpointStatus{
			models.EndpointStatusPENDINGAPPROVAL, // ignore pending approval
			models.EndpointStatusREJECTED}}).     // ignore rejected, they are considered deleted already
		Where("network = ?", networkID).
		Where("service.host = ?", config.Global.Default.Host).
		Where("service.provider = ?", models.ServiceProviderTenant).
		Suffix("FOR UPDATE OF endpoint").
		MustSql()
	if err := pgxscan.Select(ctx, tx, &endpoints, sql, args...); err != nil {
		return err
	}

	if len(endpoints) == 0 {
		log.WithField("id", endpointID).Warning("No endpoints than need update, skipping")
		return nil
	}

	refreshSegments(ctx, a.pool, endpoints, a.neutron)

	var cleanupL2 bool
	if checkAllPendingDelete(endpoints, subnetID) {
		// Consider cleaning up L2 configuration if all endpoints of a subnet are deleted
		var err error
		if err, cleanupL2 = checkCleanupL2(ctx, tx, networkID.String(),
			true, false); err != nil {
			return err
		}
	}
	err, cleanupSelfIPs := a.checkCleanupSelfIPs(ctx, tx, networkID.String(), subnetID,
		true, false)
	if err != nil {
		return err
	}

	if !cleanupL2 && endpoints[0].SegmentId == nil {
		return fmt.Errorf("could not find or fetch valid segment for endpoint %s, skipping", endpointID)
	}

	var serviceSegmentID int
	var serviceMTU int
	g, _ := errgroup.WithContext(ctx)

	if !cleanupL2 {
		g.Go(func() (err error) {
			serviceSegmentID, err = a.neutron.GetNetworkSegment(endpoints[0].ServiceNetworkId.String())
			return
		})
		g.Go(func() error {
			serviceMTU, err = a.neutron.GetNetworkMTU(endpoints[0].ServiceNetworkId.String())
			return err
		})
	}
	g.Go(func() error {
		err := a.populateEndpointPorts(endpoints)
		if err != nil && cleanupL2 {
			// ignore missing ports if all endpoints are about to be deleted, print warning instead
			log.WithError(err).
				Warning("Ignoring missing ports for endpoint(s) since endpoints are about to be deleted.")
			return nil
		}
		return err
	})

	// Wait for populating endpoints struct
	if err := g.Wait(); err != nil {
		return err
	}

	/* ==================================================
	   Layer 2 VCMP + Guest configuration
	   ================================================== */
	if !cleanupL2 {
		// VCMP configuration
		if err := a.EnsureL2(ctx, *endpoints[0].SegmentId, &serviceSegmentID, serviceMTU); err != nil {
			return err
		}
	}

	// (Re-)Sync SelfIPs of all endpoints in the same segment
	for _, ep := range endpoints {
		ep := ep
		if ep.Status != models.EndpointStatusPENDINGDELETE && ep.Status != models.EndpointStatusPENDINGREJECTED {
			// SelfIPs
			if len(ep.Port.FixedIPs) == 0 {
				return fmt.Errorf("EnsureSelfIPs: no fixedIPs found for EP port %s", ep.Port.ID)
			}
			if err := a.EnsureSelfIPs(subnetID, false); err != nil {
				return err
			}
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
	// 2. Delete lower L2 configuration if all endpoints of a segments are deleted
	logWith := log.WithField("endpoint", endpointID)

	// Get segmentID for subnet before we delete SelfIPs, since they could be the last ports holding the segment
	var segmentID int
	if cleanupL2 {
		segmentID, err = a.neutron.GetSubnetSegment(subnetID)
		if err != nil {
			if !errors.Is(err, aErrors.ErrNoPhysNetFound) {
				return err
			}
			logWith.WithError(err).Warning("ProcessEndpoint: GetSubnetSegment failed with 404, skipping L2 cleanup")
		}
	}

	if cleanupSelfIPs {
		logWith.WithField("subnet", subnetID).Info("ProcessEndpoint: deleting SelfIPs")
		if err := a.CleanupSelfIPs(subnetID); err != nil {
			return err
		}
	}

	if cleanupL2 && segmentID > 0 {
		logWith.WithField("network", networkID).Info("ProcessEndpoint: deleting L2")
		if err := a.CleanupL2(ctx, segmentID); err != nil {
			logWith.WithError(err).Error("ProcessEndpoint: CleanupL2")
		}
	}

	// 3. Finalize endpoint deletion and related ports
	for _, endpoint := range endpoints {
		switch endpoint.Status {
		case models.EndpointStatusPENDINGREJECTED:
			// Delete endpoint neutron port, if it exists and is owned by the agent
			if endpoint.Target.Port != nil && endpoint.Owned {
				if err = a.neutron.DeletePort(endpoint.Target.Port.String()); err != nil {
					if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
						return err
					}
				}
			}

			log.Debugf("ProcessEndpoint: Rejecting endpoint %s", endpoint.ID)
			sql, args = db.
				Update("endpoint").
				Set("status", models.EndpointStatusREJECTED).
				Set("updated_at", sq.Expr("NOW()")).
				MustSql()
		case models.EndpointStatusPENDINGDELETE:
			// Delete endpoint neutron port, if it exists and is owned by the agent
			if endpoint.Target.Port != nil && endpoint.Owned {
				if err = a.neutron.DeletePort(endpoint.Target.Port.String()); err != nil {
					if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
						return err
					}
				}
			}

			log.Debugf("ProcessEndpoint: Deleting endpoint %s", endpoint.ID)
			sql, args = db.
				Delete("endpoint").
				Where("id = ?", endpoint.ID).
				MustSql()
		default:
			sql, args = db.
				Update("endpoint").
				Set("status", models.EndpointStatusAVAILABLE).
				Set("updated_at", sq.Expr("NOW()")).
				Where("id = ?", endpoint.ID).
				MustSql()
		}
		if _, err = tx.Exec(ctx, sql, args...); err != nil {
			return err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	log.Info("ProcessEndpoint successful")
	return nil
}
