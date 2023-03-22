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
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/sapcc/go-bits/gopherpolicy"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/config"
)

var (
	ErrForbidden = errors.Unauthenticated("invalid credentials")
)

type Keystone struct {
	tv gopherpolicy.TokenValidator
}

func InitializeKeystone(providerClient *gophercloud.ProviderClient) (*Keystone, error) {
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

func GetProjectID(r *http.Request) string {
	return r.Header.Get("X-Project-Id")
}
