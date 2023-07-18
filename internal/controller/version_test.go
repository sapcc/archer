// Copyright 2023 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
