// SPDX-FileCopyrightText: Copyright 2026 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	log "github.com/sirupsen/logrus"
)

// BeginWithLockTimeout starts a transaction and bounds how long any statement in
// it will wait for a row/table lock via `SET LOCAL lock_timeout`. Once the
// timeout elapses the blocked statement fails immediately with SQLSTATE 55P03
// (lock_not_available) instead of hanging until the request context deadline.
//
// lock_timeout should be chosen well below the HTTP request deadline so a
// contended FOR UPDATE surfaces as a clean, retryable error rather than an
// opaque context cancellation. SET LOCAL is scoped to the transaction and is
// reset automatically on commit or rollback.
func BeginWithLockTimeout(ctx context.Context, pool PgxIface, timeout time.Duration) (pgx.Tx, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Milliseconds keeps the value integral for any sub-second timeout.
	ms := timeout.Milliseconds()
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL lock_timeout = %d", ms)); err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("failed to set lock_timeout: %w", err)
	}
	return tx, nil
}

// IsLockTimeout reports whether err is a Postgres lock_not_available (55P03)
// error, i.e. a lock wait that exceeded lock_timeout.
func IsLockTimeout(err error) bool {
	pe, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pe.Code == pgerrcode.LockNotAvailable
}

// lockBlocker is one session holding a lock on the queried relation.
type lockBlocker struct {
	PID         int32       `db:"pid"`
	Application string      `db:"application_name"`
	State       string      `db:"state"`
	RunningForS float64     `db:"running_for_s"`
	WaitType    pgtype.Text `db:"wait_event_type"`
	Query       string      `db:"query"`
}

// LogLockBlockers dumps the sessions currently holding locks on the given
// relation to the log at error level. It is meant to be called from a handler's
// lock-timeout branch to capture *who* was blocking — the tuple/relation in the
// Postgres server log only identifies the blocked row, never the holder.
//
// It runs on a fresh pool connection (the transaction that hit the lock timeout
// is left in an aborted state and cannot be reused). Because every archer
// process sets a distinct application_name (archer-api, archer-f5-agent,
// archer-ni-agent), the dump names the culprit process and its running query
// directly. This is best-effort diagnostics: any failure is logged and swallowed
// so it never masks the original lock error.
func LogLockBlockers(ctx context.Context, pool PgxIface, relation string) {
	q, args := Select(
		"a.pid",
		"a.application_name",
		"a.state",
		"COALESCE(EXTRACT(EPOCH FROM (now() - a.query_start)), 0) AS running_for_s",
		"a.wait_event_type",
		"left(a.query, 500) AS query",
	).
		From("pg_stat_activity a").
		Join("pg_locks l ON l.pid = a.pid").
		Join("pg_class c ON c.oid = l.relation").
		Where("c.relname = ?", relation).
		Where("a.pid <> pg_backend_pid()").
		Where("a.state <> 'idle'").
		OrderBy("a.query_start").
		MustSql()

	var blockers []lockBlocker
	if err := pgxscan.Select(ctx, pool, &blockers, q, args...); err != nil {
		log.WithError(err).Warnf("LogLockBlockers: failed to query lock holders for %q", relation)
		return
	}

	if len(blockers) == 0 {
		log.WithField("relation", relation).
			Warn("lock timeout: no active lock holder found (holder may have already released or is idle-in-transaction)")
		return
	}

	for _, b := range blockers {
		log.WithFields(log.Fields{
			"relation":         relation,
			"blocker_pid":      b.PID,
			"application_name": b.Application,
			"state":            b.State,
			"running_for":      time.Duration(b.RunningForS * float64(time.Second)).Round(time.Millisecond).String(),
			"wait_event_type":  b.WaitType.String, // empty when NULL
			"blocker_query":    b.Query,
		}).Error("lock timeout: blocking session holding a lock on the relation")
	}
}
