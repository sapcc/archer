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
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/go-bits/gopherpolicy"
	"github.com/sapcc/go-bits/logg"
)

// global policy engine
var Engine policy

type policy interface {
	//init initializer
	init()
	//Authorize (get_one/get_all/post/put/delete) for target(tenant)
	AuthorizeRequest(r *http.Request, t *gopherpolicy.Token, target string) bool
	//Authorize (get_all-global) for target(tenant)
	AuthorizeGetAllRequest(r *http.Request, t *gopherpolicy.Token, target string) bool
}

func SetPolicyEngine(engine string) {
	switch engine {
	case "goslo":
		Engine = gosloPolicyEngine{}
		logg.Info("Initializing goslo policy engine")
		Engine.init()
	case "noop":
		logg.Info("Initializing no-op policy engine")
		Engine = noOpPolicyEngine{}
		Engine.init()
	default:
		logg.Fatal("Policy engine '%s' not supported", engine)
	}
}

// RuleFromHTTPRequest returns policy rule key associated to a http request
func RuleFromHTTPRequest(r *http.Request) string {
	if mr := middleware.MatchedRouteFrom(r); mr != nil {
		// Access x-vendor attributes of the swagger request
		if rule, ok := mr.Operation.VendorExtensible.Extensions.GetString("x-policy"); ok {
			return rule
		}
	}
	return ""
}
