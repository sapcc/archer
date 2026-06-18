// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/jackc/pgx/v5/pgtype"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/agent/f5/as3"
	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/internal/db"
	"github.com/sapcc/archer/v2/models"
)

func (a *Agent) cleanupL2(ctx context.Context) error {
	if err := a.cleanOrphanSelfIPs(ctx); err != nil {
		log.WithError(err).Error("cleanOrphanSelfIPs")
		// continue
	}

	usedSegments, err := a.getUsedSegments(ctx)
	if err != nil {
		return err
	}
	if err := a.cleanupOrphanedTenants(usedSegments); err != nil {
		log.WithError(err).Error("cleanupOrphanedTenants")
		// continue
	}
	if err := a.cleanOrphanedRDs(usedSegments); err != nil {
		log.WithError(err).Error("cleanOrphanedRDs")
		// continue
	}
	if err := a.cleanOrphanedVLANs(usedSegments); err != nil {
		log.WithError(err).Error("cleanOrphanedVLANs")
		// continue
	}
	if err := a.cleanOrphanedVCMPVLANs(usedSegments); err != nil {
		log.WithError(err).Error("cleanOrphanedVCMPVLANs")
		// continue
	}
	if err := a.cleanOrphanedNeutronPorts(ctx, usedSegments); err != nil {
		log.WithError(err).Error("cleanOrphanedNeutronPorts")
		// continue
	}
	if err := a.cleanOrphanedSnatPorts(ctx); err != nil {
		log.WithError(err).Error("cleanOrphanedSnatPorts")
		// continue
	}
	return nil
}

// cleanOrphanSelfIPs deletes SelfIPs that are not associated with a port
func (a *Agent) cleanOrphanSelfIPs(ctx context.Context) error {
	log.Debug("Running CleanOrphanSelfIPs")
	for _, bigip := range a.devices {
		selfips, err := bigip.GetSelfIPs()
		if err != nil {
			return err
		}

		for _, selfip := range selfips {
			var portID string
			if n, err := fmt.Sscanf(selfip, "selfip-%s", &portID); err != nil || n != 1 {
				continue
			}

			_, err := a.neutron.GetPort(ctx, portID)
			if err != nil {
				log.WithError(err).WithField("port_id", portID).Info("cleanOrphanSelfIPs")
				if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
					log.WithFields(log.Fields{
						"port_id": portID,
						"host":    bigip.GetHostname(),
					}).Warning("Found orphan SelfIP, deleting")

					// port should not exist, delete selfip
					if err := bigip.DeleteSelfIP(selfip); err != nil {
						log.
							WithField("host", bigip.GetHostname()).
							WithError(err).
							Error("cleanOrphanSelfIPs")
					}
				}
			}
		}
	}
	log.Debug("Finished CleanOrphanSelfIPs")
	return nil
}

func (a *Agent) getUsedSegments(ctx context.Context) (map[int]string, error) {
	sql, args := db.Select("s.network_id", "ep.segment_id", "ep.network").
		LeftJoin("endpoint e ON s.id = e.service_id").
		LeftJoin("endpoint_port ep ON ep.endpoint_id = e.id").
		From("service s").
		Where("s.host = ?", config.Global.Default.Host).
		Where("s.provider = ?", models.ServiceProviderTenant).
		MustSql()

	rows, err := a.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	usedSegments := map[int]string{}
	for rows.Next() {
		var networkID string
		var epNetworkID pgtype.UUID
		var segmentID pgtype.Int4

		if err = rows.Scan(&networkID, &segmentID, &epNetworkID); err != nil {
			return nil, err
		}
		if epNetworkID.Valid && !segmentID.Valid {
			// refresh segmentID from neutron
			var tmp int
			if tmp, err = a.neutron.GetNetworkSegment(ctx, epNetworkID.String(),
				config.Global.Agent.PhysicalNetwork); err != nil {
				return nil, err
			}
			if err = segmentID.Scan(int64(tmp)); err != nil {
				return nil, err
			}
		}
		if segmentID.Valid && epNetworkID.Valid {
			// add endpoint to used segment map
			usedSegments[int(segmentID.Int32)] = epNetworkID.String()
		}
		serviceSegment, err := a.neutron.GetNetworkSegment(ctx, networkID, config.Global.Agent.PhysicalNetwork)
		if err != nil {
			return nil, err
		}
		usedSegments[serviceSegment] = networkID
	}

	return usedSegments, nil
}

func (a *Agent) cleanOrphanedRDs(usedSegments map[int]string) error {
	log.WithField("usedSegments", usedSegments).Debug("Running cleanOrphanedRDs")

	for _, bigip := range a.devices {
		routeDomains, err := bigip.GetRouteDomains()
		if err != nil {
			return err
		}

		for _, routeDomain := range routeDomains {
			// Check that routeDomain starts with vlan-
			var id int
			if n, err := fmt.Sscanf(routeDomain, "vlan-%d", &id); err != nil || n != 1 {
				continue
			}

			// Check if routeDomain is used
			if _, ok := usedSegments[id]; ok {
				continue
			}
			log.WithField("host", bigip.GetHostname()).
				Warningf("found orphan routeDomain %s, deleting", routeDomain)
			if err := bigip.DeleteRouteDomain(id); err != nil {
				log.Warningf("skipping routeDomain due interdependency: %s", err.Error())
			}
		}
	}
	log.Debug("Finished cleanOrphanedRDs")
	return nil
}

