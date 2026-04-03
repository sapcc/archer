// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	sq "github.com/Masterminds/squirrel"
	policy "github.com/databus23/goslo.policy"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag/conv"
	"github.com/gophercloud/gophercloud/v2/testhelper/fixture"
	"github.com/sapcc/go-bits/gopherpolicy"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
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
		Ports:       []int32{0},
		ProjectID:   testProject1,
	}
)

func (t *SuiteTest) addAgent(az *string) {
	sql, args := db.Insert("agents").
		Columns("host", "availability_zone", "physnet", "heartbeat_at").
		Values("test-host", az, config.Global.Agent.PhysicalNetwork, sq.Expr("NOW()")).
		Suffix("ON CONFLICT DO NOTHING").
		MustSql()
	if _, err := t.c.pool.Exec(context.Background(), sql, args...); err != nil {
		t.FailNow("Failed inserting agent host", err)
	}
}

func (t *SuiteTest) createService(svc models.Service) strfmt.UUID {
	t.addAgent(nil)
	t.ResetHttpServer()
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/networks/"+svc.NetworkID.String(), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/network-ip-availabilities/"+svc.NetworkID.String(), "GET",
		"", GetNetworkIpAvailabilityResponseFixture, http.StatusOK)
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1, Body: &svc},
		nil)

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceCreated{}, res)
	payload := res.(*service.PostServiceCreated).Payload
	assert.Equal(t.T(), networkId, *payload.NetworkID)
	return payload.ID
}

func (t *SuiteTest) TestGetServiceHandler() {
	serviceId := t.createService(testService)

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
	res := t.c.GetServiceHandler(service.GetServiceParams{HTTPRequest: &header, Sort: conv.Pointer("unknown")}, nil)
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
	serviceId := t.createService(testService)

	res := t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDOK{}, res)
}

func (t *SuiteTest) TestServicePostScoped() {
	serviceId := t.createService(testService)

	res := t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDOK{}, res)
}

func (t *SuiteTest) TestServiceAZPost() {
	// post and get
	testServiceWithAZ := testService
	testServiceWithAZ.AvailabilityZone = conv.Pointer("test-az")

	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/network-ip-availabilities/"+string(networkId), "GET",
		"", GetNetworkIpAvailabilityResponseFixture, http.StatusOK)

	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1,
		Body: &testServiceWithAZ}, nil)
	assert.IsType(t.T(), &service.PostServiceConflict{}, res)
	assert.Equal(t.T(), "No available host agent found.", res.(*service.PostServiceConflict).Payload.Message)

	t.addAgent(conv.Pointer("test-az"))
	res = t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1,
		Body: &testServiceWithAZ}, nil)
	assert.IsType(t.T(), &service.PostServiceCreated{}, res)
}

func (t *SuiteTest) TestServicePostPorts() {
	// post and get
	serviceId := t.createService(testService)
	testServiceOtherPort := testService
	testServiceOtherPort.Ports = []int32{8080, 9090, 10000}

	res := t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDOK{}, res)
}

func (t *SuiteTest) TestServicePostConflictPorts() {
	// post and get
	t.createService(testService)

	// same IP with same port overlap -> conflict
	testServiceOtherPort := testService
	testServiceOtherPort.Ports = []int32{0, 1234}

	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1, Body: &testServiceOtherPort},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceConflict{}, res)
	assert.Equal(t.T(), "Entry for network_id, ip_address and port(s) already exists.",
		res.(*service.PostServiceConflict).Payload.Message)
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
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject2,
		Body: &testService}, nil)

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceConflict{}, res)
	assert.Equal(t.T(), "Network not accessible.", res.(*service.PostServiceConflict).Payload.Message)
}

func (t *SuiteTest) TestServicePostNetwortNoIpAvailability() {
	t.addAgent(nil)
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/network-ip-availabilities/"+string(networkId), "GET",
		"", GetNetworkIpNoAvailabilityResponseFixture, http.StatusOK)
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1,
		Body: &testService}, nil)

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceConflict{}, res)
	assert.Equal(t.T(), "No available IP addresses in network.",
		res.(*service.PostServiceConflict).Payload.Message)
}

