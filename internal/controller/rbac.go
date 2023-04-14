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
	"github.com/go-openapi/runtime/middleware"

	"github.com/sapcc/archer/restapi/operations/rbac"
)

func (c *Controller) GetRbacPoliciesHandler(params rbac.GetRbacPoliciesParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation rbac.GetRbacPolicies has not yet been implemented")
}

func (c *Controller) PostRbacPoliciesHandler(params rbac.PostRbacPoliciesParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation rbac.PostRbacPolicies has not yet been implemented")
}

func (c *Controller) GetRbacPoliciesRbacPolicyIDHandler(params rbac.GetRbacPoliciesRbacPolicyIDParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation rbac.GetRbacPoliciesRbacPolicyID has not yet been implemented")
}

func (c *Controller) PutRbacPoliciesRbacPolicyIDHandler(params rbac.PutRbacPoliciesRbacPolicyIDParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation rbac.PutRbacPoliciesRbacPolicyID has not yet been implemented")
}

func (c *Controller) DeleteRbacPoliciesRbacPolicyIDHandler(params rbac.DeleteRbacPoliciesRbacPolicyIDParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation rbac.DeleteRbacPoliciesRbacPolicyID has not yet been implemented")
}
