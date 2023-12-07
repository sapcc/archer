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
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud"
	log "github.com/sirupsen/logrus"
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
