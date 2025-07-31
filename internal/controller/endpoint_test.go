// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/gophercloud/gophercloud/v2"
	fake "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	"github.com/gophercloud/gophercloud/v2/testhelper/fixture"
	"github.com/hashicorp/go-uuid"
	"github.com/sapcc/go-bits/gopherpolicy"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/endpoint"
	"github.com/sapcc/archer/restapi/operations/service"
)

func (t *SuiteTest) createEndpoint(serviceId strfmt.UUID, target models.EndpointTarget) *models.Endpoint {
	s := models.Endpoint{
		ServiceID: serviceId,
		Target:    target,
		ProjectID: testProject1,
	}

	t.ResetHttpServer()
	fixture.SetupHandler(t.T(), "/v2.0/networks/"+string(*target.Network), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	portID, _ := uuid.GenerateUUID()
	fixture.SetupHandler(t.T(), "/v2.0/ports", "POST", "",
		fmt.Sprintf(CreatePortResponseFixture, portID, string(*target.Network)), http.StatusCreated)
	fixture.SetupHandler(t.T(), "/v2.0/ports/"+portID, "GET", "",
		fmt.Sprintf(CreatePortResponseFixture, portID, string(*target.Network)), http.StatusOK)

	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointCreated{}, res)

	return res.(*endpoint.PostEndpointCreated).Payload
}

func (t *SuiteTest) TestEndpointList() {
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	payload := t.createEndpoint(t.createService(testService), models.EndpointTarget{
		Network: &network,
	})

	header := headerProject1
	header.URL = new(url.URL)
	res := t.c.GetEndpointHandler(endpoint.GetEndpointParams{HTTPRequest: &header},
		nil)
	assert.IsType(t.T(), &endpoint.GetEndpointOK{}, res)
	endpoints := res.(*endpoint.GetEndpointOK)
	assert.Len(t.T(), endpoints.Payload.Items, 1)
	assert.Equal(t.T(), endpoints.Payload.Items[0].ID, payload.ID)
	assert.NotNil(t.T(), endpoints.Payload.Items[0].Target.Network)
	assert.Equal(t.T(), endpoints.Payload.Items[0].Target.Network.String(), network.String())
}

func (t *SuiteTest) TestEndpointListUnknownSortColumn() {
	header := headerProject1
	header.URL = new(url.URL)
	res := t.c.GetEndpointHandler(endpoint.GetEndpointParams{HTTPRequest: &header, Sort: swag.String("unknown")},
		nil)
	assert.IsType(t.T(), &endpoint.GetEndpointBadRequest{}, res)
	assert.Equal(t.T(), "Unknown sort column.", res.(*endpoint.GetEndpointBadRequest).Payload.Message)
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

func (t *SuiteTest) TestEndpointPost() {
	// post and get
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")

	// create endpoint
	payload := t.createEndpoint(t.createService(testService), models.EndpointTarget{
		Network: &network,
	})

	assert.Equal(t.T(), network, *payload.Target.Network)

	res := t.c.GetEndpointEndpointIDHandler(
		endpoint.GetEndpointEndpointIDParams{HTTPRequest: &headerProject1, EndpointID: payload.ID},
		nil)
	assert.IsType(t.T(), &endpoint.GetEndpointEndpointIDOK{}, res)
}

func (t *SuiteTest) TestEndpointServiceNotAccessible() {
	// post and get
	serviceId := t.createService(testService)

	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &headerProject2, Body: &models.Endpoint{
		ServiceID: serviceId,
		Target:    models.EndpointTarget{Network: &network},
		ProjectID: testProject2,
	}}, nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointBadRequest{}, res)
	assert.Equal(t.T(), fmt.Sprintf("Service '%s' is not accessible.", serviceId),
		res.(*endpoint.PostEndpointBadRequest).Payload.Message)
}

