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
	"github.com/go-openapi/strfmt"
	"github.com/jedib0t/go-pretty/table"

	"github.com/sapcc/archer/client/service"
	"github.com/sapcc/archer/models"
)

var ServiceOptions struct {
	ServiceList   `command:"list" description:"List Services"`
	ServiceShow   `command:"show" description:"Show Service"`
	ServiceCreate `command:"create" description:"Create Service"`
	ServiceSet    `command:"set" description:"Update Service"`
	ServiceDelete `command:"delete" description:"Delete Service"`
}

type ServiceList struct{}

func (*ServiceList) Execute(_ []string) error {
	resp, err := ArcherClient.Service.GetService(nil, nil)
	if err != nil {
		return err
	}

	Table.AppendHeader(table.Row{"ID", "Name", "Port", "Enabled", "Project"})
	for _, sv := range resp.Payload.Items {
		Table.AppendRow(table.Row{sv.ID, sv.Name, sv.Port, *sv.Enabled, sv.ProjectID})
	}
	Table.Render()
	return nil
}

type ServiceShow struct {
	Positional struct {
		Service strfmt.UUID `description:"Service to display (ID)"`
	} `positional-args:"yes" required:"yes"`
}

func (*ServiceShow) Execute(_ []string) error {
	params := service.NewGetServiceServiceIDParams().WithServiceID(ServiceOptions.ServiceShow.Positional.Service)
	resp, err := ArcherClient.Service.GetServiceServiceID(params, nil)
	if err != nil {
		return err
	}

	return WriteTable(resp.GetPayload())
}

type ServiceCreate struct {
	Name            string        `short:"n" long:"name" description:"New service name"`
	Description     string        `long:"description" description:"Set service description"`
	Provider        *string       `long:"provider" description:"Provider type" choice:"tenant" choice:"cp"`
	Enable          bool          `long:"enable" description:"Enable service"`
	Disable         bool          `long:"disable" description:"Disable service" optional-value:"false"`
	Network         strfmt.UUID   `long:"network" description:"Network id" required:"true"`
	IPAddresses     []strfmt.IPv4 `long:"ip-address" description:"IP Addresses of the providing service, multiple addresses will be round robin load balanced." required:"true"`
	Port            int32         `long:"port" description:"Port exposed by the service" required:"true"`
	ProxyProtocol   bool          `long:"proxy-protocol" description:"Enable proxy protocol v2."`
	RequireApproval bool          `long:"require-approval" description:"Require explicit project approval for the service owner."`
	Tags            []string      `long:"tag" description:"Tag to be added to the service (repeat option to set multiple tags)"`
	Visibility      *string       `long:"visibility" description:"Set global visibility of the service. For private visibility, RBAC policies can extend the visibility to specific projects" choice:"private" choice:"public"`
}

func (*ServiceCreate) Execute(_ []string) error {
	enabled := ServiceOptions.ServiceCreate.Enable || ServiceOptions.ServiceCreate.Disable

	sv := models.Service{
		Name:            ServiceOptions.ServiceCreate.Name,
		Description:     ServiceOptions.ServiceCreate.Description,
		Provider:        ServiceOptions.ServiceCreate.Provider,
		Enabled:         &enabled,
		NetworkID:       &ServiceOptions.ServiceCreate.Network,
		IPAddresses:     ServiceOptions.ServiceCreate.IPAddresses,
		Port:            ServiceOptions.ServiceCreate.Port,
		ProxyProtocol:   &ServiceOptions.ServiceCreate.ProxyProtocol,
		RequireApproval: &ServiceOptions.ServiceCreate.RequireApproval,
		Tags:            ServiceOptions.ServiceCreate.Tags,
		Visibility:      ServiceOptions.ServiceCreate.Visibility,
	}
	resp, err := ArcherClient.Service.PostService(service.NewPostServiceParams().WithBody(&sv), nil)
	if err != nil {
		return err
	}
	return WriteTable(resp.GetPayload())
}

type ServiceSet struct {
	Positional struct {
		Service strfmt.UUID `positional-arg-name:"endpoint" description:"Service to set (ID)"`
	} `positional-args:"yes" required:"yes"`
	NoTags          bool          `long:"no-tag" description:"Clear tags associated with the service. Specify both --tag and --no-tag to overwrite current tags"`
	Tags            []string      `long:"tag" description:"Tag to be added to the service (repeat option to set multiple tags)"`
	Description     *string       `long:"description" description:"Set service description"`
	Enable          bool          `long:"enable" description:"Enable service"`
	Disable         bool          `long:"disable" description:"Disable service" optional-value:"false"`
	IPAddresses     []strfmt.IPv4 `long:"ip-address" description:"IP Addresses of the providing service, multiple addresses will be round robin load balanced."`
	Name            *string       `long:"name" description:"Service name"`
	Port            *int32        `long:"port" description:"Port exposed by the service"`
	ProxyProtocol   *bool         `long:"proxy-protocol" description:"Enable proxy protocol v2."`
	RequireApproval *bool         `long:"require-approval" description:"Require explicit project approval for the service owner."`
	Visibility      *string       `long:"visibility" description:"Set global visibility of the service. For private visibility, RBAC policies can extend the visibility to specific projects" choice:"private" choice:"public"`
}

func (*ServiceSet) Execute(_ []string) error {
	tags := make([]string, 0)
	if ServiceOptions.ServiceSet.NoTags {
		tags = append(tags, ServiceOptions.ServiceSet.Tags...)
	} else {
		params := service.
			NewGetServiceServiceIDParams().
			WithServiceID(ServiceOptions.ServiceSet.Positional.Service)
		resp, err := ArcherClient.Service.GetServiceServiceID(params, nil)
		if err != nil {
			return err
		}

		tags = append(ServiceOptions.ServiceSet.Tags, resp.Payload.Tags...)
	}

	var enabled *bool
	if ServiceOptions.ServiceSet.Enable {
		t := true
		enabled = &t
	} else if ServiceOptions.ServiceSet.Disable {
		t := false
		enabled = &t
	}
	sv := models.ServiceUpdatable{
		Description:     ServiceOptions.ServiceSet.Description,
		Enabled:         enabled,
		IPAddresses:     ServiceOptions.ServiceSet.IPAddresses,
		Name:            ServiceOptions.ServiceSet.Name,
		Port:            ServiceOptions.ServiceSet.Port,
		ProxyProtocol:   ServiceOptions.ServiceSet.ProxyProtocol,
		RequireApproval: ServiceOptions.ServiceSet.RequireApproval,
		Tags:            tags,
		Visibility:      ServiceOptions.ServiceSet.Visibility,
	}

	params := service.
		NewPutServiceServiceIDParams().
		WithServiceID(ServiceOptions.ServiceSet.Positional.Service).
		WithBody(&sv)
	resp, err := ArcherClient.Service.PutServiceServiceID(params, nil)
	if err != nil {
		return err
	}

	return WriteTable(resp.GetPayload())
}

type ServiceDelete struct {
	Positional struct {
		Service strfmt.UUID `description:"Service to delete (ID)"`
	} `positional-args:"yes" required:"yes"`
}

func (*ServiceDelete) Execute(_ []string) error {
	params := service.
		NewDeleteServiceServiceIDParams().
		WithServiceID(ServiceOptions.ServiceDelete.Positional.Service)
	_, err := ArcherClient.Service.DeleteServiceServiceID(params, nil)
	return err
}

func init() {
	if _, err := Parser.AddCommand("service", "Services",
		"Service Commands.", &ServiceOptions); err != nil {
		panic(err)
	}
}
