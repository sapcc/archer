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

// ErrNotLeader is returned when the elector determines this instance is not the leader.
var ErrNotLeader = errors.New("not the leader")

// PostgresElector implements gocron.Elector using PostgreSQL advisory locks.
// Only one instance across all API servers can hold the lock at a time.
// It holds a dedicated connection to ensure advisory lock ownership is maintained
// across calls (advisory locks are session-scoped in PostgreSQL).
type PostgresElector struct {
	pool     db.PgxIface
	conn     *pgxpool.Conn
	lockID   int64
	name     string
	isLeader bool
}

// NewPostgresElector creates a new PostgreSQL-based elector with the specified lock ID.
// Call Start() to acquire the dedicated connection before using IsLeader().
func NewPostgresElector(pool db.PgxIface, lockID int64, name string) *PostgresElector {
	return &PostgresElector{
		pool:   pool,
		lockID: lockID,
		name:   name,
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
	log.WithField("scheduler", e.name).Info("Acquired dedicated connection for leader election")
	return nil
}

// Close releases the dedicated connection and any held advisory locks.
func (e *PostgresElector) Close() {
	if e.conn != nil {
		// Explicitly release the advisory lock before releasing the connection.
		// Skip if the underlying conn is already dead — Postgres has released the
		// session-scoped lock for us and the Exec would just log a spurious error.
		if e.isLeader && !e.conn.Conn().IsClosed() {
			_, err := e.conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", e.lockID)
			if err != nil {
				log.WithError(err).Error("Failed to release advisory lock")
			}
		}
		e.isLeader = false
		e.conn.Release()
		e.conn = nil
		log.WithField("scheduler", e.name).Info("Released dedicated connection")
	}
}

// reacquireConn replaces a dead dedicated connection with a fresh one from the pool.
// Any advisory lock previously held on the old session is gone (Postgres releases
// session-scoped locks when the session dies), so isLeader is reset to false.
func (e *PostgresElector) reacquireConn(ctx context.Context) error {
	if e.conn != nil {
		e.conn.Release()
		e.conn = nil
	}
	e.isLeader = false

	conn, err := e.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	e.conn = conn
	log.WithField("scheduler", e.name).Warn("Reacquired dedicated connection after previous one closed")
	return nil
}

// IsLeader implements gocron.Elector.
// Returns nil if this instance should run jobs (is the leader),
// or an error if it should not (not the leader).
func (e *PostgresElector) IsLeader(ctx context.Context) error {
	if e.conn == nil {
		log.WithField("scheduler", e.name).Error("Dedicated connection not initialized, call Start() first")
		return errors.New("elector not started")
	}

	// If the dedicated conn died (Postgres restart, idle timeout, proxy cycling),
	// pgx surfaces this as "conn closed" / "failed to deallocate cached statement(s)".
	// Detect proactively and reacquire so leader election can recover without a process restart.
	if e.conn.Conn().IsClosed() {
		if err := e.reacquireConn(ctx); err != nil {
			log.WithError(err).Error("Failed to reacquire dedicated connection for leader election")
			return err
		}
	}

	// Try to acquire the advisory lock (non-blocking)
	var acquired bool
	err := e.conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", e.lockID).Scan(&acquired)
	if err != nil {
		// Conn may have died mid-query; try to recover once so the next tick succeeds.
		if e.conn.Conn().IsClosed() {
			if rerr := e.reacquireConn(ctx); rerr != nil {
				log.WithError(rerr).Error("Failed to reacquire dedicated connection for leader election")
				return rerr
			}
			log.WithError(err).Warn("Advisory lock query failed on dead conn; reacquired, will retry next tick")
			return ErrNotLeader
		}
		log.WithError(err).Error("Failed to check advisory lock for leader election")
		return err
	}

	if acquired {
		if !e.isLeader {
			log.WithField("scheduler", e.name).Info("This instance is now the leader")
			e.isLeader = true
		}
		return nil
	}

	if e.isLeader {
		log.WithField("scheduler", e.name).Info("This instance is no longer the leader")
		e.isLeader = false
	}
	return ErrNotLeader
}
