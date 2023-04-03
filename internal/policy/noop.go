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
)

type noOpPolicyEngine struct{}

func (p noOpPolicyEngine) init() {}

func (p noOpPolicyEngine) AuthorizeRequest(r *http.Request, t *gopherpolicy.Token, _ string) bool {
	return true
}

func (p noOpPolicyEngine) AuthorizeGetAllRequest(r *http.Request, t *gopherpolicy.Token, _ string) bool {
	return true
}
