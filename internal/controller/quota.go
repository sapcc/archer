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

	"github.com/sapcc/archer/restapi/operations/quota"
)

func (c *Controller) GetQuotasHandler(params quota.GetQuotasParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation quota.GetQuotas has not yet been implemented")
}

func (c *Controller) GetQuotasDefaultsHandler(params quota.GetQuotasDefaultsParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation quota.GetQuotasDefaults has not yet been implemented")
}

func (c *Controller) GetQuotasProjectIDHandler(params quota.GetQuotasProjectIDParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation quota.GetQuotasProjectID has not yet been implemented")
}

func (c *Controller) PutQuotasProjectIDHandler(params quota.PutQuotasProjectIDParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation quota.PutQuotasProjectID has not yet been implemented")
}

func (c *Controller) DeleteQuotasProjectIDHandler(params quota.DeleteQuotasProjectIDParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation quota.DeleteQuotasProjectID has not yet been implemented")
}