func (t *SuiteTest) TestServiceNegativeAZPost() {
	t.addAgent(conv.Pointer("test-az")) // only az-aware agent
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/network-ip-availabilities/"+string(networkId), "GET",
		"", GetNetworkIpAvailabilityResponseFixture, http.StatusOK)

	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1,
		Body: &testService}, nil)
	assert.IsType(t.T(), &service.PostServiceConflict{}, res)
	assert.Equal(t.T(), "No available host agent found.", res.(*service.PostServiceConflict).Payload.Message)
}

type TestEnforcerDenyAll struct{}

func (t *TestEnforcerDenyAll) Enforce(_ string, _ policy.Context) bool {
	return false
}

func (t *SuiteTest) TestServicePostNotTenant() {
	testServiceNotTenantProvider := testService
	testServiceNotTenantProvider.Provider = conv.Pointer("cp")
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/network-ip-availabilities/"+string(networkId), "GET",
		"", GetNetworkIpAvailabilityResponseFixture, http.StatusOK)
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1,
		Body: &testServiceNotTenantProvider}, &gopherpolicy.Token{Enforcer: &TestEnforcerDenyAll{}})

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceForbidden{}, res)
}

func (t *SuiteTest) TestServicePostNoAgentFound() {
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/network-ip-availabilities/"+string(networkId), "GET",
		"", GetNetworkIpAvailabilityResponseFixture, http.StatusOK)
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1,
		Body: &testService}, nil)

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceConflict{}, res)
	assert.Equal(t.T(), "No available host agent found.", res.(*service.PostServiceConflict).Payload.Message)
}

func (t *SuiteTest) TestServiceScopedGetFromOtherProject() {
	// post and get
	serviceId := t.createService(testService)

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
	serviceId := t.createService(testService)

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
	serviceId := t.createService(testService)

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
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/network-ip-availabilities/"+string(networkId), "GET",
		"", GetNetworkIpAvailabilityResponseFixture, http.StatusOK)

	t.addAgent(conv.Pointer("zone1"))
	s := models.Service{
		Name:             "test",
		NetworkID:        &networkId,
		IPAddresses:      []strfmt.IPv4{"1.2.3.4"},
		Ports:            []int32{0},
		AvailabilityZone: conv.Pointer("zone1"),
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
	serviceID := t.createService(s)

	// update to 1.2.3.4 -> conflict
	res = t.c.PutServiceServiceIDHandler(
		service.PutServiceServiceIDParams{HTTPRequest: &headerProject1,
			ServiceID: serviceID, Body: &models.ServiceUpdatable{IPAddresses: []strfmt.IPv4{"1.2.3.4"}}},
		nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDConflict{}, res)
}

