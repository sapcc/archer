// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"errors"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/runtime/middleware"
	"github.com/jackc/pgx/v5"

	"github.com/sapcc/archer/v2/internal/db"
	"github.com/sapcc/archer/v2/models"
	"github.com/sapcc/archer/v2/restapi/operations/agent"
)

func (c *Controller) GetAgentsHandler(params agent.GetAgentsParams, _ any) middleware.Responder {
	sql, args := db.Select("agents.*", "COUNT(service.id) AS services").
		From("agents").
		LeftJoin("service ON service.host = agents.host").
		GroupBy("agents.host").
		OrderBy("agents.host ASC").
		MustSql()

	var agentsResponse = make([]*models.Agent, 0)
	if err := pgxscan.Select(params.HTTPRequest.Context(), c.pool, &agentsResponse, sql, args...); err != nil {
		panic(err)
	}

	return agent.NewGetAgentsOK().WithPayload(&agent.GetAgentsOKBody{Items: agentsResponse})
}

func (c *Controller) GetAgentsAgentHostHandler(params agent.GetAgentsAgentHostParams, _ any) middleware.Responder {
	q := db.Select("agents.*", "COUNT(service.id) AS services").
		From("agents").
		LeftJoin("service ON service.host = agents.host").
		Where("agents.host = ?", params.AgentHost).
		GroupBy("agents.host")

	var agentResponse models.Agent
	sql, args := q.MustSql()
	if err := pgxscan.Get(params.HTTPRequest.Context(), c.pool, &agentResponse, sql, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return agent.NewGetAgentsAgentHostNotFound().WithPayload(&models.Error{
				Code:    404,
				Message: "Agent not found.",
			})
		}
		panic(err)
	}

	return agent.NewGetAgentsAgentHostOK().WithPayload(&agentResponse)
}
