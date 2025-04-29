// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"net/http"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
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
