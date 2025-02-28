// Copyright 2024 SAP SE
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
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/jackc/pgx/v5"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/internal/neutron"
	"github.com/sapcc/archer/models"
)

func checkCleanupL2(ctx context.Context, tx pgx.Tx, networkID string,
	ignorePendingEndpoint bool, ignorePendingService bool) (error, bool) {
	q := db.Select("1").
		From("service").
		Where("network_id = ?", networkID).
		Where("host = ?", config.Global.Default.Host).
		Where("provider = ?", models.ServiceProviderTenant)
	if ignorePendingService {
		q = q.Where("status != 'PENDING_DELETE'")
	}
	sql, args := q.MustSql()
	ct, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return err, false
	}
	if ct.RowsAffected() > 0 {
		// There are services in the network, don't remove l2 configuration
		return nil, false
	}

	q = db.Select("1").
		From("endpoint").
		InnerJoin("service ON endpoint.service_id = service.id").
		Join("endpoint_port ON endpoint_id = endpoint.id").
		Where("endpoint_port.network = ?", networkID).
		Where("service.host = ?", config.Global.Default.Host).
		Where("service.provider = ?", models.ServiceProviderTenant)
	if ignorePendingEndpoint {
		q = q.Where(sq.NotEq{"endpoint.status": []models.EndpointStatus{
			models.EndpointStatusPENDINGDELETE,
			models.EndpointStatusPENDINGREJECTED}})
	}
	sql, args = q.MustSql()
	ct, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		return err, false
	}
	if ct.RowsAffected() > 0 {
		// There are services in the network, don't remove l2 configuration
		return nil, false
	}

	// no dependent objects, cleanup L2
	return nil, true
}

func (a *Agent) checkCleanupSelfIPs(ctx context.Context, tx pgx.Tx, networkID string, subnetID string,
	ignorePendingEndpoint bool, ignorePendingService bool) (error, bool) {
	// Check if there are existing endpoints in the subnet
	q := db.Select("1").
		From("endpoint").
		InnerJoin("service ON endpoint.service_id = service.id").
		Join("endpoint_port ON endpoint_id = endpoint.id").
		Where("endpoint_port.subnet = ?", subnetID).
		Where("service.host = ?", config.Global.Default.Host).
		Where("service.provider = ?", models.ServiceProviderTenant)
	if ignorePendingEndpoint {
		q = q.Where(sq.NotEq{"endpoint.status": []models.EndpointStatus{
			models.EndpointStatusPENDINGDELETE,
			models.EndpointStatusPENDINGREJECTED}})
	}
	sql, args := q.MustSql()
	ct, err := tx.Exec(ctx, sql, args...)
	if err != nil {
		return err, false
	}
	if ct.RowsAffected() > 0 {
		// There are endpoints in the subnet, skip cleanup
		return nil, false
	}

	var network *neutron.NetworkMTU
	network, err = a.neutron.GetNetwork(networkID)
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		// The network is already deleted
		return nil, false
	} else if err != nil {
		return err, false
	}

	if network.Subnets[0] == subnetID {
		// There could be a service in the same subnet
		q := db.Select("1").
			From("service").
			Where("network_id = ?", networkID).
			Where("host = ?", config.Global.Default.Host).
			Where("provider = ?", models.ServiceProviderTenant)
		if ignorePendingService {
			q = q.Where("status != 'PENDING_DELETE'")
		}
		sql, args = q.MustSql()

		ct, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			return err, false
		}
		if ct.RowsAffected() > 0 {
			// There are service(s) in the subnet, don't remove self-ips
			return nil, false
		}
	}

	// no dependent objects, cleanup SelfIPs
	return nil, true
}
