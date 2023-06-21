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
	"context"
	"errors"
	"github.com/sapcc/archer/internal"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/dbscan"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sapcc/go-bits/gopherpolicy"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/auth"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/service"
)

func (c *Controller) GetServiceHandler(params service.GetServiceParams, principal any) middleware.Responder {
	q := db.Select("*").From("service")
	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return service.NewGetServiceForbidden()
	} else if projectId != "" {
		// RBAC support
		q = q.Where(
			sq.Or{
				sq.Eq{"project_id": projectId},
				sq.Eq{"visibility": "public"},
				db.Select("1").
					Prefix("EXISTS(").
					From("rbac r").
					Where("r.target_project = ?", projectId).
					Where("r.service_id = service.id").
					Suffix(")"),
			})
	}

	pagination := db.Pagination(params)
	sql, args, err := pagination.Query(c.pool, q)
	if err != nil {
		panic(err)
	}

	var servicesResponse = make([]*models.Service, 0)
	if err := pgxscan.Select(context.Background(), c.pool, &servicesResponse, sql, args...); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == pgerrcode.UndefinedColumn {
			return service.NewGetServiceBadRequest().WithPayload(&models.Error{
				Code:    400,
				Message: "Unknown sort column.",
			})
		}
		panic(err)
	}

	links := pagination.GetLinks(servicesResponse)
	return service.NewGetServiceOK().WithPayload(&service.GetServiceOKBody{Items: servicesResponse, Links: links})
}

func (c *Controller) PostServiceHandler(params service.PostServiceParams, principal any) middleware.Responder {
	ctx := params.HTTPRequest.Context()
	var serviceResponse models.Service

	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return service.NewPostServiceForbidden()
	} else if projectId != "" {
		params.Body.ProjectID = models.Project(projectId)
	}

	// Set default values
	if err := c.SetModelDefaults(params.Body); err != nil {
		panic(err)
	}

	if *params.Body.Provider != "tenant" {
		if t, ok := principal.(*gopherpolicy.Token); ok {
			if !t.Check("service:create:provider") {
				return service.NewPostServiceForbidden()
			}
		}
	}

	var port *ports.Port
	var host string
	if err := pgx.BeginFunc(context.Background(), c.pool, func(tx pgx.Tx) error {
		// schedule
		q := db.Select("agents.host", "COUNT(service.id) AS usage").
			From("agents").
			LeftJoin("service ON service.host = agents.host").
			Where(sq.And{
				sq.Eq{"agents.enabled": true},
				sq.Eq{"agents.provider": params.Body.Provider},
			}).
			OrderBy("usage ASC", "agents.updated_at DESC").
			GroupBy("agents.host").
			Limit(1)

		if params.Body.AvailabilityZone != nil {
			q.Where("agents.availability_zone = ?", *params.Body.AvailabilityZone)
		}
		sql, args, err := q.ToSql()
		if err != nil {
			return err
		}

		var usage int
		if err = c.pool.QueryRow(context.Background(), sql, args...).Scan(&host, &usage); err != nil {
			return err
		}

		logg.Info("Found host '%s' (usage=%d) for service request (provider=%s)", host, usage,
			params.Body.Provider)
		params.Body.Host = &host

		sql, args, err = db.Insert("service").
			Columns("enabled", "name", "description", "network_id", "ip_addresses", "require_approval",
				"visibility", "availability_zone", "proxy_protocol", "project_id", "port", "tags", "provider", "host").
			Values(params.Body.Enabled, params.Body.Name, params.Body.Description, params.Body.NetworkID,
				params.Body.IPAddresses, params.Body.RequireApproval, params.Body.Visibility,
				params.Body.AvailabilityZone, params.Body.ProxyProtocol, params.Body.ProjectID,
				params.Body.Port, internal.Unique(params.Body.Tags), params.Body.Provider, params.Body.Host).
			Suffix("RETURNING *").ToSql()
		if err != nil {
			return err
		}

		if err = pgxscan.Get(ctx, tx, &serviceResponse, sql, args...); err != nil {
			return err
		}

		if *params.Body.Provider == "tenant" {
			// Allocate SNAT port for provider 'tenant' (F5)
			if port, err = c.AllocateSNATNeutronPort(&serviceResponse); err != nil {
				return err
			}

			sql, args, err = db.Insert("service_port").
				Columns("service_id", "port_id").
				Values(serviceResponse.ID, port.ID).
				ToSql()
			if err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, sql, args...); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		if port != nil {
			logg.Info("Deallocating snat-port due to error %s: %s", port.ID, c.DeallocateNeutronPort(port.ID))
		}

		if errors.Is(err, pgx.ErrNoRows) {
			return service.NewPostServiceConflict().WithPayload(&models.Error{
				Code:    409,
				Message: "No available host agent found.",
			})
		}

		var pe *pgconn.PgError
		if errors.As(err, &pe) && pgerrcode.IsIntegrityConstraintViolation(pe.Code) {
			return service.NewPostServiceConflict().WithPayload(&models.Error{
				Code:    409,
				Message: "Entry for network_id, ip_address and availability_zone already exists.",
			})
		}
		panic(err)
	}

	c.notifyService(host)
	return service.NewPostServiceCreated().WithPayload(&serviceResponse)
}

