// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

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
		Updated: config.BuildTime,
		Version: config.Version,
	})
}
