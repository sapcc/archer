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

	"github.com/sapcc/archer/restapi/operations/rbac"
)

func (c *Controller) GetRbacPoliciesHandler(params rbac.GetRbacPoliciesParams, principal any) middleware.Responder {
	filter := make(map[string]any, 0)
	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return rbac.NewGetRbacPoliciesForbidden()
	} else if projectId != "" {
		filter["project_id"] = projectId
	}

	pagination := db.Pagination{
		HTTPRequest: params.HTTPRequest,
		Limit:       params.Limit,
		Marker:      params.Marker,
		PageReverse: params.PageReverse,
		Sort:        params.Sort,
	}
	rows, err := pagination.Query(c.pool, "SELECT * FROM rbac", filter)
	if err != nil {
		panic(err)
	}

	var items = make([]*models.Rbacpolicy, 0)
	if err := pgxscan.ScanAll(&items, rows); err != nil {
		panic(err)
	}
	links := pagination.GetLinks(items)
	return rbac.NewGetRbacPoliciesOK().WithPayload(&rbac.GetRbacPoliciesOKBody{Items: items, Links: links})
}

func (c *Controller) PostRbacPoliciesHandler(params rbac.PostRbacPoliciesParams, principal any) middleware.Responder {
	ctx := params.HTTPRequest.Context()
	var rbacResponse models.Rbacpolicy

	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return rbac.NewPostRbacPoliciesForbidden()
	} else if projectId != "" {
		params.Body.ProjectID = models.Project(projectId)
	}

	// Set default values
	if err := c.SetModelDefaults(params.Body); err != nil {
		panic(err)
	}

	sql, args := db.Insert("rbac").
		Set("service_id", params.Body.ServiceID).
		Set("target_project", params.Body.Target).
		Set("project_id", params.Body.ProjectID).
		Returning("id", "target_project AS target",
			"service_id", "created_at", "updated_at", "project_id").ToSQL()
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

	return rbac.NewPostRbacPoliciesCreated().WithPayload(&rbacResponse)
}

func (c *Controller) GetRbacPoliciesRbacPolicyIDHandler(params rbac.GetRbacPoliciesRbacPolicyIDParams, principal any) middleware.Responder {
	q := db.Select("id", "target_project AS target", "service_id", "created_at", "updated_at", "project_id").
		From("rbac").Where("id = ?", params.RbacPolicyID)

	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return rbac.NewGetRbacPoliciesRbacPolicyIDForbidden()
	} else if projectId != "" {
		q.Where("project_id = ?", projectId)
	}

	var rbacResponse models.Rbacpolicy
	sql, args := q.ToSQL()
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &rbacResponse, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return rbac.NewGetRbacPoliciesRbacPolicyIDNotFound()
		}
		panic(err)
	}

	return rbac.NewGetRbacPoliciesRbacPolicyIDOK().WithPayload(&rbacResponse)

}

func (c *Controller) PutRbacPoliciesRbacPolicyIDHandler(params rbac.PutRbacPoliciesRbacPolicyIDParams, principal any) middleware.Responder {
	q := db.Update("rbac").Where("id", params.RbacPolicyID)

	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return rbac.NewPutRbacPoliciesRbacPolicyIDForbidden()
	} else if projectId != "" {
		q.Where("project_id", projectId)
	}

	sql, args := q.Set("target_project", db.Coalesce{V: params.Body.Target}).
		Returning("id", "target_project AS target", "service_id", "created_at", "updated_at", "project_id").
		ToSQL()
	var rbacResponse models.Rbacpolicy
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &rbacResponse, sql, args); err != nil {
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

func (c *Controller) DeleteRbacPoliciesRbacPolicyIDHandler(params rbac.DeleteRbacPoliciesRbacPolicyIDParams, principal any) middleware.Responder {
	q := db.Delete("rbac").Where("id = ?", params.RbacPolicyID)

	if projectId, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return rbac.NewDeleteRbacPoliciesRbacPolicyIDForbidden()
	} else if projectId != "" {
		q.Where("project_id", projectId)
	}

	sql, args := q.ToSQL()
	if ct, err := c.pool.Exec(params.HTTPRequest.Context(), sql, args...); err != nil {
		panic(err)
	} else if ct.RowsAffected() == 0 {
		return rbac.NewDeleteRbacPoliciesRbacPolicyIDNotFound()
	}

	return rbac.NewDeleteRbacPoliciesRbacPolicyIDNoContent()
}
