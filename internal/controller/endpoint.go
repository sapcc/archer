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
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/runtime/middleware"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/auth"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/endpoint"
)

func (c *Controller) GetEndpointHandler(params endpoint.GetEndpointParams, principal any) middleware.Responder {
	q := db.StmtBuilder
	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return endpoint.NewGetEndpointForbidden()
	} else if projectId != "" {
		q = q.Where("project_id = ?", projectId)
	}

	pagination := db.Pagination(params)
	sql, args, err := pagination.Query(c.pool, q.
		Select("endpoint.*",
			`endpoint_port.port_id AS "target.port"`,
			`endpoint_port.network AS "target.network"`,
			`endpoint_port.subnet AS "target.subnet"`,
			"service.name AS service_name").
		From("endpoint").
		Join("endpoint_port ON endpoint_port.endpoint_id = endpoint.id").
		Join("service ON service.id = endpoint.service_id"))
	if err != nil {
		panic(err)
	}

	var items = make([]*models.Endpoint, 0)
	if err := pgxscan.Select(context.Background(), c.pool, &items, sql, args...); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == pgerrcode.UndefinedColumn {
			return endpoint.NewGetEndpointBadRequest().WithPayload(&models.Error{
				Code:    400,
				Message: "Unknown sort column.",
			})
		}
		panic(err)
	}
	links := pagination.GetLinks(items)
	return endpoint.NewGetEndpointOK().WithPayload(&endpoint.GetEndpointOKBody{Items: items, Links: links})
}

func (c *Controller) PostEndpointHandler(params endpoint.PostEndpointParams, principal any) middleware.Responder {
	ctx := params.HTTPRequest.Context()
	var endpointResponse models.Endpoint

	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return endpoint.NewGetEndpointForbidden()
	} else if projectId != "" {
		params.Body.ProjectID = models.Project(projectId)
	}

	// Set default values
	if err := c.SetModelDefaults(params.Body); err != nil {
		panic(err)
	}

	target := params.Body.Target
	if target.Subnet == nil && target.Network == nil && target.Port == nil {
		return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
			Code:    400,
			Message: "At least one of target_network, target_subnet or target_port must be specified.",
		})
	}

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Check if service is accessible
	sql, args := db.Select("1").
		From("service").
		Where(sq.Or{
			sq.Eq{"visibility": "public"},              // public service?
			sq.Eq{"project_id": params.Body.ProjectID}, // same project?
			db.Select("1").
				Prefix("EXISTS(").
				From("rbac").
				Where(sq.And{
					sq.Eq{"target_project": params.Body.ProjectID},
					sq.Eq{"service_id": params.Body.ServiceID},
				}).
				Suffix(")"), // RBAC subquery
		}).
		Suffix("FOR UPDATE"). // Lock service/rbac row in this transaction
		MustSql()

	if ct, err := tx.Exec(ctx, sql, args...); err != nil {
		panic(err)
	} else if ct.RowsAffected() < 1 {
		return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
			Code: 400,
			Message: fmt.Sprintf("Service '%s' is not accessible.",
				params.Body.ServiceID),
		})
	}

	// Insert endpoint
	sql, args = db.Insert("endpoint").
		Columns("service_id", "project_id", "tags").
		Values(params.Body.ServiceID, params.Body.ProjectID, Unique(params.Body.Tags)).
		Suffix("RETURNING id, service_id, project_id, tags, created_at, updated_at, status").
		MustSql()
	if err = pgxscan.Get(ctx, tx, &endpointResponse, sql, args...); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) {
			if pgerrcode.UniqueViolation == pe.Code {
				return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
					Code: 400,
					Message: fmt.Sprintf("Port '%s' is already used by another endpoint.",
						params.Body.Target.Port),
				})
			} else if pgerrcode.ForeignKeyViolation == pe.Code {
				return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
					Code: 400,
					Message: fmt.Sprintf("Service '%s' is not accessible.",
						params.Body.ServiceID),
				})
			}
		}
		panic(err)
	}

	port, err := c.AllocateNeutronPort(&params.Body.Target, &endpointResponse, string(params.Body.ProjectID))
	if err != nil {
		if errors.Is(err, ErrPortNotFound) {
			return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
				Code:    400,
				Message: "target_port unknown.",
			})
		}
		if errors.Is(err, ErrProjectMismatch) {
			return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
				Code:    400,
				Message: "target_port needs to be in the same project.",
			})
		}
		if errors.Is(err, ErrMissingIPAddress) {
			return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
				Code:    400,
				Message: "target_port needs at least one IP address.",
			})
		}
		panic(err)
	}

	sql, args = db.Insert("endpoint_port").
		Columns("endpoint_id", "port_id", "subnet", "network", "ip_address").
		Values(endpointResponse.ID, port.ID, port.FixedIPs[0].SubnetID, port.NetworkID, port.FixedIPs[0].IPAddress).
		Suffix("RETURNING port_id, subnet, network").
		MustSql()
	row := tx.QueryRow(params.HTTPRequest.Context(), sql, args...)
	if err := row.Scan(&endpointResponse.Target.Port, &endpointResponse.Target.Subnet,
		&endpointResponse.Target.Network); err != nil {
		logg.Error("Deallocating port %s: %s", port.ID, c.DeallocateNeutronPort(port.ID))
		panic(err)
	}

	sql, args = db.Select("name AS service_name").
		From("service").
		Where("id = ?", endpointResponse.ServiceID).
		MustSql()
	if err := tx.QueryRow(params.HTTPRequest.Context(), sql, args...).Scan(endpointResponse.ServiceName); err != nil {
		logg.Error("Deallocating port %s: %s", port.ID, c.DeallocateNeutronPort(port.ID))
		panic(err)
	}

	// done and done
	if err := tx.Commit(ctx); err != nil {
		logg.Error("Deallocating port %s: %s", port.ID, c.DeallocateNeutronPort(port.ID))
		panic(err)
	}

	return endpoint.NewPostEndpointCreated().WithPayload(&endpointResponse)
}

