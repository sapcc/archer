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
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gophercloud/gophercloud"
	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/models"
)

func (a *Agent) ProcessServices(ctx context.Context) error {
	var services []*as3.ExtendedService
	if err := pgx.BeginFunc(context.Background(), a.pool, func(tx pgx.Tx) error {
		// We need to fetch all services of this host since the AS3 tenant is shared
		sql, args := db.Select("*").
			From("service").
			Where("host = ?", config.Global.Default.Host).
			Where("provider = ?", models.ServiceProviderTenant).
			Suffix("FOR UPDATE OF service").
			MustSql()
		if err := pgxscan.Select(ctx, tx, &services, sql, args...); err != nil {
			return err
		}

		/* ==================================================
		   Populate ExtendedService instance
		   ================================================== */
		for _, service := range services {
			var err error
			// Fetch SNAT ports from neutron
			deviceIDs := a.getDeviceIDs()
			// TODO: tolerate deleted network in case the service is going to be deleted
			network, err := a.neutron.GetNetwork(service.NetworkID.String())
			if err != nil {
				var errDefault404 gophercloud.ErrDefault404
				if !errors.As(err, &errDefault404) || service.Status != models.ServiceStatusPENDINGDELETE {
					return err
				}
				log.WithError(err).WithField("service", service.ID).Warning("ProcessServices(PENDING_DELETE): network not found")
				continue
			}
			if len(network.Subnets) == 0 {
				return fmt.Errorf("service %s: no subnets found for network %s", service.ID, service.NetworkID)
			}
			service.SubnetID = network.Subnets[0]

			// Allocate SNAT ports as SelfIPs in Neutron
			service.NeutronPorts, err = a.neutron.EnsureNeutronSelfIPs(deviceIDs, service.SubnetID, false)
			if err != nil {
				var gerr gophercloud.ErrUnexpectedResponseCode
				if errors.As(err, &gerr) && gerr.Actual == 409 && bytes.Contains(gerr.Body, []byte("OverQuota")) {
					log.WithField("service", service.ID).Info(gerr.Body)
					service.Status = models.ServiceStatusERRORQUOTA
					if _, err := tx.Exec(ctx,
						`UPDATE service SET status = 'ERROR_QUOTA', updated_at = NOW() WHERE id = $1;`,
						service.ID); err != nil {
						return err
					}
					continue
				}
				// return generic error
				return err
			}

			if len(service.NeutronPorts) > 0 {
				// we only expect a valid segment if we have at least one Service port bound
				var err error
				service.SegmentId, err = a.neutron.GetNetworkSegment(service.NetworkID.String())
				if err != nil {
					return err
				}
			}
		}

		/* ==================================================
		   L2 Configuration
		   ================================================== */
		g, _ := errgroup.WithContext(ctx)
		for _, service := range services {
			service := service
			if service.Status != "PENDING_DELETE" {
				if err := a.EnsureL2(ctx, service.SegmentId, nil); err != nil {
					return err
				}
				if err := a.EnsureSelfIPs(service.SubnetID, true); err != nil {
					return err
				}
			}
		}
		var err error
		if err = g.Wait(); err != nil {
			return err
		}

		/* ==================================================
		   Post AS3 Declaration to active BigIP
		   ================================================== */
		data := as3.GetAS3Declaration(map[string]as3.Tenant{
			"Common": as3.GetServiceTenants(services),
		})

		if err = a.bigip.PostBigIP(&data, "Common"); err != nil {
			return err
		}

		/* ==================================================
		   L2 Configuration Cleanup
		   ================================================== */
		for _, service := range services {
			if service.Status == "PENDING_DELETE" {
				logWith := log.WithField("service", service.ID)
				service := service

				if service.SubnetID == "" {
					logWith.Warning("ProcessServices: no subnet found for service, skipping cleanup but continuing with deletion")
					continue
				}

				err, cleanupL2 := checkCleanupL2(ctx, tx, service.NetworkID.String(),
					false, true)
				if err != nil {
					return err
				}
				err, cleanupSelfIPs := a.checkCleanupSelfIPs(ctx, tx, service.NetworkID.String(),
					service.SubnetID, false, true)
				if err != nil {
					return err
				}

				// Get segmentID for subnet before we delete SelfIPs, since they could be the last ports holding the
				// segment
				var segmentID int
				if cleanupL2 {
					segmentID, err = a.neutron.GetSubnetSegment(service.SubnetID)
					if err != nil {
						return err
					}
				}

				if cleanupSelfIPs {
					logWith.WithField("subnet", service.SubnetID).Info("ProcessServices: deleting SelfIPs")
					if err := a.CleanupSelfIPs(service.SubnetID); err != nil {
						return err
					}

					// TODO: Remove after a while, we switched to normal f5selfips instead of f5snat
					if err := a.CleanupSNATPorts(service.NetworkID.String()); err != nil {
						return err
					}
				}

				if cleanupL2 {
					logWith.WithField("network", service.NetworkID).Info("ProcessServices: deleting L2.")
					if err := a.CleanupL2(ctx, segmentID); err != nil {
						log.
							WithFields(log.Fields{"service": service.ID, "vlan": service.SegmentId}).
							WithError(err).Error("CleanupL2")
					}
				}
			}
		}

		// Successfully updated the tenant
		for _, service := range services {
			if service.Status == models.ServiceStatusPENDINGDELETE {
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
