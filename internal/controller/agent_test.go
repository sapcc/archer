// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/restapi/operations/agent"
)

func (t *SuiteTest) TestGetAgentsHandlerEmpty() {
	res := t.c.GetAgentsHandler(
		agent.GetAgentsParams{HTTPRequest: &http.Request{}},
		nil)

	assert.IsType(t.T(), &agent.GetAgentsOK{}, res)
	payload := res.(*agent.GetAgentsOK).Payload
	assert.NotNil(t.T(), payload.Items)
	assert.Len(t.T(), payload.Items, 0)
}

func (t *SuiteTest) TestGetAgentsHandler() {
	// Add agents
	t.addAgentWithHost("agent-host-1", nil)
	t.addAgentWithHost("agent-host-2", nil)

	res := t.c.GetAgentsHandler(
		agent.GetAgentsParams{HTTPRequest: &http.Request{}},
		nil)

	assert.IsType(t.T(), &agent.GetAgentsOK{}, res)
	payload := res.(*agent.GetAgentsOK).Payload
	assert.NotNil(t.T(), payload.Items)
	assert.Len(t.T(), payload.Items, 2)
	// Should be ordered by host name
	assert.Equal(t.T(), "agent-host-1", payload.Items[0].Host)
	assert.Equal(t.T(), "agent-host-2", payload.Items[1].Host)
}

func (t *SuiteTest) TestGetAgentsHandlerWithServices() {
	// Add agent
	t.addAgent(nil)

	// Create a service assigned to this agent
	_ = t.createService(testService)

	res := t.c.GetAgentsHandler(
		agent.GetAgentsParams{HTTPRequest: &http.Request{}},
		nil)

	assert.IsType(t.T(), &agent.GetAgentsOK{}, res)
	payload := res.(*agent.GetAgentsOK).Payload
	assert.Len(t.T(), payload.Items, 1)
	assert.Equal(t.T(), "test-host", payload.Items[0].Host)
	assert.Equal(t.T(), int64(1), payload.Items[0].Services)
}

func (t *SuiteTest) TestGetAgentsAgentHostHandler() {
	// Add agent
	t.addAgentWithHost("specific-agent-host", nil)

	res := t.c.GetAgentsAgentHostHandler(
		agent.GetAgentsAgentHostParams{
			HTTPRequest: &http.Request{},
			AgentHost:   "specific-agent-host",
		},
		nil)

	assert.IsType(t.T(), &agent.GetAgentsAgentHostOK{}, res)
	payload := res.(*agent.GetAgentsAgentHostOK).Payload
	assert.Equal(t.T(), "specific-agent-host", payload.Host)
}

func (t *SuiteTest) TestGetAgentsAgentHostHandlerNotFound() {
	res := t.c.GetAgentsAgentHostHandler(
		agent.GetAgentsAgentHostParams{
			HTTPRequest: &http.Request{},
			AgentHost:   "non-existent-host",
		},
		nil)

	assert.IsType(t.T(), &agent.GetAgentsAgentHostNotFound{}, res)
	payload := res.(*agent.GetAgentsAgentHostNotFound).Payload
	assert.Equal(t.T(), int64(404), payload.Code)
	assert.Equal(t.T(), "Agent not found.", payload.Message)
}

func (t *SuiteTest) TestGetAgentsAgentHostHandlerWithServices() {
	// Add agent
	t.addAgent(nil)

	// Create services assigned to this agent
	_ = t.createService(testService)
	testService2 := testService
	testService2.IPAddresses = []strfmt.IPv4{"2.3.4.5"}
	_ = t.createService(testService2)

	res := t.c.GetAgentsAgentHostHandler(
		agent.GetAgentsAgentHostParams{
			HTTPRequest: &http.Request{},
			AgentHost:   "test-host",
		},
		nil)

	assert.IsType(t.T(), &agent.GetAgentsAgentHostOK{}, res)
	payload := res.(*agent.GetAgentsAgentHostOK).Payload
	assert.Equal(t.T(), "test-host", payload.Host)
	assert.Equal(t.T(), int64(2), payload.Services)
}

func (t *SuiteTest) TestGetAgentsAgentHostHandlerWithAZ() {
	// Add agent with specific AZ
	az := "test-az"
	sql, args := db.Insert("agents").
		Columns("host", "availability_zone", "physnet", "heartbeat_at", "provider").
		Values("az-agent-host", &az, "physnet1", sq.Expr("NOW()"), "tenant").
		Suffix("ON CONFLICT DO NOTHING").
		MustSql()
	if _, err := t.c.pool.Exec(context.Background(), sql, args...); err != nil {
		t.FailNow("Failed inserting agent host", err)
	}

	res := t.c.GetAgentsAgentHostHandler(
		agent.GetAgentsAgentHostParams{
			HTTPRequest: &http.Request{},
			AgentHost:   "az-agent-host",
		},
		nil)

	assert.IsType(t.T(), &agent.GetAgentsAgentHostOK{}, res)
	payload := res.(*agent.GetAgentsAgentHostOK).Payload
	assert.Equal(t.T(), "az-agent-host", payload.Host)
	assert.NotNil(t.T(), payload.AvailabilityZone)
	assert.Equal(t.T(), "test-az", *payload.AvailabilityZone)
}
