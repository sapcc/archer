// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package notifier

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryPendingEndpoints(t *testing.T) {
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mock.Close()

	epRows := pgxmock.NewRows([]string{"id", "name", "project_id", "service_id", "created_at", "status"}).
		AddRow("ep-1", "", "consumer-1", "svc-id-a", time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC), "PENDING_APPROVAL").
		AddRow("ep-2", "", "consumer-2", "svc-id-a", time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC), "PENDING_APPROVAL").
		AddRow("ep-3", "", "consumer-3", "svc-id-b", time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC), "PENDING_APPROVAL")

	svcRows := pgxmock.NewRows([]string{"id", "name", "project_id"}).
		AddRow("svc-id-a", "svc-a", "owner-proj-1").
		AddRow("svc-id-b", "svc-b", "owner-proj-2")

	mock.ExpectQuery("SELECT.*FROM endpoint").
		WithArgs("PENDING_APPROVAL").
		WillReturnRows(epRows)
	mock.ExpectQuery("SELECT.*FROM service").
		WithArgs("svc-id-a", "svc-id-b").
		WillReturnRows(svcRows)

	results, err := QueryPendingEndpoints(context.Background(), mock)
	require.NoError(t, err)

	assert.Len(t, results, 2)
	assert.Equal(t, "owner-proj-1", results[0].ProjectID)
	assert.Len(t, results[0].Services, 1)
	assert.Len(t, results[0].Services[0].Endpoints, 2)
	assert.Equal(t, "owner-proj-2", results[1].ProjectID)
}

func TestShouldSkipDigest_TooYoung(t *testing.T) {
	recent := time.Now().Add(-12 * time.Hour)
	assert.True(t, shouldSkipDigest(recent))
}

func TestShouldSkipDigest_OldEnough(t *testing.T) {
	old := time.Now().Add(-48 * time.Hour)
	assert.False(t, shouldSkipDigest(old))
}
