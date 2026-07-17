// SPDX-FileCopyrightText: Copyright 2026 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func TestIsLockTimeout(t *testing.T) {
	assert.True(t, IsLockTimeout(&pgconn.PgError{Code: pgerrcode.LockNotAvailable}))
	assert.False(t, IsLockTimeout(&pgconn.PgError{Code: pgerrcode.UndefinedColumn}))
	assert.False(t, IsLockTimeout(assert.AnError))
	assert.False(t, IsLockTimeout(nil))
}

// logLockBlockersSQL is the exact statement LogLockBlockers emits. Kept as a
// constant so the test fails loudly if the query builder output changes.
const logLockBlockersSQL = "SELECT a.pid, a.application_name, a.state, " +
	"COALESCE(EXTRACT(EPOCH FROM (now() - a.query_start)), 0) AS running_for_s, " +
	"a.wait_event_type, left(a.query, 500) AS query " +
	"FROM pg_stat_activity a " +
	"JOIN pg_locks l ON l.pid = a.pid " +
	"JOIN pg_class c ON c.oid = l.relation " +
	"WHERE c.relname = $1 AND a.pid <> pg_backend_pid() AND a.state <> 'idle' " +
	"ORDER BY a.query_start"

func TestLogLockBlockersQuery(t *testing.T) {
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	// One blocking session found: exercise the scan + log path.
	rows := pgxmock.NewRows([]string{
		"pid", "application_name", "state", "running_for_s", "wait_event_type", "query",
	}).AddRow(
		int32(611643), "archer-f5-agent", "idle in transaction",
		float64(12.5), "Client", "SELECT ... FOR UPDATE OF service",
	)

	dbMock.ExpectQuery(logLockBlockersSQL).
		WithArgs("service").
		WillReturnRows(rows)

	hook := logtest.NewGlobal()
	defer hook.Reset()

	LogLockBlockers(context.Background(), dbMock, "service")

	assert.NoError(t, dbMock.ExpectationsWereMet())

	// The blocker row must be logged at error level with the holder details —
	// asserting this (not just "did not panic") is what catches a scan failure.
	entry := hook.LastEntry()
	if assert.NotNil(t, entry, "expected a log entry for the blocking session") {
		assert.Equal(t, logrus.ErrorLevel, entry.Level)
		assert.Equal(t, "lock timeout: blocking session holding a lock on the relation", entry.Message)
		assert.Equal(t, int32(611643), entry.Data["blocker_pid"])
		assert.Equal(t, "archer-f5-agent", entry.Data["application_name"])
		assert.Equal(t, "idle in transaction", entry.Data["state"])
		assert.Equal(t, "Client", entry.Data["wait_event_type"])
		assert.Equal(t, "12.5s", entry.Data["running_for"])
		assert.Equal(t, "SELECT ... FOR UPDATE OF service", entry.Data["blocker_query"])
	}
}

func TestLogLockBlockersNoRows(t *testing.T) {
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	empty := pgxmock.NewRows([]string{
		"pid", "application_name", "state", "running_for_s", "wait_event_type", "query",
	})
	dbMock.ExpectQuery(logLockBlockersSQL).
		WithArgs("service").
		WillReturnRows(empty)

	hook := logtest.NewGlobal()
	defer hook.Reset()

	// No holder found: must not panic, and must not log a (false) error-level
	// "blocking session" entry.
	LogLockBlockers(context.Background(), dbMock, "service")

	assert.NoError(t, dbMock.ExpectationsWereMet())
	entry := hook.LastEntry()
	if assert.NotNil(t, entry) {
		assert.Equal(t, logrus.WarnLevel, entry.Level)
	}
}

func TestLogLockBlockersQueryError(t *testing.T) {
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	dbMock.ExpectQuery(logLockBlockersSQL).
		WithArgs("service").
		WillReturnError(assert.AnError)

	// Diagnostics are best-effort: a query failure must be swallowed, not panic.
	assert.NotPanics(t, func() {
		LogLockBlockers(context.Background(), dbMock, "service")
	})

	assert.NoError(t, dbMock.ExpectationsWereMet())
}
