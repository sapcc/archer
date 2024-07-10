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
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/jackc/pgx/v5/pgtype"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/models"
)

func (a *Agent) cleanupL2() error {
	if err := a.cleanOrphanSelfIPs(); err != nil {
		log.WithError(err).Error("cleanOrphanSelfIPs")
		// continue
	}

	usedSegments, err := a.getUsedSegments()
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
	if err := a.cleanOrphanedNeutronPorts(usedSegments); err != nil {
		log.WithError(err).Error("cleanOrphanedNeutronPorts")
		// continue
	}
	return nil
}

// cleanOrphanSelfIPs deletes SelfIPs that are not associated with a port
func (a *Agent) cleanOrphanSelfIPs() error {
	log.Debug("Running CleanOrphanSelfIPs")
	for _, bigip := range a.bigips {
		selfips, err := bigip.SelfIPs()
		if err != nil {
			return err
		}

		for _, selfip := range selfips.SelfIPs {
			var portID string
			if n, err := fmt.Sscanf(selfip.Name, "selfip-%s", &portID); err != nil || n != 1 {
				continue
			}

			_, err := a.neutron.GetPort(portID)
			if err != nil {
				log.WithError(err).WithField("port_id", portID).Info("cleanOrphanSelfIPs")
				var errDefault404 gophercloud.ErrDefault404
				if errors.As(err, &errDefault404) {
					log.WithFields(log.Fields{
						"port_id": portID,
						"host":    bigip.GetHostname(),
					}).Warning("Found orphan SelfIP, deleting")

					// port should not exist, delete selfip
					if err := bigip.DeleteSelfIP(selfip.Name); err != nil {
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

func (a *Agent) getUsedSegments() (map[int]string, error) {
	sql, args := db.Select("s.network_id", "ep.segment_id", "ep.network").
		LeftJoin("endpoint e ON s.id = e.service_id").
		LeftJoin("endpoint_port ep ON ep.endpoint_id = e.id").
		From("service s").
		Where("s.host = ?", config.Global.Default.Host).
		Where("s.provider = ?", models.ServiceProviderTenant).
		MustSql()

	rows, err := a.pool.Query(context.Background(), sql, args...)
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
		if segmentID.Valid {
			// add to used segment map
			uuid, err := epNetworkID.Value()
			if err != nil {
				return nil, err
			}
			usedSegments[int(segmentID.Int32)] = uuid.(string)
		}
		serviceSegment, err := a.neutron.GetNetworkSegment(networkID)
		if err != nil {
			return nil, err
		}
		usedSegments[serviceSegment] = networkID
	}

	return usedSegments, nil
}

func (a *Agent) cleanOrphanedRDs(usedSegments map[int]string) error {
	log.WithField("usedSegments", usedSegments).Debug("Running cleanOrphanedRDs")

	for _, bigip := range a.bigips {
		routeDomains, err := bigip.RouteDomains()
		if err != nil {
			return err
		}

		for _, routeDomain := range routeDomains.RouteDomains {
			// Check that routeDomain starts with vlan-
			var id int
			if n, err := fmt.Sscanf(routeDomain.Name, "vlan-%d", &id); err != nil || n != 1 {
				continue
			}

			// Check if routeDomain is used
			if _, ok := usedSegments[id]; ok {
				continue
			}
			log.WithField("host", bigip.GetHostname()).
				Warningf("found orphan routeDomain %s, deleting", routeDomain.Name)
			if err := bigip.DeleteRouteDomain(routeDomain.Name); err != nil {
				log.Warningf("skipping routeDomain due interdependency: %s", err.Error())
			}
		}
	}
	log.Debug("Finished cleanOrphanedRDs")
	return nil
}

func (a *Agent) cleanOrphanedVLANs(usedSegments map[int]string) error {
	log.WithField("usedSegments", usedSegments).Debug("Running cleanOrphanedVLANs")

	for _, bigip := range a.bigips {
		vlans, err := bigip.Vlans()
		if err != nil {
			return err
		}

		for _, vlan := range vlans.Vlans {
			// Check that routeDomain starts with vlan-
			var segment int
			if n, err := fmt.Sscanf(vlan.Name, "vlan-%d", &segment); err != nil || n != 1 {
				continue
			}

			// Check if routeDomain is used
			if _, ok := usedSegments[segment]; ok {
				continue
			}
			log.WithField("host", bigip.GetHostname()).
				Infof(" - Found orphan vlan %s, deleting", vlan.Name)
			if err := bigip.DeleteVlan(vlan.Name); err != nil {
				log.Error(err)
			}
		}
	}

	log.Debug("Finished cleanOrphanedVLANs")
	return nil
}

func (a *Agent) cleanOrphanedVCMPVLANs(usedSegments map[int]string) error {
	log.WithField("usedSegments", usedSegments).Debug("Running cleanOrphanedVCMPVLANs")

	for _, b := range a.vcmps {
		if err := b.SyncGuestVLANs(usedSegments); err != nil {
			return err
		}
	}

	log.Debug("Finished cleanOrphanedVCMPVLANs")
	return nil
}

func (a *Agent) cleanOrphanedNeutronPorts(usedSegments map[int]string) error {
	log.Debug("Running cleanOrphanedNeutronPorts")

	// Fetch all selfips from neutron
	selfips, err := a.neutron.FetchSelfIPPorts()
	if err != nil {
		return err
	}

	// Fetch all segments for every selfip network
	for networkID, ports := range selfips {
		segment, err := a.neutron.GetNetworkSegment(networkID)
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
			if err := a.neutron.DeletePort(port.ID); err != nil {
				log.Errorf("cleanOrphanedNeutronPorts: %s", err.Error())
			}
		}
	}

	log.Debug("Finished cleanOrphanedNeutronPorts")
	return nil
}

func (a *Agent) cleanupOrphanedTenants(usedSegments map[int]string) error {
	log.Debug("Running cleanupOrphanedTenants")

	for _, bigip := range a.bigips {
		// Fetch all partitions
		partitions, err := bigip.TMPartitions()
		if err != nil {
			return err
		}

		for _, partition := range partitions.TMPartitions {
			// skip Common partition
			if partition.Name == "Common" {
				continue
			}

			// skip non-net partitions
			if !strings.HasPrefix(partition.Name, "net-") {
				continue
			}

			// Check if partition is used
			used := false
			for _, networkID := range usedSegments {
				if as3.GetEndpointTenantName(strfmt.UUID(networkID)) == partition.Name {
					used = true
					break
				}
			}
			if used {
				continue
			}

			log.WithFields(log.Fields{"host": bigip.GetHostname(), "partition": partition.Name}).
				Warning("Found orphaned tenant, deleting")
			data := as3.GetAS3Declaration(map[string]as3.Tenant{
				partition.Name: as3.GetEndpointTenants([]*as3.ExtendedEndpoint{}),
			})
			if err := a.bigip.PostBigIP(&data, partition.Name); err != nil {
				return err
			}
		}
	}
	log.Debug("Finished cleanupOrphanedTenants")
	return nil
}
