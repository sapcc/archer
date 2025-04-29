// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"net/http"

	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/quota"
)

func (t *SuiteTest) createQuota(projectId string) {
	q := models.Quota{
		Endpoint: 1,
		Service:  2,
	}

	res := t.c.PutQuotasProjectIDHandler(quota.PutQuotasProjectIDParams{
		HTTPRequest: &http.Request{},
		ProjectID:   projectId,
		Body:        &q,
	}, nil)

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &quota.PutQuotasProjectIDOK{}, res)
	payload := res.(*quota.PutQuotasProjectIDOK).Payload
	assert.EqualValues(t.T(), 1, payload.Endpoint)
	assert.EqualValues(t.T(), 2, payload.Service)
}

func (t *SuiteTest) TestQuotaGet() {
	projectId := "abcd12345"
	t.createQuota(projectId)

	res := t.c.GetQuotasHandler(
		quota.GetQuotasParams{
			HTTPRequest: &http.Request{},
		}, nil)
	assert.IsType(t.T(), &quota.GetQuotasOK{}, res)
	payload := res.(*quota.GetQuotasOK).Payload
	assert.EqualValues(t.T(), 1, payload.Quotas[0].Quota.Endpoint)
	assert.EqualValues(t.T(), 2, payload.Quotas[0].Quota.Service)
}

func (t *SuiteTest) TestQuotaPut() {
	t.createQuota("test123456")
}

func (t *SuiteTest) TestQuotaDefaultGet() {
	config.Global.Quota.DefaultQuotaEndpoint = 666
	config.Global.Quota.DefaultQuotaService = 42

	res := t.c.GetQuotasDefaultsHandler(
		quota.GetQuotasDefaultsParams{
			HTTPRequest: &http.Request{},
		}, nil)
	assert.IsType(t.T(), &quota.GetQuotasDefaultsOK{}, res)
	payload := res.(*quota.GetQuotasDefaultsOK).Payload
	assert.EqualValues(t.T(), config.Global.Quota.DefaultQuotaEndpoint, payload.Quota.Endpoint)
	assert.EqualValues(t.T(), config.Global.Quota.DefaultQuotaService, payload.Quota.Service)
}

func (t *SuiteTest) TestQuotaPutGet() {
	projectId := "abcd12345"
	t.createQuota(projectId)

	res := t.c.GetQuotasProjectIDHandler(
		quota.GetQuotasProjectIDParams{
			HTTPRequest: &http.Request{},
			ProjectID:   projectId,
		}, nil)
	assert.IsType(t.T(), &quota.GetQuotasProjectIDOK{}, res)
	payload := res.(*quota.GetQuotasProjectIDOK).Payload
	assert.EqualValues(t.T(), 1, payload.Quota.Endpoint)
	assert.EqualValues(t.T(), 2, payload.Quota.Service)
	assert.EqualValues(t.T(), 0, payload.QuotaUsage.InUseEndpoint)
	assert.EqualValues(t.T(), 0, payload.QuotaUsage.InUseService)
}

func (t *SuiteTest) TestResetQuotas() {
	projectId := "abcd12345"
	t.createQuota(projectId)

	res := t.c.DeleteQuotasProjectIDHandler(
		quota.DeleteQuotasProjectIDParams{
			HTTPRequest: &http.Request{},
			ProjectID:   projectId,
		}, nil)
	assert.IsType(t.T(), &quota.DeleteQuotasProjectIDNoContent{}, res)

	res = t.c.GetQuotasProjectIDHandler(
		quota.GetQuotasProjectIDParams{
			HTTPRequest: &http.Request{},
			ProjectID:   projectId,
		}, nil)
	assert.IsType(t.T(), &quota.GetQuotasProjectIDOK{}, res)
}