func (t *SuiteTest) TestEndpointTargetNetworkUnknown() {
	serviceId := t.createService(testService)
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")

	notFoundBody := fmt.Sprintf(`{"NeutronError": {"type": "NetworkNotFound", "message": "Network %s could not be found.", "detail": ""}}`,
		network)
	fixture.SetupHandler(t.T(), "/v2.0/networks/"+string(network), "GET",
		"", notFoundBody, http.StatusNotFound)
	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &http.Request{}, Body: &models.Endpoint{
		ServiceID: serviceId,
		Target:    models.EndpointTarget{Network: &network},
		ProjectID: testProject1,
	}}, nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointBadRequest{}, res)
	assert.Contains(t.T(), res.(*endpoint.PostEndpointBadRequest).Payload.Message, notFoundBody)
}

func (t *SuiteTest) TestEndpointTargetForeignNetwork() {
	serviceId := t.createService(testService)
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")

	fakeServiceClient := fake.ServiceClient()
	fakeServiceClient.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		return "http://127.0.0.1:8931/", nil
	}
	token := gopherpolicy.Token{ProviderClient: fakeServiceClient.ProviderClient}
	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &http.Request{}, Body: &models.Endpoint{
		ServiceID: serviceId,
		Target:    models.EndpointTarget{Network: &network},
		ProjectID: testProject1,
	}}, &token)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointBadRequest{}, res)
	assert.Equal(t.T(), fmt.Sprintf("Expected HTTP response code [200] when accessing [GET http://127.0.0.1:8931/v2.0/networks/%s], but got 404 instead: 404 page not found", network),
		res.(*endpoint.PostEndpointBadRequest).Payload.Message)
}

func (t *SuiteTest) TestEndpointTargetPortUnknown() {
	serviceId := t.createService(testService)
	unknownPort := strfmt.UUID("aafd39b6-429d-43ff-9600-623d63de6f50")
	s := models.Endpoint{
		ServiceID: serviceId,
		Target:    models.EndpointTarget{Port: &unknownPort},
		ProjectID: testProject1,
	}

	notFoundBody := fmt.Sprintf(`{"NeutronError": {"type": "PortNotFound", "message": "Port %s could not be found.", "detail": ""}}`,
		unknownPort)
	fixture.SetupHandler(t.T(), fmt.Sprintf("/v2.0/ports/%s", unknownPort), "GET", "",
		notFoundBody, http.StatusNotFound)

	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointBadRequest{}, res)
	assert.Contains(t.T(), res.(*endpoint.PostEndpointBadRequest).Payload.Message, notFoundBody)
}

func (t *SuiteTest) TestEndpointTargetPortNotSameProject() {
	serviceId := t.createService(testService)
	portId := strfmt.UUID("89f3b416-affd-4e4f-8468-f9fc5f141cd9")
	s := models.Endpoint{
		ServiceID: serviceId,
		Target:    models.EndpointTarget{Port: &portId},
		ProjectID: testProject1,
	}

	const portFromAnotherProject = `
{
    "port": {
        "id": "89f3b416-affd-4e4f-8468-f9fc5f141cd9",
		"network_id": "8c8de75d-7ec2-4660-a7d5-50f7a60fab28",
        "fixed_ips": [
            {
                "subnet_id": "a0304c3a-4f08-4c43-88af-d796509c97d2",
                "ip_address": "10.0.0.2"
            }
        ],
		"project_id": "test-project-2"
    }
}
`
	fixture.SetupHandler(t.T(), fmt.Sprintf("/v2.0/ports/%s", portId), "GET", "",
		portFromAnotherProject, http.StatusOK)

	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointBadRequest{}, res)
	assert.Equal(t.T(), "target_port needs to be in the same project.",
		res.(*endpoint.PostEndpointBadRequest).Payload.Message)
}

