// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package notifier

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/archer/v2/models"
)

func TestRenderTemplate_Immediate(t *testing.T) {
	tmpl, err := LoadTemplates("")
	require.NoError(t, err)

	data := NotificationData{
		Type: "immediate",
		Services: []ServiceInfo{
			{
				Service: models.Service{Name: "my-service", ID: "svc-001"},
				Endpoints: []*models.Endpoint{
					{ID: "ep-001", ProjectID: "requester-project", CreatedAt: time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)},
				},
			},
		},
	}

	body, err := tmpl.RenderBody(data)
	require.NoError(t, err)
	assert.Contains(t, body, "my-service")
	assert.Contains(t, body, "ep-001")
	assert.Contains(t, body, "pending for")

	subject, err := tmpl.RenderSubject(data)
	require.NoError(t, err)
	assert.Equal(t, "Archer Endpoint Services: New endpoint(s) pending approval", subject)
}

func TestRenderTemplate_Digest(t *testing.T) {
	tmpl, err := LoadTemplates("")
	require.NoError(t, err)

	data := NotificationData{
		Type: "digest",
		Services: []ServiceInfo{
			{
				Service: models.Service{Name: "service-a", ID: "svc-001"},
				Endpoints: []*models.Endpoint{
					{ID: "ep-001", ProjectID: "proj-a", CreatedAt: time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC)},
					{ID: "ep-002", ProjectID: "proj-b", CreatedAt: time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)},
				},
			},
			{
				Service: models.Service{Name: "service-b", ID: "svc-002"},
				Endpoints: []*models.Endpoint{
					{ID: "ep-003", ProjectID: "proj-c", CreatedAt: time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)},
				},
			},
		},
	}

	body, err := tmpl.RenderBody(data)
	require.NoError(t, err)
	assert.Contains(t, body, "service-a")
	assert.Contains(t, body, "service-b")
	assert.Contains(t, body, "ep-001")
	assert.Contains(t, body, "ep-003")
	assert.Contains(t, body, "3 endpoint(s)")

	subject, err := tmpl.RenderSubject(data)
	require.NoError(t, err)
	assert.Equal(t, "Archer Endpoint Services: 3 endpoint(s) awaiting approval", subject)
}
