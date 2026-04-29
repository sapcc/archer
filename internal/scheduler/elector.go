// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/db"
)

// advisoryLockID is a fixed lock ID for the scheduler leader election.
// This value should be unique across all advisory locks used by the application.
const advisoryLockID = 8675309 // arbitrary unique identifier

// ErrNotLeader is returned when the elector determines this instance is not the leader.
var ErrNotLeader = errors.New("not the leader")

// PostgresElector implements gocron.Elector using PostgreSQL advisory locks.
// Only one instance across all API servers can hold the lock at a time.
// It holds a dedicated connection to ensure advisory lock ownership is maintained
// across calls (advisory locks are session-scoped in PostgreSQL).
type PostgresElector struct {
	pool     db.PgxIface
	conn     *pgxpool.Conn
	isLeader bool
}

// NewPostgresElector creates a new PostgreSQL-based elector.
// Call Start() to acquire the dedicated connection before using IsLeader().
func NewPostgresElector(pool db.PgxIface) *PostgresElector {
	return &PostgresElector{
		pool: pool,
	}
}

// Start acquires a dedicated connection for advisory lock operations.
// This must be called before IsLeader() to ensure lock ownership is maintained.
func (e *PostgresElector) Start(ctx context.Context) error {
	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	e.conn = conn
	log.Info("PostgresElector: acquired dedicated connection for leader election")
	return nil
}

// Close releases the dedicated connection and any held advisory locks.
func (e *PostgresElector) Close() {
	if e.conn != nil {
		// Explicitly release the advisory lock before releasing the connection
		if e.isLeader {
			_, err := e.conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockID)
			if err != nil {
				log.WithError(err).Error("Failed to release advisory lock")
			}
			e.isLeader = false
		}
		e.conn.Release()
		e.conn = nil
		log.Info("PostgresElector: released dedicated connection")
	}
}

// IsLeader implements gocron.Elector.
// Returns nil if this instance should run jobs (is the leader),
// or an error if it should not (not the leader).
func (e *PostgresElector) IsLeader(ctx context.Context) error {
	if e.conn == nil {
		log.Error("PostgresElector: dedicated connection not initialized, call Start() first")
		return errors.New("elector not started")
	}

	// Try to acquire the advisory lock (non-blocking)
	var acquired bool
	err := e.conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", advisoryLockID).Scan(&acquired)
	if err != nil {
		log.WithError(err).Error("Failed to check advisory lock for leader election")
		return err
	}

	if acquired {
		if !e.isLeader {
			log.Info("This instance is now the scheduler leader")
			e.isLeader = true
		}
		return nil
	}

	if e.isLeader {
		log.Info("This instance is no longer the scheduler leader")
		e.isLeader = false
	}
	return ErrNotLeader
}