func (t *SuiteTest) TestServiceDeleteInUse() {
	serviceId := t.createService(testService)
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

func (t *SuiteTest) TestServiceDeleteWithRejectedEndpoint() {
	serviceId := t.createService(testService)
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	ep := t.createEndpoint(serviceId, models.EndpointTarget{Network: &network})

	// delete should fail while endpoint is active
	res := t.c.DeleteServiceServiceIDHandler(
		service.DeleteServiceServiceIDParams{HTTPRequest: &http.Request{}, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.DeleteServiceServiceIDConflict{}, res)

	// set endpoint status to REJECTED directly in the DB
	sql, args := db.Update("endpoint").
		Set("status", models.EndpointStatusREJECTED).
		Where("id = ?", ep.ID).
		MustSql()
	_, err := t.c.pool.Exec(context.Background(), sql, args...)
	assert.NoError(t.T(), err)

	// delete should succeed now since only rejected endpoints remain
	res = t.c.DeleteServiceServiceIDHandler(
		service.DeleteServiceServiceIDParams{HTTPRequest: &headerProject1, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.DeleteServiceServiceIDAccepted{}, res)

	// verify service is in PENDING_DELETE state
	res = t.c.GetServiceServiceIDHandler(
		service.GetServiceServiceIDParams{HTTPRequest: &headerProject1, ServiceID: serviceId},
		nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDOK{}, res)
	assert.Equal(t.T(), models.ServiceStatusPENDINGDELETE, res.(*service.GetServiceServiceIDOK).Payload.Status)

	// verify endpoints were transitioned to PENDING_DELETE (not deleted raw) so the agent can clean up ports
	var endpointStatus models.EndpointStatus
	endpointSQL, endpointArgs := db.Select("status").From("endpoint").Where("id = ?", ep.ID).MustSql()
	err = t.c.pool.QueryRow(context.Background(), endpointSQL, endpointArgs...).Scan(&endpointStatus)
	assert.NoError(t.T(), err)
	assert.Equal(t.T(), models.EndpointStatusPENDINGDELETE, endpointStatus)
}

func (t *SuiteTest) TestGetServiceServiceIDEndpointsHandler() {
	serviceId := t.createService(testService)
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
	serviceId := t.createService(testService)
	params := service.GetServiceServiceIDEndpointsParams{HTTPRequest: &headerProject1,
		Sort: conv.Pointer("unknown"), ServiceID: serviceId}
	res := t.c.GetServiceServiceIDEndpointsHandler(params, nil)
	assert.IsType(t.T(), &service.GetServiceServiceIDEndpointsBadRequest{}, res)
	assert.Equal(t.T(), "Unknown sort column.", res.(*service.GetServiceServiceIDEndpointsBadRequest).Payload.Message)
}

func (t *SuiteTest) TestPutServiceServiceIDAcceptEndpointsHandler() {
	// create service and set require approval
	serviceId := t.createService(testService)
	params := service.PutServiceServiceIDParams{
		HTTPRequest: &headerProject1,
		Body:        &models.ServiceUpdatable{RequireApproval: conv.Pointer(true)},
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
	// create service with require approval
	svcReqApproval := testService
	svcReqApproval.RequireApproval = conv.Pointer(true)
	serviceId := t.createService(testService)

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

func (t *SuiteTest) TestPutServiceServiceIDAcceptEndpointHandlerMultipleServices() {
	// create two services with require approval
	svcReqApproval := testService
	svcReqApproval.RequireApproval = conv.Pointer(true)
	serviceID1 := t.createService(testService)

	svcReqApproval.Name = "test2"
	svcReqApproval.IPAddresses = []strfmt.IPv4{"2.3.4.5"}
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1, Body: &svcReqApproval},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceCreated{}, res)
	serviceID2 := res.(*service.PostServiceCreated).Payload.ID

	// create endpoints for both services
	var network strfmt.UUID
	network = "d714f65e-bffd-494f-8219-8eb0a85d7a2d"
	t.createEndpoint(serviceID1, models.EndpointTarget{Network: &network})
	network = "a97c6721-32d9-436d-9cd1-5327d65de67b"
	t.createEndpoint(serviceID2, models.EndpointTarget{Network: &network})

	// try accepting endpoint with from unauthorized project
	putParams := service.PutServiceServiceIDAcceptEndpointsParams{
		HTTPRequest: &headerProject1,
		ServiceID:   serviceID1,
		Body:        &models.EndpointConsumerList{ProjectIds: []models.Project{testProject1}},
	}
	// we expect only one endpoint to be returned
	res = t.c.PutServiceServiceIDAcceptEndpointsHandler(putParams, nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDAcceptEndpointsOK{}, res)
	assert.Len(t.T(), res.(*service.PutServiceServiceIDAcceptEndpointsOK).Payload, 1)
	assert.Equal(t.T(), models.EndpointStatusPENDINGCREATE,
		res.(*service.PutServiceServiceIDAcceptEndpointsOK).Payload[0].Status)
}

func (t *SuiteTest) TestServicePutConflictPorts() {
	// post and get
	svc := t.createService(testService)

	testServiceOtherPort := testService
	testServiceOtherPort.Ports = []int32{1234, 2345}

	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1, Body: &testServiceOtherPort},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PostServiceCreated{}, res)

	// update to port 2345 -> conflict
	res = t.c.PutServiceServiceIDHandler(
		service.PutServiceServiceIDParams{HTTPRequest: &headerProject1,
			ServiceID: svc, Body: &models.ServiceUpdatable{Ports: []int32{2345, 3456}}},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &service.PutServiceServiceIDConflict{}, res)
	assert.Equal(t.T(), "Entry for network_id, ip_address and port(s) already exists.",
		res.(*service.PutServiceServiceIDConflict).Payload.Message)
}

func (t *SuiteTest) addAgentWithHost(host string, az *string) {
	sql, args := db.Insert("agents").
		Columns("host", "availability_zone", "physnet", "heartbeat_at").
		Values(host, az, config.Global.Agent.PhysicalNetwork, sq.Expr("NOW()")).
		Suffix("ON CONFLICT DO NOTHING").
		MustSql()
	if _, err := t.c.pool.Exec(context.Background(), sql, args...); err != nil {
		t.FailNow("Failed inserting agent host", err)
	}
}

// TestServiceMigrateWithNullAZ tests that migrating a service with null availability_zone
// to a specific target host works correctly. This is a regression test for the issue where
// the query used "availability_zone = ?" instead of sq.Eq{"availability_zone": az} which
// doesn't properly handle NULL comparisons in SQL.
func (t *SuiteTest) TestServiceMigrateWithNullAZ() {
	// Create a service (which will have null AZ by default)
	serviceId := t.createService(testService)

	// Add a second agent with null AZ
	targetHost := "target-host-null-az"
	t.addAgentWithHost(targetHost, nil)

	// Try to migrate to the specific target host
	res := t.c.PostServiceServiceIDMigrateHandler(
		service.PostServiceServiceIDMigrateParams{
			HTTPRequest: &headerProject1,
			ServiceID:   serviceId,
			Body:        service.PostServiceServiceIDMigrateBody{TargetHost: targetHost},
		},
		nil)

	// Should succeed (not return 404)
	assert.IsType(t.T(), &service.PostServiceServiceIDMigrateOK{}, res)
	payload := res.(*service.PostServiceServiceIDMigrateOK).Payload
	assert.Equal(t.T(), targetHost, *payload.Host)
	assert.Equal(t.T(), models.ServiceStatusPENDINGUPDATE, payload.Status)
}

// TestServiceMigrateWithAZ tests that migrating a service with a specific availability_zone
// to a target host in the same AZ works correctly.
func (t *SuiteTest) TestServiceMigrateWithAZ() {
	// Create agents with AZ - we need at least 2 to migrate between them
	az := "test-az-migrate"
	t.addAgentWithHost("source-host-az", &az)
	t.addAgentWithHost("target-host-az", &az)

	// Create a service in that AZ
	testServiceWithAZ := testService
	testServiceWithAZ.AvailabilityZone = &az
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t.T(), t.fakeServer, "/v2.0/network-ip-availabilities/"+string(networkId), "GET",
		"", GetNetworkIpAvailabilityResponseFixture, http.StatusOK)

	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1, Body: &testServiceWithAZ},
		nil)
	assert.IsType(t.T(), &service.PostServiceCreated{}, res)
	serviceId := res.(*service.PostServiceCreated).Payload.ID
	currentHost := *res.(*service.PostServiceCreated).Payload.Host

	// Determine which host to migrate to (the other one)
	var targetHost string
	if currentHost == "source-host-az" {
		targetHost = "target-host-az"
	} else {
		targetHost = "source-host-az"
	}

	// Try to migrate to the target host in the same AZ
	res = t.c.PostServiceServiceIDMigrateHandler(
		service.PostServiceServiceIDMigrateParams{
			HTTPRequest: &headerProject1,
			ServiceID:   serviceId,
			Body:        service.PostServiceServiceIDMigrateBody{TargetHost: targetHost},
		},
		nil)

	assert.IsType(t.T(), &service.PostServiceServiceIDMigrateOK{}, res)
	payload := res.(*service.PostServiceServiceIDMigrateOK).Payload
	assert.Equal(t.T(), targetHost, *payload.Host)
}

