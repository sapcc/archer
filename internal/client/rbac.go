// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"github.com/go-openapi/strfmt"
	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/sapcc/archer/client/rbac"
	"github.com/sapcc/archer/models"
)

var RbacOptions struct {
	RbacList   `command:"list" description:"List RBACs"`
	RbacCreate `command:"create" description:"Create RBACs"`
	RbacShow   `command:"show" description:"Show RBAC policy detail"`
	RbacSet    `command:"set" description:"Set RBAC properties"`
	RbacDelete `command:"delete" description:"Delete RBAC policy"`
}

type RbacList struct{}

func (*RbacList) Execute(_ []string) error {
	params := rbac.NewGetRbacPoliciesParams()
	resp, err := ArcherClient.Rbac.GetRbacPolicies(params, nil)
	if err != nil {
		return err
	}

	Table.AppendHeader(table.Row{"ID", "Target Type", "Target", "Service", "Created", "Updated"})
	for _, r := range resp.Payload.Items {
		Table.AppendRow(table.Row{r.ID, r.TargetType, r.Target, r.ServiceID, r.CreatedAt, r.UpdatedAt})
	}
	Table.Render()
	return nil
}

type RbacCreate struct {
	Service    strfmt.UUID `long:"service" description:"The ID of the service resource." required:"true"`
	Target     string      `long:"target" description:"The ID of the project to which the RBAC policy will be enforced."`
	TargetType string      `long:"target-type" description:"RBAC Policy Target Type." choice:"project"`
}

func (*RbacCreate) Execute(_ []string) error {
	params := rbac.NewPostRbacPoliciesParams().
		WithBody(&models.Rbacpolicy{
			ServiceID:  &RbacOptions.RbacCreate.Service,
			Target:     RbacOptions.RbacCreate.Target,
			TargetType: &RbacOptions.RbacCreate.TargetType,
		})
	resp, err := ArcherClient.Rbac.PostRbacPolicies(params, nil)
	if err != nil {
		return err
	}

	return WriteTable(resp.GetPayload())
}

type RbacShow struct {
	Positional struct {
		RbacPolicy strfmt.UUID `description:"RBAC Policy to display (ID)"`
	} `positional-args:"yes" required:"yes"`
}

func (*RbacShow) Execute(_ []string) error {
	params := rbac.NewGetRbacPoliciesRbacPolicyIDParams().
		WithRbacPolicyID(RbacOptions.RbacShow.Positional.RbacPolicy)
	resp, err := ArcherClient.Rbac.GetRbacPoliciesRbacPolicyID(params, nil)
	if err != nil {
		return err
	}

	return WriteTable(resp.GetPayload())
}

type RbacDelete struct {
	Positional struct {
		RbacPolicy strfmt.UUID `description:"RBAC Policy to display (ID)"`
	} `positional-args:"yes" required:"yes"`
}

func (*RbacDelete) Execute(_ []string) error {
	params := rbac.NewDeleteRbacPoliciesRbacPolicyIDParams().
		WithRbacPolicyID(RbacOptions.RbacDelete.Positional.RbacPolicy)
	_, err := ArcherClient.Rbac.DeleteRbacPoliciesRbacPolicyID(params, nil)
	return err
}

type RbacSet struct {
	Positional struct {
		RbacPolicy strfmt.UUID `description:"RBAC Policy to display (ID)"`
	} `positional-args:"yes" required:"yes"`
	Target     *string `long:"target" description:"The ID of the project to which the RBAC policy will be enforced."`
	TargetType *string `long:"target-type" description:"RBAC Policy Target Type." choice:"project"`
}

func (*RbacSet) Execute(_ []string) error {
	params := rbac.NewPutRbacPoliciesRbacPolicyIDParams().
		WithRbacPolicyID(RbacOptions.RbacSet.Positional.RbacPolicy).
		WithBody(&models.Rbacpolicycommon{
			Target: RbacOptions.RbacSet.Target,
		})
	if RbacOptions.RbacSet.TargetType != nil {
		params.Body.TargetType = RbacOptions.RbacSet.TargetType
	}

	resp, err := ArcherClient.Rbac.PutRbacPoliciesRbacPolicyID(params, nil)
	if err != nil {
		return err
	}

	return WriteTable(resp.GetPayload())
}

func init() {
	if _, err := Parser.AddCommand("rbac", "RBACs",
		"RBAC Policy Commands.", &RbacOptions); err != nil {
		panic(err)
	}
}
