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
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/runtime/middleware"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

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
	ctx := params.HTTPRequest.Context()
	var endpointResponse models.Endpoint

	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return endpoint.NewGetEndpointForbidden()
	} else if projectId != "" {
		params.Body.ProjectID = models.Project(projectId)
	}

	// Set default values
	if err := c.SetModelDefaults(params.Body); err != nil {
		panic(err)
	}

	if params.Body.Target.Subnet == nil && params.Body.Target.Network == nil && params.Body.Target.Port == nil {
		return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
			Code:    400,
			Message: "Only one of target_network, target_subnet or target_port must be specified.",
		})
	}

	sql, args := db.Insert("endpoint").
		Set("service_id", params.Body.ServiceID).
		Set("project_id", params.Body.ProjectID).
		Set("\"target.network\"", params.Body.Target.Network).
		Set("\"target.subnet\"", params.Body.Target.Subnet).
		Set("\"target.port\"", params.Body.Target.Port).
		Returning("*").ToSQL()
	if err := pgxscan.Get(ctx, c.pool, &endpointResponse, sql, args...); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pgerrcode.IsIntegrityConstraintViolation(pe.Code) {
			// Todo
			return endpoint.NewPostEndpointForbidden()
		}
		panic(err)
	}

	return endpoint.NewPostEndpointOK().WithPayload(&endpointResponse)
}

func (c *Controller) GetEndpointEndpointIDHandler(params endpoint.GetEndpointEndpointIDParams, principal any) middleware.Responder {
	q := db.Select("*").From("endpoint").Where("id = ?", params.EndpointID)

	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return endpoint.NewGetEndpointEndpointIDForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	var endpointResponse models.Endpoint
	sql, args := q.ToSQL()
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &endpointResponse, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return endpoint.NewGetEndpointEndpointIDNotFound()
		}
		panic(err)
	}

	return endpoint.NewGetEndpointEndpointIDOK().WithPayload(&endpointResponse)
}

func (c *Controller) DeleteEndpointEndpointIDHandler(params endpoint.DeleteEndpointEndpointIDParams, principal any) middleware.Responder {
	q := db.Update("endpoint").
		Set("status", "PENDING_DELETE").
		Where("id", params.EndpointID)

	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return endpoint.NewDeleteEndpointEndpointIDForbidden()
	} else if projectId != "" {
		q.Where("project_id", projectId)
	}

	sql, args := q.ToSQL()
	if ct, err := c.pool.Exec(params.HTTPRequest.Context(), sql, args); err != nil {
		panic(err)
	} else if ct.RowsAffected() == 0 {
		return endpoint.NewDeleteEndpointEndpointIDNotFound()
	}

	return endpoint.NewDeleteEndpointEndpointIDNoContent()
}