// TestServiceMigrateToWrongAZFails tests that migrating a service to an agent
// in a different availability zone fails.
func (t *SuiteTest) TestServiceMigrateToWrongAZFails() {
	// Create a service with null AZ
	serviceId := t.createService(testService)

	// Add an agent with a specific AZ (different from null)
	differentAZ := "different-az"
	t.addAgentWithHost("target-host-different-az", &differentAZ)

	// Try to migrate to the agent with different AZ - should fail
	res := t.c.PostServiceServiceIDMigrateHandler(
		service.PostServiceServiceIDMigrateParams{
			HTTPRequest: &headerProject1,
			ServiceID:   serviceId,
			Body:        service.PostServiceServiceIDMigrateBody{TargetHost: "target-host-different-az"},
		},
		nil)

	// Should return 404 because target agent is in different AZ
	assert.IsType(t.T(), &service.PostServiceServiceIDMigrateNotFound{}, res)
	assert.Equal(t.T(), "Target agent not found or not healthy",
		res.(*service.PostServiceServiceIDMigrateNotFound).Payload.Message)
}

// TestServiceMigrateAutoSelectsHost tests that migrating without a target host
// automatically selects an available agent.
func (t *SuiteTest) TestServiceMigrateAutoSelectsHost() {
	// Create a service
	serviceId := t.createService(testService)

	// Add additional agents with null AZ
	t.addAgentWithHost("auto-target-host-1", nil)
	t.addAgentWithHost("auto-target-host-2", nil)

	// Migrate without specifying target host
	res := t.c.PostServiceServiceIDMigrateHandler(
		service.PostServiceServiceIDMigrateParams{
			HTTPRequest: &headerProject1,
			ServiceID:   serviceId,
			Body:        service.PostServiceServiceIDMigrateBody{},
		},
		nil)

	assert.IsType(t.T(), &service.PostServiceServiceIDMigrateOK{}, res)
	payload := res.(*service.PostServiceServiceIDMigrateOK).Payload
	// Should have migrated to a different host than the original
	assert.NotEqual(t.T(), "test-host", *payload.Host)
}

