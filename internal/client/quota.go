/*
 *   Copyright 2023 SAP SE
 *
 *   Licensed under the Apache License, Version 2.0 (the "License");
 *   you may not use this file except in compliance with the License.
 *   You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 */

package client

import (
	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/sapcc/archer/client/quota"
	"github.com/sapcc/archer/models"
)

var QuotaOptions struct {
	QuotaList         `command:"list" description:"List Quotas"`
	QuotaShow         `command:"show" description:"Show Quota detail"`
	QuotaShowDefaults `command:"defaults" description:"Show Quota defaults"`
	QuotaSet          `command:"set" description:"Set Quota"`
	QuotaDelete       `command:"reset" description:"Reset all Quota of a project"`
}

type QuotaList struct {
	Project *string `long:"project" description:"Project (ID)"`
}

func (*QuotaList) Execute(_ []string) error {
	params := quota.NewGetQuotasParams().WithProjectID(QuotaOptions.QuotaList.Project)
	resp, err := ArcherClient.Quota.GetQuotas(params, nil)
	if err != nil {
		return err
	}

	Table.AppendHeader(table.Row{"Endpoint", "Endpoint in use", "Service", "Service in use", "Project"})
	for _, q := range resp.Payload.Quotas {
		Table.AppendRow(table.Row{q.Endpoint, q.InUseEndpoint, q.Service, q.InUseService, q.ProjectID})
	}
	Table.Render()
	return nil
}

type QuotaShow struct {
	Project string `long:"project" description:"Project (ID)" required:"true"`
}

func (*QuotaShow) Execute(_ []string) error {
	params := quota.NewGetQuotasProjectIDParams().WithProjectID(QuotaOptions.QuotaShow.Project)
	resp, err := ArcherClient.Quota.GetQuotasProjectID(params, nil)
	if err != nil {
		return err
	}

	return WriteTable(resp.GetPayload())
}

type QuotaShowDefaults struct{}

func (*QuotaShowDefaults) Execute(_ []string) error {
	resp, err := ArcherClient.Quota.GetQuotasDefaults(quota.NewGetQuotasDefaultsParams().WithDefaults(), nil)
	if err != nil {
		return err
	}

	return WriteTable(resp.GetPayload().Quota)
}

type QuotaDelete struct {
	Project string `long:"project" description:"Project (ID)" required:"true"`
}

func (*QuotaDelete) Execute(_ []string) error {
	params := quota.NewDeleteQuotasProjectIDParams().WithProjectID(QuotaOptions.QuotaDelete.Project)
	_, err := ArcherClient.Quota.DeleteQuotasProjectID(params, nil)
	return err
}

type QuotaSet struct {
	Project  string `long:"project" description:"Project (ID)" required:"true"`
	Endpoint *int64 `long:"endpoints" description:"The configured endpoint quota limit. A setting of null means it is using the deployment default quota. A setting of -1 means unlimited."`
	Service  *int64 `long:"services" description:"The configured service quota limit. A setting of null means it is using the deployment default quota. A setting of -1 means unlimited."`
}

func (*QuotaSet) Execute(_ []string) error {
	getParams := quota.NewGetQuotasProjectIDParams().WithProjectID(QuotaOptions.QuotaSet.Project)
	getResp, err := ArcherClient.Quota.GetQuotasProjectID(getParams, nil)
	if err != nil {
		return err
	}
	quotas := models.Quota{
		Endpoint: getResp.Payload.Quota.Endpoint,
		Service:  getResp.Payload.Quota.Service,
	}

	if QuotaOptions.QuotaSet.Endpoint != nil {
		quotas.Endpoint = *QuotaOptions.QuotaSet.Endpoint
	}
	if QuotaOptions.QuotaSet.Service != nil {
		quotas.Service = *QuotaOptions.QuotaSet.Service
	}

	params := quota.NewPutQuotasProjectIDParams().
		WithProjectID(QuotaOptions.QuotaSet.Project).
		WithBody(&quotas)
	resp, err := ArcherClient.Quota.PutQuotasProjectID(params, nil)
	if err != nil {
		return err
	}

	return WriteTable(resp.GetPayload())
}

func init() {
	if _, err := Parser.AddCommand("quota", "Quotas",
		"Quota Commands.", &QuotaOptions); err != nil {
		panic(err)
	}
}
