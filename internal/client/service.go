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
	"context"
	"errors"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/sethvargo/go-retry"

	"github.com/sapcc/archer/client/service"
	"github.com/sapcc/archer/models"
)

var ServiceOptions struct {
	ServiceList     `command:"list" description:"List Services"`
	ServiceEndpoint `command:"endpoint" description:"Service Endpoint Commands"`
	ServiceShow     `command:"show" description:"Show Service"`
	ServiceCreate   `command:"create" description:"Create Service"`
	ServiceSet      `command:"set" description:"Update Service"`
	ServiceDelete   `command:"delete" description:"Delete Service"`
}

type ServiceList struct {
	Tags       []string `long:"tags" description:"List services which have all given tag(s) (repeat option for multiple tags)"`
	AnyTags    []string `long:"any-tags" description:"List services which have any given tag(s) (repeat option for multiple tags)"`
	NotTags    []string `long:"not-tags" description:"Exclude services which have all given tag(s) (repeat option for multiple tags)"`
	NotAnyTags []string `long:"not-any-tags" description:"Exclude services which have any given tag(s) (repeat option for multiple tags)"`
	Project    *string  `short:"p" long:"project" description:"List services in the given project (ID)"`
}

func (*ServiceList) Execute(_ []string) error {
	params := service.NewGetServiceParams().
		WithTags(ServiceOptions.ServiceList.Tags).
		WithTagsAny(ServiceOptions.ServiceList.AnyTags).
		WithNotTags(ServiceOptions.ServiceList.NotTags).
		WithNotTagsAny(ServiceOptions.ServiceList.NotAnyTags).
		WithProjectID(ServiceOptions.ServiceList.Project)
	resp, err := ArcherClient.Service.GetService(params, nil)
	if err != nil {
		return err
	}
	DefaultColumns = []string{"id", "name", "port", "enabled", "provider", "status", "visibility", "availability_zone", "project_id"}
	return WriteTable(resp.GetPayload().Items)
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
	Name             string        `short:"n" long:"name" description:"New service name"`
	Description      string        `long:"description" description:"Set service description"`
	Provider         *string       `long:"provider" description:"Provider type" choice:"tenant" choice:"cp"`
	Enable           bool          `long:"enable" description:"Enable service"`
	Disable          bool          `long:"disable" description:"Disable service"`
	Network          strfmt.UUID   `long:"network" description:"Network id" required:"true"`
	IPAddresses      []strfmt.IPv4 `long:"ip-address" description:"IP Addresses of the providing service, multiple addresses will be round robin load balanced." required:"true"`
	Port             int32         `long:"port" description:"Port exposed by the service" required:"true"`
	ProxyProtocol    bool          `long:"proxy-protocol" description:"Enable proxy protocol v2."`
	RequireApproval  bool          `long:"require-approval" description:"Require explicit project approval for the service owner."`
	Tags             []string      `long:"tag" description:"Tag to be added to the service (repeat option to set multiple tags)"`
	Visibility       *string       `long:"visibility" description:"Set global visibility of the service. For private visibility, RBAC policies can extend the visibility to specific projects" choice:"private" choice:"public"`
	Wait             bool          `long:"wait" description:"Wait for service to be ready"`
	AvailabilityZone *string       `long:"availability-zone" description:"Availability zone for the service"`
}

func (*ServiceCreate) Execute(_ []string) error {
	enabled := ServiceOptions.ServiceCreate.Enable || !ServiceOptions.ServiceCreate.Disable

	sv := models.Service{
		Name:             ServiceOptions.ServiceCreate.Name,
		Description:      ServiceOptions.ServiceCreate.Description,
		Provider:         ServiceOptions.ServiceCreate.Provider,
		Enabled:          &enabled,
		NetworkID:        &ServiceOptions.ServiceCreate.Network,
		IPAddresses:      ServiceOptions.ServiceCreate.IPAddresses,
		Port:             ServiceOptions.ServiceCreate.Port,
		ProxyProtocol:    &ServiceOptions.ServiceCreate.ProxyProtocol,
		RequireApproval:  &ServiceOptions.ServiceCreate.RequireApproval,
		Tags:             ServiceOptions.ServiceCreate.Tags,
		Visibility:       ServiceOptions.ServiceCreate.Visibility,
		AvailabilityZone: ServiceOptions.ServiceCreate.AvailabilityZone,
	}
	resp, err := ArcherClient.Service.PostService(service.NewPostServiceParams().WithBody(&sv), nil)
	if err != nil {
		return err
	}

	var res *models.Service
	res = resp.GetPayload()
	if ServiceOptions.ServiceCreate.Wait {
		if res, err = waitForService(res.ID, false); err != nil {
			return err
		}
	}
	return WriteTable(res)
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
	Wait            bool          `long:"wait" description:"Wait for service to be ready"`
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
			var getServiceServiceIDNotFound *service.GetServiceServiceIDNotFound
			if errors.As(err, &getServiceServiceIDNotFound) {
				return errors.New("Not Found")
			}

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

	var res *models.Service
	res = resp.GetPayload()
	if ServiceOptions.ServiceSet.Wait {
		if res, err = waitForService(res.ID, false); err != nil {
			return err
		}
	}
	return WriteTable(res)
}

