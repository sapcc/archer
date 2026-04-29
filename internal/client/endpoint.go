// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/sethvargo/go-retry"

	"github.com/sapcc/archer/v2/client/endpoint"
	"github.com/sapcc/archer/v2/client/service"
	"github.com/sapcc/archer/v2/models"
)

var EndpointOptions struct {
	EndpointList   `command:"list" description:"List Endpoints"`
	EndpointShow   `command:"show" description:"Show Endpoint"`
	EndpointCreate `command:"create" description:"Create Endpoint"`
	EndpointSet    `command:"set" description:"Set Endpoint"`
	EndpointDelete `command:"delete" description:"Delete Endpoint"`
}

type EndpointList struct {
	Tags       []string `long:"tags" description:"List endpoints which have all given tag(s) (repeat option for multiple tags)"`
	AnyTags    []string `long:"any-tags" description:"List endpoints which have any given tag(s) (repeat option for multiple tags)"`
	NotTags    []string `long:"not-tags" description:"Exclude endpoints which have all given tag(s) (repeat option for multiple tags)"`
	NotAnyTags []string `long:"not-any-tags" description:"Exclude endpoints which have any given tag(s) (repeat option for multiple tags)"`
	Project    *string  `short:"p" long:"project" description:"List endpoints in the given project (ID)"`
	Service    *string  `short:"s" long:"service" description:"List endpoints for the given service (name or ID)"`
}

type cliEndpoint struct {
	*models.Endpoint
	ServiceName  string  `json:"service_name"`
	ServicePorts []int32 `json:"service_ports"`
}

func (*EndpointList) Execute(_ []string) error {
	var serviceID *strfmt.UUID
	if EndpointOptions.EndpointList.Service != nil {
		id, err := ResolveServiceID(*EndpointOptions.EndpointList.Service)
		if err != nil {
			return err
		}
		serviceID = &id
	}

	params := endpoint.NewGetEndpointParams().
		WithTags(EndpointOptions.EndpointList.Tags).
		WithTagsAny(EndpointOptions.EndpointList.AnyTags).
		WithNotTags(EndpointOptions.EndpointList.NotTags).
		WithNotTagsAny(EndpointOptions.EndpointList.NotAnyTags).
		WithProjectID(EndpointOptions.EndpointList.Project)
	resp, err := ArcherClient.Endpoint.GetEndpoint(params, nil)
	if err != nil {
		return err
	}

	DefaultColumns = []string{"id", "name", "service_id", "target.port", "status", "ip_address"}
	var items []*models.Endpoint
	for _, item := range resp.GetPayload().Items {
		if serviceID == nil || item.ServiceID == *serviceID {
			items = append(items, item)
		}
	}
	return WriteTable(items)
}

type EndpointShow struct {
	Positional struct {
		Endpoint string `positional-arg-name:"endpoint" description:"Endpoint to display (name or ID)"`
	} `positional-args:"yes" required:"yes"`
}

func (*EndpointShow) Execute(_ []string) error {
	endpointID, err := ResolveEndpointID(EndpointOptions.EndpointShow.Positional.Endpoint)
	if err != nil {
		return err
	}

	var e cliEndpoint
	params := endpoint.NewGetEndpointEndpointIDParams().WithEndpointID(endpointID)
	resp, err := ArcherClient.Endpoint.GetEndpointEndpointID(params, nil)
	if err != nil {
		return err
	}
	e.Endpoint = resp.GetPayload()

	if resp, err := ArcherClient.Service.GetServiceServiceID(
		service.NewGetServiceServiceIDParams().WithServiceID(e.ServiceID),
		nil); resp != nil && err == nil {
		e.ServiceName = resp.GetPayload().Name
		e.ServicePorts = resp.GetPayload().Ports
	}

	return WriteTable(e)
}

type EndpointCreate struct {
	Name                string   `short:"n" long:"name" description:"New endpoint name"`
	Description         string   `long:"description" description:"Set endpoint description"`
	Tags                []string `long:"tag" description:"Tag to be added to the endpoint (repeat option to set multiple tags)"`
	Network             *string  `long:"network" description:"Endpoint network (name or ID)"`
	Port                *string  `long:"port" description:"Endpoint port (ID)"`
	Subnet              *string  `long:"subnet" description:"Endpoint subnet (ID)"`
	ConnectionMirroring bool     `long:"connection-mirroring" description:"Enable BIG-IP connection mirroring for HA failover (only affects provider type 'tenant')"`
	Wait                bool     `long:"wait" description:"Wait for endpoint to be ready"`
	Positional          struct {
		Service string `positional-arg-name:"service" description:"Service to reference (name or ID)"`
	} `positional-args:"yes" required:"yes"`
}

