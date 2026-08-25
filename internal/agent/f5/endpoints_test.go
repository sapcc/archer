// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag/conv"
	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/gophercloud/gophercloud/v2/testhelper/fixture"
	"github.com/pashagolub/pgxmock/v5"

	"github.com/sapcc/archer/v2/internal/agent/f5/as3"
	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/internal/neutron"
	"github.com/sapcc/archer/v2/models"
)

const GetNetworkResponseFixture = `
{
    "network": {
        "id": "35a3ca82-62af-4e0a-9472-92331500fb3a",
        "subnets": ["e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e"],
		"project_id": "test-project-1",
		"segments": [
			{
				"provider:physical_network": "physnet1",
				"provider:network_type": "vlan",
				"provider:segmentation_id": 123
			}
		]
    }
}
`

const GetNetworkResponseAnotherOneFixture = `
{
    "network": {
        "id": "ba578650-e29d-4c63-8847-59ebeedf4629",
        "subnets": ["e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e"],
		"project_id": "test-project-1",
		"segments": [
			{
				"provider:physical_network": "physnet1",
				"provider:network_type": "vlan",
				"provider:segmentation_id": 999
			}
		]
    }
}
`

const GetSubnetResponseFixture = `
{
	"subnet": {
        "cidr": "192.0.0.0/8",
        "network_id": "35a3ca82-62af-4e0a-9472-92331500fb3a"
	}
}
`

const GetServiceNetworkResponseFixture = `
{
    "network": {
        "id": "b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b",
		"segments": [
			{
				"provider:physical_network": "physnet1",
				"provider:network_type": "vlan",
				"provider:segmentation_id": 666
			}
		]
    }
}
`

const GetPortListResponseFixture = `
{
	"ports": [
		{
			"fixed_ips": [
				{
					"subnet_id": "e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e",
					"ip_address": "2.3.4.5"
				}
			],
			"device_owner": "network:f5snat",
			"id": "c0c0c0c0-c0c0-4c0c-8c0c-0c0c0c0c0c0c",
			"network_id": "35a3ca82-62af-4e0a-9472-92331500fb3a",
			"project_id": "test-project-1"
		},
		{
			"name": "local-dummybigiphost",
			"fixed_ips": [
				{
					"subnet_id": "e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e",
					"ip_address": "42.42.42.42"
				}
			],
			"device_owner": "network:f5selfip",
			"id": "5a8ad669-4ffe-4133-b9f9-6de62cd654a4",
			"network_id": "35a3ca82-62af-4e0a-9472-92331500fb3a",
			"project_id": "test-project-1"
		}
	]
}
`

const EmptyPortListResponseFixture = `
{
	"ports": []
}
`

// GetPortListWithEndpointFixture extends GetPortListResponseFixture with the
// endpoint port used in TestAgent_DeleteEndpointBlockedByPendingDeleteService.
const GetPortListWithEndpointFixture = `
{
	"ports": [
		{
			"fixed_ips": [{"subnet_id": "e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e", "ip_address": "2.3.4.5"}],
			"device_owner": "network:f5snat",
			"id": "c0c0c0c0-c0c0-4c0c-8c0c-0c0c0c0c0c0c",
			"network_id": "35a3ca82-62af-4e0a-9472-92331500fb3a",
			"project_id": "test-project-1"
		},
		{
			"name": "local-dummybigiphost",
			"fixed_ips": [{"subnet_id": "e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e", "ip_address": "42.42.42.42"}],
			"device_owner": "network:f5selfip",
			"id": "5a8ad669-4ffe-4133-b9f9-6de62cd654a4",
			"network_id": "35a3ca82-62af-4e0a-9472-92331500fb3a",
			"project_id": "test-project-1"
		},
		{
			"fixed_ips": [{"subnet_id": "e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e", "ip_address": "10.0.0.1"}],
			"id": "c1c1c1c1-c1c1-4c1c-8c1c-1c1c1c1c1c1c",
			"network_id": "35a3ca82-62af-4e0a-9472-92331500fb3a",
			"project_id": "test-project-1"
		}
	]
}
`

