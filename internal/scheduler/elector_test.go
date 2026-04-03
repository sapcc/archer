// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostgresElector_IsLeader_NotStarted(t *testing.T) {
	// Create elector without calling Start()
	elector := &PostgresElector{}

	err := elector.IsLeader(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

func TestPostgresElector_Close_NilConn(t *testing.T) {
	// Close should not panic when conn is nil
	elector := &PostgresElector{}
	elector.Close() // Should not panic
}
