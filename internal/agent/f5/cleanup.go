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

// scanAndClean scans all selfips on all bigips and deletes them if they are not in the database
func (a *Agent) cleanOrphanSelfIPs() {
	log.Debug("Running CleanOrphanSelfIPs")
	for _, bigip := range a.bigips {
		selfips, err := bigip.SelfIPs()
		if err != nil {
			log.Error(err)
			return
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
						log.Error(err)
						return
					}
				}
			}
		}
	}
	log.Debug("Finished CleanOrphanSelfIPs")
}

func (a *Agent) cleanOrphanRDsAndVLANs() error {
	log.Debug("Running cleanOrphanRDsAndVLANs")

	sql, args := db.Select("s.network_id", "ep.segment_id").
		Join("endpoint e ON s.id = e.service_id").
		Join("endpoint_port ep ON ep.endpoint_id = e.id").
		From("service s").
		Where("s.host = ?", config.Global.Default.Host).
		Where("s.provider = 'tenant'").
		MustSql()

	rows, err := a.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	usedSegments := map[int]struct{}{}
	for rows.Next() {
		var networkID string
		var segmentID int

		if err = rows.Scan(&networkID, &segmentID); err != nil {
			return err
		}
		usedSegments[segmentID] = struct{}{}
		serviceSegment, err := a.neutron.GetNetworkSegment(networkID)
		if err != nil {
			return err
		}
		usedSegments[serviceSegment] = struct{}{}
	}

	// Print used segments
	log.Debugf("cleanOrphanRDsAndVLANs: Used segments: %v", usedSegments)

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
				Infof(" - found orphan routeDomain %s, deleting", routeDomain.Name)
			if err := bigip.DeleteRouteDomain(routeDomain.Name); err != nil {
				log.Warningf("- skipping due interdependency: %s", err.Error())
			}
		}
	}

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

	log.Debug("Finished cleanOrphanRDsAndVLANs")
	return nil
}
