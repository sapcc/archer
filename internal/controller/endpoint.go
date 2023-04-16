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
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/runtime/middleware"

	"github.com/sapcc/archer/internal/auth"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/endpoint"
)

func (c *Controller) GetEndpointHandler(params endpoint.GetEndpointParams, principal any) middleware.Responder {
	filter := make(map[string]any, 0)
	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return endpoint.NewGetEndpointForbidden()
	} else if projectId != "" {
		filter["project_id"] = projectId
	}

	pagination := db.Pagination(params)
	rows, err := pagination.Query(c.pool, "endpoint", filter)
	if err != nil {
		panic(err)
	}

	var items = make([]*models.Endpoint, 0)
	if err := pgxscan.ScanAll(&items, rows); err != nil {
		panic(err)
	}
	links := pagination.GetLinks(items)
	return endpoint.NewGetEndpointOK().WithPayload(&endpoint.GetEndpointOKBody{Items: items, Links: links})
}

func (c *Controller) PostEndpointHandler(params endpoint.PostEndpointParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation endpoint.PostEndpoint has not yet been implemented")
}

func (c *Controller) GetEndpointEndpointIDHandler(params endpoint.GetEndpointEndpointIDParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation endpoint.GetEndpointEndpointID has not yet been implemented")
}

func (c *Controller) DeleteEndpointEndpointIDHandler(params endpoint.DeleteEndpointEndpointIDParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation endpoint.DeleteEndpointEndpointID has not yet been implemented")
}
