// Code generated by go-swagger; DO NOT EDIT.

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

package rbac

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/sapcc/archer/models"
)

// GetRbacPoliciesRbacPolicyIDOKCode is the HTTP code returned for type GetRbacPoliciesRbacPolicyIDOK
const GetRbacPoliciesRbacPolicyIDOKCode int = 200

/*
GetRbacPoliciesRbacPolicyIDOK RBAC Policy

swagger:response getRbacPoliciesRbacPolicyIdOK
*/
type GetRbacPoliciesRbacPolicyIDOK struct {

	/*
	  In: Body
	*/
	Payload *models.Rbacpolicy `json:"body,omitempty"`
}

// NewGetRbacPoliciesRbacPolicyIDOK creates GetRbacPoliciesRbacPolicyIDOK with default headers values
func NewGetRbacPoliciesRbacPolicyIDOK() *GetRbacPoliciesRbacPolicyIDOK {

	return &GetRbacPoliciesRbacPolicyIDOK{}
}

// WithPayload adds the payload to the get rbac policies rbac policy Id o k response
func (o *GetRbacPoliciesRbacPolicyIDOK) WithPayload(payload *models.Rbacpolicy) *GetRbacPoliciesRbacPolicyIDOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get rbac policies rbac policy Id o k response
func (o *GetRbacPoliciesRbacPolicyIDOK) SetPayload(payload *models.Rbacpolicy) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetRbacPoliciesRbacPolicyIDOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetRbacPoliciesRbacPolicyIDForbiddenCode is the HTTP code returned for type GetRbacPoliciesRbacPolicyIDForbidden
const GetRbacPoliciesRbacPolicyIDForbiddenCode int = 403

/*
GetRbacPoliciesRbacPolicyIDForbidden Forbidden

swagger:response getRbacPoliciesRbacPolicyIdForbidden
*/
type GetRbacPoliciesRbacPolicyIDForbidden struct {
}

// NewGetRbacPoliciesRbacPolicyIDForbidden creates GetRbacPoliciesRbacPolicyIDForbidden with default headers values
func NewGetRbacPoliciesRbacPolicyIDForbidden() *GetRbacPoliciesRbacPolicyIDForbidden {

	return &GetRbacPoliciesRbacPolicyIDForbidden{}
}

// WriteResponse to the client
func (o *GetRbacPoliciesRbacPolicyIDForbidden) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(403)
}

// GetRbacPoliciesRbacPolicyIDNotFoundCode is the HTTP code returned for type GetRbacPoliciesRbacPolicyIDNotFound
const GetRbacPoliciesRbacPolicyIDNotFoundCode int = 404

/*
GetRbacPoliciesRbacPolicyIDNotFound Not Found

swagger:response getRbacPoliciesRbacPolicyIdNotFound
*/
type GetRbacPoliciesRbacPolicyIDNotFound struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetRbacPoliciesRbacPolicyIDNotFound creates GetRbacPoliciesRbacPolicyIDNotFound with default headers values
func NewGetRbacPoliciesRbacPolicyIDNotFound() *GetRbacPoliciesRbacPolicyIDNotFound {

	return &GetRbacPoliciesRbacPolicyIDNotFound{}
}

// WithPayload adds the payload to the get rbac policies rbac policy Id not found response
func (o *GetRbacPoliciesRbacPolicyIDNotFound) WithPayload(payload *models.Error) *GetRbacPoliciesRbacPolicyIDNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get rbac policies rbac policy Id not found response
func (o *GetRbacPoliciesRbacPolicyIDNotFound) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetRbacPoliciesRbacPolicyIDNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}