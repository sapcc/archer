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

// DeleteEndpointEndpointIDAcceptedCode is the HTTP code returned for type DeleteEndpointEndpointIDAccepted
const DeleteEndpointEndpointIDAcceptedCode int = 202

/*
DeleteEndpointEndpointIDAccepted Delete request successfully accepted.

swagger:response deleteEndpointEndpointIdAccepted
*/
type DeleteEndpointEndpointIDAccepted struct {
}

// NewDeleteEndpointEndpointIDAccepted creates DeleteEndpointEndpointIDAccepted with default headers values
func NewDeleteEndpointEndpointIDAccepted() *DeleteEndpointEndpointIDAccepted {

	return &DeleteEndpointEndpointIDAccepted{}
}

// WriteResponse to the client
func (o *DeleteEndpointEndpointIDAccepted) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(202)
}

// DeleteEndpointEndpointIDUnauthorizedCode is the HTTP code returned for type DeleteEndpointEndpointIDUnauthorized
const DeleteEndpointEndpointIDUnauthorizedCode int = 401

/*
DeleteEndpointEndpointIDUnauthorized Unauthorized

swagger:response deleteEndpointEndpointIdUnauthorized
*/
type DeleteEndpointEndpointIDUnauthorized struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewDeleteEndpointEndpointIDUnauthorized creates DeleteEndpointEndpointIDUnauthorized with default headers values
func NewDeleteEndpointEndpointIDUnauthorized() *DeleteEndpointEndpointIDUnauthorized {

	return &DeleteEndpointEndpointIDUnauthorized{}
}

// WithPayload adds the payload to the delete endpoint endpoint Id unauthorized response
func (o *DeleteEndpointEndpointIDUnauthorized) WithPayload(payload *models.Error) *DeleteEndpointEndpointIDUnauthorized {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete endpoint endpoint Id unauthorized response
func (o *DeleteEndpointEndpointIDUnauthorized) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteEndpointEndpointIDUnauthorized) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(401)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// DeleteEndpointEndpointIDForbiddenCode is the HTTP code returned for type DeleteEndpointEndpointIDForbidden
const DeleteEndpointEndpointIDForbiddenCode int = 403

/*
DeleteEndpointEndpointIDForbidden Forbidden

swagger:response deleteEndpointEndpointIdForbidden
*/
type DeleteEndpointEndpointIDForbidden struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewDeleteEndpointEndpointIDForbidden creates DeleteEndpointEndpointIDForbidden with default headers values
func NewDeleteEndpointEndpointIDForbidden() *DeleteEndpointEndpointIDForbidden {

	return &DeleteEndpointEndpointIDForbidden{}
}

// WithPayload adds the payload to the delete endpoint endpoint Id forbidden response
func (o *DeleteEndpointEndpointIDForbidden) WithPayload(payload *models.Error) *DeleteEndpointEndpointIDForbidden {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete endpoint endpoint Id forbidden response
func (o *DeleteEndpointEndpointIDForbidden) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteEndpointEndpointIDForbidden) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(403)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// DeleteEndpointEndpointIDNotFoundCode is the HTTP code returned for type DeleteEndpointEndpointIDNotFound
const DeleteEndpointEndpointIDNotFoundCode int = 404

/*
DeleteEndpointEndpointIDNotFound Not Found

swagger:response deleteEndpointEndpointIdNotFound
*/
type DeleteEndpointEndpointIDNotFound struct {
}

// NewDeleteEndpointEndpointIDNotFound creates DeleteEndpointEndpointIDNotFound with default headers values
func NewDeleteEndpointEndpointIDNotFound() *DeleteEndpointEndpointIDNotFound {

	return &DeleteEndpointEndpointIDNotFound{}
}

// WriteResponse to the client
func (o *DeleteEndpointEndpointIDNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(404)
}

// DeleteEndpointEndpointIDUnprocessableEntityCode is the HTTP code returned for type DeleteEndpointEndpointIDUnprocessableEntity
const DeleteEndpointEndpointIDUnprocessableEntityCode int = 422

/*
DeleteEndpointEndpointIDUnprocessableEntity Unprocessable Content

swagger:response deleteEndpointEndpointIdUnprocessableEntity
*/
type DeleteEndpointEndpointIDUnprocessableEntity struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewDeleteEndpointEndpointIDUnprocessableEntity creates DeleteEndpointEndpointIDUnprocessableEntity with default headers values
func NewDeleteEndpointEndpointIDUnprocessableEntity() *DeleteEndpointEndpointIDUnprocessableEntity {

	return &DeleteEndpointEndpointIDUnprocessableEntity{}
}

// WithPayload adds the payload to the delete endpoint endpoint Id unprocessable entity response
func (o *DeleteEndpointEndpointIDUnprocessableEntity) WithPayload(payload *models.Error) *DeleteEndpointEndpointIDUnprocessableEntity {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete endpoint endpoint Id unprocessable entity response
func (o *DeleteEndpointEndpointIDUnprocessableEntity) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteEndpointEndpointIDUnprocessableEntity) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(422)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}