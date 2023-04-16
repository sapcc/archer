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
	"fmt"

	"github.com/go-openapi/runtime/middleware"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/version"
)

func (c *Controller) GetVersionHandler(params version.GetParams) middleware.Responder {
	var capabilities []string
	if !config.Global.ApiSettings.DisablePagination {
		capabilities = append(capabilities, "pagination")
	}
	if !config.Global.ApiSettings.DisableSorting {
		capabilities = append(capabilities, "sorting")
	}
	if !config.Global.ApiSettings.DisableCors {
		capabilities = append(capabilities, "cors")
	}
	if config.Global.ApiSettings.AuthStrategy != "none" {
		capabilities = append(capabilities, config.Global.ApiSettings.AuthStrategy)
	}
	if config.Global.ApiSettings.RateLimit > 0 {
		capabilities = append(capabilities, fmt.Sprintf("ratelimit=%.2f",
			config.Global.ApiSettings.RateLimit))
	}
	if config.Global.ApiSettings.PaginationMaxLimit > 0 {
		capabilities = append(capabilities, fmt.Sprintf("pagination_max=%d",
			config.Global.ApiSettings.PaginationMaxLimit))
	}
	return version.NewGetOK().WithPayload(&models.Version{
		Capabilities: capabilities,
		Links: []*models.Link{{
			Href: config.GetApiBaseUrl(params.HTTPRequest),
			Rel:  "self",
		}},
		Updated: "now", // TODO: build time
		Version: c.spec.Spec().Info.Version,
	})
}