var PostBigIPFixture = &as3.AS3{
	Persist: false,
	Class:   "AS3",
	Action:  "deploy",
	Declaration: as3.ADC{
		Class:         "ADC",
		SchemaVersion: "3.36.0",
		UpdateMode:    "selective",
		Id:            "urn:uuid:07649173-4AF7-48DF-963F-84000C70F0DD",
		Tenants: map[string]as3.Tenant{
			"net-35a3ca82-62af-4e0a-9472-92331500fb3a": {
				Class: "Tenant",
				Applications: map[string]as3.Application{
					"si-endpoints": {
						Class:    "Application",
						Template: "generic", Services: map[string]any{
							"endpoint-80-95dbe813-62f9-47f1-90ba-09f2dadcaefa": as3.Service{
								Label:      "endpoint-80-95dbe813-62f9-47f1-90ba-09f2dadcaefa",
								Class:      "Service_L4",
								AllowVlans: []string{"/Common/vlan-123"},
								IRules:     []as3.Pointer{}, Mirroring: "none",
								PersistenceMethods: []string{}, Pool: as3.Pointer{
									BigIP: "/Common/Shared/pool-a0a0a0a0-a0a0-4a0a-8a0a-0a0a0a0a0a0a-80"},
								ProfileL4:  &as3.Pointer{BigIP: "/Common/cc_fastL4_noaging_profile"},
								ProfileTCP: &as3.Pointer{BigIP: "/Common/cc_tcp_archer_profile"},
								Snat: as3.Pointer{
									BigIP: "/Common/Shared/snatpool-a0a0a0a0-a0a0-4a0a-8a0a-0a0a0a0a0a0a",
								},
								VirtualAddresses:    []string{"2.3.4.5%123"},
								TranslateServerPort: true, VirtualPort: 80},
						},
					},
				},
			},
		},
	},
}

var BigIPCleanupFixture = &as3.AS3{
	Persist: false,
	Class:   "AS3",
	Action:  "deploy",
	Declaration: as3.ADC{
		Class: "ADC", SchemaVersion: "3.36.0",
		UpdateMode: "selective",
		Id:         "urn:uuid:07649173-4AF7-48DF-963F-84000C70F0DD",
		Tenants: map[string]as3.Tenant{
			"net-35a3ca82-62af-4e0a-9472-92331500fb3a": {
				Class:        "Tenant",
				Applications: map[string]as3.Application(nil),
			},
		},
	},
}

