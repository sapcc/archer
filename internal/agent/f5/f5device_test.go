// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
)

func TestGetF5DeviceSession(t *testing.T) {
	config.Global.Agent.MaxDuration = 0 // disable timeout for tests
	config.Global.Agent.MaxRetries = 0

	_, err := GetF5DeviceSession("http://localhost")
	assert.ErrorContains(t, err, "BIGIP_USER required for host 'localhost'")

	_, err = GetF5DeviceSession("http://user@localhost")
	assert.ErrorContains(t, err, "BIGIP_PASSWORD required for host 'localhost'")

	_, err = GetF5DeviceSession("http://user:password@localhost")
	assert.ErrorContains(t, err, "connection refused")
}
