// SPDX-FileCopyrightText: Copyright 2026 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package haproxy

import (
	"github.com/sapcc/archer/internal/agent/ni/models"
)

type HAProxy interface {
	CollectStats()
	IsRunning(string) bool
	AddInstance(injection *models.ServiceInjection) error
	RemoveInstance(networkID string) error
}
