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

package service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/sapcc/archer/models"
)

// PutServiceServiceIDAcceptEndpointsOKCode is the HTTP code returned for type PutServiceServiceIDAcceptEndpointsOK
const PutServiceServiceIDAcceptEndpointsOKCode int = 200

/*
PutServiceServiceIDAcceptEndpointsOK Ok

swagger:response putServiceServiceIdAcceptEndpointsOK
*/
type PutServiceServiceIDAcceptEndpointsOK struct {

	/*
	  In: Body
	*/
	Payload []*models.EndpointConsumer `json:"body,omitempty"`
}

// NewPutServiceServiceIDAcceptEndpointsOK creates PutServiceServiceIDAcceptEndpointsOK with default headers values
func NewPutServiceServiceIDAcceptEndpointsOK() *PutServiceServiceIDAcceptEndpointsOK {

	return &PutServiceServiceIDAcceptEndpointsOK{}
}

// WithPayload adds the payload to the put service service Id accept endpoints o k response
func (o *PutServiceServiceIDAcceptEndpointsOK) WithPayload(payload []*models.EndpointConsumer) *PutServiceServiceIDAcceptEndpointsOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the put service service Id accept endpoints o k response
func (o *PutServiceServiceIDAcceptEndpointsOK) SetPayload(payload []*models.EndpointConsumer) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PutServiceServiceIDAcceptEndpointsOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	payload := o.Payload
	if payload == nil {
		// return empty array
		payload = make([]*models.EndpointConsumer, 0, 50)
	}

	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

// PutServiceServiceIDAcceptEndpointsForbiddenCode is the HTTP code returned for type PutServiceServiceIDAcceptEndpointsForbidden
const PutServiceServiceIDAcceptEndpointsForbiddenCode int = 403

/*
PutServiceServiceIDAcceptEndpointsForbidden Forbidden

swagger:response putServiceServiceIdAcceptEndpointsForbidden
*/
type PutServiceServiceIDAcceptEndpointsForbidden struct {
}

// NewPutServiceServiceIDAcceptEndpointsForbidden creates PutServiceServiceIDAcceptEndpointsForbidden with default headers values
func NewPutServiceServiceIDAcceptEndpointsForbidden() *PutServiceServiceIDAcceptEndpointsForbidden {

	return &PutServiceServiceIDAcceptEndpointsForbidden{}
}

// WriteResponse to the client
func (o *PutServiceServiceIDAcceptEndpointsForbidden) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(403)
}

// PutServiceServiceIDAcceptEndpointsNotFoundCode is the HTTP code returned for type PutServiceServiceIDAcceptEndpointsNotFound
const PutServiceServiceIDAcceptEndpointsNotFoundCode int = 404

/*
PutServiceServiceIDAcceptEndpointsNotFound Not Found

swagger:response putServiceServiceIdAcceptEndpointsNotFound
*/
type PutServiceServiceIDAcceptEndpointsNotFound struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewPutServiceServiceIDAcceptEndpointsNotFound creates PutServiceServiceIDAcceptEndpointsNotFound with default headers values
func NewPutServiceServiceIDAcceptEndpointsNotFound() *PutServiceServiceIDAcceptEndpointsNotFound {

	return &PutServiceServiceIDAcceptEndpointsNotFound{}
}

// WithPayload adds the payload to the put service service Id accept endpoints not found response
func (o *PutServiceServiceIDAcceptEndpointsNotFound) WithPayload(payload *models.Error) *PutServiceServiceIDAcceptEndpointsNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the put service service Id accept endpoints not found response
func (o *PutServiceServiceIDAcceptEndpointsNotFound) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PutServiceServiceIDAcceptEndpointsNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}