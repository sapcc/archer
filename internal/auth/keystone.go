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

package auth

import (
	"errors"
	"github.com/sapcc/archer/internal/policy"
	"net/http"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/sapcc/go-bits/gopherpolicy"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/config"
)

var (
	ErrForbidden = errors.New("forbidden")
)

type Keystone struct {
	tv gopherpolicy.TokenValidator
}

func InitializeKeystone() (*Keystone, error) {
	authInfo := clientconfig.AuthInfo(config.Global.ServiceAuth)
	providerClient, err := clientconfig.AuthenticatedClient(&clientconfig.ClientOpts{
		AuthInfo: &authInfo})
	if err != nil {
		return nil, err
	}

	keystoneV3, err := openstack.NewIdentityV3(providerClient, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	tv := gopherpolicy.TokenValidator{
		IdentityV3: keystoneV3,
		Cacher:     gopherpolicy.InMemoryCacher(),
	}
	if err := tv.LoadPolicyFile(config.Global.ApiSettings.PolicyFile); err != nil {
		return nil, err
	}

	return &Keystone{tv}, nil
}

func (k *Keystone) AuthenticateToken(tokenStr string) (any, error) {
	token := k.tv.CheckCredentials(tokenStr, func() gopherpolicy.TokenResult {
		return tokens.Get(k.tv.IdentityV3, tokenStr)
	})
	token.Context.Logger = logg.Debug
	logg.Debug("token has auth = %v", token.Context.Auth)
	logg.Debug("token has roles = %v", token.Context.Roles)

	if token.Err != nil {
		return nil, ErrForbidden
	}

	return token, nil
}

func AuthenticatePrincipal(r *http.Request, principal any) (string, error) {
	if t, ok := principal.(*gopherpolicy.Token); ok {
		rule := policy.RuleFromHTTPRequest(r)
		if t.Check(rule + "-global") {
			return "", nil
		} else if t.Check(rule) {
			return t.ProjectScopeUUID(), nil
		} else {
			return "", ErrForbidden
		}
	}

	return "", nil
}
