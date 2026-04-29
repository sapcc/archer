// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"
	"regexp"

	"github.com/go-openapi/runtime/middleware"

	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/models"
	"github.com/sapcc/archer/v2/restapi/operations/version"
)

// extractMicroversion extracts a valid Keystone version string from a full version string.
// Keystone accepts: 'v1', 'v1.2', '1.2.3', 'v1.2.3', etc.
// e.g., "v2.2.0-1-ge4ce034" -> "v2.2.0", "2.2.0" -> "2.2.0"
func extractMicroversion(fullVersion string) string {
	re := regexp.MustCompile(`^(v?\d+(?:\.\d+)*)`)
	matches := re.FindStringSubmatch(fullVersion)
	if len(matches) >= 2 {
		return matches[1]
	}
	return "1.0.0"
}

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

	baseURL := config.GetApiBaseUrl(params.HTTPRequest)
	links := []*models.Link{
		{Href: baseURL, Rel: "self"},
	}

	// Extract clean microversion for Keystone compatibility
	microversion := extractMicroversion(config.Version)

	return version.NewGetOK().WithPayload(&models.Versions{
		// Keystone-compatible versions array
		Versions: []*models.Version{
			{
				ID:           "v1",
				Status:       "CURRENT",
				Capabilities: capabilities,
				Links: []*models.Link{
					{Href: baseURL, Rel: "self"},
					{Href: baseURL, Rel: "collection"},
				},
				Updated: config.BuildTime,
				Version: microversion,
			},
		},
		// Backward-compatible root-level fields
		Capabilities: capabilities,
		Links:        links,
		Updated:      config.BuildTime,
		Version:      config.Version,
	})
}