func (c *Controller) GetServiceServiceIDHandler(params service.GetServiceServiceIDParams, principal any) middleware.Responder {
	q := db.Select("*").From("service").Where("id = ?", params.ServiceID)

	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return service.NewGetServiceServiceIDForbidden()
	} else if projectId != "" {
		// RBAC support
		q = q.Where(
			sq.Or{
				sq.Eq{"project_id": projectId},
				db.Select("1").
					Prefix("EXISTS(").
					From("rbac r").
					Where("r.target_project = ?", projectId).
					Where("r.service_id = service.id").
					Suffix(")"),
			})
	}

	var servicesResponse models.Service
	sql, args := q.MustSql()
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &servicesResponse, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.NewGetServiceServiceIDNotFound()
		}
		panic(err)
	}
	return service.NewGetServiceServiceIDOK().WithPayload(&servicesResponse)
}

func (c *Controller) PutServiceServiceIDHandler(params service.PutServiceServiceIDParams, principal any) middleware.Responder {
	upd := db.Update("service")

	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return service.NewPutServiceServiceIDForbidden()
	} else if projectId != "" {
		upd = upd.Where("project_id = ?", projectId)
	}

	upd = upd.Set("enabled", sq.Expr("COALESCE(?, enabled)", params.Body.Enabled)).
		Set("name", sq.Expr("COALESCE(?, name)", params.Body.Name)).
		Set("description", sq.Expr("COALESCE(?, description)", params.Body.Description)).
		Set("require_approval", sq.Expr("COALESCE(?, require_approval)", params.Body.RequireApproval)).
		Set("proxy_protocol", sq.Expr("COALESCE(?, proxy_protocol)", params.Body.ProxyProtocol)).
		Set("port", sq.Expr("COALESCE(?, port)", params.Body.Port)).
		Set("ip_addresses", sq.Expr("COALESCE(?, ip_addresses)", params.Body.IPAddresses)).
		Set("visibility", sq.Expr("COALESCE(?, visibility)", params.Body.Visibility)).
		Set("tags", sq.Expr("COALESCE(?, tags)", internal.Unique(params.Body.Tags))).
		Set("status", "PENDING_UPDATE").
		Where("id = ?", params.ServiceID).
		Suffix("RETURNING *")

	sql, args := upd.MustSql()
	var serviceResponse models.Service
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &serviceResponse, sql, args...); err != nil {
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

	c.notifyService(*serviceResponse.Host)
	return service.NewPutServiceServiceIDOK().WithPayload(&serviceResponse)
}

