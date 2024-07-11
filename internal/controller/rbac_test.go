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
	"net/http"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations/rbac"
	"github.com/sapcc/archer/restapi/operations/service"
)

func (t *SuiteTest) createRbac(target string) strfmt.UUID {
	service := t.createService(testService)
	s := models.Rbacpolicy{
		ServiceID:  &service,
		Target:     target,
		TargetType: swag.String(models.RbacpolicyTargetTypeProject),
	}

	res := t.c.PostRbacPoliciesHandler(rbac.PostRbacPoliciesParams{HTTPRequest: &http.Request{}, Body: &s},
		nil)

	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &rbac.PostRbacPoliciesCreated{}, res)
	payload := res.(*rbac.PostRbacPoliciesCreated).Payload
	assert.Equal(t.T(), target, payload.Target)
	return payload.ID
}

func (t *SuiteTest) TestRbacEmptyGet() {
	// empty get from random uuid
	u := strfmt.UUID("30971832-4f4d-4068-97fb-b0cfe816cae0")

	res := t.c.GetRbacPoliciesRbacPolicyIDHandler(
		rbac.GetRbacPoliciesRbacPolicyIDParams{
			HTTPRequest:  &http.Request{},
			RbacPolicyID: u,
		}, nil)

	// not found
	assert.IsType(t.T(), &rbac.GetRbacPoliciesRbacPolicyIDNotFound{}, res)
}

func (t *SuiteTest) TestRbacPost() {
	// post and get
	u := t.createRbac("8bfb0b1d-483b-49ba-b11e-f2f83fd9e1b6")

	res := t.c.GetRbacPoliciesRbacPolicyIDHandler(
		rbac.GetRbacPoliciesRbacPolicyIDParams{
			HTTPRequest:  &http.Request{},
			RbacPolicyID: u,
		}, nil)
	assert.IsType(t.T(), &rbac.GetRbacPoliciesRbacPolicyIDOK{}, res)
}

func (t *SuiteTest) TestRbacPut() {
	u := t.createRbac("8bfb0b1d-483b-49ba-b11e-f2f83fd9e1b6")

	// update target
	newTarget := "84b08420-7be8-471b-ae56-b656af97cea0"
	res := t.c.PutRbacPoliciesRbacPolicyIDHandler(
		rbac.PutRbacPoliciesRbacPolicyIDParams{
			HTTPRequest: &http.Request{},
			Body: &models.Rbacpolicycommon{
				Target: &newTarget,
			},
			RbacPolicyID: u,
		}, nil)

	assert.IsType(t.T(), &rbac.PutRbacPoliciesRbacPolicyIDOK{}, res)
	payload := res.(*rbac.PutRbacPoliciesRbacPolicyIDOK).Payload
	assert.Equal(t.T(), newTarget, payload.Target)

	res = t.c.GetRbacPoliciesRbacPolicyIDHandler(
		rbac.GetRbacPoliciesRbacPolicyIDParams{
			HTTPRequest:  &http.Request{},
			RbacPolicyID: u,
		}, nil)
	assert.IsType(t.T(), &rbac.GetRbacPoliciesRbacPolicyIDOK{}, res)
	payload = res.(*rbac.GetRbacPoliciesRbacPolicyIDOK).Payload
	assert.Equal(t.T(), newTarget, payload.Target)
}

func (t *SuiteTest) TestRbacDelete() {
	u := t.createRbac("8bfb0b1d-483b-49ba-b11e-f2f83fd9e1b6")

	// delete
	res := t.c.DeleteRbacPoliciesRbacPolicyIDHandler(
		rbac.DeleteRbacPoliciesRbacPolicyIDParams{
			HTTPRequest:  &http.Request{},
			RbacPolicyID: u,
		}, nil)
	assert.IsType(t.T(), &rbac.DeleteRbacPoliciesRbacPolicyIDNoContent{}, res)

	// not found
	res = t.c.DeleteRbacPoliciesRbacPolicyIDHandler(
		rbac.DeleteRbacPoliciesRbacPolicyIDParams{
			HTTPRequest:  &http.Request{},
			RbacPolicyID: u,
		}, nil)
	assert.IsType(t.T(), &rbac.DeleteRbacPoliciesRbacPolicyIDNotFound{}, res)
}

func (t *SuiteTest) TestRbacConflict() {
	target := "faa90930-9518-469e-ac9f-d3622110e09b"
	s := t.createService(testService)
	p := models.Rbacpolicy{
		ServiceID:  &s,
		Target:     target,
		TargetType: swag.String(models.RbacpolicyTargetTypeProject),
	}

	// post
	res := t.c.PostRbacPoliciesHandler(rbac.PostRbacPoliciesParams{HTTPRequest: &http.Request{}, Body: &p},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &rbac.PostRbacPoliciesCreated{}, res)
	payload := res.(*rbac.PostRbacPoliciesCreated).Payload

	// conflict
	res = t.c.PostRbacPoliciesHandler(rbac.PostRbacPoliciesParams{HTTPRequest: &http.Request{}, Body: &p},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &rbac.PostRbacPoliciesConflict{}, res)

	// new target should succeed
	p.Target = "1382cc18-d25e-40aa-873d-c3fb8466fdc0"
	res = t.c.PostRbacPoliciesHandler(rbac.PostRbacPoliciesParams{HTTPRequest: &http.Request{}, Body: &p},
		nil)
	assert.NotNil(t.T(), res)
	assert.IsType(t.T(), &rbac.PostRbacPoliciesCreated{}, res)

	// update old target to new one target
	res = t.c.PutRbacPoliciesRbacPolicyIDHandler(
		rbac.PutRbacPoliciesRbacPolicyIDParams{
			HTTPRequest: &http.Request{},
			Body: &models.Rbacpolicycommon{
				Target: &p.Target,
			},
			RbacPolicyID: payload.ID,
		}, nil)

	assert.IsType(t.T(), &rbac.PutRbacPoliciesRbacPolicyIDConflict{}, res)
}

func (t *SuiteTest) TestRbacServiceCascadeDelete() {
	u := t.createRbac("8bfb0b1d-483b-49ba-b11e-f2f83fd9e1b6")

	// get service
	res := t.c.GetRbacPoliciesRbacPolicyIDHandler(
		rbac.GetRbacPoliciesRbacPolicyIDParams{
			HTTPRequest:  &http.Request{},
			RbacPolicyID: u,
		}, nil)
	assert.IsType(t.T(), &rbac.GetRbacPoliciesRbacPolicyIDOK{}, res)
	serviceId := res.(*rbac.GetRbacPoliciesRbacPolicyIDOK).Payload.ServiceID

	// delete service
	res = t.c.DeleteServiceServiceIDHandler(service.DeleteServiceServiceIDParams{
		HTTPRequest: &http.Request{},
		ServiceID:   *serviceId,
	}, nil)
	assert.IsType(t.T(), &service.DeleteServiceServiceIDAccepted{}, res)

	// emulate real delete from backend
	sql := `DELETE FROM service WHERE id = $1 and status = 'PENDING_DELETE'`
	ct, err := t.c.pool.Exec(context.Background(), sql, serviceId)
	assert.Nil(t.T(), err)
	assert.NotZero(t.T(), ct.RowsAffected())

	// expect deleted
	res = t.c.GetRbacPoliciesRbacPolicyIDHandler(
		rbac.GetRbacPoliciesRbacPolicyIDParams{
			HTTPRequest:  &http.Request{},
			RbacPolicyID: u,
		}, nil)
	assert.IsType(t.T(), &rbac.GetRbacPoliciesRbacPolicyIDNotFound{}, res)
}
