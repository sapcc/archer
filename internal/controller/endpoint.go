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
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/sapcc/archer/internal"
	"github.com/sapcc/archer/internal/auth"
	"github.com/sapcc/archer/internal/db"
	aerr "github.com/sapcc/archer/internal/errors"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/endpoint"
	"github.com/sapcc/go-bits/logg"
)

func (c *Controller) GetEndpointHandler(params endpoint.GetEndpointParams, principal any) middleware.Responder {
	q := db.StmtBuilder
	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return endpoint.NewGetEndpointForbidden()
	} else if projectId != "" {
		q = q.Where("endpoint.project_id = ?", projectId)
	}

	pagination := db.Pagination(params)
	sql, args, err := pagination.Query(c.pool, q.
		Select("endpoint.*",
			`endpoint_port.port_id AS "target.port"`,
			`endpoint_port.network AS "target.network"`,
			`endpoint_port.subnet AS "target.subnet"`).
		From("endpoint").
		Join("endpoint_port ON endpoint_port.endpoint_id = endpoint.id"))
	if err != nil {
		panic(err)
	}

	var items = make([]*models.Endpoint, 0)
	if err := pgxscan.Select(params.HTTPRequest.Context(), c.pool, &items, sql, args...); err != nil {
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
	var host string

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
	sql, args := db.Select("host").
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
		Where("id = ?", params.Body.ServiceID).
		Suffix("FOR UPDATE"). // Lock service/rbac row in this transaction
		MustSql()

	if err = tx.QueryRow(ctx, sql, args...).Scan(&host); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
				Code: 400,
				Message: fmt.Sprintf("Service '%s' is not accessible.",
					params.Body.ServiceID),
			})
		}
		panic(err)
	}

	// Insert endpoint
	sql, args = db.Insert("endpoint").
		Columns("service_id", "project_id", "tags", "name", "description").
		Values(params.Body.ServiceID, params.Body.ProjectID, internal.Unique(params.Body.Tags),
			params.Body.Name, params.Body.Description).
		Suffix("RETURNING id, name, description, service_id, project_id, tags, created_at, updated_at, status").
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

	port, err := c.neutron.AllocateNeutronEndpointPort(&params.Body.Target, &endpointResponse, string(params.Body.ProjectID),
		host)
	if err != nil {
		if errors.Is(err, aerr.ErrPortNotFound) {
			return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
				Code:    400,
				Message: "target_port unknown.",
			})
		}
		if errors.Is(err, aerr.ErrProjectMismatch) {
			return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
				Code:    400,
				Message: "target_port needs to be in the same project.",
			})
		}
		if errors.Is(err, aerr.ErrMissingIPAddress) {
			return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
				Code:    400,
				Message: "target_port needs at least one IP address.",
			})
		}
		if gopherCloudErr, ok := err.(gophercloud.StatusCodeError); ok {
			return endpoint.NewPostEndpointBadRequest().WithPayload(&models.Error{
				Code:    int64(gopherCloudErr.GetStatusCode()),
				Message: gopherCloudErr.Error(),
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
		logg.Error("Deallocating port %s: %s", port.ID, c.neutron.DeletePort(port.ID))
		panic(err)
	}

	// done and done
	if err := tx.Commit(ctx); err != nil {
		logg.Error("Deallocating port %s: %s", port.ID, c.neutron.DeletePort(port.ID))
		panic(err)
	}

	c.notifyEndpoint(host, endpointResponse.ID)
	return endpoint.NewPostEndpointCreated().WithPayload(&endpointResponse)
}

func (c *Controller) GetEndpointEndpointIDHandler(params endpoint.GetEndpointEndpointIDParams, principal any) middleware.Responder {
	q := db.Select("endpoint.*",
		`endpoint_port.port_id AS "target.port"`,
		`endpoint_port.network AS "target.network"`,
		`endpoint_port.subnet AS "target.subnet"`).
		From("endpoint").
		Join("endpoint_port ON endpoint_port.endpoint_id = endpoint.id").
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
		`endpoint_port.subnet AS "target.subnet"`).
		PrefixExpr(db.Update("endpoint").
			Prefix("WITH endpoint AS (").
			Set("tags", sq.Expr("COALESCE(?, tags)", internal.Unique(params.Body.Tags))).
			Set("name", sq.Expr("COALESCE(?, name)", params.Body.Name)).
			Set("description", sq.Expr("COALESCE(?, description)", params.Body.Description)).
			Set("updated_at", sq.Expr("NOW()")).
			Where("id = ?", params.EndpointID).
			Suffix("RETURNING *)")).
		From("endpoint").
		Join("endpoint_port ON endpoint_port.endpoint_id = endpoint.id")

	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return endpoint.NewPutEndpointEndpointIDForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	sql, args := q.MustSql()
	var endpointResponse models.Endpoint
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &endpointResponse, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return endpoint.NewPutEndpointEndpointIDNotFound().WithPayload(&models.Error{
				Code:    404,
				Message: fmt.Sprintf("Endpoint with id '%s' not found.", params.EndpointID),
			})
		}
		panic(err)
	}

	var host string
	if err := c.pool.QueryRow(params.HTTPRequest.Context(), `SELECT host FROM service WHERE id = $1`,
		params.EndpointID).Scan(&host); err == nil {
		c.notifyEndpoint(host, params.EndpointID)
	}
	return endpoint.NewPutEndpointEndpointIDOK().WithPayload(&endpointResponse)
}

func (c *Controller) DeleteEndpointEndpointIDHandler(params endpoint.DeleteEndpointEndpointIDParams, principal any) middleware.Responder {
	var serviceID strfmt.UUID
	var host string

	q := db.Update("endpoint").
		Set("status", "PENDING_DELETE").
		Where("id = ?", params.EndpointID).
		Suffix("RETURNING service_id")

	if projectId, ok := auth.AuthenticatePrincipal(params.HTTPRequest, principal); !ok {
		return endpoint.NewDeleteEndpointEndpointIDForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	sql, args := q.MustSql()

	if err := pgx.BeginFunc(params.HTTPRequest.Context(), c.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(params.HTTPRequest.Context(), sql, args...).Scan(&serviceID); err != nil {
			return err
		}

		if err := tx.QueryRow(params.HTTPRequest.Context(), `SELECT host FROM service WHERE id = $1`, serviceID).
			Scan(&host); err != nil {
			return err
		}

		return nil
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return endpoint.NewDeleteEndpointEndpointIDNotFound()
		}
	}

	c.notifyEndpoint(host, params.EndpointID)
	return endpoint.NewDeleteEndpointEndpointIDAccepted()
}
