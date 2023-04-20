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
	"net/http"

	"github.com/georgysavva/scany/v2/dbscan"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

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
	rows, err := pagination.Query(c.pool, "SELECT * FROM service", filter)
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

	sql, args := db.Insert("service").
		Set("enabled", params.Body.Enabled).
		Set("name", params.Body.Name).
		Set("description", params.Body.Description).
		Set("network_id", params.Body.NetworkID).
		Set("ip_addresses", params.Body.IPAddresses).
		Set("require_approval", params.Body.RequireApproval).
		Set("visibility", params.Body.Visibility).
		Set("availability_zone", params.Body.AvailabilityZone).
		Set("proxy_protocol", params.Body.ProxyProtocol).
		Set("project_id", params.Body.ProjectID).
		Set("port", params.Body.Port).
		Set("tags", params.Body.Tags).
		Returning("*").
		ToSQL()
	if err := pgxscan.Get(ctx, c.pool, &serviceResponse, sql, args...); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pgerrcode.IsIntegrityConstraintViolation(pe.Code) {
			return service.NewPostServiceConflict().WithPayload(&models.Error{
				Code:    409,
				Message: "Entry for network_id, ip_address and availability_zone already exists.",
			})
		}
		panic(err)
	}

	return service.NewPostServiceCreated().WithPayload(&serviceResponse)
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
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &servicesResponse, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.NewGetServiceServiceIDNotFound()
		}
		panic(err)
	}
	return service.NewGetServiceServiceIDOK().WithPayload(&servicesResponse)
}

func (c *Controller) PutServiceServiceIDHandler(params service.PutServiceServiceIDParams, principal any) middleware.Responder {
	q := db.Update("service").Where("id", params.ServiceID)

	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return service.NewPutServiceServiceIDForbidden()
	} else if projectId != "" {
		q.Where("project_id", projectId)
	}

	q.Set("enabled", db.Coalesce{V: params.Body.Enabled}).
		Set("name", db.Coalesce{V: params.Body.Name}).
		Set("description", db.Coalesce{V: params.Body.Description}).
		Set("require_approval", db.Coalesce{V: params.Body.RequireApproval}).
		Set("proxy_protocol", db.Coalesce{V: params.Body.ProxyProtocol}).
		Set("port", db.Coalesce{V: params.Body.Port}).
		Set("ip_addresses", db.Coalesce{V: params.Body.IPAddresses}).
		Set("tags", db.Coalesce{V: params.Body.Tags}).
		Returning("*")

	sql, args := q.ToSQL()
	var serviceResponse models.Service
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &serviceResponse, sql, args); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.NewPutServiceServiceIDNotFound()
		}

		var pe *pgconn.PgError
		if errors.As(err, &pe) && pgerrcode.IsIntegrityConstraintViolation(pe.Code) {
			return service.NewPutServiceServiceIDConflict().WithPayload(&models.Error{
				Code:    409,
				Message: "Entry for network_id, ip_address and availability_zone already exists.",
			})
		}
		panic(err)
	}

	return service.NewPutServiceServiceIDOK().WithPayload(&serviceResponse)

}

func (c *Controller) DeleteServiceServiceIDHandler(params service.DeleteServiceServiceIDParams, principal any) middleware.Responder {
	q := db.Delete("service").Where("id = ?", params.ServiceID)

	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return service.NewDeleteServiceServiceIDForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	sql, args := q.ToSQL()
	if ct, err := c.pool.Exec(params.HTTPRequest.Context(), sql, args...); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pgerrcode.IsIntegrityConstraintViolation(pe.Code) {
			return service.NewDeleteServiceServiceIDConflict().WithPayload(&models.Error{
				Code:    409,
				Message: "Service in use",
			})
		}
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
	if ct, err := c.pool.Exec(params.HTTPRequest.Context(), sql, args...); err != nil {
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
	rows, err := pagination.Query(c.pool, "SELECT id, project_id, status FROM endpoint", filter)
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
	endpointConsumers, err := commonEndpointsActionHandler(c.pool, params, principal)
	switch err {
	case auth.ErrForbidden:
		return service.NewPutServiceServiceIDAcceptEndpointsForbidden()
	case ErrBadRequest:
		return service.NewPutServiceServiceIDAcceptEndpointsBadRequest().WithPayload(
			&models.Error{
				Code:    400,
				Message: "Must declare at least one, endpoint_id(s) or project_id(s)",
			})
	case dbscan.ErrNotFound:
		return service.NewPutServiceServiceIDAcceptEndpointsNotFound()
	}
	return service.NewPutServiceServiceIDAcceptEndpointsOK().WithPayload(endpointConsumers)
}

func (c *Controller) PutServiceServiceIDRejectEndpointsHandler(params service.PutServiceServiceIDRejectEndpointsParams, principal any) middleware.Responder {
	endpointConsumers, err := commonEndpointsActionHandler(c.pool, params, principal)
	switch err {
	case auth.ErrForbidden:
		return service.NewPutServiceServiceIDRejectEndpointsForbidden()
	case ErrBadRequest:
		return service.NewPutServiceServiceIDRejectEndpointsBadRequest().WithPayload(
			&models.Error{
				Code:    400,
				Message: "Must declare at least one, endpoint_id(s) or project_id(s)",
			})
	case dbscan.ErrNotFound:
		return service.NewPutServiceServiceIDRejectEndpointsNotFound()
	}
	return service.NewPutServiceServiceIDRejectEndpointsOK().WithPayload(endpointConsumers)
}

func commonEndpointsActionHandler(pool *pgxpool.Pool, body any, principal any) ([]*models.EndpointConsumer, error) {
	var serviceId strfmt.UUID
	var httpRequest *http.Request
	var consumerList *models.EndpointConsumerList

	q := db.Update("endpoint")
	switch params := body.(type) {
	case service.PutServiceServiceIDAcceptEndpointsParams:
		q.Set("status", "PENDING_CREATE")
		serviceId = params.ServiceID
		httpRequest = params.HTTPRequest
		consumerList = params.Body
	case service.PutServiceServiceIDRejectEndpointsParams:
		q.Set("status", "PENDING_REJECTED")
		serviceId = params.ServiceID
		httpRequest = params.HTTPRequest
		consumerList = params.Body
	}

	if projectId, err := auth.AuthenticatePrincipal(httpRequest, principal); err != nil {
		return nil, err
	} else if projectId != "" {
		q.Where("service.project_id", projectId)
	}

	q.From("service").
		Where("endpoint.service_id", db.Raw("service.id")).
		Where("service.id", serviceId).
		Returning("endpoint.id", "endpoint.status", "endpoint.project_id")

	if len(consumerList.EndpointIds) > 0 {
		q.Where("endpoint.id", consumerList.EndpointIds)
	} else if len(consumerList.ProjectIds) > 0 {
		q.Where("endpoint.project_id", consumerList.ProjectIds)
	} else {
		return nil, ErrBadRequest
	}

	sql, args := q.ToSQL()
	var endpointConsumers []*models.EndpointConsumer
	rows, err := pool.Query(httpRequest.Context(), sql, args)
	if err != nil {
		return nil, err
	}

	if err := pgxscan.ScanAll(&endpointConsumers, rows); err != nil {
		return nil, err
	}
	if len(endpointConsumers) == 0 {
		return nil, dbscan.ErrNotFound
	}

	return endpointConsumers, nil
}
