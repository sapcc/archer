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

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/service"
)

var (
	networkId = strfmt.UUID("7e0be670-deb6-45d2-af1a-f8ca524f5ac4")
)

func (t *SuiteTest) createService() strfmt.UUID {
	s := models.Service{
		Name:        "test",
		NetworkID:   &networkId,
		IPAddresses: []strfmt.IPv4{"1.2.3.4"},
	}

	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceCreated{}, res)
	payload := res.(*service.PostServiceCreated).Payload
	assert.Equal(t.T(), networkId, *payload.NetworkID)
	return payload.ID
}

func (t *SuiteTest) TestServiceEmptyGet() {
	// empty get from random uuid
	u := strfmt.UUID("30971832-4f4d-4068-97fb-b0cfe816cae0")

	res := t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: u},
		nil)

	// not found
	assert.IsType(t.T(), &service.GetServiceServiceIDNotFound{}, res)
}

func (t *SuiteTest) TestServicePost() {
	// post and get
	serviceId := t.createService()

	res := t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDOK{}, res)
}

func (t *SuiteTest) TestServicePut() {
	// post and get
	serviceId := t.createService()

	res := t.c.PutServiceServiceIDHandler(
		service.PutServiceServiceIDParams{HTTPRequest: &http.Request{},
			ServiceID: serviceId, Body: &models.ServiceUpdatable{Name: "test2"}},
		nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDOK{}, res)

	res = t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDOK{}, res)
	assert.Equal(t.T(), "test2", res.(*service.GetServiceServiceIDOK).Payload.Name)
}

func (t *SuiteTest) TestServiceDelete() {
	// create, delete, get
	// post and get
	serviceId := t.createService()

	// delete
	res := t.c.DeleteServiceServiceIDHandler(
		service.DeleteServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.DeleteServiceServiceIDNoContent{}, res)

	// not found
	res = t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDNotFound{}, res)
}

func (t *SuiteTest) TestServiceDuplicatePayload() {
	// create, delete, get
	az := "abc"
	s := models.Service{
		Name:             "test",
		NetworkID:        &networkId,
		IPAddresses:      []strfmt.IPv4{"1.2.3.4"},
		AvailabilityZone: &az,
	}

	// post two identical services
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)
	assert.IsType(t.T(), &service.PostServiceCreated{}, res)

	res = t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)
	assert.IsType(t.T(), &service.PostServiceConflict{}, res)

	// create a second service with a different ip
	s.IPAddresses = []strfmt.IPv4{"1.2.3.5"}
	res = t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)
	assert.IsType(t.T(), &service.PostServiceCreated{}, res)
	payload := res.(*service.PostServiceCreated).Payload

	// update to 1.2.3.4 -> conflict
	res = t.c.PutServiceServiceIDHandler(
		service.PutServiceServiceIDParams{HTTPRequest: &http.Request{},
			ServiceID: payload.ID, Body: &models.ServiceUpdatable{IPAddresses: []strfmt.IPv4{"1.2.3.4"}}},
		nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDConflict{}, res)
}

func (t *SuiteTest) TestServiceDeleteInUse() {
	serviceId := t.createService()
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	t.createEndpoint(serviceId, models.EndpointTarget{Network: &network})

	// delete conflict
	res := t.c.DeleteServiceServiceIDHandler(
		service.DeleteServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.DeleteServiceServiceIDConflict{}, res)

	// get ok
	res = t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDOK{}, res)
}
