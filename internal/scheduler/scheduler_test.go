// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultConfig() Config {
	return Config{
		StaleTimeout:           5 * time.Minute,
		CheckInterval:          1 * time.Minute,
		RebalanceDelay:         10 * time.Minute,
		RebalanceThreshold:     0.5,
		RebalanceMaxMigrations: 5,
	}
}

func TestNewServiceScheduler(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	cfg := defaultConfig()
	notifyCalled := false
	notifyFunc := func(host string) { notifyCalled = true }

	scheduler := NewServiceScheduler(mock, cfg, notifyFunc)

	assert.NotNil(t, scheduler)
	assert.Equal(t, cfg.StaleTimeout, scheduler.config.StaleTimeout)
	assert.Equal(t, cfg.RebalanceThreshold, scheduler.config.RebalanceThreshold)

	// Test that notify function is stored
	scheduler.notify("test-host")
	assert.True(t, notifyCalled)
}

func TestServiceScheduler_FindLeastLoadedAgent(t *testing.T) {
	ctx := context.Background()
	cfg := defaultConfig()
	az := "az1"

	t.Run("finds least loaded agent", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		scheduler := NewServiceScheduler(mock, cfg, nil)

		mock.ExpectQuery("SELECT agents.host, COUNT").
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"host", "usage"}).AddRow("agent-1", 2))

		host, err := scheduler.FindLeastLoadedAgent(ctx, "cp", &az, "")

		assert.NoError(t, err)
		assert.Equal(t, "agent-1", host)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("excludes specified host", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		scheduler := NewServiceScheduler(mock, cfg, nil)

		mock.ExpectQuery("SELECT agents.host, COUNT").
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"host", "usage"}).AddRow("agent-2", 3))

		host, err := scheduler.FindLeastLoadedAgent(ctx, "cp", &az, "agent-1")

		assert.NoError(t, err)
		assert.Equal(t, "agent-2", host)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when no agents available", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		scheduler := NewServiceScheduler(mock, cfg, nil)

		mock.ExpectQuery("SELECT agents.host, COUNT").
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"host", "usage"}))

		_, err = scheduler.FindLeastLoadedAgent(ctx, "cp", &az, "")

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestServiceScheduler_ConfigValues(t *testing.T) {
	cfg := Config{
		StaleTimeout:           10 * time.Minute,
		CheckInterval:          2 * time.Minute,
		RebalanceDelay:         15 * time.Minute,
		RebalanceThreshold:     0.3,
		RebalanceMaxMigrations: 10,
	}

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	scheduler := NewServiceScheduler(mock, cfg, nil)

	assert.Equal(t, 10*time.Minute, scheduler.config.StaleTimeout)
	assert.Equal(t, 2*time.Minute, scheduler.config.CheckInterval)
	assert.Equal(t, 15*time.Minute, scheduler.config.RebalanceDelay)
	assert.Equal(t, 0.3, scheduler.config.RebalanceThreshold)
	assert.Equal(t, 10, scheduler.config.RebalanceMaxMigrations)
}

func TestServiceScheduler_NotifyCallback(t *testing.T) {
	cfg := defaultConfig()

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	t.Run("notify is called when set", func(t *testing.T) {
		notifiedHosts := []string{}
		notifyFunc := func(host string) {
			notifiedHosts = append(notifiedHosts, host)
		}

		scheduler := NewServiceScheduler(mock, cfg, notifyFunc)

		scheduler.notify("host-1")
		scheduler.notify("host-2")

		assert.Equal(t, []string{"host-1", "host-2"}, notifiedHosts)
	})

	t.Run("nil notify does not panic", func(t *testing.T) {
		scheduler := NewServiceScheduler(mock, cfg, nil)

		// Should not panic when notify is nil
		assert.Nil(t, scheduler.notify)
	})
}

// TestServiceScheduler_MigrateService_SkipsInProgress verifies that a service
// already PENDING_UPDATE is not re-migrated: MigrateService reads the status
// under FOR UPDATE and returns without issuing any host UPDATE or notify, so
// rebalance/stale retries cannot reset an in-flight migration.
func TestServiceScheduler_MigrateService_SkipsInProgress(t *testing.T) {
	ctx := context.Background()
	cfg := defaultConfig()

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	notified := 0
	scheduler := NewServiceScheduler(mock, cfg, func(string) { notified++ })

	serviceID := strfmt.UUID("46ca20cf-84c3-4210-a360-3f79875f6b9b")
	az := "az1"

	mock.ExpectBegin()
	// FOR UPDATE select now also returns status; PENDING_UPDATE means in flight.
	mock.ExpectQuery("SELECT provider, availability_zone, status FROM service").
		WithArgs(serviceID).
		WillReturnRows(pgxmock.NewRows([]string{"provider", "availability_zone", "status"}).
			AddRow("tenant", &az, "PENDING_UPDATE"))
	// No UPDATE expected — the guard returns before any write.
	mock.ExpectCommit()
	mock.ExpectRollback() // BeginFunc's deferred rollback (no-op after commit)

	err = scheduler.MigrateService(ctx, serviceID, "lb011-01", "lb017-archer")
	assert.NoError(t, err)
	assert.Equal(t, 0, notified, "must not notify agents when skipping in-flight migration")
	assert.NoError(t, mock.ExpectationsWereMet())
}
