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
	"github.com/sapcc/archer/internal/policy"
	"testing"

	"github.com/go-openapi/loads"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sapcc/go-bits/osext"
	"github.com/stretchr/testify/suite"
	"github.com/z0ne-dev/mgx/v2"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db/migrations"
)

type SuiteTest struct {
	suite.Suite
	c *Controller
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(SuiteTest))
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

	// need to load from file due to cyclic dependency of restapi
	spec, err := loads.Spec("swagger.yaml")
	if err != nil {
		t.FailNow("Failed loading swagger spec - ensure running test from source root", err)
	}

	// initialize controller
	t.c = NewController(pool, spec)

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
}

// Run After a Test
func (t *SuiteTest) AfterTest(suiteName, testName string) {
	// clear
	sql := `
		DELETE FROM rbac;
		DELETE FROM endpoint;
		DELETE FROM service;
	`

	if _, err := t.c.pool.Exec(context.Background(), sql); err != nil {
		t.FailNow("Failed cleanup", err)
	}
}
