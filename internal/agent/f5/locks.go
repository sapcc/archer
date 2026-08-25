// SPDX-FileCopyrightText: Copyright 2026 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"context"
	"errors"
	"hash/fnv"

	"github.com/jackc/pgx/v5"

	"github.com/sapcc/archer/v2/internal/config"
)

// errAdvisoryLockHeld is returned by tryAdvisoryLock when another run holds the
// lock; callers match it with errors.Is to skip the cycle (vs a real DB error).
var errAdvisoryLockHeld = errors.New("advisory lock already held by another run")

// Advisory lock IDs serializing the agent's own runs. They replace the former
// `FOR UPDATE OF service` / `FOR UPDATE OF endpoint` row locks, whose job was
// only to stop two concurrent runs racing on the shared AS3 tenant. Being
// independent of table rows, they do NOT block the API's row-level `FOR UPDATE`
// in the DELETE/PUT handlers — so the agent holding one across seconds of
// BigIP/Neutron I/O no longer starves API requests into a lock timeout (503).
//
// Lock IDs are derived per-host so agents managing different F5 hosts do not
// serialize each other (cf. scheduler.advisoryLockID = 8675309).
const (
	advisoryLockProcessServices  = 0
	advisoryLockProcessEndpoints = 1
)

// hostLockID returns a stable int64 advisory lock ID for the given host and
// lock slot (0 = ProcessServices, 1 = ProcessEndpoints).
func hostLockID(slot int64) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(config.Global.Default.Host))
	// XOR with slot so ProcessServices and ProcessEndpoints get distinct IDs.
	return int64(h.Sum64()^uint64(slot)) & 0x7fffffffffffffff // keep positive
}

// tryAdvisoryLock takes the given advisory lock without blocking, via a
// transaction-scoped `pg_try_advisory_xact_lock`. The returned lockTx MUST be
// held open for the whole run and rolled back when done — Postgres releases the
// xact-scoped lock then, so there is no separate unlock. lockTx holds no
// `service`/`endpoint` row locks; all real work must use a.pool or a separate
// write tx, never lockTx.
//
// Returns errAdvisoryLockHeld (match with errors.Is) when another run holds the
// lock; any other error is a genuine DB failure. lockTx is non-nil only when
// err is nil (the tx is rolled back before returning on any error).
func (a *Agent) tryAdvisoryLock(ctx context.Context, slot int64) (lockTx pgx.Tx, err error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	var acquired bool
	if err = tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", hostLockID(slot)).Scan(&acquired); err != nil {
		_ = tx.Rollback(ctx)
		return nil, err
	}

	if !acquired {
		_ = tx.Rollback(ctx)
		return nil, errAdvisoryLockHeld
	}

	return tx, nil
}
