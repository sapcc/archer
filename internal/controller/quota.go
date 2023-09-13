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

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/runtime/middleware"
	"github.com/jackc/pgx/v5"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/quota"
)

func (c *Controller) GetQuotasHandler(params quota.GetQuotasParams, _ any) middleware.Responder {
	q := db.Select("quota.*", "COUNT(DISTINCT s.id) AS in_use_service", "COUNT(DISTINCT e.id) AS in_use_endpoint").
		From("quota").
		InnerJoin("service s ON quota.project_id = s.project_id").
		InnerJoin("endpoint e ON quota.project_id = e.project_id").
		GroupBy("quota.project_id")

	if params.ProjectID != nil {
		q.Where("project_id = ?", params.ProjectID)
	}

	var quotas = make([]*quota.GetQuotasOKBodyQuotasItems0, 0)
	sql, args := q.MustSql()
	if err := pgxscan.Select(params.HTTPRequest.Context(), c.pool, &quotas, sql, args...); err != nil {
		panic(err)
	}

	return quota.NewGetQuotasOK().WithPayload(&quota.GetQuotasOKBody{Quotas: quotas})
}

func (c *Controller) GetQuotasDefaultsHandler(_ quota.GetQuotasDefaultsParams, _ any) middleware.Responder {
	return quota.NewGetQuotasDefaultsOK().WithPayload(&quota.GetQuotasDefaultsOKBody{
		Quota: &models.Quota{
			Endpoint: config.Global.Quota.DefaultQuotaEndpoint,
			Service:  config.Global.Quota.DefaultQuotaService,
		},
	})
}

func getQuotaDetails(ctx context.Context, tx pgx.Tx, projectID string, body *quota.GetQuotasProjectIDOKBody) error {
	sql, args, err := db.Select("quota.service", "quota.endpoint", "COUNT(DISTINCT s.id)", "COUNT(DISTINCT e.id)").
		From("quota").
		Where("quota.project_id = ?", projectID).
		LeftJoin("service s ON quota.project_id = s.project_id").
		LeftJoin("endpoint e ON quota.project_id = e.project_id").
		GroupBy("quota.project_id").
		ToSql()

	if err != nil {
		return err
	}

	return tx.QueryRow(ctx, sql, args...).
		Scan(
			&body.Quota.Service,
			&body.Quota.Endpoint,
			&body.QuotaUsage.InUseService,
			&body.QuotaUsage.InUseEndpoint)
}

func insertDefaultQuota(ctx context.Context, tx pgx.Tx, projectID string) error {
	sql, args, err := db.Insert("quota").
		Columns("service", "endpoint", "project_id").
		Values(
			config.Global.Quota.DefaultQuotaService,
			config.Global.Quota.DefaultQuotaEndpoint,
			projectID).
		ToSql()
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, sql, args...)
	return err
}

func (c *Controller) GetQuotasProjectIDHandler(params quota.GetQuotasProjectIDParams, _ any) middleware.Responder {
	q := quota.GetQuotasProjectIDOKBody{
		Quota:      models.Quota{},
		QuotaUsage: models.QuotaUsage{},
	}

	if err := pgx.BeginFunc(context.Background(), c.pool, func(tx pgx.Tx) error {
		err := getQuotaDetails(params.HTTPRequest.Context(), tx, params.ProjectID, &q)
		if err == nil {
			return nil
		}

		if errors.Is(err, pgx.ErrNoRows) {
			// insert default quotas
			if err := insertDefaultQuota(params.HTTPRequest.Context(), tx, params.ProjectID); err != nil {
				return err
			}
			return getQuotaDetails(params.HTTPRequest.Context(), tx, params.ProjectID, &q)
		}
		return err
	}); err != nil {
		panic(err)
	}
	return quota.NewGetQuotasProjectIDOK().WithPayload(&q)
}

func (c *Controller) PutQuotasProjectIDHandler(params quota.PutQuotasProjectIDParams, _ any) middleware.Responder {
	sql, args := db.Insert("quota").
		Columns("service", "endpoint", "project_id").
		Values(params.Body.Service, params.Body.Endpoint, params.ProjectID).
		Suffix("ON CONFLICT (project_id) DO UPDATE SET service = ?, endpoint = ?",
			params.Body.Service, params.Body.Endpoint).
		Suffix("RETURNING service, endpoint").
		MustSql()

	var quotaResponse models.Quota
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &quotaResponse, sql, args...); err != nil {
		panic(err)
	}

	return quota.NewPutQuotasProjectIDOK().WithPayload(&quotaResponse)
}

func (c *Controller) DeleteQuotasProjectIDHandler(params quota.DeleteQuotasProjectIDParams, _ any) middleware.Responder {
	sql, args := db.Delete("quota").Where("project_id = ?", params.ProjectID).MustSql()
	if ct, err := c.pool.Exec(params.HTTPRequest.Context(), sql, args...); err != nil {
		panic(err)
	} else if ct.RowsAffected() == 0 {
		return quota.NewDeleteQuotasProjectIDNotFound()
	}

	return quota.NewDeleteQuotasProjectIDNoContent()
}
