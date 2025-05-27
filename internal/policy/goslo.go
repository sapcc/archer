// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"net/http"

	"github.com/sapcc/go-bits/gopherpolicy"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
)

type gosloPolicyEngine struct{}

func (p gosloPolicyEngine) init() {
	if config.Global.ApiSettings.AuthStrategy != "keystone" {
		log.Fatal("Policy engine goslo supports only api_settings.auth_strategy = 'keystone'")
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