func TestAgent_ProcessEndpoint(t *testing.T) {
	endpoint := strfmt.UUID("95dbe813-62f9-47f1-90ba-09f2dadcaefa")
	port := strfmt.UUID("c0c0c0c0-c0c0-4c0c-8c0c-0c0c0c0c0c0c")
	network := strfmt.UUID("35a3ca82-62af-4e0a-9472-92331500fb3a")
	subnet := strfmt.UUID("e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e")
	service := strfmt.UUID("a0a0a0a0-a0a0-4a0a-8a0a-0a0a0a0a0a0a")
	serviceNetwork := strfmt.UUID("b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b")

	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	config.Global.Agent.L4Profile = "/Common/cc_fastL4_noaging_profile"
	config.Global.Agent.TCPProfile = "/Common/cc_tcp_archer_profile"
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+network.String(), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+serviceNetwork.String(), "GET",
		"", GetServiceNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "",
		GetPortListResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+subnet.String(), "GET", "",
		GetSubnetResponseFixture, http.StatusOK)

	ctx := context.Background()
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	f5DeviceHost := NewMockF5Device(t)
	f5DeviceHost.On("GetHostname").Return("dummybigiphost")
	f5DeviceHost.EXPECT().
		EnsureVLAN(123, 0).
		Return(nil)
	f5DeviceHost.EXPECT().
		EnsureRouteDomain(123, conv.Pointer(666)).
		Return(nil)
	f5DeviceHost.EXPECT().
		EnsureBigIPSelfIP(
			"selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4",
			"42.42.42.42%123/8",
			123,
		).Return(nil)
	f5DeviceHost.EXPECT().
		PostAS3(PostBigIPFixture, "net-35a3ca82-62af-4e0a-9472-92331500fb3a").
		Return(nil)

	config.Global.Default.Host = "host-123"
	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	neutronClient.InitCache()
	a := &Agent{
		pool:    dbMock,
		neutron: &neutronClient,
		devices: []F5Device{f5DeviceHost},
		hosts:   []F5Device{},
		active:  f5DeviceHost,
	}
	dbMock.
		ExpectBegin()
	dbMock.ExpectQuery("SELECT pg_try_advisory_xact_lock($1)").
		WithArgs(hostLockID(advisoryLockProcessEndpoints)).
		WillReturnRows(pgxmock.NewRows([]string{"pg_try_advisory_xact_lock"}).AddRow(true))
	dbMock.ExpectQuery("SELECT network, subnet FROM endpoint_port WHERE endpoint_id = $1").
		WithArgs(endpoint).
		WillReturnRows(pgxmock.NewRows([]string{"network", "subnet"}).AddRow(network, subnet.String()))
	dbMock.ExpectQuery("SELECT endpoint.*, service.ports AS service_ports, service.proxy_protocol, service.network_id AS service_network_id, service.status AS service_status, endpoint_port.segment_id, endpoint_port.port_id AS \"target.port\", endpoint_port.network AS \"target.network\", endpoint_port.subnet AS \"target.subnet\", endpoint_port.owned FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint.status NOT IN ($1,$2) AND network = $3 AND service.host = $4 AND service.provider = $5").
		WithArgs(models.EndpointStatusPENDINGAPPROVAL, models.EndpointStatusREJECTED, network, config.Global.Default.Host, models.ServiceProviderTenant).
		WillReturnRows(pgxmock.
			NewRows([]string{"id", "service_id", "name", "service_ports", "proxy_protocol", "service_network_id", "service_status", "segment_id", "target.port", "target.network", "target.subnet"}).
			AddRow(endpoint, service, "test-service", []int32{80}, false, serviceNetwork, string(models.ServiceStatusAVAILABLE), nil, &port, &network, &subnet))
	dbMock.ExpectExec("UPDATE endpoint_port SET segment_id = $1 WHERE endpoint_id = $2").
		WithArgs(123, endpoint).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	dbMock.ExpectExec("SELECT 1 FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint_port.subnet = $1 AND service.host = $2 AND service.provider = $3 AND endpoint.status NOT IN ($4,$5)").
		WithArgs(subnet.String(), config.Global.Default.Host, models.ServiceProviderTenant, models.EndpointStatusPENDINGDELETE, models.EndpointStatusPENDINGREJECTED).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
	// Short write transaction for the status transition.
	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE endpoint SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3").
		WithArgs(models.EndpointStatusAVAILABLE, endpoint, models.EndpointStatus("")).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	dbMock.ExpectCommit()
	dbMock.ExpectRollback() // write tx defer (no-op after commit)
	dbMock.ExpectRollback() // advisory-lock tx defer

	if err := a.ProcessEndpoint(ctx, endpoint); err != nil {
		t.Errorf("Agent.ProcessEndpoint() error = %v", err)
	}
	if err := dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// TestAgent_ProcessEndpointDefersWhenServiceNotReady verifies that during a