type ServiceDelete struct {
	Positional struct {
		Service strfmt.UUID `description:"Service to delete (ID)"`
	} `positional-args:"yes" required:"yes"`
	Wait bool `long:"wait" description:"Wait for endpoint to be deleted"`
}

func (*ServiceDelete) Execute(_ []string) error {
	params := service.
		NewDeleteServiceServiceIDParams().
		WithServiceID(ServiceOptions.ServiceDelete.Positional.Service)
	_, err := ArcherClient.Service.DeleteServiceServiceID(params, nil)
	if err != nil {
		return err
	}

	if ServiceOptions.ServiceDelete.Wait {
		if _, err = waitForService(params.ServiceID, true); err != nil {
			return err
		}
	}
	return err
}

type ServiceEndpoint struct {
	Service               strfmt.UUID `long:"service" description:"Service" required:"true"`
	ServiceEndpointList   `command:"list" description:"List Service Endpoints"`
	ServiceEndpointAccept `command:"accept" description:"Accept Service Endpoint"`
	ServiceEndpointReject `command:"reject" description:"Reject Service Endpoint"`
}

type ServiceEndpointList struct {
}

func (*ServiceEndpointList) Execute(_ []string) error {
	params := service.NewGetServiceServiceIDEndpointsParams().
		WithServiceID(ServiceOptions.Service)
	resp, err := ArcherClient.Service.GetServiceServiceIDEndpoints(params, nil)
	if err != nil {
		return err
	}

	Table.AppendHeader(table.Row{"ID", "Project", "Status", "Service"})
	for _, ep := range resp.Payload.Items {
		Table.AppendRow(table.Row{ep.ID, ep.ProjectID, ep.Status, ServiceOptions.Service})
	}
	Table.Render()
	return nil
}

type ServiceEndpointAccept struct {
	Endpoints []strfmt.UUID `long:"endpoint" description:"Accept endpoint (repeat option to accept multiple endpoints)"`
	Projects  []strfmt.UUID `long:"project" description:"Accept all endpoints of project (repeat option to accept multiple projects)"`
}

func (*ServiceEndpointAccept) Execute(_ []string) error {
	var projects []models.Project
	for _, project := range ServiceOptions.ServiceEndpointAccept.Projects {
		projects = append(projects, models.Project(project.String()))
	}
	consumerList := models.EndpointConsumerList{
		EndpointIds: ServiceOptions.ServiceEndpointAccept.Endpoints,
		ProjectIds:  projects,
	}

	params := service.
		NewPutServiceServiceIDAcceptEndpointsParams().
		WithServiceID(ServiceOptions.Service).
		WithBody(&consumerList)
	resp, err := ArcherClient.Service.PutServiceServiceIDAcceptEndpoints(params, nil)
	if err != nil {
		return err
	}

	Table.AppendHeader(table.Row{"ID", "Project", "Status", "Service"})
	for _, ep := range resp.Payload {
		Table.AppendRow(table.Row{ep.ID, ep.ProjectID, ep.Status, ServiceOptions.Service})
	}
	Table.Render()
	return nil
}

type ServiceEndpointReject struct {
	Endpoints []strfmt.UUID `long:"endpoint" description:"Reject endpoint (repeat option to reject multiple endpoints)"`
	Projects  []strfmt.UUID `long:"project" description:"Reject all endpoints of project (repeat option to reject multiple projects)"`
}

func (*ServiceEndpointReject) Execute(_ []string) error {
	var projects []models.Project
	for _, project := range ServiceOptions.ServiceEndpointReject.Projects {
		projects = append(projects, models.Project(project.String()))
	}
	consumerList := models.EndpointConsumerList{
		EndpointIds: ServiceOptions.ServiceEndpointReject.Endpoints,
		ProjectIds:  projects,
	}

	params := service.
		NewPutServiceServiceIDRejectEndpointsParams().
		WithServiceID(ServiceOptions.Service).
		WithBody(&consumerList)
	resp, err := ArcherClient.Service.PutServiceServiceIDRejectEndpoints(params, nil)
	if err != nil {
		return err
	}

	Table.AppendHeader(table.Row{"ID", "Project", "Status", "Service"})
	for _, ep := range resp.Payload {
		Table.AppendRow(table.Row{ep.ID, ep.ProjectID, ep.Status, ServiceOptions.Service})
	}
	Table.Render()
	return nil
}

func init() {
	if _, err := Parser.AddCommand("service", "Services",
		"Service Commands.", &ServiceOptions); err != nil {
		panic(err)
	}
}

func waitForService(id strfmt.UUID, deleted bool) (*models.Service, error) {
	var res *models.Service

	b := retry.NewConstant(1 * time.Second)
	b = retry.WithMaxDuration(60*time.Second, b)
	if err := retry.Do(context.Background(), b, func(ctx context.Context) error {
		params := service.NewGetServiceServiceIDParams().WithServiceID(id)
		r, err := ArcherClient.Service.GetServiceServiceID(params, nil)
		if err != nil {
			var getServiceServiceIDNotFound *service.GetServiceServiceIDNotFound
			if errors.As(err, &getServiceServiceIDNotFound) && deleted {
				return nil
			}
			return err
		}

		res = r.GetPayload()
		if deleted || res.Status != "AVAILABLE" {
			return retry.RetryableError(errors.New("service not processed"))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}