func (t *SuiteTest) TestServiceMigrateServiceNotFound() {
	unknown := strfmt.UUID("00000000-1111-2222-3333-444444444444")
	res := t.c.PostServiceServiceIDMigrateHandler(
		service.PostServiceServiceIDMigrateParams{
			HTTPRequest: &headerProject1,
			ServiceID:   unknown,
			Body:        service.PostServiceServiceIDMigrateBody{},
		},
		nil)

	assert.IsType(t.T(), &service.PostServiceServiceIDMigrateNotFound{}, res)
}

func (t *SuiteTest) TestServiceMigrateSameHostFails() {
	// Create a service
	serviceId := t.createService(testService)

	// Try to migrate to the same host (test-host is the default)
	res := t.c.PostServiceServiceIDMigrateHandler(
		service.PostServiceServiceIDMigrateParams{
			HTTPRequest: &headerProject1,
			ServiceID:   serviceId,
			Body:        service.PostServiceServiceIDMigrateBody{TargetHost: "test-host"},
		},
		nil)

	assert.IsType(t.T(), &service.PostServiceServiceIDMigrateBadRequest{}, res)
	assert.Equal(t.T(), "Service is already on the target host",
		res.(*service.PostServiceServiceIDMigrateBadRequest).Payload.Message)
}

