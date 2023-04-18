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
	"github.com/jackc/pgx/v5"

	"github.com/sapcc/archer/internal/auth"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/quota"
)

func (c *Controller) GetQuotasHandler(params quota.GetQuotasParams, principal any) middleware.Responder {
	if _, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return quota.NewGetQuotasForbidden()
	}

	q := db.Select("quota.*", "COUNT(DISTINCT s.id) AS in_use_service", "COUNT(DISTINCT e.id) AS in_use_endpoint").
		From("quota").
		RawJoin("INNER JOIN service s ON quota.project_id = s.project_id").
		RawJoin("INNER JOIN endpoint e ON quota.project_id = e.project_id").
		Group("quota.project_id")

	if params.ProjectID != nil {
		q.Where("project_id = ?", params.ProjectID)
	}

	var quotas = make([]*quota.GetQuotasOKBodyQuotasItems0, 0)
	sql, args := q.ToSQL()
	if err := pgxscan.Select(params.HTTPRequest.Context(), c.pool, &quotas, sql, args...); err != nil {
		panic(err)
	}

	return quota.NewGetQuotasOK().WithPayload(&quota.GetQuotasOKBody{Quotas: quotas})
}

func (c *Controller) GetQuotasDefaultsHandler(params quota.GetQuotasDefaultsParams, principal any) middleware.Responder {
	if _, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return quota.NewGetQuotasDefaultsForbidden()
	}

	return quota.NewGetQuotasDefaultsOK().WithPayload(&quota.GetQuotasDefaultsOKBody{
		Quota: &models.Quota{
			Endpoint: &config.Global.Quota.DefaultQuotaEndpoint,
			Service:  &config.Global.Quota.DefaultQuotaService,
		},
	})
}

func (c *Controller) GetQuotasProjectIDHandler(params quota.GetQuotasProjectIDParams, principal any) middleware.Responder {
	if _, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return quota.NewGetQuotasForbidden()
	}

	q := db.Select("quota.service", "quota.endpoint", "COUNT(DISTINCT s.id)", "COUNT(DISTINCT e.id)").
		From("quota").
		Where("quota.project_id = ?", params.ProjectID).
		RawJoin("INNER JOIN service s ON quota.project_id = s.project_id").
		RawJoin("INNER JOIN endpoint e ON quota.project_id = e.project_id").
		Group("quota.project_id")

	var quotaAvail models.Quota
	var quotaUsage models.QuotaUsage
	sql, args := q.ToSQL()
	if err := c.pool.QueryRow(params.HTTPRequest.Context(), sql, args...).
		Scan(&quotaAvail.Service, &quotaAvail.Endpoint, &quotaUsage.InUseService, &quotaUsage.InUseEndpoint); err != nil {
		panic(err)
	}

	return quota.NewGetQuotasProjectIDOK().WithPayload(&quota.GetQuotasProjectIDOKBody{
		Quota: struct {
			models.Quota
			models.QuotaUsage
		}{quotaAvail, quotaUsage},
	})
}

func (c *Controller) PutQuotasProjectIDHandler(params quota.PutQuotasProjectIDParams, principal any) middleware.Responder {
	if _, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return quota.NewPutQuotasProjectIDForbidden()
	}

	q := db.Update("quota").
		Where("project_id", params.ProjectID).Returning("service", "endpoint")

	if params.Quota.Quota.Service != nil {
		q.Set("service", db.Coalesce{V: *params.Quota.Quota.Service})
	}
	if params.Quota.Quota.Endpoint != nil {
		q.Set("endpoint", db.Coalesce{V: *params.Quota.Quota.Endpoint})
	}

	sql, args := q.ToSQL()
	var quotaResponse models.Quota
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &quotaResponse, sql, args); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return quota.NewPutQuotasProjectIDNotFound()
		}
		panic(err)
	}

	return quota.NewPutQuotasProjectIDOK().WithPayload(&quota.PutQuotasProjectIDOKBody{Quota: &quotaResponse})
}

func (c *Controller) DeleteQuotasProjectIDHandler(params quota.DeleteQuotasProjectIDParams, principal any) middleware.Responder {
	if _, err := auth.AuthenticatePrincipal(params.HTTPRequest, principal); err != nil {
		return quota.NewDeleteQuotasProjectIDForbidden()
	}

	sql, args := db.Delete("quota").Where("project_id = ?", params.ProjectID).ToSQL()
	if ct, err := c.pool.Exec(params.HTTPRequest.Context(), sql, args...); err != nil {
		panic(err)
	} else if ct.RowsAffected() == 0 {
		return quota.NewDeleteQuotasProjectIDNotFound()
	}

	return quota.NewDeleteQuotasProjectIDNoContent()
}
