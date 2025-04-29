// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"net/http"

	"github.com/sapcc/go-bits/gopherpolicy"
)

type noOpPolicyEngine struct{}

func (p noOpPolicyEngine) init() {}

func (p noOpPolicyEngine) AuthorizeRequest(_ *http.Request, _ *gopherpolicy.Token, _ string) bool {
	return true
}

func (p noOpPolicyEngine) AuthorizeGetAllRequest(_ *http.Request, _ *gopherpolicy.Token, _ string) bool {
	return true
}
