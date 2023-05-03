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
	"net/http"
	"net/url"

	"github.com/go-openapi/strfmt"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/endpoint"
)

func (t *SuiteTest) createEndpoint(serviceId strfmt.UUID, target models.EndpointTarget) *models.Endpoint {
	s := models.Endpoint{
		ServiceID: serviceId,
		Target:    target,
	}

	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointCreated{}, res)

	return res.(*endpoint.PostEndpointCreated).Payload
}

func (t *SuiteTest) TestEndpointList() {
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	payload := t.createEndpoint(t.createService(), models.EndpointTarget{
		Network: &network,
	})

	res := t.c.GetEndpointHandler(endpoint.GetEndpointParams{HTTPRequest: &http.Request{URL: new(url.URL)}},
		nil)
	assert.IsType(t.T(), &endpoint.GetEndpointOK{}, res)
	endpoints := res.(*endpoint.GetEndpointOK)
	assert.Len(t.T(), endpoints.Payload.Items, 1)
	assert.Equal(t.T(), endpoints.Payload.Items[0].ID, payload.ID)
	assert.NotNil(t.T(), endpoints.Payload.Items[0].Target.Network)
	assert.Equal(t.T(), endpoints.Payload.Items[0].Target.Network.String(), network.String())
}

func (t *SuiteTest) TestEndpointEmptyGet() {
	// empty get from random uuid
	u := strfmt.UUID("30971832-4f4d-4068-97fb-b0cfe816cae0")

	res := t.c.GetEndpointEndpointIDHandler(
		endpoint.GetEndpointEndpointIDParams{HTTPRequest: &http.Request{}, EndpointID: u},
		nil)

	// not found
	assert.IsType(t.T(), &endpoint.GetEndpointEndpointIDNotFound{}, res)
}

func (t *SuiteTest) TestEndpointPostMissingTarget() {
	s := models.Endpoint{
		ServiceID: t.createService(),
	}

	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointBadRequest{}, res)
}

func (t *SuiteTest) TestEndpointPost() {
	// post and get
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	payload := t.createEndpoint(t.createService(), models.EndpointTarget{
		Network: &network,
	})

	assert.Equal(t.T(), network, *payload.Target.Network)

	res := t.c.GetEndpointEndpointIDHandler(
		endpoint.GetEndpointEndpointIDParams{HTTPRequest: &http.Request{}, EndpointID: payload.ID},
		nil)
	assert.IsType(t.T(), &endpoint.GetEndpointEndpointIDOK{}, res)
}

func (t *SuiteTest) TestEndpointPut() {
	// post and get
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	payload := t.createEndpoint(t.createService(), models.EndpointTarget{
		Network: &network,
	})

	res := t.c.PutEndpointEndpointIDHandler(
		endpoint.PutEndpointEndpointIDParams{HTTPRequest: &http.Request{}, EndpointID: payload.ID,
			Body: endpoint.PutEndpointEndpointIDBody{Tags: []string{"a", "b", "c"}}},
		nil)
	assert.IsType(t.T(), &endpoint.PutEndpointEndpointIDOK{}, res)
	assert.EqualValues(t.T(), []string{"a", "b", "c"}, res.(*endpoint.PutEndpointEndpointIDOK).Payload.Tags)
	assert.Equal(t.T(), network, *res.(*endpoint.PutEndpointEndpointIDOK).Payload.Target.Network)
}

func (t *SuiteTest) TestEndpointDelete() {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	// create, delete, get
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	payload := t.createEndpoint(t.createService(), models.EndpointTarget{
		Network: &network,
	})

	// delete
	res := t.c.DeleteEndpointEndpointIDHandler(
		endpoint.DeleteEndpointEndpointIDParams{HTTPRequest: &http.Request{}, EndpointID: payload.ID},
		nil)
	assert.IsType(t.T(), &endpoint.DeleteEndpointEndpointIDAccepted{}, res)

	// pending delete
	res = t.c.GetEndpointEndpointIDHandler(
		endpoint.GetEndpointEndpointIDParams{HTTPRequest: &http.Request{}, EndpointID: payload.ID},
		nil)
	assert.IsType(t.T(), &endpoint.GetEndpointEndpointIDOK{}, res)
	p2 := res.(*endpoint.GetEndpointEndpointIDOK).Payload
	assert.NotNil(t.T(), p2)
	assert.Equal(t.T(), models.EndpointStatusPENDINGDELETE, p2.Status)
}
