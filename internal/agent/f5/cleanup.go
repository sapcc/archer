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

			// check if port exists in archer database
			sql, args, err := db.
				Select("1").
				From("service_port").
				Where("port_id = ?", portID).
				ToSql()
			if err != nil {
				log.Error(err)
				return
			}
			ct, err := a.pool.Exec(context.Background(), sql, args...)
			if err != nil {
				log.Error(err)
				return
			}

			if ct.RowsAffected() != 0 {
				// port exists, nothing to do
				continue
			}

			log.WithFields(log.Fields{
				"port_id": portID,
				"host":    bigip.GetHostname(),
			}).Warning("Found orphan SelfIP, deleting")

			// Delete neutron port, but don't fail if it doesn't exist
			if err := a.neutron.DeletePort(portID); err != nil {
				var errDefault404 gophercloud.ErrDefault404
				if !errors.As(err, &errDefault404) {
					log.Error(err)
					return
				}
			}

			// port should not exist, delete selfip
			if err := bigip.DeleteSelfIP(selfip.Name); err != nil {
				log.Error(err)
				return
			}
		}
	}
	log.Debug("Finished CleanOrphanSelfIPs")
}
