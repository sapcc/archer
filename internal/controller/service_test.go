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
	"context"
	"fmt"
	"net/http"
	"net/url"

	policy "github.com/databus23/goslo.policy"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/gophercloud/gophercloud/testhelper/fixture"
	"github.com/sapcc/go-bits/gopherpolicy"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/endpoint"
	"github.com/sapcc/archer/restapi/operations/service"
)

var (
	networkId      = strfmt.UUID("7e0be670-deb6-45d2-af1a-f8ca524f5ac4")
	testProject1   = models.Project("test-project-1")
	testProject2   = models.Project("test-project-2")
	headerProject1 = http.Request{Header: http.Header{"X-Project-Id": []string{string(testProject1)}}, URL: &url.URL{}}
	headerProject2 = http.Request{Header: http.Header{"X-Project-Id": []string{string(testProject2)}}, URL: &url.URL{}}
	testService    = models.Service{
		Name:        "test",
		NetworkID:   &networkId,
		IPAddresses: []strfmt.IPv4{"1.2.3.4"},
		ProjectID:   testProject1,
	}
)

func (t *SuiteTest) addAgent(az string) {
	sql := `INSERT INTO agents (host, availability_zone) VALUES ('test-host', '%s')`
	if _, err := t.c.pool.Exec(context.Background(), fmt.Sprintf(sql, az)); err != nil {
		t.FailNow("Failed inserting agent host", err)
	}
}

func (t *SuiteTest) createService() strfmt.UUID {
	t.addAgent("")
	fixture.SetupHandler(t.T(), "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1, Body: &testService},
		nil)

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceCreated{}, res)
	payload := res.(*service.PostServiceCreated).Payload
	assert.Equal(t.T(), networkId, *payload.NetworkID)
	return payload.ID
}

func (t *SuiteTest) TestGetServiceHandler() {
	serviceId := t.createService()

	header := headerProject1
	header.URL = new(url.URL)
	res := t.c.GetServiceHandler(service.GetServiceParams{HTTPRequest: &header}, nil)
	assert.IsType(t.T(), &service.GetServiceOK{}, res)
	services := res.(*service.GetServiceOK)
	assert.Len(t.T(), services.Payload.Items, 1)
	assert.Equal(t.T(), serviceId, services.Payload.Items[0].ID)
	assert.Equal(t.T(), &networkId, services.Payload.Items[0].NetworkID)
}

func (t *SuiteTest) TestGetServiceHandlerUnknownSortColumn() {
	header := headerProject1
	header.URL = new(url.URL)
	res := t.c.GetServiceHandler(service.GetServiceParams{HTTPRequest: &header, Sort: swag.String("unknown")}, nil)
	assert.IsType(t.T(), &service.GetServiceBadRequest{}, res)
	assert.Equal(t.T(), "Unknown sort column.", res.(*service.GetServiceBadRequest).Payload.Message)
}

func (t *SuiteTest) TestGetServiceHandlerEmptyGet() {
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

func (t *SuiteTest) TestServicePostScoped() {
	serviceId := t.createService()

	res := t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDOK{}, res)
}

func (t *SuiteTest) TestServicePostQuotaMet() {
	config.Global.Quota.Enabled = true
	config.Global.Quota.DefaultQuotaService = 0
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1, Body: &testService},
		nil)
	config.Global.Quota.Enabled = false
	fmt.Print(res)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceForbidden{}, res)
	assert.Equal(t.T(), "Quota has been met for Resource: service", res.(*service.PostServiceForbidden).Payload.Message)
}

func (t *SuiteTest) TestServicePostNetworkNotFound() {
	testServiceUnknownNetwork := testService
	unknownNetwork := strfmt.UUID("c655688a-f4e3-4117-a4fe-30f73fce2950")
	testServiceUnknownNetwork.NetworkID = &unknownNetwork
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1,
		Body: &testServiceUnknownNetwork}, nil)

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceConflict{}, res)
	assert.Equal(t.T(), "Network not found.", res.(*service.PostServiceConflict).Payload.Message)
}

func (t *SuiteTest) TestServicePostNetworkNotAccessible() {
	fixture.SetupHandler(t.T(), "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject2,
		Body: &testService}, nil)

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceConflict{}, res)
	assert.Equal(t.T(), "Network not accessible.", res.(*service.PostServiceConflict).Payload.Message)
}

