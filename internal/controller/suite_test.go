// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/loads"
	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/z0ne-dev/mgx/v2"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db/migrations"
	"github.com/sapcc/archer/internal/neutron"
	"github.com/sapcc/archer/internal/policy"
)

var (
	_, b, _, _ = runtime.Caller(0)
	rootpath   = filepath.Join(filepath.Dir(b), "../..")
)

type SuiteTest struct {
	suite.Suite
	c           *Controller
	fakeServer  th.FakeServer
	ctx         context.Context
	pgContainer *postgres.PostgresContainer
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(SuiteTest))
}

const CreatePortResponseFixture = `
{
    "port": {
        "status": "DOWN",
        "id": "%s",
		"network_id": "%s",
        "fixed_ips": [
            {
                "subnet_id": "a0304c3a-4f08-4c43-88af-d796509c97d2",
                "ip_address": "10.0.0.2"
            }
        ],
		"project_id": "test-project-1"
    }
}
`

const GetNetworkResponseFixture = `
{
    "network": {
        "id": "d714f65e-bffd-494f-8219-8eb0a85d7a2d",
        "subnets": ["a0304c3a-4f08-4c43-88af-d796509c97d2"],
		"project_id": "test-project-1",
		"segments": [
			{
				"provider:physical_network": "physnet1",
				"provider:network_type": "vlan",
				"provider:segmentation_id": 100
			}
		]
    }
}
`

const GetNetworkIpAvailabilityResponseFixture = `
{
	"network_ip_availability": {
		"network_id": "d714f65e-bffd-494f-8219-8eb0a85d7a2d",
		"network_name": "test-network-1",
		"tenant_id": "test-project-1",
		"total_ips": 256,
		"used_ips": 0
	}
}
`

const GetNetworkIpNoAvailabilityResponseFixture = `
{
	"network_ip_availability": {
		"network_id": "d714f65e-bffd-494f-8219-8eb0a85d7a2d",
		"network_name": "test-network-1",
		"tenant_id": "test-project-1",
		"total_ips": 0,
		"used_ips": 0
	}
}
`

type MockedController struct {
	*Controller
	db pgxmock.PgxPoolIface
}

func (c *MockedController) Close() {
	c.db.Close()
}

func (t *SuiteTest) GetMockedController() *MockedController {
	// need to load from file due to cyclic dependency of restapi package
	spec, err := loads.Spec(path.Join(rootpath, "swagger.yaml"))
	if err != nil {
		t.FailNow("Failed loading swagger spec - ensure running test from source root", err)
	}

	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.FailNow(err.Error())
	}

	c := NewController(dbMock, spec, &neutron.NeutronClient{ServiceClient: fake.ServiceClient(t.fakeServer)})
	return &MockedController{c, dbMock}
}

func (t *SuiteTest) ResetHttpServer() {
	t.fakeServer.Teardown()
	t.fakeServer = th.SetupPersistentPortHTTP(t.T(), 8931)
}

// Setup db value
func (t *SuiteTest) SetupSuite() {
	var ok bool
	if config.Global.Database.Connection, ok = os.LookupEnv("DB_URL"); !ok {
		t.ctx = context.Background()
		var err error
		if t.pgContainer, err = postgres.Run(t.ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("test-db"),
			postgres.WithUsername("postgres"),
			postgres.WithPassword("postgres"),
			postgres.BasicWaitStrategies(),
		); err != nil {
			t.FailNow("Failed starting postgres container", err)
		}

		if config.Global.Database.Connection, err = t.pgContainer.ConnectionString(t.ctx, "sslmode=disable"); err != nil {
			t.FailNow("Failed getting connection string", err)
		}
	}

	pool, err := pgxpool.New(context.Background(), config.Global.Database.Connection)
	if err != nil {
		t.FailNow("Failed connecting to Database", err)
	}

	// Use it globally
	config.Global.ApiSettings.PaginationMaxLimit = 1000
	config.Global.ApiSettings.AuthStrategy = "none"
	config.Global.Agent.PhysicalNetwork = "physnet1"
	policy.SetPolicyEngine("noop")

	// need to load from file due to cyclic dependency of restapi package
	spec, err := loads.Spec(path.Join(rootpath, "swagger.yaml"))
	if err != nil {
		t.FailNow("Failed loading swagger spec - ensure running test from source root", err)
	}

	t.fakeServer = th.SetupPersistentPortHTTP(t.T(), 8931)
	t.c = NewController(pool, spec, &neutron.NeutronClient{ServiceClient: fake.ServiceClient(t.fakeServer)})
	t.c.neutron.InitCache()

	// Run migration
	migrator, err := mgx.New(migrations.Migrations)
	if err != nil {
		t.FailNow("Failed migration", err)
	}

	if err := migrator.Migrate(context.Background(), t.c.pool); err != nil {
		t.FailNow("Failed migration", err)
	}
}

// Run After All Test Done
func (t *SuiteTest) TearDownSuite() {
	// clear
	sql := `
		DROP SCHEMA public CASCADE;
		CREATE SCHEMA public;
	`

	if _, err := t.c.pool.Exec(context.Background(), sql); err != nil {
		t.FailNow("Failed cleanup", err)
	}

	if t.pgContainer != nil {
		if err := t.pgContainer.Terminate(t.ctx); err != nil {
			t.FailNow("Failed terminating postgres container", err)
		}
	}

	t.fakeServer.Teardown()
}

// Run After a Test
func (t *SuiteTest) AfterTest(_, _ string) {
	// clear
	sql := `
		DELETE FROM rbac;
		DELETE FROM endpoint;
		DELETE FROM service;
		DELETE FROM quota;
		DELETE FROM agents;
	`

	if _, err := t.c.pool.Exec(context.Background(), sql); err != nil {
		t.FailNow("Failed cleanup", err)
	}

	t.ResetHttpServer()
	t.c.neutron.ResetCache()
}
