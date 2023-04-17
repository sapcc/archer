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
	"github.com/sapcc/archer/restapi/operations/service"
)

func (c *Controller) GetServiceHandler(params service.GetServiceParams, principal any) middleware.Responder {
	filter := make(map[string]any, 0)
	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return service.NewGetServiceForbidden()
	} else if projectId != "" {
		filter["project_id"] = projectId
	}

	pagination := db.Pagination(params)
	rows, err := pagination.Query(c.pool, "service", filter)
	if err != nil {
		panic(err)
	}

	var servicesResponse = make([]*models.Service, 0)
	if err := pgxscan.ScanAll(&servicesResponse, rows); err != nil {
		panic(err)
	}

	links := pagination.GetLinks(servicesResponse)
	return service.NewGetServiceOK().WithPayload(&service.GetServiceOKBody{Items: servicesResponse, Links: links})
}

func (c *Controller) PostServiceHandler(params service.PostServiceParams, principal any) middleware.Responder {
	ctx := params.HTTPRequest.Context()
	var serviceResponse models.Service

	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return service.NewPostServiceForbidden()
	} else if projectId != "" {
		params.Body.ProjectID = models.Project(projectId)
	}

	// Set default values
	if err := c.SetModelDefaults(params.Body); err != nil {
		panic(err)
	}

	q := db.
		Insert("service").
		Columns("enabled", "name", "description", "network_id", "ip_addresses", "require_approval",
			"visibility", "availability_zone", "proxy_protocol", "project_id", "port", "tags").
		Values(params.Body.Enabled, params.Body.Name, params.Body.Description, params.Body.NetworkID,
			params.Body.IPAddresses, params.Body.RequireApproval, params.Body.Visibility, params.Body.AvailabilityZone,
			params.Body.ProxyProtocol, params.Body.ProjectID, params.Body.Port, params.Body.Tags).
		Returning("*")
	sql, args := q.ToSQL()
	rows, err := c.pool.Query(ctx, sql, *args...)
	if err != nil {
		panic(err)
	}
	if err := pgxscan.ScanOne(&serviceResponse, rows); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pgerrcode.IsIntegrityConstraintViolation(pe.Code) {
			return service.NewPostServiceConflict().WithPayload(&models.Error{
				Code:    409,
				Message: "Entry for network_id, ip_address and availability_zone already exists.",
			})
		}
		panic(err)
	}

	return service.NewPostServiceOK().WithPayload(&serviceResponse)
}

func (c *Controller) GetServiceServiceIDHandler(params service.GetServiceServiceIDParams, principal any) middleware.Responder {
	q := db.Select("*").From("service").Where("id = ?", params.ServiceID)

	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return service.NewGetServiceServiceIDForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	var servicesResponse models.Service
	sql, args := q.ToSQL()
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &servicesResponse, sql, *args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.NewGetServiceServiceIDNotFound()
		}
		panic(err)
	}
	return service.NewGetServiceServiceIDOK().WithPayload(&servicesResponse)
}

func (c *Controller) PutServiceServiceIDHandler(params service.PutServiceServiceIDParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation service.GetServiceServiceID has not yet been implemented")
}

func (c *Controller) DeleteServiceServiceIDHandler(params service.DeleteServiceServiceIDParams, principal any) middleware.Responder {
	q := db.Delete("service").Where("id = ?", params.ServiceID)

	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return service.NewDeleteServiceServiceIDForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	sql, args := q.ToSQL()
	if ct, err := c.pool.Exec(params.HTTPRequest.Context(), sql, *args...); err != nil {
		//TODO: check for conflict (service in use)
		panic(err)
	} else if ct.RowsAffected() == 0 {
		return service.NewDeleteServiceServiceIDNotFound()
	}

	return service.NewDeleteServiceServiceIDNoContent()
}

func (c *Controller) GetServiceServiceIDEndpointsHandler(params service.GetServiceServiceIDEndpointsParams, principal any) middleware.Responder {
	q := db.Select("1").From("service").Where("id = ?", params.ServiceID)
	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return service.NewGetServiceServiceIDEndpointsForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	sql, args := q.ToSQL()
	if ct, err := c.pool.Exec(params.HTTPRequest.Context(), sql, *args...); err != nil {
		panic(err)
	} else if ct.RowsAffected() == 0 {
		return service.NewGetServiceServiceIDEndpointsNotFound()
	}

	pagination := db.Pagination{
		HTTPRequest: params.HTTPRequest,
		Limit:       params.Limit,
		Marker:      params.Marker,
		PageReverse: params.PageReverse,
		Sort:        params.Sort,
	}
	filter := map[string]any{"service_id": params.ServiceID}
	rows, err := pagination.Query(c.pool, "endpoint", filter)
	if err != nil {
		panic(err)
	}

	var endpointsResponse = make([]*models.EndpointConsumer, 0)
	if err := pgxscan.ScanAll(&endpointsResponse, rows); err != nil {
		panic(err)
	}

	links := pagination.GetLinks(endpointsResponse)
	return service.NewGetServiceServiceIDEndpointsOK().
		WithPayload(&service.GetServiceServiceIDEndpointsOKBody{Items: endpointsResponse, Links: links})
}

func (c *Controller) PutServiceServiceIDAcceptEndpointsHandler(params service.PutServiceServiceIDAcceptEndpointsParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation service.PutServiceServiceIDAcceptEndpoints has not yet been implemented")
}

func (c *Controller) PutServiceServiceIDRejectEndpointsHandler(params service.PutServiceServiceIDRejectEndpointsParams, principal any) middleware.Responder {
	return middleware.NotImplemented("operation service.PutServiceServiceIDRejectEndpoints has not yet been implemented")
}