type TestEnforcerDenyAll struct{}

func (t *TestEnforcerDenyAll) Enforce(_ string, _ policy.Context) bool {
	return false
}

func (t *SuiteTest) TestServicePostNotTenant() {
	testServiceNotTenantProvider := testService
	testServiceNotTenantProvider.Provider = swag.String("cp")
	fixture.SetupHandler(t.T(), "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1,
		Body: &testServiceNotTenantProvider}, &gopherpolicy.Token{Enforcer: &TestEnforcerDenyAll{}})

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceForbidden{}, res)
}

func (t *SuiteTest) TestServicePostNoAgentFound() {
	fixture.SetupHandler(t.T(), "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1,
		Body: &testService}, nil)

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceConflict{}, res)
	assert.Equal(t.T(), "No available host agent found.", res.(*service.PostServiceConflict).Payload.Message)
}

func (t *SuiteTest) TestServiceScopedGetFromOtherProject() {
	// post and get
	serviceId := t.createService()

	res := t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &headerProject2, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDNotFound{}, res)

	// change visibility to public
	visibility := models.ServiceVisibilityPublic
	res = t.c.PutServiceServiceIDHandler(
		service.PutServiceServiceIDParams{HTTPRequest: &headerProject1, ServiceID: serviceId,
			Body: &models.ServiceUpdatable{Visibility: &visibility}},
		nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDOK{}, res)

	res = t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &headerProject2, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDOK{}, res)
}

func (t *SuiteTest) TestServicePut() {
	// post and get
	serviceId := t.createService()

	name := "test2"
	res := t.c.PutServiceServiceIDHandler(
		service.PutServiceServiceIDParams{HTTPRequest: &http.Request{},
			ServiceID: serviceId, Body: &models.ServiceUpdatable{Name: &name}},
		nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDOK{}, res)

	// get -> updated
	res = t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDOK{}, res)
	assert.Equal(t.T(), "test2", res.(*service.GetServiceServiceIDOK).Payload.Name)

	// not found
	res = t.c.PutServiceServiceIDHandler(
		service.PutServiceServiceIDParams{HTTPRequest: &http.Request{},
			ServiceID: "11fb8be3-6154-4244-80f1-6b0bef94aa1e",
			Body:      &models.ServiceUpdatable{Name: &name}},
		nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDNotFound{}, res)

}

func (t *SuiteTest) TestServiceDelete() {
	// create, delete, get
	// post and get
	serviceId := t.createService()

	// delete
	res := t.c.DeleteServiceServiceIDHandler(
		service.DeleteServiceServiceIDParams{HTTPRequest: &headerProject1, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.DeleteServiceServiceIDAccepted{}, res)

	// not found
	res = t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &headerProject1, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDOK{}, res)
	assert.Equal(t.T(), models.ServiceStatusPENDINGDELETE, res.(*service.GetServiceServiceIDOK).Payload.Status)

	// delete not found
	res = t.c.DeleteServiceServiceIDHandler(
		service.DeleteServiceServiceIDParams{HTTPRequest: &headerProject1,
			ServiceID: "11fb8be3-6154-4244-80f1-6b0bef94aa1e"},
		nil)
	assert.IsType(t.T(), &service.DeleteServiceServiceIDNotFound{}, res)
}

