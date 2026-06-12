// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/agent/f5/as3"
	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/internal/db"
	internal "github.com/sapcc/archer/v2/internal/errors"
	"github.com/sapcc/archer/v2/models"
)

func (a *Agent) getExtendedService(ctx context.Context, s *models.Service) (*as3.ExtendedService, error) {
	// Fetch SNAT ports from neutron
	service := &as3.ExtendedService{Service: *s}
	deviceIDs := a.getDeviceIDs()
	pendingDelete := service.Status == models.ServiceStatusPENDINGDELETE

	network, err := a.neutron.GetNetwork(ctx, service.NetworkID.String())
	if err != nil {
		if pendingDelete {
			// We tolerate deleted networks in case the service is going to be deleted
			return service, nil
		}
		return nil, fmt.Errorf("GetNetwork: %w", err)
	}

	service.MTU = network.MTU

	if len(network.Subnets) == 0 {
		return nil, fmt.Errorf("GetNetwork: %w", internal.ErrNoSubnetFound)
	}

	// allocate picks between service-scoped SNAT ports (snat_pool_size set) and per-device
	// SelfIP ports. The two paths share identical 409/quota handling below, so the only thing
	// that varies per subnet is which Neutron helper to call and the error label used in wraps.
	allocate := func(subnetID string) (map[string]*ports.Port, string, error) {
		if service.SnatPoolSize != nil {
			ports, err := a.neutron.EnsureServiceSnatPorts(ctx,
				service.ID, subnetID, int(*service.SnatPoolSize), pendingDelete)
			return ports, "EnsureServiceSnatPorts", err
		}
		ports, err := a.neutron.EnsureNeutronSelfIPs(ctx, deviceIDs, subnetID, pendingDelete)
		return ports, "EnsureNeutronSelfIPs", err
	}

	// Try to allocate SNAT ports from the subnet that has the service in it
	err = nil
	var label string
	for i, subnetID := range network.Subnets {
		var subnet *subnets.Subnet
		if subnet, err = a.neutron.GetSubnet(ctx, subnetID); err != nil {
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

		service.NeutronPorts, label, err = allocate(subnetID)
		if err == nil {
			service.SubnetID = subnetID
			break
		}

		if gerr, ok := errors.AsType[gophercloud.ErrUnexpectedResponseCode](err); ok && gerr.Actual == 409 {
			if bytes.Contains(gerr.Body, []byte("OverQuota")) {
				return nil, fmt.Errorf("%s: %w, %w", label, err, internal.ErrQuotaExceeded)
			} else if bytes.Contains(gerr.Body, []byte("No more IP addresses available")) {
				continue // try next subnet
			}
		}

		// unexpected error from neutron
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if err != nil || service.SubnetID == "" {
		// All subnet IPs are exhausted
		return nil, fmt.Errorf("%s: %w, %w", label, err, internal.ErrNoIPsAvailable)
	}

	// Fetch segmentID for the network
	if len(service.NeutronPorts) > 0 {
		// we only expect a valid segment if we have at least one Service port bound
		if service.SegmentId, err = a.neutron.GetNetworkSegment(ctx, service.NetworkID.String(),
			config.Global.Agent.PhysicalNetwork); err != nil {
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
		if extendedService, err := a.getExtendedService(ctx, service); err != nil {
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
		if service.Status != models.ServiceStatusPENDINGDELETE {
			if err := a.EnsureL2(ctx, service.SegmentId, nil, service.MTU); err != nil {
				return err
			}
			if err := a.EnsureSelfIPs(ctx, service.SubnetID, true); err != nil {
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

	if err = a.active.PostAS3(&data, "Common"); err != nil {
		return err
	}

	/* ==================================================
	   Clean up orphaned endpoint tenants (e.g. after service migration)
	   ================================================== */
	var usedSegments map[int]string
	if usedSegments, err = a.getUsedSegments(ctx); err != nil {
		log.WithError(err).Warning("ProcessServices: failed to get used segments for orphan cleanup")
	} else if err = a.cleanupOrphanedTenants(usedSegments); err != nil {
		log.WithError(err).Warning("ProcessServices: failed to clean up orphaned tenants")
	}

	/* ==================================================
	   L2 Configuration Cleanup
	   ================================================== */
	for _, service := range services {
		if service.Status == models.ServiceStatusPENDINGDELETE {
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
				segmentID, err = a.neutron.GetSubnetSegment(ctx, service.SubnetID, config.Global.Agent.PhysicalNetwork)
				if errors.Is(err, internal.ErrNoPhysNetFound) {
					// No segment found, skip L2 cleanup
					cleanupL2 = false
				} else if err != nil {
					return err
				}
			}

			if cleanupSelfIPs {
				logWith.WithField("subnet", service.SubnetID).Info("ProcessServices: deleting SelfIPs")
				if err := a.CleanupSelfIPs(ctx, service.SubnetID); err != nil {
					return err
				}
			}

			// Delete service-scoped SNAT ports if the service uses them. Independent of the
			// L2/SelfIP cleanup gating because these ports belong to the service, not the
			// shared subnet pool.
			if service.SnatPoolSize != nil {
				if err = a.neutron.CleanupServiceSnatPorts(ctx, service.ID); err != nil {
					return fmt.Errorf("CleanupServiceSnatPorts: %w", err)
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
