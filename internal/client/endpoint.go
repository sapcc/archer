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

	"github.com/sapcc/archer/client/endpoint"
	"github.com/sapcc/archer/models"
)

var EndpointOptions struct {
	EndpointList   `command:"list" description:"List Endpoints"`
	EndpointShow   `command:"show" description:"Show Endpoint"`
	EndpointCreate `command:"create" description:"Create Endpoint"`
	EndpointSet    `command:"set" description:"Set Endpoint"`
	EndpointDelete `command:"delete" description:"Delete Endpoint"`
}

type EndpointList struct{}

func (*EndpointList) Execute(_ []string) error {
	resp, err := ArcherClient.Endpoint.GetEndpoint(nil, nil)
	if err != nil {
		return err
	}

	Table.AppendHeader(table.Row{"ID", "Service ID", "Service Name", "Port", "Status", "Project"})
	for _, ep := range resp.Payload.Items {
		Table.AppendRow(table.Row{ep.ID, ep.ServiceID, ep.ServiceName, ep.Target.Port, ep.Status, ep.ProjectID})
	}
	Table.Render()
	return nil
}

type EndpointShow struct {
	Positional struct {
		Endpoint strfmt.UUID `positional-arg-name:"endpoint" description:"Endpoint to display (ID)"`
	} `positional-args:"yes" required:"yes"`
}

func (*EndpointShow) Execute(_ []string) error {
	params := endpoint.NewGetEndpointEndpointIDParams().WithEndpointID(EndpointOptions.EndpointShow.Positional.Endpoint)
	resp, err := ArcherClient.Endpoint.GetEndpointEndpointID(params, nil)
	if err != nil {
		return err
	}

	return WriteTable(resp.GetPayload())
}

type EndpointCreate struct {
	Tags       []string     `long:"tag" description:"Tag to be added to the endpoint (repeat option to set multiple tags)"`
	Network    *strfmt.UUID `long:"network" description:"Endpoint network (ID)"`
	Port       *strfmt.UUID `long:"port" description:"Endpoint port (ID)"`
	Subnet     *strfmt.UUID `long:"subnet" description:"Endpoint subnet (ID)"`
	Positional struct {
		Service strfmt.UUID `positional-arg-name:"service" description:"Service to reference (ID)"`
	} `positional-args:"yes" required:"yes"`
}

func (*EndpointCreate) Execute(_ []string) error {
	sv := models.Endpoint{
		ServiceID: EndpointOptions.EndpointCreate.Positional.Service,
		Tags:      EndpointOptions.EndpointCreate.Tags,
		Target: models.EndpointTarget{
			Network: EndpointOptions.Network,
			Port:    EndpointOptions.Port,
			Subnet:  EndpointOptions.Subnet,
		},
	}
	resp, err := ArcherClient.Endpoint.PostEndpoint(endpoint.NewPostEndpointParams().WithBody(&sv), nil)
	if err != nil {
		return err
	}
	return WriteTable(resp.GetPayload())
}

type EndpointDelete struct {
	Positional struct {
		Endpoint strfmt.UUID `description:"Endpoint to set (ID)"`
	} `positional-args:"yes" required:"yes"`
}

func (*EndpointDelete) Execute(_ []string) error {
	params := endpoint.
		NewDeleteEndpointEndpointIDParams().
		WithEndpointID(EndpointOptions.EndpointDelete.Positional.Endpoint)
	_, err := ArcherClient.Endpoint.DeleteEndpointEndpointID(params, nil)
	return err
}

type EndpointSet struct {
	Positional struct {
		Endpoint strfmt.UUID `positional-arg-name:"endpoint" description:"Endpoint to set (ID)"`
	} `positional-args:"yes" required:"yes"`
	NoTags bool     `long:"no-tag" description:"Clear tags associated with the endpoint. Specify both --tag and --no-tag to overwrite current tags"`
	Tags   []string `long:"tag" description:"Tag to be added to the endpoint (repeat option to set multiple tags)"`
}

func (*EndpointSet) Execute(_ []string) error {
	tags := make([]string, 0)
	if EndpointOptions.EndpointSet.NoTags {
		tags = append(tags, EndpointOptions.EndpointSet.Tags...)
	} else {
		params := endpoint.
			NewGetEndpointEndpointIDParams().
			WithEndpointID(EndpointOptions.EndpointSet.Positional.Endpoint)
		resp, err := ArcherClient.Endpoint.GetEndpointEndpointID(params, nil)
		if err != nil {
			return err
		}

		tags = append(EndpointOptions.EndpointSet.Tags, resp.Payload.Tags...)
	}

	params := endpoint.
		NewPutEndpointEndpointIDParams().
		WithEndpointID(EndpointOptions.EndpointSet.Positional.Endpoint).
		WithBody(endpoint.PutEndpointEndpointIDBody{Tags: tags})
	resp, err := ArcherClient.Endpoint.PutEndpointEndpointID(params, nil)
	if err != nil {
		return err
	}

	return WriteTable(resp.GetPayload())
}

func init() {
	if _, err := Parser.AddCommand("endpoint", "Endpoints",
		"Endpoint Commands.", &EndpointOptions); err != nil {
		panic(err)
	}
}
