// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package db

import "github.com/Masterminds/squirrel"

// Status-guarded writes for reconcile loops. An agent reads a row's status, does
// seconds of BigIP/Neutron I/O, then persists the transition. Guarding the final
// write on the status the run acted on makes a row the API changed meanwhile
// (e.g. a delete) a 0-row no-op that reconciles on the next notification, instead
// of clobbering it back to AVAILABLE.

// UpdateStatusGuarded builds an UPDATE that sets status/updated_at only while the
// row still holds fromStatus.
func UpdateStatusGuarded(table string, id any, fromStatus, toStatus any) (string, []any) {
	return Update(table).
		Set("status", toStatus).
		Set("updated_at", squirrel.Expr("NOW()")).
		Where("id = ?", id).
		Where("status = ?", fromStatus).
		MustSql()
}

// DeleteIfStatus builds a DELETE that removes the row only while it holds
// fromStatus (typically PENDING_DELETE).
func DeleteIfStatus(table string, id any, fromStatus any) (string, []any) {
	return Delete(table).
		Where("id = ?", id).
		Where("status = ?", fromStatus).
		MustSql()
}
