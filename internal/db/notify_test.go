// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
)

func TestNotifyService(t *testing.T) {
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	dbMock.ExpectExec("SELECT pg_notify('service', $1)").
		WithArgs("test-host").
		WillReturnResult(pgxmock.NewResult("SELECT", 1))

	NotifyService(dbMock, "test-host")

	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestNotifyEndpoint(t *testing.T) {
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	endpointID := strfmt.UUID("12345678-1234-1234-1234-123456789abc")

	dbMock.ExpectExec("SELECT pg_notify('endpoint', $1)").
		WithArgs("test-host:12345678-1234-1234-1234-123456789abc").
		WillReturnResult(pgxmock.NewResult("SELECT", 1))

	NotifyEndpoint(dbMock, "test-host", endpointID)

	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestNotifyServiceWithError(t *testing.T) {
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	dbMock.ExpectExec("SELECT pg_notify('service', $1)").
		WithArgs("error-host").
		WillReturnError(assert.AnError)

	// Should not panic, just log the error
	NotifyService(dbMock, "error-host")

	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestNotifyEndpointWithError(t *testing.T) {
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	endpointID := strfmt.UUID("12345678-1234-1234-1234-123456789abc")

	dbMock.ExpectExec("SELECT pg_notify('endpoint', $1)").
		WithArgs("error-host:12345678-1234-1234-1234-123456789abc").
		WillReturnError(assert.AnError)

	// Should not panic, just log the error
	NotifyEndpoint(dbMock, "error-host", endpointID)

	if err = dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
