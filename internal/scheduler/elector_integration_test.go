// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sapcc/go-bits/osext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const testLockID = 8675309 // test lock ID

// TestPostgresElector_Integration tests the elector with a real PostgreSQL database
// to verify that advisory locks work correctly across multiple connections.
func TestPostgresElector_Integration(t *testing.T) {
	if osext.GetenvBool("CHECK_SKIPS_FUNCTIONAL_TEST") {
		t.Skip("Skipping integration test as CHECK_SKIPS_FUNCTIONAL_TEST is set")
	}

	ctx := context.Background()

	// Start postgres container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "Failed starting postgres container")
	defer func() {
		require.NoError(t, pgContainer.Terminate(ctx), "Failed terminating postgres container")
	}()

	connStr := pgContainer.MustConnectionString(ctx, "sslmode=disable")

	t.Run("single elector becomes leader", func(t *testing.T) {
		pool, err := pgxpool.New(ctx, connStr)
		require.NoError(t, err)
		defer pool.Close()

		elector := NewPostgresElector(pool, testLockID, "test")
		require.NoError(t, elector.Start(ctx))
		defer elector.Close()

		err = elector.IsLeader(ctx)
		assert.NoError(t, err, "Single elector should become leader")
		assert.True(t, elector.isLeader)
	})

	t.Run("second elector cannot become leader while first holds lock", func(t *testing.T) {
		pool1, err := pgxpool.New(ctx, connStr)
		require.NoError(t, err)
		defer pool1.Close()

		pool2, err := pgxpool.New(ctx, connStr)
		require.NoError(t, err)
		defer pool2.Close()

		// First elector acquires leadership
		elector1 := NewPostgresElector(pool1, testLockID, "test")
		require.NoError(t, elector1.Start(ctx))
		defer elector1.Close()

		err = elector1.IsLeader(ctx)
		require.NoError(t, err, "First elector should become leader")

		// Second elector cannot become leader
		elector2 := NewPostgresElector(pool2, testLockID, "test")
		require.NoError(t, elector2.Start(ctx))
		defer elector2.Close()

		err = elector2.IsLeader(ctx)
		assert.ErrorIs(t, err, ErrNotLeader, "Second elector should not become leader")
		assert.False(t, elector2.isLeader)
	})

	t.Run("second elector becomes leader after first closes", func(t *testing.T) {
		pool1, err := pgxpool.New(ctx, connStr)
		require.NoError(t, err)
		defer pool1.Close()

		pool2, err := pgxpool.New(ctx, connStr)
		require.NoError(t, err)
		defer pool2.Close()

		// First elector acquires leadership
		elector1 := NewPostgresElector(pool1, testLockID, "test")
		require.NoError(t, elector1.Start(ctx))

		err = elector1.IsLeader(ctx)
		require.NoError(t, err, "First elector should become leader")

		// Second elector cannot become leader yet
		elector2 := NewPostgresElector(pool2, testLockID, "test")
		require.NoError(t, elector2.Start(ctx))
		defer elector2.Close()

		err = elector2.IsLeader(ctx)
		assert.ErrorIs(t, err, ErrNotLeader)

		// Close first elector - releases the lock
		elector1.Close()

		// Now second elector should be able to become leader
		err = elector2.IsLeader(ctx)
		assert.NoError(t, err, "Second elector should become leader after first closes")
		assert.True(t, elector2.isLeader)
	})

	t.Run("same elector can reacquire lock", func(t *testing.T) {
		pool, err := pgxpool.New(ctx, connStr)
		require.NoError(t, err)
		defer pool.Close()

		elector := NewPostgresElector(pool, testLockID, "test")
		require.NoError(t, elector.Start(ctx))
		defer elector.Close()

		// First acquisition
		err = elector.IsLeader(ctx)
		assert.NoError(t, err)
		assert.True(t, elector.isLeader)

		// Second acquisition (should succeed - same session can reacquire)
		err = elector.IsLeader(ctx)
		assert.NoError(t, err)
		assert.True(t, elector.isLeader)
	})

	t.Run("concurrent electors only one leader", func(t *testing.T) {
		pool1, err := pgxpool.New(ctx, connStr)
		require.NoError(t, err)
		defer pool1.Close()

		pool2, err := pgxpool.New(ctx, connStr)
		require.NoError(t, err)
		defer pool2.Close()

		// Create and start both electors
		elector1 := NewPostgresElector(pool1, testLockID, "test")
		require.NoError(t, elector1.Start(ctx))
		defer elector1.Close()

		elector2 := NewPostgresElector(pool2, testLockID, "test")
		require.NoError(t, elector2.Start(ctx))
		defer elector2.Close()

		// Check both electors
		err1 := elector1.IsLeader(ctx)
		err2 := elector2.IsLeader(ctx)

		// Exactly one should be leader
		isLeader1 := err1 == nil
		isLeader2 := err2 == nil

		// XOR - exactly one should be true
		assert.True(t, isLeader1 != isLeader2, "Exactly one elector should be leader, got elector1=%v, elector2=%v", isLeader1, isLeader2)
	})

	t.Run("recovers when dedicated connection is killed", func(t *testing.T) {
		pool, err := pgxpool.New(ctx, connStr)
		require.NoError(t, err)
		defer pool.Close()

		elector := NewPostgresElector(pool, testLockID, "test")
		require.NoError(t, elector.Start(ctx))
		defer elector.Close()

		// Become leader and capture the backend pid of the dedicated session.
		require.NoError(t, elector.IsLeader(ctx))
		require.True(t, elector.isLeader)

		var pid int32
		require.NoError(t, elector.conn.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid))

		// Kill that backend from a separate connection. This simulates a Postgres
		// restart / proxy cycling / idle timeout that drops the dedicated conn.
		killConn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		_, err = killConn.Exec(ctx, "SELECT pg_terminate_backend($1)", pid)
		require.NoError(t, err)
		killConn.Release()

		// Next IsLeader call must recover (not return a permanent error).
		// Either path is acceptable: it reacquires and immediately becomes leader
		// (the lock is free again because the prior session died), or it bounces
		// once with ErrNotLeader and succeeds on the following tick.
		err = elector.IsLeader(ctx)
		if err != nil {
			assert.ErrorIs(t, err, ErrNotLeader, "post-kill error must be recoverable, got %v", err)
			// Subsequent tick must succeed.
			require.NoError(t, elector.IsLeader(ctx))
		}
		assert.True(t, elector.isLeader, "elector should be leader again after recovery")
	})

	t.Run("leadership stable across multiple checks", func(t *testing.T) {
		pool, err := pgxpool.New(ctx, connStr)
		require.NoError(t, err)
		defer pool.Close()

		elector := NewPostgresElector(pool, testLockID, "test")
		require.NoError(t, elector.Start(ctx))
		defer elector.Close()

		// Acquire leadership
		err = elector.IsLeader(ctx)
		require.NoError(t, err, "Should become leader")

		// Verify leadership is stable across multiple checks
		for i := 0; i < 10; i++ {
			err = elector.IsLeader(ctx)
			assert.NoError(t, err, "Leadership should be stable on check %d", i)
		}
	})
}