func (*EndpointCreate) Execute(_ []string) error {
	serviceID, err := ResolveServiceID(EndpointOptions.EndpointCreate.Positional.Service)
	if err != nil {
		return err
	}

	var networkID, portID, subnetID *strfmt.UUID
	if EndpointOptions.EndpointCreate.Network != nil {
		id, err := ResolveNetworkID(*EndpointOptions.EndpointCreate.Network)
		if err != nil {
			return err
		}
		networkID = &id
	}
	if EndpointOptions.EndpointCreate.Port != nil {
		id := strfmt.UUID(*EndpointOptions.EndpointCreate.Port)
		portID = &id
	}
	if EndpointOptions.EndpointCreate.Subnet != nil {
		id := strfmt.UUID(*EndpointOptions.EndpointCreate.Subnet)
		subnetID = &id
	}

	sv := models.Endpoint{
		Name:                EndpointOptions.EndpointCreate.Name,
		Description:         EndpointOptions.EndpointCreate.Description,
		ServiceID:           serviceID,
		Tags:                EndpointOptions.EndpointCreate.Tags,
		ConnectionMirroring: boolFlag(EndpointOptions.EndpointCreate.ConnectionMirroring, false),
		Target: models.EndpointTarget{
			Network: networkID,
			Port:    portID,
			Subnet:  subnetID,
		},
	}
	resp, err := ArcherClient.Endpoint.PostEndpoint(endpoint.NewPostEndpointParams().WithBody(&sv), nil)
	if err != nil {
		return err
	}
	var res *models.Endpoint
	res = resp.GetPayload()
	if EndpointOptions.EndpointCreate.Wait {
		if res, err = waitForEndpoint(res.ID, false); err != nil {
			return err
		}
	}
	return WriteTable(res)
}

type EndpointDelete struct {
	Positional struct {
		Endpoint string `description:"Endpoint to delete (name or ID)"`
	} `positional-args:"yes" required:"yes"`
	Wait bool `long:"wait" description:"Wait for endpoint to be deleted"`
}

func (*EndpointDelete) Execute(_ []string) error {
	endpointID, err := ResolveEndpointID(EndpointOptions.EndpointDelete.Positional.Endpoint)
	if err != nil {
		return err
	}

	params := endpoint.
		NewDeleteEndpointEndpointIDParams().
		WithEndpointID(endpointID)
	_, err = ArcherClient.Endpoint.DeleteEndpointEndpointID(params, nil)
	if err != nil {
		return err
	}

	if EndpointOptions.EndpointDelete.Wait {
		if _, err = waitForEndpoint(params.EndpointID, true); err != nil {
			return err
		}
	}
	return err
}

type EndpointSet struct {
	Positional struct {
		Endpoint string `positional-arg-name:"endpoint" description:"Endpoint to set (name or ID)"`
	} `positional-args:"yes" required:"yes"`
	Name                  *string  `short:"n" long:"name" description:"New endpoint name"`
	Description           *string  `long:"description" description:"Set endpoint description"`
	ConnectionMirroring   bool     `long:"connection-mirroring" description:"Enable BIG-IP connection mirroring for HA failover (only affects provider type 'tenant')"`
	NoConnectionMirroring bool     `long:"no-connection-mirroring" description:"Disable BIG-IP connection mirroring"`
	NoTags                bool     `long:"no-tag" description:"Clear tags associated with the endpoint. Specify both --tag and --no-tag to overwrite current tags"`
	Tags                  []string `long:"tag" description:"Tag to be added to the endpoint (repeat option to set multiple tags)"`
	Wait                  bool     `long:"wait" description:"Wait for endpoint to be ready"`
}

func (*EndpointSet) Execute(_ []string) error {
	endpointID, err := ResolveEndpointID(EndpointOptions.EndpointSet.Positional.Endpoint)
	if err != nil {
		return err
	}

	tags := make([]string, 0)
	if EndpointOptions.EndpointSet.NoTags {
		tags = append(tags, EndpointOptions.EndpointSet.Tags...)
	} else {
		params := endpoint.
			NewGetEndpointEndpointIDParams().
			WithEndpointID(endpointID)
		resp, err := ArcherClient.Endpoint.GetEndpointEndpointID(params, nil)
		if err != nil {
			return err
		}

		tags = append(EndpointOptions.EndpointSet.Tags, resp.Payload.Tags...)
	}

	params := endpoint.
		NewPutEndpointEndpointIDParams().
		WithEndpointID(endpointID).
		WithBody(endpoint.PutEndpointEndpointIDBody{
			Name:                EndpointOptions.EndpointSet.Name,
			Description:         EndpointOptions.EndpointSet.Description,
			ConnectionMirroring: boolFlag(EndpointOptions.EndpointSet.ConnectionMirroring, EndpointOptions.EndpointSet.NoConnectionMirroring),
			Tags:                tags,
		})
	resp, err := ArcherClient.Endpoint.PutEndpointEndpointID(params, nil)
	if err != nil {
		return err
	}
	var res *models.Endpoint
	res = resp.GetPayload()
	if EndpointOptions.EndpointSet.Wait {
		if res, err = waitForEndpoint(res.ID, false); err != nil {
			return err
		}
	}
	return WriteTable(res)
}

func init() {
	if _, err := Parser.AddCommand("endpoint", "Endpoints",
		"Endpoint Commands.", &EndpointOptions); err != nil {
		panic(err)
	}
}

func waitForEndpoint(id strfmt.UUID, deleted bool) (*models.Endpoint, error) {
	var res *models.Endpoint
	b := retry.NewConstant(1 * time.Second)
	b = retry.WithMaxDuration(opts.Timeout, b)
	if err := retry.Do(context.Background(), b, func(ctx context.Context) error {
		params := endpoint.NewGetEndpointEndpointIDParams().WithEndpointID(id)
		r, err := ArcherClient.Endpoint.GetEndpointEndpointID(params, nil)
		if err != nil {
			var getEndpointEndpointIDNotFound *endpoint.GetEndpointEndpointIDNotFound
			if errors.As(err, &getEndpointEndpointIDNotFound) && deleted {
				// endpoint deleted
				return nil
			}
			return err
		}

		res = r.GetPayload()
		if deleted || res.Status != models.EndpointStatusAVAILABLE {
			return retry.RetryableError(errors.New("endpoint not processed"))
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return res, nil
}
