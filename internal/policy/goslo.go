/*
 *   Copyright 2020 SAP SE
 *
 *   Licensed under the Apache License, Version 2.0 (the "License");
 *   you may not use this file except in compliance with the License.
 *   You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 */

package policy

import (
	"github.com/sapcc/go-bits/gopherpolicy"
	"net/http"

	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/config"
)

type gosloPolicyEngine struct{}

func (p gosloPolicyEngine) init() {
	if config.Global.ApiSettings.AuthStrategy != "keystone" {
		logg.Fatal("Policy engine goslo supports only api_settings.auth_strategy = 'keystone'")
	}
}

func (p gosloPolicyEngine) AuthorizeRequest(r *http.Request, t *gopherpolicy.Token, target string) bool {
	rule := RuleFromHTTPRequest(r)

	if t != nil {
		t.Context.Request = map[string]string{
			"project_id": target,
		}
		return t.Check(rule)
	}
	// Ignore disabled keystone middleware
	return true
}

func (p gosloPolicyEngine) AuthorizeGetAllRequest(r *http.Request, t *gopherpolicy.Token, target string) bool {
	rule := RuleFromHTTPRequest(r)

	if t != nil {
		t.Context.Request = map[string]string{
			"project_id": target,
		}
		return t.Check(rule + "-global")
	}
	// Ignore disabled keystone middleware
	return true
}
