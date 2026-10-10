// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ni

import (
	"context"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/archer/v2/internal/agent/ni/haproxy"
	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/models"
)

// svcID and epID are stable UUIDs used across health scrape tests.
var (
	healthSvcID = strfmt.UUID("aaaaaaaa-0000-0000-0000-000000000001")
	healthEpID  = strfmt.UUID("bbbbbbbb-0000-0000-0000-000000000001")
)

// setupHealthMock creates a pgxmock pool and an Agent wired up for HealthScrapeLoop tests.
func setupHealthMock(t *testing.T) (pgxmock.PgxPoolIface, *Agent) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { mock.Close() })

	config.Global.Default.Host = "test-host"

	a := &Agent{
		pool:    mock,
		haproxy: haproxy.NewFakeHaproxy(),
	}
	return mock, a
}

// expectLeftJoinQuery sets up the single LEFT JOIN SELECT expectation.
// rows is the list of (service_id, endpoint_id) pairs; pass nil endpoint_id for services with no endpoints.
func expectLeftJoinQuery(mock pgxmock.PgxPoolIface, rows [][]any) {
	result := pgxmock.NewRows([]string{"service_id", "endpoint_id"})
	for _, row := range rows {
		result.AddRow(row...)
	}
	// LEFT JOIN query has 3 args: endpoint status (in JOIN ON), host, service status.
	mock.ExpectQuery("SELECT s.id AS service_id, e.id AS endpoint_id FROM service s").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(result)
}

// expectHealthUpdate sets up the UPDATE expectation including the no-op guard arg.
func expectHealthUpdate(mock pgxmock.PgxPoolIface, status string, svcID strfmt.UUID) {
	mock.ExpectExec("UPDATE service SET health_status").
		WithArgs(status, svcID, status).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
}

func TestHealthScrapeLoop_AllRunning_Online(t *testing.T) {
	mock, a := setupHealthMock(t)
	a.haproxy.(*haproxy.FakeHaproxy).Running = true

	expectLeftJoinQuery(mock, [][]any{{healthSvcID, &healthEpID}})
	expectHealthUpdate(mock, models.ServiceHealthStatusONLINE, healthSvcID)

	require.NoError(t, a.HealthScrapeLoop(context.Background()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthScrapeLoop_HaproxyDown_Offline(t *testing.T) {
	mock, a := setupHealthMock(t)
	a.haproxy.(*haproxy.FakeHaproxy).Running = false

	expectLeftJoinQuery(mock, [][]any{{healthSvcID, &healthEpID}})
	expectHealthUpdate(mock, models.ServiceHealthStatusOFFLINE, healthSvcID)

	require.NoError(t, a.HealthScrapeLoop(context.Background()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthScrapeLoop_NoEndpoints_Unchecked(t *testing.T) {
	mock, a := setupHealthMock(t)

	// LEFT JOIN returns the service with NULL endpoint_id (no AVAILABLE endpoints).
	expectLeftJoinQuery(mock, [][]any{{healthSvcID, (*strfmt.UUID)(nil)}})
	expectHealthUpdate(mock, models.ServiceHealthStatusUNCHECKED, healthSvcID)

	require.NoError(t, a.HealthScrapeLoop(context.Background()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthScrapeLoop_NoServices_Noop(t *testing.T) {
	mock, a := setupHealthMock(t)

	expectLeftJoinQuery(mock, [][]any{})

	require.NoError(t, a.HealthScrapeLoop(context.Background()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthScrapeLoop_JoinQueryError_Propagates(t *testing.T) {
	mock, a := setupHealthMock(t)

	mock.ExpectQuery("SELECT s.id AS service_id, e.id AS endpoint_id FROM service s").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(errHealthQueryFailed)

	err := a.HealthScrapeLoop(context.Background())
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthScrapeLoop_MultipleEndpoints_OneDown_Offline(t *testing.T) {
	ep2 := strfmt.UUID("bbbbbbbb-0000-0000-0000-000000000002")

	mock, a := setupHealthMock(t)
	a.haproxy.(*haproxy.FakeHaproxy).Running = false

	expectLeftJoinQuery(mock, [][]any{
		{healthSvcID, &healthEpID},
		{healthSvcID, &ep2},
	})
	expectHealthUpdate(mock, models.ServiceHealthStatusOFFLINE, healthSvcID)

	require.NoError(t, a.HealthScrapeLoop(context.Background()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

// errHealthQueryFailed is a sentinel error for the DB-error test above.
var errHealthQueryFailed = assert.AnError
