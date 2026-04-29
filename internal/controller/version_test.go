// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"net/http"

	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/restapi/operations/version"
)

func (t *SuiteTest) TestVersion() {
	config.Global.ApiSettings.AuthStrategy = "keystone"
	config.Global.ApiSettings.RateLimit = 100
	res := t.c.GetVersionHandler(version.GetParams{HTTPRequest: &http.Request{}})
	assert.IsType(t.T(), &version.GetOK{}, res)
	payload := res.(*version.GetOK).Payload

	// Test backward-compatible root-level fields (full version string)
	assert.Equal(t.T(), config.Version, payload.Version)
	assert.Contains(t.T(), payload.Capabilities, "pagination_max=1000")
	assert.Contains(t.T(), payload.Capabilities, "pagination")
	assert.Contains(t.T(), payload.Capabilities, "sorting")
	assert.Contains(t.T(), payload.Capabilities, "cors")
	assert.Contains(t.T(), payload.Capabilities, "keystone")
	assert.Contains(t.T(), payload.Capabilities, "ratelimit=100.00")
	assert.Len(t.T(), payload.Links, 1)
	assert.Equal(t.T(), "self", payload.Links[0].Rel)

	// Test Keystone-compatible versions array (clean microversion)
	assert.Len(t.T(), payload.Versions, 1)
	v := payload.Versions[0]
	assert.Equal(t.T(), "v1", v.ID)
	assert.Equal(t.T(), "CURRENT", v.Status)
	// Keystone expects a clean major.minor version, not full git version
	assert.Equal(t.T(), extractMicroversion(config.Version), v.Version)
	assert.Contains(t.T(), v.Capabilities, "pagination_max=1000")
	assert.Contains(t.T(), v.Capabilities, "pagination")
	assert.Contains(t.T(), v.Capabilities, "sorting")
	assert.Contains(t.T(), v.Capabilities, "cors")
	assert.Contains(t.T(), v.Capabilities, "keystone")
	assert.Contains(t.T(), v.Capabilities, "ratelimit=100.00")
	assert.Len(t.T(), v.Links, 2)
	assert.Equal(t.T(), "self", v.Links[0].Rel)
	assert.Equal(t.T(), "collection", v.Links[1].Rel)

	config.Global.ApiSettings.AuthStrategy = "none"
}

func (t *SuiteTest) TestExtractMicroversion() {
	tests := []struct {
		input    string
		expected string
	}{
		{"v2.2.0-1-ge4ce034", "v2.2.0"},
		{"v2.2.0", "v2.2.0"},
		{"2.2.0", "2.2.0"},
		{"v1.0.0", "v1.0.0"},
		{"v10.20.30", "v10.20.30"},
		{"v1", "v1"},
		{"v1.2", "v1.2"},
		{"1.2.3", "1.2.3"},
		{"123", "123"},
		{"invalid", "1.0.0"},
		{"", "1.0.0"},
	}

	for _, tc := range tests {
		result := extractMicroversion(tc.input)
		assert.Equal(t.T(), tc.expected, result, "extractMicroversion(%q)", tc.input)
	}
}
