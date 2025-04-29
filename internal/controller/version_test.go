// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"net/http"

	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/restapi/operations/version"
)

func (t *SuiteTest) TestVersion() {
	config.Global.ApiSettings.AuthStrategy = "keystone"
	config.Global.ApiSettings.RateLimit = 100
	res := t.c.GetVersionHandler(version.GetParams{HTTPRequest: &http.Request{}})
	assert.IsType(t.T(), &version.GetOK{}, res)
	payload := res.(*version.GetOK).Payload
	assert.Equal(t.T(), config.Version, payload.Version)
	assert.Contains(t.T(), payload.Capabilities, "pagination_max=1000")
	assert.Contains(t.T(), payload.Capabilities, "pagination")
	assert.Contains(t.T(), payload.Capabilities, "sorting")
	assert.Contains(t.T(), payload.Capabilities, "cors")
	assert.Contains(t.T(), payload.Capabilities, "keystone")
	assert.Contains(t.T(), payload.Capabilities, "ratelimit=100.00")
	config.Global.ApiSettings.AuthStrategy = "none"
}
