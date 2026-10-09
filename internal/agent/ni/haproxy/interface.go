// SPDX-FileCopyrightText: Copyright 2026 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package haproxy

import (
	"context"

	"github.com/sapcc/archer/v2/internal/agent/ni/models"
)

type HAProxy interface {
	CollectStats()
	IsRunning(endpointID string) bool
	AddInstance(injection *models.ServiceInjection) error
	RemoveInstance(endpointID string) error
	Run(ctx context.Context)
}