func (c *Controller) DeleteServiceServiceIDHandler(params service.DeleteServiceServiceIDParams, principal any) middleware.Responder {
	var host string
	q := db.Select("host").
		From("service").
		Where("id = ?", params.ServiceID).
		Suffix("FOR UPDATE")

	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return service.NewDeleteServiceServiceIDForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	tx, err := c.pool.Begin(params.HTTPRequest.Context())
	if err != nil {
		panic(err)
	}
	defer func() { _ = tx.Rollback(params.HTTPRequest.Context()) }()

	// First check if service exists and is "accessible", and lock the row
	sql, args := q.MustSql()
	if err := tx.QueryRow(params.HTTPRequest.Context(), sql, args...).Scan(&host); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.NewDeleteServiceServiceIDNotFound()
		}
		panic(err)
	}

	// Update status if no endpoints are attached.
	u := db.Update("service").
		Set("status", models.ServiceStatusPENDINGDELETE).
		Where(sq.And{
			sq.Eq{"id": params.ServiceID},
			db.Select("1").
				From("endpoint").
				Where("service_id = service.id").
				Prefix("NOT EXISTS(").
				Suffix(")"), // RBAC subquery
		})
	sql, args = u.MustSql()
	if ct, err := tx.Exec(params.HTTPRequest.Context(), sql, args...); err != nil {
		panic(err)
	} else if ct.RowsAffected() == 0 {
		return service.NewDeleteServiceServiceIDConflict().WithPayload(&models.Error{
			Code:    409,
			Message: "Service in use",
		})
	}
	if err = tx.Commit(params.HTTPRequest.Context()); err != nil {
		panic(err)
	}

	c.notifyService(host)
	return service.NewDeleteServiceServiceIDAccepted()
}

func (c *Controller) GetServiceServiceIDEndpointsHandler(params service.GetServiceServiceIDEndpointsParams, principal any) middleware.Responder {
	q := db.Select("1").
		From("service").
		Where("id = ?", params.ServiceID)

	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return service.NewGetServiceServiceIDEndpointsForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	sql, args := q.MustSql()
	tx, err := c.pool.Begin(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	ct, err := tx.Exec(params.HTTPRequest.Context(), sql, args...)
	if err != nil {
		panic(err)
	}
	if ct.RowsAffected() == 0 {
		return service.NewGetServiceServiceIDEndpointsNotFound()
	}

	pagination := db.Pagination{
		HTTPRequest: params.HTTPRequest,
		Limit:       params.Limit,
		Marker:      params.Marker,
		PageReverse: params.PageReverse,
		Sort:        params.Sort,
	}
	q = db.Select("id", "project_id", "status").
		From("endpoint").
		Where("service_id = ?", params.ServiceID)
	sql, args, err = pagination.Query(c.pool, q)
	if err != nil {
		panic(err)
	}

	var endpointsResponse = make([]*models.EndpointConsumer, 0)
	if err := pgxscan.Select(context.Background(), c.pool, &endpointsResponse, sql, args...); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == pgerrcode.UndefinedColumn {
			return service.NewGetServiceServiceIDEndpointsBadRequest().WithPayload(&models.Error{
				Code:    400,
				Message: "Unknown sort column.",
			})
		}
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

	q := db.Update("endpoint").
		From("service").
		Suffix("RETURNING endpoint.id, endpoint.status, endpoint.project_id")

	switch params := body.(type) {
	case service.PutServiceServiceIDAcceptEndpointsParams:
		q = q.Set("status", models.EndpointStatusPENDINGCREATE)
		serviceId = params.ServiceID
		httpRequest = params.HTTPRequest
		consumerList = params.Body
	case service.PutServiceServiceIDRejectEndpointsParams:
		q = q.Set("status", models.EndpointStatusPENDINGREJECTED)
		serviceId = params.ServiceID
		httpRequest = params.HTTPRequest
		consumerList = params.Body
	}

	if projectId, ok := auth.AuthenticatePrincipal(httpRequest, principal); !ok {
		return nil, auth.ErrForbidden
	} else if projectId != "" {
		q = q.Where(db.Select("1").
			Prefix("EXISTS(").
			From("service").
			Where("project_id = ?", projectId).
			Where("id = ?", serviceId).
			Suffix(")"), // service subquery
		)
	}

	if len(consumerList.EndpointIds) == 0 && len(consumerList.ProjectIds) == 0 {
		return nil, ErrBadRequest
	}

	q = q.Where(sq.Or{
		sq.Eq{"endpoint.id": consumerList.EndpointIds},
		sq.Eq{"endpoint.project_id": consumerList.ProjectIds},
	})

	sql, args := q.MustSql()
	var endpointConsumers []*models.EndpointConsumer
	rows, err := pool.Query(httpRequest.Context(), sql, args...)
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