// migration, an endpoint whose owning service has not yet reached AVAILABLE on
// this host (its /Common/Shared snatpool is not posted yet) is NOT pushed to
// BigIP. Posting would fail with a 422; instead the run returns quietly and
// PendingSyncLoop re-drives it once ProcessServices makes the service AVAILABLE.
func TestAgent_ProcessEndpointDefersWhenServiceNotReady(t *testing.T) {
	endpoint := strfmt.UUID("95dbe813-62f9-47f1-90ba-09f2dadcaefa")
	port := strfmt.UUID("c0c0c0c0-c0c0-4c0c-8c0c-0c0c0c0c0c0c")
	network := strfmt.UUID("35a3ca82-62af-4e0a-9472-92331500fb3a")
	subnet := strfmt.UUID("e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e")
	service := strfmt.UUID("a0a0a0a0-a0a0-4a0a-8a0a-0a0a0a0a0a0a")
	serviceNetwork := strfmt.UUID("b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b")

	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	config.Global.Agent.L4Profile = "/Common/cc_fastL4_noaging_profile"
	config.Global.Agent.TCPProfile = "/Common/cc_tcp_archer_profile"
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+network.String(), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+serviceNetwork.String(), "GET",
		"", GetServiceNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "",
		GetPortListResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+subnet.String(), "GET", "",
		GetSubnetResponseFixture, http.StatusOK)

	ctx := context.Background()
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	// L2/SelfIP ensures are idempotent and still run before the readiness guard.
	f5DeviceHost := NewMockF5Device(t)
	f5DeviceHost.On("GetHostname").Return("dummybigiphost")
	f5DeviceHost.EXPECT().EnsureVLAN(123, 0).Return(nil)
	f5DeviceHost.EXPECT().EnsureRouteDomain(123, conv.Pointer(666)).Return(nil)
	f5DeviceHost.EXPECT().EnsureBigIPSelfIP(
		"selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4",
		"42.42.42.42%123/8",
		123,
	).Return(nil)
	// Crucially: no PostAS3 expectation — the run must defer before posting.

	config.Global.Default.Host = "host-123"
	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	neutronClient.InitCache()
	a := &Agent{
		pool:    dbMock,
		neutron: &neutronClient,
		devices: []F5Device{f5DeviceHost},
		hosts:   []F5Device{},
		active:  f5DeviceHost,
	}
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT pg_try_advisory_xact_lock($1)").
		WithArgs(hostLockID(advisoryLockProcessEndpoints)).
		WillReturnRows(pgxmock.NewRows([]string{"pg_try_advisory_xact_lock"}).AddRow(true))
	dbMock.ExpectQuery("SELECT network, subnet FROM endpoint_port WHERE endpoint_id = $1").
		WithArgs(endpoint).
		WillReturnRows(pgxmock.NewRows([]string{"network", "subnet"}).AddRow(network, subnet.String()))
	dbMock.ExpectQuery("SELECT endpoint.*, service.ports AS service_ports, service.proxy_protocol, service.network_id AS service_network_id, service.status AS service_status, endpoint_port.segment_id, endpoint_port.port_id AS \"target.port\", endpoint_port.network AS \"target.network\", endpoint_port.subnet AS \"target.subnet\", endpoint_port.owned FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint.status NOT IN ($1,$2) AND network = $3 AND service.host = $4 AND service.provider = $5").
		WithArgs(models.EndpointStatusPENDINGAPPROVAL, models.EndpointStatusREJECTED, network, config.Global.Default.Host, models.ServiceProviderTenant).
		WillReturnRows(pgxmock.
			NewRows([]string{"id", "service_id", "name", "service_ports", "proxy_protocol", "service_network_id", "service_status", "segment_id", "target.port", "target.network", "target.subnet"}).
			// Service is still PENDING_CREATE: snatpool not yet posted to BigIP.
			AddRow(endpoint, service, "test-service", []int32{80}, false, serviceNetwork, string(models.ServiceStatusPENDINGCREATE), nil, &port, &network, &subnet))
	dbMock.ExpectExec("UPDATE endpoint_port SET segment_id = $1 WHERE endpoint_id = $2").
		WithArgs(123, endpoint).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	dbMock.ExpectExec("SELECT 1 FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint_port.subnet = $1 AND service.host = $2 AND service.provider = $3 AND endpoint.status NOT IN ($4,$5)").
		WithArgs(subnet.String(), config.Global.Default.Host, models.ServiceProviderTenant, models.EndpointStatusPENDINGDELETE, models.EndpointStatusPENDINGREJECTED).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
	// No write transaction: the run defers before persisting any status change.
	dbMock.ExpectRollback() // advisory-lock tx defer

	if err := a.ProcessEndpoint(ctx, endpoint); err != nil {
		t.Errorf("Agent.ProcessEndpoint() error = %v", err)
	}
	if err := dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestAgent_DeleteEndpointWithDeletedNetwork(t *testing.T) {
	endpoint := strfmt.UUID("95dbe813-62f9-47f1-90ba-09f2dadcaefa")
	port := strfmt.UUID("c0c0c0c0-c0c0-4c0c-8c0c-0c0c0c0c0c0c")
	network := strfmt.UUID("35a3ca82-62af-4e0a-9472-92331500fb3a")
	subnet := strfmt.UUID("e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e")
	service := strfmt.UUID("a0a0a0a0-a0a0-4a0a-8a0a-0a0a0a0a0a0a")
	serviceNetwork := strfmt.UUID("b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b")

	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+network.String(), "GET",
		"", GetNetworkResponseFixture, http.StatusNotFound)
	// Ports are deleted along with network - return empty list
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET",
		"", EmptyPortListResponseFixture, http.StatusOK)

	ctx := context.Background()
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	f5DeviceHost := NewMockF5Device(t)

	config.Global.Default.Host = "host-123"
	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	neutronClient.InitCache()
	a := &Agent{
		pool:    dbMock,
		neutron: &neutronClient,
		devices: []F5Device{f5DeviceHost},
		hosts:   []F5Device{},
		active:  f5DeviceHost,
	}

	dbMock.
		ExpectBegin()
	dbMock.ExpectQuery("SELECT pg_try_advisory_xact_lock($1)").
		WithArgs(hostLockID(advisoryLockProcessEndpoints)).
		WillReturnRows(pgxmock.NewRows([]string{"pg_try_advisory_xact_lock"}).AddRow(true))
	dbMock.ExpectQuery("SELECT network, subnet FROM endpoint_port WHERE endpoint_id = $1").
		WithArgs(endpoint).
		WillReturnRows(pgxmock.NewRows([]string{"network", "subnet"}).AddRow(network, subnet.String()))
	dbMock.ExpectQuery("SELECT endpoint.*, service.ports AS service_ports, service.proxy_protocol, service.network_id AS service_network_id, service.status AS service_status, endpoint_port.segment_id, endpoint_port.port_id AS \"target.port\", endpoint_port.network AS \"target.network\", endpoint_port.subnet AS \"target.subnet\", endpoint_port.owned FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint.status NOT IN ($1,$2) AND network = $3 AND service.host = $4 AND service.provider = $5").
		WithArgs(models.EndpointStatusPENDINGAPPROVAL, models.EndpointStatusREJECTED, network, config.Global.Default.Host, models.ServiceProviderTenant).
		WillReturnRows(pgxmock.
			NewRows([]string{"id", "service_id", "status", "name", "service_ports", "proxy_protocol", "service_network_id", "service_status", "segment_id", "target.port", "target.network", "target.subnet"}).
			AddRow(endpoint, service, models.EndpointStatusPENDINGDELETE, "test-service", []int32{80}, false, serviceNetwork, string(models.ServiceStatusPENDINGDELETE), nil, &port, &network, &subnet))
	dbMock.ExpectExec("SELECT 1 FROM service WHERE network_id = $1 AND host = $2 AND provider = $3").
		WithArgs(network.String(), config.Global.Default.Host, models.ServiceProviderTenant).
		WillReturnResult(pgxmock.NewResult("SELECT", 0))
	dbMock.ExpectExec("SELECT 1 FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint_port.network = $1 AND service.host = $2 AND service.provider = $3 AND endpoint.status NOT IN ($4,$5)").
		WithArgs(network.String(), config.Global.Default.Host, models.ServiceProviderTenant, models.EndpointStatusPENDINGDELETE, models.EndpointStatusPENDINGREJECTED).
		WillReturnResult(pgxmock.NewResult("SELECT", 0))
	dbMock.ExpectExec("SELECT 1 FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint_port.subnet = $1 AND service.host = $2 AND service.provider = $3 AND endpoint.status NOT IN ($4,$5)").
		WithArgs(subnet.String(), config.Global.Default.Host, models.ServiceProviderTenant, models.EndpointStatusPENDINGDELETE, models.EndpointStatusPENDINGREJECTED).
		WillReturnResult(pgxmock.NewResult("SELECT", 0))
	f5DeviceHost.EXPECT().
		PostAS3(BigIPCleanupFixture, "net-35a3ca82-62af-4e0a-9472-92331500fb3a").
		Return(nil)
	// Short write transaction for the endpoint deletion.
	dbMock.ExpectBegin()
	dbMock.ExpectExec("DELETE FROM endpoint WHERE id = $1 AND status = $2").
		WithArgs(endpoint, models.EndpointStatusPENDINGDELETE).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
	dbMock.ExpectCommit()
	dbMock.ExpectRollback() // write tx defer (no-op after commit)
	dbMock.ExpectRollback() // advisory-lock tx defer

	if err = a.ProcessEndpoint(ctx, endpoint); err != nil {
		t.Errorf("Agent.ProcessEndpoint() error = %v", err)
	}
	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// TestAgent_DeleteEndpointBlockedByPendingDeleteService verifies that a
