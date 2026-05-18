// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package notifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/archer/v2/models"
)

func TestNotifier_SendNotification(t *testing.T) {
	var receivedReq CampfireRequest
	var receivedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("X-Auth-Token")
		err := json.NewDecoder(r.Body).Decode(&receivedReq)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockClient := &gophercloud.ProviderClient{
		TokenID: "test-token-123",
	}

	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	n, err := New(Config{
		CampfireURL:    server.URL,
		DigestCron:     "0 9 * * *",
		ProviderClient: mockClient,
	}, mock)
	require.NoError(t, err)

	data := NotificationData{
		Type: "immediate",
		Services: []ServiceInfo{
			{
				Service: models.Service{Name: "test-service", ID: "svc-123"},
				Endpoints: []*models.Endpoint{
					{ID: "ep-456", ProjectID: "consumer-proj", CreatedAt: time.Now()},
				},
			},
		},
	}

	err = n.SendNotification(context.Background(), "owner-project-id", data)
	require.NoError(t, err)

	assert.Equal(t, "owner-project-id", receivedReq.ProjectID)
	assert.Equal(t, "text/plain", receivedReq.MimeType)
	assert.Contains(t, receivedReq.Subject, "pending approval")
	assert.Contains(t, receivedReq.MailText, "test-service")
	assert.Equal(t, "test-token-123", receivedToken)
}

func TestNotifier_BuildSubject_Immediate(t *testing.T) {
	data := NotificationData{Type: "immediate"}
	subject := buildSubject(data)
	assert.Equal(t, "Archer Endpoint Services: New endpoint(s) pending approval", subject)
}

func TestNotifier_BuildSubject_Digest(t *testing.T) {
	data := NotificationData{
		Type: "digest",
		Services: []ServiceInfo{
			{Endpoints: []*models.Endpoint{{}, {}}},
			{Endpoints: []*models.Endpoint{{}}},
		},
	}
	subject := buildSubject(data)
	assert.Equal(t, "Archer Endpoint Services: 3 endpoint(s) awaiting approval", subject)
}

func TestNotifier_StopWithoutStart(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	n, err := New(Config{
		CampfireURL: "http://localhost",
		DigestCron:  "0 9 * * *",
	}, mock)
	require.NoError(t, err)

	err = n.Stop()
	assert.NoError(t, err)
}

func TestNotifier_StartFailsWithoutConnection(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	mock.Close()

	n, err := New(Config{
		CampfireURL: "http://localhost",
		DigestCron:  "0 9 * * *",
	}, mock)
	require.NoError(t, err)

	err = n.Start(context.Background())
	assert.Error(t, err)
}

func TestNotifier_ScheduleImmediate(t *testing.T) {
	received := make(chan CampfireRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CampfireRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusOK)
		received <- req
	}))
	defer server.Close()

	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mock.Close()

	n, err := New(Config{
		CampfireURL:    server.URL,
		DigestCron:     "0 9 * * *",
		ProviderClient: &gophercloud.ProviderClient{TokenID: "test"},
	}, mock)
	require.NoError(t, err)
	n.gocronScheduler.Start()
	defer func() { _ = n.gocronScheduler.Shutdown() }()

	rows := pgxmock.NewRows([]string{"name", "project_id"}).
		AddRow("my-service", "owner-project-123")
	mock.ExpectQuery("SELECT .+ FROM service").
		WithArgs(strfmt.UUID("svc-id-1")).
		WillReturnRows(rows)

	ep := &models.Endpoint{
		ID:        "ep-id-1",
		ProjectID: "consumer-project",
		CreatedAt: time.Now(),
	}

	n.ScheduleImmediate(context.Background(), mock, strfmt.UUID("svc-id-1"), ep)

	select {
	case req := <-received:
		assert.Equal(t, "owner-project-123", req.ProjectID)
		assert.Contains(t, req.Subject, "pending approval")
		assert.Contains(t, req.MailText, "my-service")
	case <-time.After(2 * time.Second):
		t.Fatal("immediate notification was not sent")
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestNotifier_ScheduleImmediate_ParentContextCancelled verifies that cancelling the
// caller's context after ScheduleImmediate returns does NOT prevent the async job from
// completing. This regresses against an earlier bug where the HTTP request context was
// captured into the gocron task closure: net/http cancels that ctx on handler return,
// causing both the DB lookup and the Campfire HTTP call to fail with context canceled.
func TestNotifier_ScheduleImmediate_ParentContextCancelled(t *testing.T) {
	received := make(chan CampfireRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CampfireRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusOK)
		received <- req
	}))
	defer server.Close()

	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer mock.Close()

	n, err := New(Config{
		CampfireURL:    server.URL,
		DigestCron:     "0 9 * * *",
		ProviderClient: &gophercloud.ProviderClient{TokenID: "test"},
	}, mock)
	require.NoError(t, err)
	n.gocronScheduler.Start()
	defer func() { _ = n.gocronScheduler.Shutdown() }()

	rows := pgxmock.NewRows([]string{"name", "project_id"}).
		AddRow("my-service", "owner-project-123")
	mock.ExpectQuery("SELECT .+ FROM service").
		WithArgs(strfmt.UUID("svc-id-1")).
		WillReturnRows(rows)

	ep := &models.Endpoint{
		ID:        "ep-id-1",
		ProjectID: "consumer-project",
		CreatedAt: time.Now(),
	}

	// Cancel BEFORE scheduling so the gocron task is guaranteed to observe
	// an already-cancelled parent. Without context.WithoutCancel in
	// ScheduleImmediate, the task's DB lookup would fail with "context
	// canceled" and the Campfire HTTP server would never receive a request.
	parentCtx, cancel := context.WithCancel(context.Background())
	cancel()
	n.ScheduleImmediate(parentCtx, mock, strfmt.UUID("svc-id-1"), ep)

	select {
	case req := <-received:
		assert.Equal(t, "owner-project-123", req.ProjectID)
		assert.Contains(t, req.Subject, "pending approval")
		assert.Contains(t, req.MailText, "my-service")
	case <-time.After(2 * time.Second):
		t.Fatal("async notification job did not complete after parent context was cancelled")
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}
