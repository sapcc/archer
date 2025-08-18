// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"github.com/sapcc/go-bits/gopherpolicy"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

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
	if err := tv.LoadPolicyFile(config.Global.ApiSettings.PolicyFile, yaml.Unmarshal); err != nil {
		return nil, err
	}

	return &Keystone{tv}, nil
}

func (k *Keystone) AuthenticateToken(tokenStr string) (any, error) {
	ctx := context.Background()
	token := k.tv.CheckCredentials(ctx, tokenStr, func() gopherpolicy.TokenResult {
		return tokens.Get(ctx, k.tv.IdentityV3, tokenStr)
	})
	token.Context.Logger = log.Debugf
	log.Debugf("token has auth = %v", token.Context.Auth)
	log.Debugf("token has roles = %v", token.Context.Roles)

	if token.Err != nil {
		return nil, ErrForbidden
	}

	return token, nil
}

func GetProjectID(r *http.Request) string {
	return r.Header.Get("X-Project-Id")
}