func (a *Agent) cleanOrphanedVLANs(usedSegments map[int]string) error {
	log.WithField("usedSegments", usedSegments).Debug("Running cleanOrphanedVLANs")

	for _, device := range a.devices {
		vlans, err := device.GetVLANs()
		if err != nil {
			return err
		}

		for _, vlan := range vlans {
			// Check that routeDomain starts with vlan-
			var segment int
			if n, err := fmt.Sscanf(vlan, "vlan-%d", &segment); err != nil || n != 1 {
				continue
			}

			// Check if routeDomain is used
			if _, ok := usedSegments[segment]; ok {
				continue
			}
			log.WithField("host", device.GetHostname()).
				Infof(" - Found orphan vlan %s, deleting", vlan)
			if err := device.DeleteVLAN(segment); err != nil {
				log.Error(err)
			}
		}
	}

	log.Debug("Finished cleanOrphanedVLANs")
	return nil
}

func (a *Agent) cleanOrphanedVCMPVLANs(usedSegments map[int]string) error {
	log.WithField("usedSegments", usedSegments).Debug("Running cleanOrphanedVCMPVLANs")

	for _, h := range a.hosts {
		if err := h.SyncGuestVLANs(usedSegments); err != nil {
			return err
		}
	}

	log.Debug("Finished cleanOrphanedVCMPVLANs")
	return nil
}

func (a *Agent) cleanOrphanedNeutronPorts(ctx context.Context, usedSegments map[int]string) error {
	log.Debug("Running cleanOrphanedNeutronPorts")

	// Fetch all selfips from neutron
	selfips, err := a.neutron.FetchSelfIPPorts(ctx)
	if err != nil {
		return err
	}

	// Fetch all segments for every selfip network
	for networkID, ports := range selfips {
		segment, err := a.neutron.GetNetworkSegment(ctx, networkID, config.Global.Agent.PhysicalNetwork)
		if err != nil {
			log.Errorf("cleanOrphanedNeutronPorts: %s", err.Error())
			continue
		}
		if _, ok := usedSegments[segment]; ok {
			// SelfIP in use
			continue
		}

		// SelfIP is part of an unused segment, delete it
		for _, port := range ports {
			log.WithFields(log.Fields{"network": networkID, "port": port.ID, "segment": segment}).
				Warningf("found orphan SelfIP port '%s', deleting", port.Name)
			if err := a.neutron.DeletePort(ctx, port.ID); err != nil {
				log.Errorf("cleanOrphanedNeutronPorts: %s", err.Error())
			}
		}
	}

	log.Debug("Finished cleanOrphanedNeutronPorts")
	return nil
}

// cleanOrphanedSnatPorts deletes SNAT pool ports whose owning service no longer
// exists in the DB on this host. SNAT ports are normally cleaned up by the
// PENDING_DELETE branch of ProcessServices via CleanupServiceSnatPorts; this
// catches the leftover cases — e.g. a row removed without going through that
// flow, or a CleanupServiceSnatPorts that errored after the row was gone.
//
// The orphan signal here is per-service (device_id), not per-segment as for
// SelfIPs, because SNAT pools are owned by a single service rather than shared
// across a subnet.
func (a *Agent) cleanOrphanedSnatPorts(ctx context.Context) error {
	log.Debug("Running cleanOrphanedSnatPorts")

	snats, err := a.neutron.FetchSnatPorts(ctx)
	if err != nil {
		return err
	}
	if len(snats) == 0 {
		log.Debug("Finished cleanOrphanedSnatPorts (no ports)")
		return nil
	}

	sql, args := db.Select("id").From("service").
		Where("host = ?", config.Global.Default.Host).
		Where("provider = ?", models.ServiceProviderTenant).
		MustSql()
	rows, err := a.pool.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("cleanOrphanedSnatPorts: list services: %w", err)
	}
	defer rows.Close()
	live := make(map[string]struct{})
	for rows.Next() {
		var id strfmt.UUID
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("cleanOrphanedSnatPorts: scan: %w", err)
		}
		live[id.String()] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("cleanOrphanedSnatPorts: rows: %w", err)
	}

	for _, port := range snats {
		if _, ok := live[port.DeviceID]; ok {
			continue // service still exists, port is live
		}
		log.WithFields(log.Fields{"port": port.ID, "service": port.DeviceID}).
			Warningf("found orphan SNAT port '%s', deleting", port.Name)
		if err := a.neutron.DeletePort(ctx, port.ID); err != nil {
			log.Errorf("cleanOrphanedSnatPorts: %s", err.Error())
		}
	}

	log.Debug("Finished cleanOrphanedSnatPorts")
	return nil
}

func (a *Agent) cleanupOrphanedTenants(usedSegments map[int]string) error {
	log.Debug("Running cleanupOrphanedTenants")

	for _, bigip := range a.devices {
		// Fetch all partitions
		partitions, err := bigip.GetPartitions()
		if err != nil {
			return err
		}

		for _, partition := range partitions {
			// skip Common partition
			if partition == "Common" {
				continue
			}

			// skip non-net partitions
			if !strings.HasPrefix(partition, "net-") {
				continue
			}

			// Check if partition is used
			used := false
			for _, networkID := range usedSegments {
				if as3.GetEndpointTenantName(strfmt.UUID(networkID)) == partition {
					used = true
					break
				}
			}
			if used {
				continue
			}

			log.WithFields(log.Fields{"host": bigip.GetHostname(), "partition": partition}).
				Warning("Found orphaned tenant, deleting")
			data := as3.GetAS3Declaration(map[string]as3.Tenant{
				partition: as3.GetEndpointTenants([]*as3.ExtendedEndpoint{}),
			})
			if err := a.active.PostAS3(&data, partition); err != nil {
				return err
			}
		}
	}
	log.Debug("Finished cleanupOrphanedTenants")
	return nil
}
