// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/go-bits/gopherpolicy"
	log "github.com/sirupsen/logrus"
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
		log.Info("Initializing goslo policy engine")
		Engine.init()
	case "noop":
		log.Info("Initializing no-op policy engine")
		Engine = noOpPolicyEngine{}
		Engine.init()
	default:
		log.Fatalf("Policy engine '%s' not supported", engine)
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