func (t *SuiteTest) TestEndpointTargetPortMissingIPAdddress() {
	serviceId := t.createService(testService)
	portId := strfmt.UUID("89f3b416-affd-4e4f-8468-f9fc5f141cd9")
	s := models.Endpoint{
		ServiceID: serviceId,
		Target:    models.EndpointTarget{Port: &portId},
		ProjectID: testProject1,
	}

	const PortWithoutIPAddress = `{
    	"port": {
        	"status": "DOWN",
        	"id": "89f3b416-affd-4e4f-8468-f9fc5f141cd9",
			"network_id": "e780305d-18a4-4648-b916-2e01615fed1d",
        	"fixed_ips": [],
			"project_id": "test-project-1"
    	}
	}`
	fixture.SetupHandler(t.T(), fmt.Sprintf("/v2.0/ports/%s", portId), "GET", "",
		PortWithoutIPAddress, http.StatusOK)

	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointBadRequest{}, res)
	assert.Equal(t.T(), "target_port needs at least one IP address.",
		res.(*endpoint.PostEndpointBadRequest).Payload.Message)
}

func (t *SuiteTest) TestEndpointTargetPortSameNetworkAsService() {
	serviceId := t.createService(testService)
	portId := strfmt.UUID("89f3b416-affd-4e4f-8468-f9fc5f141cd9")
	s := models.Endpoint{
		ServiceID: serviceId,
		Target:    models.EndpointTarget{Port: &portId},
		ProjectID: testProject1,
	}

	fixture.SetupHandler(t.T(), fmt.Sprintf("/v2.0/ports/%s", portId), "GET", "",
		fmt.Sprintf(CreatePortResponseFixture, portId, networkId), http.StatusOK)

	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointBadRequest{}, res)
	assert.Equal(t.T(), "target_port needs to be in a different network than the service.",
		res.(*endpoint.PostEndpointBadRequest).Payload.Message)
}

func (t *SuiteTest) TestEndpointScopes() {
	// post and get
	serviceId := t.createService(testService)

	// prepare endpoint
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	fixture.SetupHandler(t.T(), "/v2.0/networks/"+string(network), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	portID, _ := uuid.GenerateUUID()
	fixture.SetupHandler(t.T(), "/v2.0/ports", "POST", "",
		fmt.Sprintf(CreatePortResponseFixture, portID, string(network)), http.StatusCreated)
	s := models.Endpoint{
		ServiceID: serviceId,
		Target:    models.EndpointTarget{Network: &network},
		ProjectID: testProject2,
	}

	// test endpoint creation - should fail
	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &headerProject2, Body: &s},
		nil)
	assert.IsType(t.T(), &endpoint.PostEndpointBadRequest{}, res)

	// change visibility to public
	visibility := models.ServiceVisibilityPublic
	res = t.c.PutServiceServiceIDHandler(
		service.PutServiceServiceIDParams{HTTPRequest: &headerProject1, ServiceID: serviceId,
			Body: &models.ServiceUpdatable{Visibility: &visibility}},
		nil)
	assert.IsType(t.T(), &service.PutServiceServiceIDOK{}, res)

	// retry - should succeed
	res = t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &headerProject2, Body: &s},
		nil)
	assert.IsType(t.T(), &endpoint.PostEndpointCreated{}, res)
}

func (t *SuiteTest) TestEndpointWithQuota() {
	network := strfmt.UUID("037d5b08-e113-4567-9d43-901fd89d27cf")
	s := models.Endpoint{
		ServiceID: t.createService(testService),
		Target: models.EndpointTarget{
			Network: &network,
		},
		ProjectID: testProject1,
	}

	fixture.SetupHandler(t.T(), "/v2.0/networks/"+string(network), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	portID, _ := uuid.GenerateUUID()
	fixture.SetupHandler(t.T(), "/v2.0/ports", "POST", "",
		fmt.Sprintf(CreatePortResponseFixture, portID, string(network)), http.StatusCreated)

	config.Global.Quota.Enabled = true
	t.createQuota(string(testProject1))
	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &headerProject1, Body: &s},
		nil)
	config.Global.Quota.Enabled = false
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointCreated{}, res)
}

func (t *SuiteTest) TestEndpointPostMissingTarget() {
	s := models.Endpoint{
		ServiceID: t.createService(testService),
	}

	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointBadRequest{}, res)
}

