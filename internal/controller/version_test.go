// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"net/http"

	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/version"
)

func (t *SuiteTest) TestVersion() {
	config.Global.ApiSettings.AuthStrategy = "keystone"
	config.Global.ApiSettings.RateLimit = 100
	res := t.c.GetVersionHandler(version.GetParams{HTTPRequest: &http.Request{}})
	assert.IsType(t.T(), &version.GetOK{}, res)
	root := res.(*version.GetOK).Payload
	assert.Len(t.T(), root.Versions, 1)
	v := root.Versions[0]
	assert.Equal(t.T(), "v1", v.ID)
	assert.Equal(t.T(), models.VersionStatusCURRENT, v.Status)
	assert.Equal(t.T(), config.Version, v.Version)
	assert.Contains(t.T(), v.Capabilities, "pagination_max=1000")
	assert.Contains(t.T(), v.Capabilities, "pagination")
	assert.Contains(t.T(), v.Capabilities, "sorting")
	assert.Contains(t.T(), v.Capabilities, "cors")
	assert.Contains(t.T(), v.Capabilities, "keystone")
	assert.Contains(t.T(), v.Capabilities, "ratelimit=100.00")
	assert.Len(t.T(), v.Links, 1)
	assert.Equal(t.T(), "self", v.Links[0].Rel)
	config.Global.ApiSettings.AuthStrategy = "none"
}
