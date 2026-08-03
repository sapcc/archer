// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateStatusGuarded(t *testing.T) {
	sql, args := UpdateStatusGuarded("endpoint", "the-id", "PENDING_UPDATE", "AVAILABLE")
	assert.Equal(t,
		"UPDATE endpoint SET status = $1, updated_at = NOW() WHERE id = $2 AND status = $3",
		sql)
	assert.Equal(t, []any{"AVAILABLE", "the-id", "PENDING_UPDATE"}, args)
}

func TestDeleteIfStatus(t *testing.T) {
	sql, args := DeleteIfStatus("service", "the-id", "PENDING_DELETE")
	assert.Equal(t, "DELETE FROM service WHERE id = $1 AND status = $2", sql)
	assert.Equal(t, []any{"the-id", "PENDING_DELETE"}, args)
}
