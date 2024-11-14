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
	"net"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
	internal "github.com/sapcc/archer/internal/errors"
	"github.com/sapcc/archer/models"
)

func (a *Agent) getExtendedService(s *models.Service) (*as3.ExtendedService, error) {
	// Fetch SNAT ports from neutron
	service := &as3.ExtendedService{Service: *s}
	deviceIDs := a.getDeviceIDs()
	pendingDelete := service.Status == models.ServiceStatusPENDINGDELETE

	network, err := a.neutron.GetNetwork(service.NetworkID.String())
	if err != nil {
		if pendingDelete {
			// We tolerate deleted networks in case the service is going to be deleted
			return service, nil
		}
		return nil, fmt.Errorf("GetNetwork: %w", err)
	}

	if len(network.Subnets) == 0 {
		return nil, fmt.Errorf("GetNetwork: %w", internal.ErrNoSubnetFound)
	}

	// Try to allocate SNAT ports from the subnet hat has the service in it
	err = nil
	for i, subnetID := range network.Subnets {
		var subnet *subnets.Subnet
		if subnet, err = a.neutron.GetSubnet(subnetID); err != nil {
			return nil, fmt.Errorf("GetSubnet: %w", err)
		}

		var ipnet *net.IPNet
		if _, ipnet, err = net.ParseCIDR(subnet.CIDR); err != nil {
			return nil, fmt.Errorf("ParseCIDR: %w", err)
		}

		found := false
		for _, ip := range service.IPAddresses {
			var ipAddress net.IP
			if ipAddress, _, err = net.ParseCIDR(ip.String()); err != nil {
				return nil, fmt.Errorf("ParseCIDR: %w", err)
			}

			if ipnet.Contains(ipAddress) {
				found = true
				break
			}
		}
		if !found && i < len(network.Subnets)-1 {
			// Skip this subnet and try the next if there are more subnets
			// for backwards compatibility - else just allow invalid configuration.
			continue
		}

		// Allocate SNAT ports as SelfIPs in Neutron
		service.NeutronPorts, err = a.neutron.EnsureNeutronSelfIPs(deviceIDs, subnetID, pendingDelete)
		if err == nil {
			service.SubnetID = subnetID
			break
		}

		var gerr gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &gerr) && gerr.Actual == 409 {
			if bytes.Contains(gerr.Body, []byte("OverQuota")) {
				return nil, fmt.Errorf("EnsureNeutronSelfIPs: %w, %w", err, internal.ErrQuotaExceeded)
			} else if bytes.Contains(gerr.Body, []byte("No more IP addresses available")) {
				continue // try next subnet
			}
		}

		// unexpected error from neutron
		return nil, fmt.Errorf("EnsureNeutronSelfIPs: %w", err)
	}
	if err != nil || service.SubnetID == "" {
		// All subnet IPs are exhausted
		return nil, fmt.Errorf("EnsureNeutronSelfIPs: %w, %w", err, internal.ErrNoIPsAvailable)
	}

	// Fetch segmentID for the network
	if len(service.NeutronPorts) > 0 {
		// we only expect a valid segment if we have at least one Service port bound
		if service.SegmentId, err = a.neutron.GetNetworkSegment(service.NetworkID.String()); err != nil {
			return nil, fmt.Errorf("GetNetworkSegment: %w", err)
		}
	}

	return service, nil
}

func (a *Agent) ProcessServices(ctx context.Context) error {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var dbServices []*models.Service
	// We need to fetch all services of this host since the AS3 tenant is shared
	sql, args := db.Select("*").
		From("service").
		Where("host = ?", config.Global.Default.Host).
		Where("provider = ?", models.ServiceProviderTenant).
		Suffix("FOR UPDATE OF service").
		MustSql()
	if err = pgxscan.Select(ctx, tx, &dbServices, sql, args...); err != nil {
		return err
	}

	/* ==================================================
	   Populate ExtendedService instance
	   ================================================== */
	var services []*as3.ExtendedService
	for _, service := range dbServices {
		if extendedService, err := a.getExtendedService(service); err != nil {
			l := log.WithFields(log.Fields{"service": service.ID, "network": service.NetworkID})
			if errors.Is(err, internal.ErrQuotaExceeded) {
				service.Status = models.ServiceStatusERRORQUOTA
				if _, err = tx.Exec(ctx,
					`UPDATE service SET status = 'ERROR_QUOTA', updated_at = NOW() WHERE id = $1;`,
					service.ID); err != nil {
					return err
				}
			} else if errors.Is(err, internal.ErrNoIPsAvailable) {
				// No more IPs available in the subnet
				l.WithError(err).Warning("ProcessServices: no more IPs available in any subnet")
			} else if errors.Is(err, internal.ErrNoPhysNetFound) {
				l.WithError(err).Warning("ProcessServices: no segment/physnet found for network")
			} else if errors.Is(err, internal.ErrNoSubnetFound) {
				l.WithError(err).Warning("ProcessServices: no subnet found for network")
			} else {
				// Unexpected error, don't skip the service but abort update
				return err
			}
		} else {
			services = append(services, extendedService)
		}
	}

	/* ==================================================
	   L2 Configuration
	   ================================================== */
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
				if errors.Is(err, internal.ErrNoPhysNetFound) {
					// No segment found, skip L2 cleanup
					cleanupL2 = false
				} else if err != nil {
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
	_ = tx.Commit(ctx)
	return nil
}
