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

package controller

import (
	"context"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-openapi/loads"
	fake "github.com/gophercloud/gophercloud/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pashagolub/pgxmock/v2"
	"github.com/sapcc/go-bits/osext"
	"github.com/stretchr/testify/suite"
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
	c *Controller
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(SuiteTest))
}

const CreatePortResponseFixture = `
{
    "port": {
        "status": "DOWN",
        "id": "65c0ee9f-d634-4522-8954-51021b570b0d",
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
        "id": "d714f65e-bffd-494f-8219-8eb0a85d7a2d]",
        "subnets": ["a0304c3a-4f08-4c43-88af-d796509c97d2"],
		"project_id": "test-project-1"
    }
}
`

type mockedController struct {
	*Controller
	db pgxmock.PgxPoolIface
}

func (c *mockedController) Close() {
	c.db.Close()
}

func (t *SuiteTest) GetMockedController() *mockedController {
	// need to load from file due to cyclic dependency of restapi package
	spec, err := loads.Spec(path.Join(rootpath, "swagger.yaml"))
	if err != nil {
		t.FailNow("Failed loading swagger spec - ensure running test from source root", err)
	}

	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.FailNow(err.Error())
	}

	c := NewController(dbMock, spec, &neutron.NeutronClient{ServiceClient: fake.ServiceClient()})
	return &mockedController{c, dbMock}
}

func (t *SuiteTest) ResetHttpServer() {
	th.TeardownHTTP()
	th.SetupPersistentPortHTTP(t.T(), 8931)
}

// Setup db value
func (t *SuiteTest) SetupSuite() {
	config.Global.Database.Connection = osext.GetenvOrDefault(
		"DB_URL", "postgresql://localhost/test_suite_controller")
	pool, err := pgxpool.New(context.Background(), config.Global.Database.Connection)
	if err != nil {
		t.FailNow("Failed connecting to Database", err)
	}

	// Use it globally
	config.Global.ApiSettings.PaginationMaxLimit = 1000
	config.Global.ApiSettings.AuthStrategy = "none"
	policy.SetPolicyEngine("noop")

	// need to load from file due to cyclic dependency of restapi package
	spec, err := loads.Spec(path.Join(rootpath, "swagger.yaml"))
	if err != nil {
		t.FailNow("Failed loading swagger spec - ensure running test from source root", err)
	}

	th.SetupPersistentPortHTTP(t.T(), 8931)
	t.c = NewController(pool, spec, &neutron.NeutronClient{ServiceClient: fake.ServiceClient()})

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

	th.TeardownHTTP()
}

// Run After a Test
func (t *SuiteTest) AfterTest(suiteName, testName string) {
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
}
