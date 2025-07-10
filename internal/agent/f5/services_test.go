// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-openapi/strfmt"
	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/gophercloud/gophercloud/v2/testhelper/fixture"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/neutron"
	"github.com/sapcc/archer/models"
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

	th.SetupPersistentPortHTTP(t, 8931)
	defer th.TeardownHTTP()
	config.Global.Agent.PhysicalNetwork = "physnet1"
	fixture.SetupHandler(t, "/v2.0/networks/"+network.String(), "GET",
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
	f5DeviceHost.On("GetHostname").Return("dummybigiphost")

	config.Global.Default.Host = "host-123"
	neutronClient := neutron.NeutronClient{ServiceClient: fake.ServiceClient()}
	neutronClient.InitCache()
	a := &Agent{
		pool:    dbMock,
		neutron: &neutronClient,
		devices: []F5Device{f5DeviceHost},
		hosts:   []F5Device{},
		active:  f5DeviceHost,
	}

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT * FROM service WHERE host = $1 AND provider = $2 FOR UPDATE OF service").
		WithArgs("host-123", models.ServiceProviderTenant).
		WillReturnRows(dbMock.NewRows([]string{"id", "network_id", "status"}).AddRow(service, &network, models.ServiceStatusPENDINGDELETE))
	f5DeviceHost.EXPECT().
		PostAS3(PostAs3BigipFixture, "Common").
		Return(nil)
	// delete service
	dbMock.ExpectExec("DELETE FROM service WHERE id = $1 AND status = 'PENDING_DELETE';").
		WithArgs(service).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	dbMock.ExpectCommit()
	// beginFuncExec does always a rollback at the end
	dbMock.ExpectRollback()

	if err := a.ProcessServices(ctx); err != nil {
		t.Errorf("Agent.ProcessServices() error = %v", err)
	}
	if err := dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
