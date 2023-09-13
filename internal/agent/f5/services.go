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
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/models"
)

func (a *Agent) fetchOrAllocateSNATPorts(ctx context.Context, tx pgx.Tx, service *as3.ExtendedService) error {
	sql, args := db.Select("port_id").
		From("service_port").
		Where("service_id = ?", service.ID).
		MustSql()
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var portID strfmt.UUID
		err = rows.Scan(&portID)
		if err != nil {
			return err
		}

		port, err := a.neutron.GetPort(portID.String())
		if err != nil {
			// todo: handle missing port
			return err
		}
		hostname := strings.TrimPrefix(port.Name, "local-")
		service.SnatPorts[hostname] = port
	}

	if rows.Err() != nil {
		return err
	}

	if service.Status == "PENDING_DELETE" {
		// done if service is being deleted
		return nil
	}

	for _, bigip := range a.bigips {
		hostname := bigip.GetHostname()
		if _, ok := service.SnatPorts[hostname]; ok {
			// already allocated, nothing to do
			continue
		}

		// No SNAT ports allocated yet, allocate them
		var port *ports.Port
		if port, err = a.neutron.AllocateSNATNeutronPort(&service.Service, hostname); err != nil {
			return err
		}

		sql, args, err = db.Insert("service_port").
			Columns("service_id", "port_id").
			Values(service.ID, port.ID).
			ToSql()
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, sql, args...); err != nil {
			return err
		}

		service.SnatPorts[hostname] = port
	}

	return nil
}

func (a *Agent) ProcessServices(ctx context.Context) error {
	var services []*as3.ExtendedService
	if err := pgx.BeginFunc(context.Background(), a.pool, func(tx pgx.Tx) error {
		// We need to fetch all services of this host since the AS3 tenant is shared
		sql, args := db.Select("*").
			From("service").
			Where("host = ?", config.Global.Default.Host).
			Where("provider = 'tenant'").
			Suffix("FOR UPDATE OF service").
			MustSql()
		if err := pgxscan.Select(ctx, tx, &services, sql, args...); err != nil {
			return err
		}

		/* ==================================================
		   Populate ExtendedService instance
		   ================================================== */
		for _, service := range services {
			// Fetch SNAT ports from neutron
			service.SnatPorts = make(map[string]*ports.Port, len(a.bigips))
			err := a.fetchOrAllocateSNATPorts(ctx, tx, service)
			if err != nil {
				var gerr gophercloud.ErrUnexpectedResponseCode
				if errors.As(err, &gerr) && gerr.Actual == 409 && bytes.Contains(gerr.Body, []byte("OverQuota")) {
					log.WithField("service", service.ID).Info(gerr.Body)
					if _, err := tx.Exec(ctx, `UPDATE service SET status = 'ERROR_QUOTA', updated_at = NOW() WHERE id = $1;`,
						service.ID); err != nil {
						return err
					}
					return nil
				}
				return err
			}

			if len(service.SnatPorts) > 0 {
				// we only expect a valid segment if we have at least one SNAT port bound
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

				// Ensure SNAT neutron ports and segment ids on VCMP guests
				for _, bigip := range a.bigips {
					bigip := bigip
					g.Go(func() error {
						return service.EnsureSNATPort(bigip, a.neutron)
					})
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
				service := service

				for _, bigip := range a.bigips {
					bigip := bigip
					g.Go(func() error {
						port, ok := service.SnatPorts[bigip.GetHostname()]
						if ok {
							if err := bigip.CleanupSelfIP(port); err != nil {
								log.
									WithFields(log.Fields{"service": service.ID, "port": port.ID}).
									Error(err)
							}
						} else {
							log.
								WithFields(log.Fields{"service": service.ID, "host": bigip.GetHostname()}).
								Info("CleanupSelfIP: No SelfIP registered for this host")
						}
						return nil
					})
				}

				if err = g.Wait(); err != nil {
					return err
				}

				// Ensure L2 configuration is no longer needed
				var skipCleanup bool
				for _, s := range services {
					// check if other service uses the same segment
					if s.ID != service.ID && s.Status != "PENDING_DELETE" && s.SegmentId == service.SegmentId {
						skipCleanup = true
					}
				}
				// Check if there are still endpoints using the same segment
				ct, err := tx.Exec(ctx, "SELECT 1 FROM endpoint_port WHERE network = $1", service.NetworkID)
				if err != nil {
					return err
				}
				if ct.RowsAffected() > 0 {
					skipCleanup = true
				}

				if service.SegmentId == 0 {
					skipCleanup = true
				}

				if !skipCleanup {
					if err := a.CleanupL2(ctx, service.SegmentId); err != nil {
						log.
							WithFields(log.Fields{"service": service.ID, "vlan": service.SegmentId}).
							WithError(err).
							Error("CleanupL2")
					}
				} else {
					log.
						WithFields(log.Fields{"service": service.ID, "vlan": service.SegmentId}).
						Info("Skipping CleanupL2")
				}
			}
		}

		// Successfully updated the tenant
		for _, service := range services {
			if service.Status == models.ServiceStatusPENDINGDELETE {
				for _, snatPort := range service.SnatPorts {
					if err = a.neutron.DeletePort(snatPort.ID); err != nil {
						var errDefault404 gophercloud.ErrDefault404
						if !errors.As(err, &errDefault404) {
							return err
						}
					}
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
