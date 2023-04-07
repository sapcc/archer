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

package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/agent/as3"
	"github.com/sapcc/archer/internal/config"
)

func (a *Agent) WorkerThread() {
	for job := range a.jobQueue {
		logg.Info("received message %v", job)

		switch job.model {
		case "service":
			if err := a.ProcessServices(); err != nil {
				logg.Error(err.Error())
			}
		case "endpoint":
			if err := a.ProcessEndpoint(job.id); err != nil {
				logg.Error(err.Error())
			}

		}
	}
}

func (a *Agent) ProcessServices() (err error) {
	var services []*ExtendedService
	var tx pgx.Tx
	tx, err = a.pool.Begin(context.Background())
	if err != nil {
		return err
	}

	defer func(tx pgx.Tx, services []*ExtendedService) {
		if err == nil {
			// Try commit
			if err = tx.Commit(context.Background()); err != nil {
				logg.Error(err.Error())
			}
		}

		if err != nil {
			// Revert ports that have been created in this transaction
			for _, service := range services {
				if service.TXAllocated {
					logg.Error("Orphaned neutron SNAT port due rollback, deleting '%s'", service.SnatPortId)
					if err := a.DeleteSNATPort(service); err != nil {
						logg.Error(err.Error())
					}
				}
			}

			if err := tx.Rollback(context.Background()); err != nil {
				logg.Error(err.Error())
			}
		}
	}(tx, services)

	// We need to fetch all services of this host since the AS3 tenant is shared
	if err = pgxscan.Select(context.Background(), tx, &services,
		`SELECT id, enabled, network_id, proxy_protocol, port, ip_addresses, sap.port_id AS snat_port_id
              FROM service 
                  LEFT JOIN service_snat_port sap ON service.id = sap.service_id 
              WHERE host = $1`,
		config.HostName(),
	); err != nil {
		return err
	}

	// Ensure SNAT neutron ports and segment ids
	for _, service := range services {

		// Fetch SNAT port from neutron
		if service.SnatPortId != nil {
			service.SnatPort, err = a.GetSNATPort(service.SnatPortId)
			if err != nil {
				return err
			}
		} else {
			service.SnatPort, err = a.AllocateSNATPort(service)
			if err != nil {
				return err
			}
			// set allocated flag, for deletion during rollback
			service.TXAllocated = true

			if _, err = tx.Exec(context.Background(), `INSERT INTO service_snat_port(service_id, port_id) VALUES ($1, $2)`,
				service.ID, service.SnatPort.ID); err != nil {
				return err
			}
			if err = tx.Commit(context.Background()); err != nil {
				return err
			}

			return nil
		}

		// Fetch segment ID from neutron
		service.SegmentId, err = a.GetNetworkSegment(service.NetworkID.String())
		if err != nil {
			return err
		}
		if err := a.EnsureVLAN(service.SegmentId); err != nil {
			return err
		}
		if err := a.EnsureRouteDomain(service.SegmentId); err != nil {
			return err
		}
	}

	data := GetAS3Declaration(map[string]as3.Tenant{
		"Common": GetServiceTenants(services),
	})

	var js []byte
	js, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		logg.Fatal(err.Error())
	}

	if config.IsDebug() {
		fmt.Printf("-------------------> ProcessServices %s\n%s\n-------------------\n", a.bigip.Host, js)
	}

	var successfulTenants string
	err, successfulTenants, _ = a.bigip.PostAs3Bigip(string(js), "Common")
	if err != nil {
		return err
	}

	var ids []strfmt.UUID
	for _, service := range services {
		ids = append(ids, service.ID)
	}
	if _, err = tx.Exec(
		context.Background(),
		`UPDATE service SET status = 'AVAILABLE' WHERE id = ANY($1);`,
		ids); err != nil {
		return err
	}

	logg.Info("ProcessService successful for %s", successfulTenants)
	return nil
}

func (a *Agent) ProcessEndpoint(networkId strfmt.UUID) error {
	var endpoints []*ExtendedEndpoint
	var segmentId int

	tx, err := a.pool.Begin(context.Background())
	if err != nil {
		return err
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		// Rollback is safe to call even if the tx is already closed, so if
		// the tx commits successfully, this is a no-op
		_ = tx.Rollback(ctx)
	}(tx, context.Background())

	err = pgxscan.Select(context.Background(), tx, &endpoints,
		`SELECT endpoint.*, service.port AS service_port_nr 
              FROM endpoint
                  INNER JOIN service ON service.id = service_id and service.status = 'AVAILABLE' 
              WHERE endpoint."target.network" = $1`,
		networkId)
	if err != nil {
		return err
	}

	if len(endpoints) == 0 {
		return nil
	}

	// Fetch segment ID from neutron
	segmentId, err = a.GetNetworkSegment(networkId.String())
	if err != nil {
		return err
	}

	// Ensure VLAN and Route Domain
	if err := a.EnsureVLAN(segmentId); err != nil {
		return err
	}
	if err := a.EnsureRouteDomain(segmentId); err != nil {
		return err
	}

	// Fetch ports from neutron
	var opts PortListOpts
	for _, endpoint := range endpoints {
		opts.IDs = append(opts.IDs, endpoint.Target.Port.String())
	}

	var pages pagination.Page
	pages, err = ports.List(a.neutron, opts).AllPages()
	if err != nil {
		return err
	}
	endpointPorts, err := ports.ExtractPorts(pages)
	if err != nil {
		return err
	}
	for _, port := range endpointPorts {
		for _, endpoint := range endpoints {
			if endpoint.Target.Port.String() == port.ID {
				endpoint.Port = &port
				endpoint.SegmentId = segmentId
			}
		}
	}

	tenantName := GetEndpointTenantName(networkId)
	data := GetAS3Declaration(map[string]as3.Tenant{
		tenantName: GetEndpointTenants(endpoints),
	})

	var js []byte
	js, err = json.MarshalIndent(data, "", "  ")
	if err != nil {
		logg.Fatal(err.Error())
	}

	if config.IsDebug() {
		fmt.Printf("-------------------> %s\n%s\n-------------------\n", a.bigip.Host, js)
	}

	var successfulTenants string
	err, successfulTenants, _ = a.bigip.PostAs3Bigip(string(js), tenantName)
	if err != nil {
		return err
	}

	var ids []strfmt.UUID
	for _, endpoint := range endpoints {
		ids = append(ids, endpoint.ID)
	}
	if _, err = tx.Exec(
		context.Background(),
		`UPDATE endpoint SET status = 'AVAILABLE' WHERE id = ANY($1);`,
		ids); err != nil {
		return err
	}

	if err = tx.Commit(context.Background()); err != nil {
		return err
	}

	logg.Info("ProcessEndpoint successful for %s", successfulTenants)
	return nil
}