// TestMaskCPServiceIPAddresses tests the IP address masking for CP services
func (t *SuiteTest) TestMaskCPServiceIPAddresses() {
	cpProvider := "cp"

	// Test with nil service - should not panic
	maskCPServiceIPAddresses(nil, nil)

	// Test with CP service and regular user (no principal) - IPs should be masked
	svc := &models.Service{
		Provider:    &cpProvider,
		IPAddresses: []strfmt.IPv4{"1.2.3.4"},
	}
	maskCPServiceIPAddresses(svc, nil)
	assert.Nil(t.T(), svc.IPAddresses)

	// Test with non-CP service - IPs should not be masked
	tenantProvider := "tenant"
	svc2 := &models.Service{
		Provider:    &tenantProvider,
		IPAddresses: []strfmt.IPv4{"1.2.3.4"},
	}
	maskCPServiceIPAddresses(svc2, nil)
	assert.NotNil(t.T(), svc2.IPAddresses)
	assert.Len(t.T(), svc2.IPAddresses, 1)

	// Test with CP service and cloud_admin (service:read-global) - IPs should NOT be masked
	svc3 := &models.Service{
		Provider:    &cpProvider,
		IPAddresses: []strfmt.IPv4{"5.6.7.8"},
	}
	maskCPServiceIPAddresses(svc3, &gopherpolicy.Token{Enforcer: &TestEnforcerAllowReadGlobal{}})
	assert.NotNil(t.T(), svc3.IPAddresses)
	assert.Len(t.T(), svc3.IPAddresses, 1)

	// Test with CP service and non-admin user - IPs should be masked
	svc4 := &models.Service{
		Provider:    &cpProvider,
		IPAddresses: []strfmt.IPv4{"9.10.11.12"},
	}
	maskCPServiceIPAddresses(svc4, &gopherpolicy.Token{Enforcer: &TestEnforcerDenyAll{}})
	assert.Nil(t.T(), svc4.IPAddresses)
}

type TestEnforcerAllowReadGlobal struct{}

func (t *TestEnforcerAllowReadGlobal) Enforce(rule string, _ policy.Context) bool {
	return rule == "service:read-global"
}

func (t *SuiteTest) TestServicePostCPProviderWithoutNetworkID() {
	// Add an agent for CP provider
	t.addCPAgent("cp-test-host", nil)

	cpProvider := "cp"
	cpService := models.Service{
		Name:        "cp-service-test",
		Provider:    &cpProvider,
		IPAddresses: []strfmt.IPv4{"192.168.1.100"},
		Ports:       []int32{8080},
		ProjectID:   testProject1,
	}

	// Create CP service without network_id - should succeed with placeholder
	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1, Body: &cpService},
		&gopherpolicy.Token{Enforcer: &TestEnforcerAllowProvider{}})

	assert.IsType(t.T(), &service.PostServiceCreated{}, res)
	payload := res.(*service.PostServiceCreated).Payload
	assert.NotNil(t.T(), payload.NetworkID)
	assert.Equal(t.T(), CPNetworkID, *payload.NetworkID)
	assert.Equal(t.T(), "cp", *payload.Provider)
}

func (t *SuiteTest) TestServicePostTenantProviderWithoutNetworkIDFails() {
	// Try to create a tenant service without network_id - should fail
	svc := models.Service{
		Name:        "tenant-no-network",
		IPAddresses: []strfmt.IPv4{"1.2.3.4"},
		Ports:       []int32{0},
		ProjectID:   testProject1,
	}

	res := t.c.PostServiceHandler(service.PostServiceParams{HTTPRequest: &headerProject1, Body: &svc},
		nil)

	assert.IsType(t.T(), &service.PostServiceUnprocessableEntity{}, res)
	assert.Equal(t.T(), "network_id is required for tenant provider",
		res.(*service.PostServiceUnprocessableEntity).Payload.Message)
}

type TestEnforcerAllowProvider struct{}

func (t *TestEnforcerAllowProvider) Enforce(_ string, _ policy.Context) bool {
	return true
}

func (t *SuiteTest) addCPAgent(host string, az *string) {
	sql, args := db.Insert("agents").
		Columns("host", "availability_zone", "provider", "heartbeat_at").
		Values(host, az, "cp", sq.Expr("NOW()")).
		Suffix("ON CONFLICT DO NOTHING").
		MustSql()
	if _, err := t.c.pool.Exec(context.Background(), sql, args...); err != nil {
		t.FailNow("Failed inserting cp agent host", err)
	}
}
