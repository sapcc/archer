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

package endpoint

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/sapcc/archer/models"
)

// GetEndpointEndpointIDOKCode is the HTTP code returned for type GetEndpointEndpointIDOK
const GetEndpointEndpointIDOKCode int = 200

/*
GetEndpointEndpointIDOK An endpoint detail.

swagger:response getEndpointEndpointIdOK
*/
type GetEndpointEndpointIDOK struct {

	/*
	  In: Body
	*/
	Payload *models.Endpoint `json:"body,omitempty"`
}

// NewGetEndpointEndpointIDOK creates GetEndpointEndpointIDOK with default headers values
func NewGetEndpointEndpointIDOK() *GetEndpointEndpointIDOK {

	return &GetEndpointEndpointIDOK{}
}

// WithPayload adds the payload to the get endpoint endpoint Id o k response
func (o *GetEndpointEndpointIDOK) WithPayload(payload *models.Endpoint) *GetEndpointEndpointIDOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get endpoint endpoint Id o k response
func (o *GetEndpointEndpointIDOK) SetPayload(payload *models.Endpoint) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetEndpointEndpointIDOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetEndpointEndpointIDForbiddenCode is the HTTP code returned for type GetEndpointEndpointIDForbidden
const GetEndpointEndpointIDForbiddenCode int = 403

/*
GetEndpointEndpointIDForbidden Forbidden

swagger:response getEndpointEndpointIdForbidden
*/
type GetEndpointEndpointIDForbidden struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetEndpointEndpointIDForbidden creates GetEndpointEndpointIDForbidden with default headers values
func NewGetEndpointEndpointIDForbidden() *GetEndpointEndpointIDForbidden {

	return &GetEndpointEndpointIDForbidden{}
}

// WithPayload adds the payload to the get endpoint endpoint Id forbidden response
func (o *GetEndpointEndpointIDForbidden) WithPayload(payload *models.Error) *GetEndpointEndpointIDForbidden {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get endpoint endpoint Id forbidden response
func (o *GetEndpointEndpointIDForbidden) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetEndpointEndpointIDForbidden) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(403)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetEndpointEndpointIDNotFoundCode is the HTTP code returned for type GetEndpointEndpointIDNotFound
const GetEndpointEndpointIDNotFoundCode int = 404

/*
GetEndpointEndpointIDNotFound Not Found

swagger:response getEndpointEndpointIdNotFound
*/
type GetEndpointEndpointIDNotFound struct {
}

// NewGetEndpointEndpointIDNotFound creates GetEndpointEndpointIDNotFound with default headers values
func NewGetEndpointEndpointIDNotFound() *GetEndpointEndpointIDNotFound {

	return &GetEndpointEndpointIDNotFound{}
}

// WriteResponse to the client
func (o *GetEndpointEndpointIDNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(404)
}