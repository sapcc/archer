// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/runtime/middleware"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/sapcc/archer/internal/auth"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/rbac"
)

func (c *Controller) GetRbacPoliciesHandler(params rbac.GetRbacPoliciesParams, _ any) middleware.Responder {
	q := db.Select("id", "target_project AS target", "'project' AS target_type", "service_id",
		"created_at", "updated_at", "project_id").
		From("rbac")
	if projectId := auth.GetProjectID(params.HTTPRequest); projectId != "" {
		q = q.Where("project_id = ?", projectId)
	}

	pagination := db.Pagination{
		HTTPRequest: params.HTTPRequest,
		Limit:       params.Limit,
		Marker:      params.Marker,
		PageReverse: params.PageReverse,
		Sort:        params.Sort,
	}

	sql, args, err := pagination.Query(c.pool, q)
	if err != nil {
		panic(err)
	}

	var items = make([]*models.Rbacpolicy, 0)
	if err := pgxscan.Select(context.Background(), c.pool, &items, sql, args...); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == pgerrcode.UndefinedColumn {
			return rbac.NewGetRbacPoliciesBadRequest().WithPayload(&models.Error{
				Code:    400,
				Message: "Unknown sort column.",
			})
		}
		panic(err)
	}
	links := pagination.GetLinks(items)
	return rbac.NewGetRbacPoliciesOK().WithPayload(&rbac.GetRbacPoliciesOKBody{Items: items, Links: links})
}

func (c *Controller) PostRbacPoliciesHandler(params rbac.PostRbacPoliciesParams, _ any) middleware.Responder {
	ctx := params.HTTPRequest.Context()
	var rbacResponse models.Rbacpolicy

	if projectId := auth.GetProjectID(params.HTTPRequest); projectId != "" {
		params.Body.ProjectID = models.Project(projectId)
	}

	// Set default values
	if err := c.SetModelDefaults(params.Body); err != nil {
		panic(err)
	}

	sql, args := db.Insert("rbac").
		Columns("service_id", "target_project", "project_id").
		Values(params.Body.ServiceID, params.Body.Target, params.Body.ProjectID).
		Suffix("RETURNING id, target_project AS target, 'project' AS target_type, service_id, created_at, updated_at, project_id").
		MustSql()
	if err := pgxscan.Get(ctx, c.pool, &rbacResponse, sql, args...); err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pgerrcode.IsIntegrityConstraintViolation(pe.Code) {
			// Todo
			if pgerrcode.UniqueViolation == pe.Code {
				return rbac.NewPostRbacPoliciesConflict().WithPayload(&models.Error{
					Code:    409,
					Message: "Duplicate RBAC Policy, service_id and target project combination already exists",
				})
			}
			return rbac.NewPostRbacPoliciesNotFound()
		}
		panic(err)
	}

	return rbac.NewPostRbacPoliciesCreated().WithXTargetID(rbacResponse.ID).WithPayload(&rbacResponse)
}

func (c *Controller) GetRbacPoliciesRbacPolicyIDHandler(params rbac.GetRbacPoliciesRbacPolicyIDParams, _ any) middleware.Responder {
	q := db.
		Select("id", "target_project AS target", "'project' AS target_type", "service_id", "created_at",
			"updated_at", "project_id").
		From("rbac").
		Where("id = ?", params.RbacPolicyID)

	if projectId := auth.GetProjectID(params.HTTPRequest); projectId != "" {
		q = q.Where("project_id = ?", projectId)
	}

	var rbacResponse models.Rbacpolicy
	sql, args := q.MustSql()
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &rbacResponse, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return rbac.NewGetRbacPoliciesRbacPolicyIDNotFound()
		}
		panic(err)
	}

	return rbac.NewGetRbacPoliciesRbacPolicyIDOK().WithPayload(&rbacResponse)

}

func (c *Controller) PutRbacPoliciesRbacPolicyIDHandler(params rbac.PutRbacPoliciesRbacPolicyIDParams, _ any) middleware.Responder {
	q := db.Update("rbac").Where("id = ?", params.RbacPolicyID)

	if projectId := auth.GetProjectID(params.HTTPRequest); projectId != "" {
		q = q.Where("project_id = ?", projectId)
	}

	sql, args := q.Set("target_project", sq.Expr("COALESCE(?, target_project)", params.Body.Target)).
		Suffix("RETURNING id, target_project AS target, 'project' AS target_type, service_id, created_at, updated_at, project_id").
		MustSql()
	var rbacResponse models.Rbacpolicy
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &rbacResponse, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return rbac.NewPutRbacPoliciesRbacPolicyIDNotFound()
		}

		var pe *pgconn.PgError
		if errors.As(err, &pe) && pgerrcode.IsIntegrityConstraintViolation(pe.Code) {
			return rbac.NewPutRbacPoliciesRbacPolicyIDConflict().WithPayload(&models.Error{
				Code:    409,
				Message: "Duplicate RBAC Policy, service_id and target project combination already exists",
			})
		}
		panic(err)
	}

	return rbac.NewPutRbacPoliciesRbacPolicyIDOK().WithPayload(&rbacResponse)
}

func (c *Controller) DeleteRbacPoliciesRbacPolicyIDHandler(params rbac.DeleteRbacPoliciesRbacPolicyIDParams, _ any) middleware.Responder {
	q := db.Delete("rbac").
		Where("id = ?", params.RbacPolicyID)

	if projectId := auth.GetProjectID(params.HTTPRequest); projectId != "" {
		q = q.Where("project_id = ?", projectId)
	}

	sql, args := q.MustSql()
	if ct, err := c.pool.Exec(params.HTTPRequest.Context(), sql, args...); err != nil {
		panic(err)
	} else if ct.RowsAffected() == 0 {
		return rbac.NewDeleteRbacPoliciesRbacPolicyIDNotFound()
	}

	return rbac.NewDeleteRbacPoliciesRbacPolicyIDNoContent()
}
