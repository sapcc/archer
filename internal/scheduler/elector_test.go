// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresElector_IsLeader_Acquired(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT pg_try_advisory_lock").
		WithArgs(advisoryLockID).
		WillReturnRows(pgxmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))

	elector := NewPostgresElector(mock)
	err = elector.IsLeader(context.Background())

	assert.NoError(t, err)
	assert.True(t, elector.isLeader)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresElector_IsLeader_NotAcquired(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectQuery("SELECT pg_try_advisory_lock").
		WithArgs(advisoryLockID).
		WillReturnRows(pgxmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(false))

	elector := NewPostgresElector(mock)
	err = elector.IsLeader(context.Background())

	assert.ErrorIs(t, err, ErrNotLeader)
	assert.False(t, elector.isLeader)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresElector_IsLeader_TransitionToLeader(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// First call: not leader
	mock.ExpectQuery("SELECT pg_try_advisory_lock").
		WithArgs(advisoryLockID).
		WillReturnRows(pgxmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(false))

	// Second call: becomes leader
	mock.ExpectQuery("SELECT pg_try_advisory_lock").
		WithArgs(advisoryLockID).
		WillReturnRows(pgxmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))

	elector := NewPostgresElector(mock)

	// First check: not leader
	err = elector.IsLeader(context.Background())
	assert.ErrorIs(t, err, ErrNotLeader)
	assert.False(t, elector.isLeader)

	// Second check: becomes leader
	err = elector.IsLeader(context.Background())
	assert.NoError(t, err)
	assert.True(t, elector.isLeader)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresElector_IsLeader_TransitionFromLeader(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// First call: is leader
	mock.ExpectQuery("SELECT pg_try_advisory_lock").
		WithArgs(advisoryLockID).
		WillReturnRows(pgxmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))

	// Second call: loses leadership
	mock.ExpectQuery("SELECT pg_try_advisory_lock").
		WithArgs(advisoryLockID).
		WillReturnRows(pgxmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(false))

	elector := NewPostgresElector(mock)

	// First check: is leader
	err = elector.IsLeader(context.Background())
	assert.NoError(t, err)
	assert.True(t, elector.isLeader)

	// Second check: loses leadership
	err = elector.IsLeader(context.Background())
	assert.ErrorIs(t, err, ErrNotLeader)
	assert.False(t, elector.isLeader)

	assert.NoError(t, mock.ExpectationsWereMet())
}