// non-delete endpoint whose service is PENDING_DELETE is processed (not
// deferred) without an AS3 post, unblocking the subsequent service deletion.
func TestAgent_DeleteEndpointBlockedByPendingDeleteService(t *testing.T) {
	endpoint := strfmt.UUID("bfab83f8-9582-4e88-b3dd-96a006ba95d6")
	port := strfmt.UUID("c1c1c1c1-c1c1-4c1c-8c1c-1c1c1c1c1c1c")
	network := strfmt.UUID("35a3ca82-62af-4e0a-9472-92331500fb3a")
	subnet := strfmt.UUID("e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e")
	service := strfmt.UUID("dad9b89f-5619-4738-aecf-f7ce76b98b79")
	serviceNetwork := strfmt.UUID("b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b")

	fakeServer := th.SetupPersistentPortHTTP(t, 8932)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	config.Global.Agent.L4Profile = "/Common/cc_fastL4_noaging_profile"
	config.Global.Agent.TCPProfile = "/Common/cc_tcp_archer_profile"
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+network.String(), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+serviceNetwork.String(), "GET",
		"", GetServiceNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "",
		GetPortListWithEndpointFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+subnet.String(), "GET", "",
		GetSubnetResponseFixture, http.StatusOK)

	ctx := context.Background()
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	f5DeviceHost := NewMockF5Device(t)
	f5DeviceHost.On("GetHostname").Return("dummybigiphost")
	f5DeviceHost.EXPECT().EnsureVLAN(123, 0).Return(nil)
	f5DeviceHost.EXPECT().EnsureRouteDomain(123, conv.Pointer(666)).Return(nil)
	f5DeviceHost.EXPECT().EnsureBigIPSelfIP(
		"selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4",
		"42.42.42.42%123/8",
		123,
	).Return(nil)
	// PostAS3 must NOT be called — service is PENDING_DELETE.

	config.Global.Default.Host = "host-123"
	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	neutronClient.InitCache()
	a := &Agent{
		pool:    dbMock,
		neutron: &neutronClient,
		devices: []F5Device{f5DeviceHost},
		hosts:   []F5Device{},
		active:  f5DeviceHost,
	}

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT pg_try_advisory_xact_lock($1)").
		WithArgs(hostLockID(advisoryLockProcessEndpoints)).
		WillReturnRows(pgxmock.NewRows([]string{"pg_try_advisory_xact_lock"}).AddRow(true))
	dbMock.ExpectQuery("SELECT network, subnet FROM endpoint_port WHERE endpoint_id = $1").
		WithArgs(endpoint).
		WillReturnRows(pgxmock.NewRows([]string{"network", "subnet"}).AddRow(network, subnet.String()))
	dbMock.ExpectQuery("SELECT endpoint.*, service.ports AS service_ports, service.proxy_protocol, service.network_id AS service_network_id, service.status AS service_status, endpoint_port.segment_id, endpoint_port.port_id AS \"target.port\", endpoint_port.network AS \"target.network\", endpoint_port.subnet AS \"target.subnet\", endpoint_port.owned FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint.status NOT IN ($1,$2) AND network = $3 AND service.host = $4 AND service.provider = $5").
		WithArgs(models.EndpointStatusPENDINGAPPROVAL, models.EndpointStatusREJECTED, network, config.Global.Default.Host, models.ServiceProviderTenant).
		WillReturnRows(pgxmock.
			NewRows([]string{"id", "service_id", "status", "name", "service_ports", "proxy_protocol", "service_network_id", "service_status", "segment_id", "target.port", "target.network", "target.subnet"}).
			// PENDING_CREATE endpoint whose service is PENDING_DELETE
			AddRow(endpoint, service, models.EndpointStatusPENDINGCREATE, "test-service", []int32{80}, false, serviceNetwork, string(models.ServiceStatusPENDINGDELETE), nil, &port, &network, &subnet))
	dbMock.ExpectExec("UPDATE endpoint_port SET segment_id = $1 WHERE endpoint_id = $2").
		WithArgs(123, endpoint).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	dbMock.ExpectExec("SELECT 1 FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint_port.subnet = $1 AND service.host = $2 AND service.provider = $3 AND endpoint.status NOT IN ($4,$5)").
		WithArgs(subnet.String(), config.Global.Default.Host, models.ServiceProviderTenant, models.EndpointStatusPENDINGDELETE, models.EndpointStatusPENDINGREJECTED).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
	// Write transaction: endpoint transitions to AVAILABLE (postAS3=false, writes still run).
	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE endpoint SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3").
		WithArgs(models.EndpointStatusAVAILABLE, endpoint, models.EndpointStatusPENDINGCREATE).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	dbMock.ExpectCommit()
	dbMock.ExpectRollback() // write tx defer (no-op after commit)
	dbMock.ExpectRollback() // advisory-lock tx defer

	if err = a.ProcessEndpoint(ctx, endpoint); err != nil {
		t.Errorf("Agent.ProcessEndpoint() error = %v", err)
	}
	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestAgent_TestEndpointRequiringApproval(t *testing.T) {
	endpoint := strfmt.UUID("95dbe813-62f9-47f1-90ba-09f2dadcaefa")
	network := strfmt.UUID("35a3ca82-62af-4e0a-9472-92331500fb3a")
	subnet := strfmt.UUID("e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e")
	config.Global.Default.Host = "host-123"

	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()
	agent := &Agent{pool: dbMock}

	dbMock.
		ExpectBegin()
	dbMock.ExpectQuery("SELECT pg_try_advisory_xact_lock($1)").
		WithArgs(hostLockID(advisoryLockProcessEndpoints)).
		WillReturnRows(pgxmock.NewRows([]string{"pg_try_advisory_xact_lock"}).AddRow(true))
	dbMock.ExpectQuery("SELECT network, subnet FROM endpoint_port WHERE endpoint_id = $1").
		WithArgs(endpoint).
		WillReturnRows(pgxmock.NewRows([]string{"network", "subnet"}).AddRow(network, subnet.String()))
	dbMock.ExpectQuery("SELECT endpoint.*, service.ports AS service_ports, service.proxy_protocol, service.network_id AS service_network_id, service.status AS service_status, endpoint_port.segment_id, endpoint_port.port_id AS \"target.port\", endpoint_port.network AS \"target.network\", endpoint_port.subnet AS \"target.subnet\", endpoint_port.owned FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint.status NOT IN ($1,$2) AND network = $3 AND service.host = $4 AND service.provider = $5").
		WithArgs(models.EndpointStatusPENDINGAPPROVAL, models.EndpointStatusREJECTED, network, config.Global.Default.Host, models.ServiceProviderTenant).
		WillReturnRows(pgxmock.
			NewRows([]string{"id", "service_id", "status", "name", "service_ports", "proxy_protocol", "service_network_id", "service_status", "segment_id", "target.port", "target.network", "target.subnet"}))
	dbMock.ExpectRollback() // advisory-lock tx defer (no endpoints to update, early return)

	if err = agent.ProcessEndpoint(context.Background(), endpoint); err != nil {
		t.Errorf("Agent.ProcessEndpoint() error = %v", err)
	}
	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// TestAgent_DeleteEndpointWithMissingNeutronPort verifies that ProcessEndpoint can delete
// an endpoint when the Neutron port has already been deleted. We use 2 endpoints in the same
// network: one PENDING_DELETE (with missing port) and one AVAILABLE (so L2 cleanup is skipped).
func TestAgent_DeleteEndpointWithMissingNeutronPort(t *testing.T) {
	endpoint1 := strfmt.UUID("11111111-1111-4111-8111-111111111111") // PENDING_DELETE, port missing
	endpoint2 := strfmt.UUID("95dbe813-62f9-47f1-90ba-09f2dadcaefa") // AVAILABLE, port exists
	port1 := strfmt.UUID("d0d0d0d0-d0d0-4d0d-8d0d-0d0d0d0d0d0d")     // missing from Neutron
	port2 := strfmt.UUID("c0c0c0c0-c0c0-4c0c-8c0c-0c0c0c0c0c0c")     // exists in Neutron
	network := strfmt.UUID("35a3ca82-62af-4e0a-9472-92331500fb3a")
	subnet := strfmt.UUID("e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e")
	service := strfmt.UUID("a0a0a0a0-a0a0-4a0a-8a0a-0a0a0a0a0a0a")
	serviceNetwork := strfmt.UUID("b0b0b0b0-b0b0-4b0b-8b0b-0b0b0b0b0b0b")

	fakeServer := th.SetupPersistentPortHTTP(t, 8935)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	config.Global.Agent.L4Profile = "/Common/cc_fastL4_noaging_profile"
	config.Global.Agent.TCPProfile = "/Common/cc_tcp_archer_profile"
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+network.String(), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+serviceNetwork.String(), "GET",
		"", GetServiceNetworkResponseFixture, http.StatusOK)
	// Only port2 (c0c0c0c0...) exists - port1 (d0d0d0d0...) is missing from Neutron
	fixture.SetupHandler(t, fakeServer, "/v2.0/ports", "GET", "", GetPortListResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+subnet.String(), "GET", "",
		GetSubnetResponseFixture, http.StatusOK)

	ctx := context.Background()
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	f5DeviceHost := NewMockF5Device(t)
	f5DeviceHost.On("GetHostname").Return("dummybigiphost")
	f5DeviceHost.EXPECT().EnsureVLAN(123, 0).Return(nil)
	f5DeviceHost.EXPECT().EnsureRouteDomain(123, conv.Pointer(666)).Return(nil)
	f5DeviceHost.EXPECT().EnsureBigIPSelfIP(
		"selfip-5a8ad669-4ffe-4133-b9f9-6de62cd654a4",
		"42.42.42.42%123/8",
		123,
	).Return(nil)
	// AS3 declaration will only include endpoint2 (endpoint1 is being deleted)
	f5DeviceHost.EXPECT().PostAS3(PostBigIPFixture, "net-35a3ca82-62af-4e0a-9472-92331500fb3a").Return(nil)

	config.Global.Default.Host = "host-123"
	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	neutronClient.InitCache()
	a := &Agent{
		pool:    dbMock,
		neutron: &neutronClient,
		devices: []F5Device{f5DeviceHost},
		hosts:   []F5Device{},
		active:  f5DeviceHost,
	}

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT pg_try_advisory_xact_lock($1)").
		WithArgs(hostLockID(advisoryLockProcessEndpoints)).
		WillReturnRows(pgxmock.NewRows([]string{"pg_try_advisory_xact_lock"}).AddRow(true))
	dbMock.ExpectQuery("SELECT network, subnet FROM endpoint_port WHERE endpoint_id = $1").
		WithArgs(endpoint1).
		WillReturnRows(pgxmock.NewRows([]string{"network", "subnet"}).AddRow(network, subnet.String()))
	// Return both endpoints: endpoint1 (PENDING_DELETE) and endpoint2 (AVAILABLE)
	dbMock.ExpectQuery("SELECT endpoint.*, service.ports AS service_ports, service.proxy_protocol, service.network_id AS service_network_id, service.status AS service_status, endpoint_port.segment_id, endpoint_port.port_id AS \"target.port\", endpoint_port.network AS \"target.network\", endpoint_port.subnet AS \"target.subnet\", endpoint_port.owned FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint.status NOT IN ($1,$2) AND network = $3 AND service.host = $4 AND service.provider = $5").
		WithArgs(models.EndpointStatusPENDINGAPPROVAL, models.EndpointStatusREJECTED, network, config.Global.Default.Host, models.ServiceProviderTenant).
		WillReturnRows(pgxmock.
			NewRows([]string{"id", "service_id", "status", "name", "service_ports", "proxy_protocol", "service_network_id", "service_status", "segment_id", "target.port", "target.network", "target.subnet"}).
			AddRow(endpoint1, service, models.EndpointStatusPENDINGDELETE, "test-service-1", []int32{80}, false, serviceNetwork, string(models.ServiceStatusAVAILABLE), conv.Pointer(123), &port1, &network, &subnet).
			AddRow(endpoint2, service, models.EndpointStatusAVAILABLE, "test-service-2", []int32{80}, false, serviceNetwork, string(models.ServiceStatusAVAILABLE), conv.Pointer(123), &port2, &network, &subnet))
	// endpoint2 is not PENDING_DELETE, so no L2/SelfIP cleanup checks needed
	dbMock.ExpectExec("SELECT 1 FROM endpoint INNER JOIN service ON endpoint.service_id = service.id JOIN endpoint_port ON endpoint_id = endpoint.id WHERE endpoint_port.subnet = $1 AND service.host = $2 AND service.provider = $3 AND endpoint.status NOT IN ($4,$5)").
		WithArgs(subnet.String(), config.Global.Default.Host, models.ServiceProviderTenant, models.EndpointStatusPENDINGDELETE, models.EndpointStatusPENDINGREJECTED).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
	// Short write transaction: delete endpoint1, update endpoint2 status
	dbMock.ExpectBegin()
	dbMock.ExpectExec("DELETE FROM endpoint WHERE id = $1 AND status = $2").
		WithArgs(endpoint1, models.EndpointStatusPENDINGDELETE).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	dbMock.ExpectExec("UPDATE endpoint SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3").
		WithArgs(models.EndpointStatusAVAILABLE, endpoint2, models.EndpointStatusAVAILABLE).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	dbMock.ExpectCommit()
	dbMock.ExpectRollback() // write tx defer (no-op after commit)
	dbMock.ExpectRollback() // advisory-lock tx defer

	if err = a.ProcessEndpoint(ctx, endpoint1); err != nil {
		t.Errorf("Agent.ProcessEndpoint() error = %v", err)
	}
	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