func (t *SuiteTest) TestEndpointQuotaMet() {
	s := models.Endpoint{
		ServiceID: t.createService(testService),
	}

	config.Global.Quota.Enabled = true
	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &headerProject1, Body: &s},
		nil)
	config.Global.Quota.Enabled = false
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointForbidden{}, res)
	assert.Equal(t.T(), "Quota has been met for Resource: endpoint", res.(*endpoint.PostEndpointForbidden).Payload.Message)
}

func (t *SuiteTest) TestEndpointPortAlreadyUsed() {
	serviceId := t.createService(testService)
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	payload := t.createEndpoint(serviceId, models.EndpointTarget{
		Network: &network,
	})
	assert.NotNil(t.T(), payload)

	// create another endpoint with the same port
	s := models.Endpoint{
		ServiceID: serviceId,
		Target: models.EndpointTarget{
			Port: payload.Target.Port,
		},
		ProjectID: testProject1,
	}
	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &endpoint.PostEndpointBadRequest{}, res)
	expectedMessage := fmt.Sprintf("Port '%s' is already used by another endpoint.", payload.Target.Port)
	assert.Equal(t.T(), expectedMessage, res.(*endpoint.PostEndpointBadRequest).Payload.Message)
}

func (t *SuiteTest) TestEndpointSegmentCouldNotBeFound() {
	serviceId := t.createService(testService)
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	s := models.Endpoint{
		ServiceID: serviceId,
		Target:    models.EndpointTarget{Network: &network},
		ProjectID: testProject1,
	}

	t.ResetHttpServer()
	fixture.SetupHandler(t.T(), "/v2.0/networks/"+string(network), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	portID, _ := uuid.GenerateUUID()
	fixture.SetupHandler(t.T(), "/v2.0/ports", "POST", "",
		fmt.Sprintf(CreatePortResponseFixture, portID, string(network)), http.StatusCreated)
	fixture.SetupHandler(t.T(), "/v2.0/ports/"+portID, "GET", "",
		fmt.Sprintf(CreatePortResponseFixture, portID, string(network)), http.StatusOK)

	// manipulate physical of the agent to require a segment that does not exist
	_, _ = t.c.pool.Exec(context.Background(), `UPDATE agents SET physnet = 'phys_unknown';`)
	defer func() {
		_, _ = t.c.pool.Exec(context.Background(), `UPDATE agents SET physnet=$1;`,
			config.Global.Agent.PhysicalNetwork)
	}()

	// create endpoint
	res := t.c.PostEndpointHandler(endpoint.PostEndpointParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)

	// assert that the error message is as expected
	assert.IsType(t.T(), &endpoint.PostEndpointBadRequest{}, res)
	assert.Contains(t.T(), res.(*endpoint.PostEndpointBadRequest).Payload.Message, "No segment found for network")
}

func (t *SuiteTest) TestEndpointPut() {
	// put not found
	res := t.c.PutEndpointEndpointIDHandler(
		endpoint.PutEndpointEndpointIDParams{HTTPRequest: &headerProject1,
			EndpointID: "99c3f0bc-7389-45a3-b1e9-a9544214a004",
			Body:       endpoint.PutEndpointEndpointIDBody{Tags: []string{"a", "b", "c"}}},
		nil)
	assert.IsType(t.T(), &endpoint.PutEndpointEndpointIDNotFound{}, res)

	// post and get
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")

	// create endpoint
	payload := t.createEndpoint(t.createService(testService), models.EndpointTarget{
		Network: &network,
	})

	// Listen for Postgres notifications
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	conn, err := t.c.pool.Acquire(ctx)
	assert.NoError(t.T(), err, "Failed to acquire connection")
	defer conn.Release()

	_, err = conn.Exec(ctx, "LISTEN endpoint;")
	assert.NoError(t.T(), err, "Failed to listen to endpoint")

	res = t.c.PutEndpointEndpointIDHandler(
		endpoint.PutEndpointEndpointIDParams{HTTPRequest: &http.Request{}, EndpointID: payload.ID,
			Body: endpoint.PutEndpointEndpointIDBody{Tags: []string{"a", "b", "c"}}},
		nil)
	assert.IsType(t.T(), &endpoint.PutEndpointEndpointIDOK{}, res)
	assert.EqualValues(t.T(), []string{"a", "b", "c"}, res.(*endpoint.PutEndpointEndpointIDOK).Payload.Tags)
	assert.Equal(t.T(), network, *res.(*endpoint.PutEndpointEndpointIDOK).Payload.Target.Network)

	n, err := conn.Conn().WaitForNotification(ctx)
	assert.Nil(t.T(), err)
	assert.NotNil(t.T(), n, "Notification not received")
	assert.Equal(t.T(), "endpoint", n.Channel)
	assert.Equal(t.T(), fmt.Sprintf("%s:%s", "test-host", payload.ID), n.Payload)
}

func (t *SuiteTest) TestEndpointRequireApproval() {
	// create service with require approval
	t.addAgent(nil)
	fixture.SetupHandler(t.T(), "/v2.0/networks/"+string(networkId), "GET",
		"", GetNetworkResponseFixture, http.StatusOK)
	fixture.SetupHandler(t.T(), "/v2.0/network-ip-availabilities/"+string(networkId), "GET",
		"", GetNetworkIpAvailabilityResponseFixture, http.StatusOK)
	serviceCopy := testService
	serviceCopy.RequireApproval = swag.Bool(true)
	res := t.c.PostServiceHandler(
		service.PostServiceParams{HTTPRequest: &headerProject1, Body: &serviceCopy}, nil)
	assert.EqualValues(t.T(), *res.(*service.PostServiceCreated).Payload.RequireApproval, true)

	// create endpoint
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")
	payload := t.createEndpoint(res.(*service.PostServiceCreated).Payload.ID, models.EndpointTarget{
		Network: &network,
	})
	assert.Equal(t.T(), models.EndpointStatusPENDINGAPPROVAL, payload.Status)

	// changes won't change the approval status
	res = t.c.PutEndpointEndpointIDHandler(
		endpoint.PutEndpointEndpointIDParams{HTTPRequest: &http.Request{}, EndpointID: payload.ID,
			Body: endpoint.PutEndpointEndpointIDBody{Name: swag.String("testPut")}}, nil)
	assert.IsType(t.T(), &endpoint.PutEndpointEndpointIDOK{}, res)
	payload = res.(*endpoint.PutEndpointEndpointIDOK).Payload
	assert.Equal(t.T(), "testPut", payload.Name)
	assert.Equal(t.T(), models.EndpointStatusPENDINGAPPROVAL, payload.Status)

	// deletion should succeed
	res = t.c.DeleteEndpointEndpointIDHandler(
		endpoint.DeleteEndpointEndpointIDParams{HTTPRequest: &http.Request{}, EndpointID: payload.ID}, nil)
	assert.IsType(t.T(), &endpoint.DeleteEndpointEndpointIDAccepted{}, res)

	// pending delete
	res = t.c.GetEndpointEndpointIDHandler(
		endpoint.GetEndpointEndpointIDParams{HTTPRequest: &http.Request{}, EndpointID: payload.ID}, nil)
	assert.IsType(t.T(), &endpoint.GetEndpointEndpointIDOK{}, res)
	payload = res.(*endpoint.GetEndpointEndpointIDOK).Payload
	assert.NotNil(t.T(), payload)
	assert.Equal(t.T(), models.EndpointStatusPENDINGDELETE, payload.Status)
}

func (t *SuiteTest) TestEndpointDelete() {
	// create, delete, get
	network := strfmt.UUID("d714f65e-bffd-494f-8219-8eb0a85d7a2d")

	// delete not found
	res := t.c.DeleteEndpointEndpointIDHandler(
		endpoint.DeleteEndpointEndpointIDParams{HTTPRequest: &headerProject1,
			EndpointID: "f02605c2-e8d6-4f14-9daa-f5ba7dc65b41"},
		nil)
	assert.IsType(t.T(), &endpoint.DeleteEndpointEndpointIDNotFound{}, res)

	// create endpoint
	payload := t.createEndpoint(t.createService(testService), models.EndpointTarget{
		Network: &network,
	})

	// delete
	res = t.c.DeleteEndpointEndpointIDHandler(
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
