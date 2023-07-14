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

package db

import (
	"context"
	"net/http"
	"testing"

	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/errors"
)

func TestCheckQuotaMet(t *testing.T) {
	config.Global.Quota.Enabled = true
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
		config.Global.Quota.Enabled = false
	}()

	dbMock.
		ExpectExec("INSERT INTO quota (project_id,service,endpoint) VALUES ($1,$2,$3) ON CONFLICT (project_id) DO NOTHING").
		WithArgs("", int64(0), int64(0)).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	dbMock.ExpectQuery("SELECT service, (SELECT COUNT(id) FROM service WHERE project_id = quota.project_id) AS use FROM quota WHERE project_id = $1").
		WithArgs("").
		WillReturnRows(pgxmock.NewRows([]string{"service", "use"}).AddRow(1, 1))

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/example", nil)

	err = CheckQuota(dbMock, r, "service")

	assert.Equal(t, errors.ErrQuotaExceeded, err)
	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCheckQuota(t *testing.T) {
	config.Global.Quota.Enabled = true
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
		config.Global.Quota.Enabled = false
	}()

	dbMock.
		ExpectExec("INSERT INTO quota (project_id,service,endpoint) VALUES ($1,$2,$3) ON CONFLICT (project_id) DO NOTHING").
		WithArgs("", int64(0), int64(0)).
		WillReturnResult(pgxmock.NewResult("INSERT", 0))
	dbMock.ExpectQuery("SELECT service, (SELECT COUNT(id) FROM service WHERE project_id = quota.project_id) AS use FROM quota WHERE project_id = $1").
		WithArgs("").
		WillReturnRows(pgxmock.NewRows([]string{"service", "use"}).AddRow(1, 0))

	r, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/example", nil)

	assert.Nil(t, CheckQuota(dbMock, r, "service"))
	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
