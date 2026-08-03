// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag/conv"
	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/gophercloud/gophercloud/v2/testhelper/fixture"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/mock"

	agentpkg "github.com/sapcc/archer/v2/internal/agent"
	"github.com/sapcc/archer/v2/internal/agent/f5/as3"
	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/internal/neutron"
	"github.com/sapcc/archer/v2/models"
)

var PostAs3BigipFixture = &as3.AS3{
	Persist: false,
	Class:   "AS3",
	Action:  "deploy", Declaration: as3.ADC{
		Class:         "ADC",
		SchemaVersion: "3.36.0",
		UpdateMode:    "selective",
		Id:            "urn:uuid:07649173-4AF7-48DF-963F-84000C70F0DD",
		Tenants: map[string]as3.Tenant{
			"Common": {
				Class: "Tenant",
				Applications: map[string]as3.Application{
					"Shared": {
						Class:    "Application",
						Template: "shared",
						Services: map[string]any{},
					},
				},
			},
		},
	},
}

func TestProcessServicesWithDeletedNetwork(t *testing.T) {
	network := strfmt.UUID("3cf2f3fb-7527-45aa-accc-6880e783e5c8")
	service := strfmt.UUID("2975c302-4a0d-47ab-82df-42e7597ae41f")

	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+network.String(), "GET",
		"", GetNetworkResponseFixture, http.StatusNotFound)

	ctx := context.Background()
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	f5DeviceHost := NewMockF5Device(t)
	// GetHostname is no longer called from getExtendedService (the SelfIP-as-
	// SNAT branch was removed); the early-return path here doesn't reach any
	// other caller.

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

	// Advisory-lock transaction: acquired successfully.
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT pg_try_advisory_xact_lock($1)").
		WithArgs(advisoryLockProcessServices).
		WillReturnRows(dbMock.NewRows([]string{"pg_try_advisory_xact_lock"}).AddRow(true))
	// Services are read without FOR UPDATE now.
	dbMock.ExpectQuery("SELECT * FROM service WHERE host = $1 AND provider = $2").
		WithArgs("host-123", models.ServiceProviderTenant).
		WillReturnRows(dbMock.NewRows([]string{"id", "network_id", "status"}).AddRow(service, &network, models.ServiceStatusPENDINGDELETE))
	f5DeviceHost.EXPECT().
		PostAS3(PostAs3BigipFixture, "Common").
		Return(nil)
	// Short write transaction: delete the pending-delete service.
	dbMock.ExpectBegin()
	dbMock.ExpectExec("DELETE FROM service WHERE id = $1 AND status = 'PENDING_DELETE';").
		WithArgs(service).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	dbMock.ExpectCommit()
	// beginFuncExec does always a rollback at the end; the write tx rollback is a
	// no-op after commit, followed by the advisory-lock tx rollback (releases the
	// xact-scoped advisory lock).
	dbMock.ExpectRollback()
	dbMock.ExpectRollback()

	if err := a.ProcessServices(ctx); err != nil {
		t.Errorf("Agent.ProcessServices() error = %v", err)
	}
	if err := dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// TestProcessServicesRepostsCommonDespiteServiceError verifies that an
// unexpected getExtendedService error on one service no longer short-circuits
// the whole run: the Common tenant is still re-posted (so services that have
// left this host, e.g. after a migration, are pruned by absence), and the
// unexpected error is surfaced as the return value so the run is retried.
func TestProcessServicesRepostsCommonDespiteServiceError(t *testing.T) {
	deletingNet := strfmt.UUID("3cf2f3fb-7527-45aa-accc-6880e783e5c8")
	deletingSvc := strfmt.UUID("2975c302-4a0d-47ab-82df-42e7597ae41f")
	brokenNet := strfmt.UUID("11111111-1111-4111-8111-111111111111")
	brokenSvc := strfmt.UUID("22222222-2222-4222-8222-222222222222")

	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	// The pending-delete service tolerates a 404 network.
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+deletingNet.String(), "GET",
		"", "", http.StatusNotFound)
	// The other service hits an unexpected 500 -> deferred error.
	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+brokenNet.String(), "GET",
		"", "", http.StatusInternalServerError)

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

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT pg_try_advisory_xact_lock($1)").
		WithArgs(advisoryLockProcessServices).
		WillReturnRows(dbMock.NewRows([]string{"pg_try_advisory_xact_lock"}).AddRow(true))
	dbMock.ExpectQuery("SELECT * FROM service WHERE host = $1 AND provider = $2").
		WithArgs("host-123", models.ServiceProviderTenant).
		WillReturnRows(dbMock.NewRows([]string{"id", "network_id", "status"}).
			AddRow(deletingSvc, &deletingNet, models.ServiceStatusPENDINGDELETE).
			AddRow(brokenSvc, &brokenNet, models.ServiceStatusPENDINGCREATE))

	// The Common re-post must still happen even though brokenSvc errored. We
	// don't pin the exact declaration here (the deleting service contributes
	// nothing and the broken one is absent) - only that Common is posted.
	var posted bool
	f5DeviceHost.EXPECT().
		PostAS3(mock.MatchedBy(func(d *as3.AS3) bool {
			adc, ok := d.Declaration.(as3.ADC)
			return ok && len(adc.Tenants) == 1
		}), "Common").
		Run(func(*as3.AS3, string) { posted = true }).
		Return(nil)

	// Statuses are now persisted before the deferred error is returned: the
	// pending-delete service (tolerated 404 network) is deleted in a short tx.
	dbMock.ExpectBegin()
	dbMock.ExpectExec("DELETE FROM service WHERE id = $1 AND status = 'PENDING_DELETE';").
		WithArgs(deletingSvc).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	dbMock.ExpectCommit()
	dbMock.ExpectRollback() // persist tx defer (no-op after commit)
	dbMock.ExpectRollback() // advisory-lock tx defer

	if err := a.ProcessServices(ctx); err == nil {
		t.Error("Agent.ProcessServices() expected an error from the broken service, got nil")
	}
	if !posted {
		t.Error("expected Common to be re-posted despite the service error")
	}
	if err := dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// TestProcessServicesSkipsWhenLockHeld verifies that when the advisory lock is