func (t *SuiteTest) TestServiceDuplicatePayload() {
	fixture.SetupHandler(t.T(), "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	t.addAgent("zone1")
	s := models.Service{
		Name:             "test",
		NetworkID:        &networkId,
		IPAddresses:      []strfmt.IPv4{"1.2.3.4"},
		AvailabilityZone: swag.String("zone1"),
	}

	// post two identical services
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1, Body: &s},
		nil)
	assert.IsType(t.T(), &service.PostServiceCreated{}, res)

	res = t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1, Body: &s},
		nil)
	assert.IsType(t.T(), &service.PostServiceConflict{}, res)

	// create a second service with a different ip
	s.IPAddresses = []strfmt.IPv4{"1.2.3.5"}
	res = t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1, Body: &s},
		nil)
	assert.IsType(t.T(), &service.PostServiceCreated{}, res)
	payload := res.(*service.PostServiceCreated).Payload

	// update to 1.2.3.4 -> conflict
	res = t.c.PutServiceServiceIDHandler(
		service.PutServiceServiceIDParams{HTTPRequest: &headerProject1,
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

func (t *SuiteTest) TestGetServiceServiceIDEndpointsHandler() {
	serviceId := t.createService()
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	ep := t.createEndpoint(serviceId, models.EndpointTarget{Network: &network})

	// Get associated endpoints
	params := service.GetServiceServiceIDEndpointsParams{HTTPRequest: &headerProject1, ServiceID: serviceId}
	res := t.c.GetServiceServiceIDEndpointsHandler(params, nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDEndpointsOK{}, res)
	assert.Len(t.T(), res.(*service.GetServiceServiceIDEndpointsOK).Payload.Items, 1)
	assert.Equal(t.T(), ep.ID, res.(*service.GetServiceServiceIDEndpointsOK).Payload.Items[0].ID)
}

func (t *SuiteTest) TestGetServiceServiceIDEndpointsHandlerNotFound() {
	unknown := strfmt.UUID("50a1e876-5171-45c4-9e03-6388512ee418")

	// expect not found
	params := service.GetServiceServiceIDEndpointsParams{HTTPRequest: &headerProject1, ServiceID: unknown}
	res := t.c.GetServiceServiceIDEndpointsHandler(params, nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.GetServiceServiceIDEndpointsNotFound{}, res)
}

func (t *SuiteTest) TestGetServiceServiceIDEndpointsHandlerUnknownSortColumn() {
	serviceId := t.createService()
	params := service.GetServiceServiceIDEndpointsParams{HTTPRequest: &headerProject1,
		Sort: swag.String("unknown"), ServiceID: serviceId}
	res := t.c.GetServiceServiceIDEndpointsHandler(params, nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDEndpointsBadRequest{}, res)
	assert.Equal(t.T(), "Unknown sort column.", res.(*service.GetServiceServiceIDEndpointsBadRequest).Payload.Message)
}

func (t *SuiteTest) TestPutServiceServiceIDAcceptEndpointsHandler() {
	// create service and set require approval
	serviceId := t.createService()
	params := service.PutServiceServiceIDParams{
		HTTPRequest: &headerProject1,
		Body:        &models.ServiceUpdatable{RequireApproval: swag.Bool(true)},
		ServiceID:   serviceId,
	}
	assert.IsType(t.T(), &service.PutServiceServiceIDOK{}, t.c.PutServiceServiceIDHandler(params, nil))

	// create endpoint
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	ep := t.createEndpoint(serviceId, models.EndpointTarget{Network: &network})

	// validate endpoint is status PENDING_APPROVAL
	epParams := endpoint.GetEndpointEndpointIDParams{HTTPRequest: &headerProject1, EndpointID: ep.ID}
	epRes := t.c.GetEndpointEndpointIDHandler(epParams, nil)
	assert.IsType(t.T(), &endpoint.GetEndpointEndpointIDOK{}, epRes)
	assert.Equal(t.T(), models.EndpointStatusPENDINGAPPROVAL, epRes.(*endpoint.GetEndpointEndpointIDOK).Payload.Status)

	putParams := service.PutServiceServiceIDAcceptEndpointsParams{
		HTTPRequest: &headerProject1,
		ServiceID:   serviceId,
		Body:        &models.EndpointConsumerList{EndpointIds: []strfmt.UUID{ep.ID}},
	}

	// try accepting endpoint with from unauthorized project
	unauthorizedParams := putParams
	unauthorizedParams.HTTPRequest = &headerProject2
	res := t.c.PutServiceServiceIDAcceptEndpointsHandler(unauthorizedParams, nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDAcceptEndpointsNotFound{}, res)

	// try accepting with invalid endpoint id
	invalidEPIDParams := putParams
	invalidEPIDParams.Body = &models.EndpointConsumerList{
		EndpointIds: []strfmt.UUID{"50a1e876-5171-45c4-9e03-6388512ee418"}}
	res = t.c.PutServiceServiceIDAcceptEndpointsHandler(invalidEPIDParams, nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDAcceptEndpointsNotFound{}, res)

	// try accepting without correct consumer list
	missingConsumerListParams := putParams
	missingConsumerListParams.Body = &models.EndpointConsumerList{}
	res = t.c.PutServiceServiceIDAcceptEndpointsHandler(missingConsumerListParams, nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDAcceptEndpointsBadRequest{}, res)
	assert.Equal(t.T(), "Must declare at least one, endpoint_id(s) or project_id(s)",
		res.(*service.PutServiceServiceIDAcceptEndpointsBadRequest).Payload.Message)

	// accept endpoint and validate status is PENDING_CREATE
	res = t.c.PutServiceServiceIDAcceptEndpointsHandler(putParams, nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDAcceptEndpointsOK{}, res)
	assert.Len(t.T(), res.(*service.PutServiceServiceIDAcceptEndpointsOK).Payload, 1)
	assert.Equal(t.T(), models.EndpointStatusPENDINGCREATE,
		res.(*service.PutServiceServiceIDAcceptEndpointsOK).Payload[0].Status)
}

func (t *SuiteTest) TestPutServiceServiceIDRejectEndpointsHandler() {
	// create service and set require approval
	serviceId := t.createService()
	params := service.PutServiceServiceIDParams{
		HTTPRequest: &headerProject1,
		Body:        &models.ServiceUpdatable{RequireApproval: swag.Bool(true)},
		ServiceID:   serviceId,
	}
	assert.IsType(t.T(), &service.PutServiceServiceIDOK{}, t.c.PutServiceServiceIDHandler(params, nil))

	// create endpoint
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	ep := t.createEndpoint(serviceId, models.EndpointTarget{Network: &network})

	// validate endpoint is status PENDING_APPROVAL
	epParams := endpoint.GetEndpointEndpointIDParams{HTTPRequest: &headerProject1, EndpointID: ep.ID}
	epRes := t.c.GetEndpointEndpointIDHandler(epParams, nil)
	assert.IsType(t.T(), &endpoint.GetEndpointEndpointIDOK{}, epRes)
	assert.Equal(t.T(), models.EndpointStatusPENDINGAPPROVAL, epRes.(*endpoint.GetEndpointEndpointIDOK).Payload.Status)

	putParams := service.PutServiceServiceIDRejectEndpointsParams{
		HTTPRequest: &headerProject1,
		ServiceID:   serviceId,
		Body:        &models.EndpointConsumerList{EndpointIds: []strfmt.UUID{ep.ID}},
	}

	// try Rejecting endpoint with from unauthorized project
	unauthorizedParams := putParams
	unauthorizedParams.HTTPRequest = &headerProject2
	res := t.c.PutServiceServiceIDRejectEndpointsHandler(unauthorizedParams, nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDRejectEndpointsNotFound{}, res)

	// try Rejecting with invalid endpoint id
	invalidEPIDParams := putParams
	invalidEPIDParams.Body = &models.EndpointConsumerList{
		EndpointIds: []strfmt.UUID{"50a1e876-5171-45c4-9e03-6388512ee418"}}
	res = t.c.PutServiceServiceIDRejectEndpointsHandler(invalidEPIDParams, nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDRejectEndpointsNotFound{}, res)

	// try Rejecting without correct consumer list
	missingConsumerListParams := putParams
	missingConsumerListParams.Body = &models.EndpointConsumerList{}
	res = t.c.PutServiceServiceIDRejectEndpointsHandler(missingConsumerListParams, nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDRejectEndpointsBadRequest{}, res)
	assert.Equal(t.T(), "Must declare at least one, endpoint_id(s) or project_id(s)",
		res.(*service.PutServiceServiceIDRejectEndpointsBadRequest).Payload.Message)

	// Reject endpoint and validate status is PENDING_REJECTED
	res = t.c.PutServiceServiceIDRejectEndpointsHandler(putParams, nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDRejectEndpointsOK{}, res)
	assert.Len(t.T(), res.(*service.PutServiceServiceIDRejectEndpointsOK).Payload, 1)
	assert.Equal(t.T(), models.EndpointStatusPENDINGREJECTED,
		res.(*service.PutServiceServiceIDRejectEndpointsOK).Payload[0].Status)
}
