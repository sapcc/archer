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

	"github.com/gophercloud/gophercloud"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
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

func (a *Agent) getUsedSegments() (map[int]struct{}, error) {
	sql, args := db.Select("s.network_id", "ep.segment_id").
		LeftJoin("endpoint e ON s.id = e.service_id").
		LeftJoin("endpoint_port ep ON ep.endpoint_id = e.id").
		From("service s").
		Where("s.host = ?", config.Global.Default.Host).
		Where("s.provider = 'tenant'").
		MustSql()

	rows, err := a.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	usedSegments := map[int]struct{}{}
	for rows.Next() {
		var networkID string
		var segmentID int

		if err = rows.Scan(&networkID, &segmentID); err != nil {
			return nil, err
		}
		if segmentID != 0 {
			// add to used segment map
			usedSegments[segmentID] = struct{}{}
		}
		serviceSegment, err := a.neutron.GetNetworkSegment(networkID)
		if err != nil {
			return nil, err
		}
		usedSegments[serviceSegment] = struct{}{}
	}

	return usedSegments, nil
}

func (a *Agent) cleanOrphanedRDs(usedSegments map[int]struct{}) error {
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

func (a *Agent) cleanOrphanedVLANs(usedSegments map[int]struct{}) error {
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

func (a *Agent) cleanOrphanedVCMPVLANs(usedSegments map[int]struct{}) error {
	log.WithField("usedSegments", usedSegments).Debug("Running cleanOrphanedVCMPVLANs")

	for _, b := range a.vcmps {
		if err := b.SyncGuestVLANs(usedSegments); err != nil {
			return err
		}
	}

	log.Debug("Finished cleanOrphanedVCMPVLANs")
	return nil
}