// already held by another run, ProcessServices returns without reading or
// mutating any service rows — i.e. it never takes a `service` row lock that
// could block the API's DELETE/PUT handlers.
func TestProcessServicesSkipsWhenLockHeld(t *testing.T) {
	ctx := context.Background()
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	config.Global.Default.Host = "host-locked"
	a := &Agent{pool: dbMock}

	// Advisory-lock transaction: not acquired -> rolled back, no further queries.
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT pg_try_advisory_xact_lock($1)").
		WithArgs(advisoryLockProcessServices).
		WillReturnRows(dbMock.NewRows([]string{"pg_try_advisory_xact_lock"}).AddRow(false))
	dbMock.ExpectRollback()

	// A held lock returns ErrProcessBusy so the caller reschedules promptly; no rows are read or mutated.
	if err := a.ProcessServices(ctx); !errors.Is(err, agentpkg.ErrProcessBusy) {
		t.Errorf("Agent.ProcessServices() error = %v, want ErrProcessBusy", err)
	}
	if err := dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// TestGetExtendedServiceAllocatesSnatPortsByDefault verifies that
// getExtendedService routes every service through the dedicated SNAT pool
// path (network:f5snat ports) and never reuses SelfIPs as SNAT. The previous
// branch on SnatPoolSize == nil is gone — the field is now mandatory at the
// model layer.
func TestGetExtendedServiceAllocatesSnatPortsByDefault(t *testing.T) {
	const networkID = "35a3ca82-62af-4e0a-9472-92331500fb3a"
	const subnetID = "e0e0e0e0-e0e0-4e0e-8e0e-0e0e0e0e0e0e"
	serviceID := strfmt.UUID("2975c302-4a0d-47ab-82df-42e7597ae41f")

	fakeServer := th.SetupPersistentPortHTTP(t, 8931)
	defer fakeServer.Teardown()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	config.Global.Default.Host = "host-snat"

	fixture.SetupHandler(t, fakeServer, "/v2.0/networks/"+networkID, "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t, fakeServer, "/v2.0/subnets/"+subnetID, "GET",
		"", GetSubnetResponseFixture, http.StatusOK)

	// Track which Neutron port flows are exercised. Listing on f5snat is
	// expected (EnsureServiceSnatPorts); listing on f5selfip would mean the
	// removed legacy branch is still being taken.
	var (
		snatList    int
		selfipList  int
		createCalls int
	)
	fakeServer.Mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			q := r.URL.Query()
			switch q.Get("device_owner") {
			case "network:f5snat":
				snatList++
			case "network:f5selfip":
				selfipList++
			}
			_, _ = w.Write([]byte(`{"ports": []}`))
		case http.MethodPost:
			createCalls++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"port": {
				"id": "p0p0p0p0-p0p0-4p0p-8p0p-0p0p0p0p0p0p",
				"name": "snat-` + serviceID.String() + `-0",
				"device_owner": "network:f5snat",
				"device_id": "` + serviceID.String() + `",
				"network_id": "` + networkID + `",
				"fixed_ips": [{"subnet_id": "` + subnetID + `", "ip_address": "192.0.0.10"}]
			}}`))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient(fakeServer)}
	neutronClient.InitCache()
	a := &Agent{neutron: &neutronClient}

	netUUID := strfmt.UUID(networkID)
	svc := &models.Service{
		ID:           serviceID,
		NetworkID:    &netUUID,
		IPAddresses:  []models.InetAddress{"192.0.0.5/32"},
		Status:       models.ServiceStatusPENDINGCREATE,
		SnatPoolSize: conv.Pointer(int32(1)),
	}

	got, err := a.getExtendedService(context.Background(), svc)
	if err != nil {
		t.Fatalf("getExtendedService() error = %v", err)
	}
	if got.SubnetID != subnetID {
		t.Errorf("expected SubnetID %s, got %s", subnetID, got.SubnetID)
	}
	if len(got.NeutronPorts) != 1 {
		t.Fatalf("expected 1 SNAT port, got %d", len(got.NeutronPorts))
	}
	if snatList == 0 {
		t.Error("expected at least one list of network:f5snat ports")
	}
	if selfipList != 0 {
		t.Errorf("expected zero lists of network:f5selfip from getExtendedService SNAT path, got %d", selfipList)
	}
	if createCalls != 1 {
		t.Errorf("expected one Neutron port to be created, got %d", createCalls)
	}
}