func (c *Controller) GetEndpointEndpointIDHandler(params endpoint.GetEndpointEndpointIDParams, principal any) middleware.Responder {
	q := db.Select("endpoint.*",
		`endpoint_port.port_id AS "target.port"`,
		`endpoint_port.network AS "target.network"`,
		`endpoint_port.subnet AS "target.subnet"`,
		"service.name AS service_name").
		From("endpoint").
		Join("endpoint_port ON endpoint_port.endpoint_id = endpoint.id").
		Join("service ON service.id = endpoint.service_id").
		Where("endpoint.id = ?", params.EndpointID)

	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return endpoint.NewGetEndpointEndpointIDForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	var endpointResponse models.Endpoint
	sql, args := q.MustSql()
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &endpointResponse, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return endpoint.NewGetEndpointEndpointIDNotFound()
		}
		panic(err)
	}

	return endpoint.NewGetEndpointEndpointIDOK().WithPayload(&endpointResponse)
}

func (c *Controller) PutEndpointEndpointIDHandler(params endpoint.PutEndpointEndpointIDParams, principal any) middleware.Responder {
	q := db.Select("endpoint.*",
		`endpoint_port.port_id AS "target.port"`,
		`endpoint_port.network AS "target.network"`,
		`endpoint_port.subnet AS "target.subnet"`,
		"service.name AS service_name").
		PrefixExpr(db.Update("endpoint").
			Prefix("WITH endpoint AS (").
			Set("tags", Unique(params.Body.Tags)).
			Where("id = ?", params.EndpointID).
			Suffix("RETURNING *)")).
		From("endpoint").
		Join("endpoint_port ON endpoint_port.endpoint_id = endpoint.id").
		Join("service ON service.id = endpoint.service_id")

	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return endpoint.NewPutEndpointEndpointIDForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	sql, args := q.MustSql()
	var endpointResponse models.Endpoint
	if err := pgxscan.Get(context.Background(), c.pool, &endpointResponse, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return endpoint.NewPutEndpointEndpointIDNotFound().WithPayload(&models.Error{
				Code:    404,
				Message: fmt.Sprintf("Endpoint with id '%s' not found.", params.EndpointID),
			})
		}
		panic(err)
	}

	return endpoint.NewPutEndpointEndpointIDOK().WithPayload(&endpointResponse)
}

func (c *Controller) DeleteEndpointEndpointIDHandler(params endpoint.DeleteEndpointEndpointIDParams, principal any) middleware.Responder {
	q := db.Update("endpoint").
		Set("status", "PENDING_DELETE").
		Where("id = ?", params.EndpointID)

	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return endpoint.NewDeleteEndpointEndpointIDForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	sql, args := q.MustSql()
	if ct, err := c.pool.Exec(params.HTTPRequest.Context(), sql, args...); err != nil {
		panic(err)
	} else if ct.RowsAffected() == 0 {
		return endpoint.NewDeleteEndpointEndpointIDNotFound()
	}

	return endpoint.NewDeleteEndpointEndpointIDAccepted()
}
