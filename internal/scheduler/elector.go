// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"errors"

	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/db"
)

// advisoryLockID is a fixed lock ID for the scheduler leader election.
// This value should be unique across all advisory locks used by the application.
const advisoryLockID = 8675309 // arbitrary unique identifier

// ErrNotLeader is returned when the elector determines this instance is not the leader.
var ErrNotLeader = errors.New("not the leader")

// PostgresElector implements gocron.Elector using PostgreSQL advisory locks.
// Only one instance across all API servers can hold the lock at a time.
type PostgresElector struct {
	pool     db.PgxIface
	isLeader bool
}

// NewPostgresElector creates a new PostgreSQL-based elector.
func NewPostgresElector(pool db.PgxIface) *PostgresElector {
	return &PostgresElector{
		pool: pool,
	}
}

// IsLeader implements gocron.Elector.
// Returns nil if this instance should run jobs (is the leader),
// or an error if it should not (not the leader).
func (e *PostgresElector) IsLeader(ctx context.Context) error {
	// Try to acquire the advisory lock (non-blocking)
	var acquired bool
	err := e.pool.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", advisoryLockID).Scan(&acquired)
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
