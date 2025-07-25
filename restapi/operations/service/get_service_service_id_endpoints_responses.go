// Code generated by go-swagger; DO NOT EDIT.

// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/sapcc/archer/models"
)

// GetServiceServiceIDEndpointsOKCode is the HTTP code returned for type GetServiceServiceIDEndpointsOK
const GetServiceServiceIDEndpointsOKCode int = 200

/*
GetServiceServiceIDEndpointsOK An array of service endpoint consumers.

swagger:response getServiceServiceIdEndpointsOK
*/
type GetServiceServiceIDEndpointsOK struct {

	/*
	  In: Body
	*/
	Payload *GetServiceServiceIDEndpointsOKBody `json:"body,omitempty"`
}

// NewGetServiceServiceIDEndpointsOK creates GetServiceServiceIDEndpointsOK with default headers values
func NewGetServiceServiceIDEndpointsOK() *GetServiceServiceIDEndpointsOK {

	return &GetServiceServiceIDEndpointsOK{}
}

// WithPayload adds the payload to the get service service Id endpoints o k response
func (o *GetServiceServiceIDEndpointsOK) WithPayload(payload *GetServiceServiceIDEndpointsOKBody) *GetServiceServiceIDEndpointsOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get service service Id endpoints o k response
func (o *GetServiceServiceIDEndpointsOK) SetPayload(payload *GetServiceServiceIDEndpointsOKBody) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetServiceServiceIDEndpointsOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetServiceServiceIDEndpointsBadRequestCode is the HTTP code returned for type GetServiceServiceIDEndpointsBadRequest
const GetServiceServiceIDEndpointsBadRequestCode int = 400

/*
GetServiceServiceIDEndpointsBadRequest Bad request

swagger:response getServiceServiceIdEndpointsBadRequest
*/
type GetServiceServiceIDEndpointsBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetServiceServiceIDEndpointsBadRequest creates GetServiceServiceIDEndpointsBadRequest with default headers values
func NewGetServiceServiceIDEndpointsBadRequest() *GetServiceServiceIDEndpointsBadRequest {

	return &GetServiceServiceIDEndpointsBadRequest{}
}

// WithPayload adds the payload to the get service service Id endpoints bad request response
func (o *GetServiceServiceIDEndpointsBadRequest) WithPayload(payload *models.Error) *GetServiceServiceIDEndpointsBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get service service Id endpoints bad request response
func (o *GetServiceServiceIDEndpointsBadRequest) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetServiceServiceIDEndpointsBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetServiceServiceIDEndpointsUnauthorizedCode is the HTTP code returned for type GetServiceServiceIDEndpointsUnauthorized
const GetServiceServiceIDEndpointsUnauthorizedCode int = 401

/*
GetServiceServiceIDEndpointsUnauthorized Unauthorized

swagger:response getServiceServiceIdEndpointsUnauthorized
*/
type GetServiceServiceIDEndpointsUnauthorized struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetServiceServiceIDEndpointsUnauthorized creates GetServiceServiceIDEndpointsUnauthorized with default headers values
func NewGetServiceServiceIDEndpointsUnauthorized() *GetServiceServiceIDEndpointsUnauthorized {

	return &GetServiceServiceIDEndpointsUnauthorized{}
}

// WithPayload adds the payload to the get service service Id endpoints unauthorized response
func (o *GetServiceServiceIDEndpointsUnauthorized) WithPayload(payload *models.Error) *GetServiceServiceIDEndpointsUnauthorized {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get service service Id endpoints unauthorized response
func (o *GetServiceServiceIDEndpointsUnauthorized) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetServiceServiceIDEndpointsUnauthorized) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(401)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetServiceServiceIDEndpointsForbiddenCode is the HTTP code returned for type GetServiceServiceIDEndpointsForbidden
const GetServiceServiceIDEndpointsForbiddenCode int = 403

/*
GetServiceServiceIDEndpointsForbidden Forbidden

swagger:response getServiceServiceIdEndpointsForbidden
*/
type GetServiceServiceIDEndpointsForbidden struct {
}

// NewGetServiceServiceIDEndpointsForbidden creates GetServiceServiceIDEndpointsForbidden with default headers values
func NewGetServiceServiceIDEndpointsForbidden() *GetServiceServiceIDEndpointsForbidden {

	return &GetServiceServiceIDEndpointsForbidden{}
}

// WriteResponse to the client
func (o *GetServiceServiceIDEndpointsForbidden) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(403)
}

// GetServiceServiceIDEndpointsNotFoundCode is the HTTP code returned for type GetServiceServiceIDEndpointsNotFound
const GetServiceServiceIDEndpointsNotFoundCode int = 404

/*
GetServiceServiceIDEndpointsNotFound Not Found

swagger:response getServiceServiceIdEndpointsNotFound
*/
type GetServiceServiceIDEndpointsNotFound struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetServiceServiceIDEndpointsNotFound creates GetServiceServiceIDEndpointsNotFound with default headers values
func NewGetServiceServiceIDEndpointsNotFound() *GetServiceServiceIDEndpointsNotFound {

	return &GetServiceServiceIDEndpointsNotFound{}
}

// WithPayload adds the payload to the get service service Id endpoints not found response
func (o *GetServiceServiceIDEndpointsNotFound) WithPayload(payload *models.Error) *GetServiceServiceIDEndpointsNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get service service Id endpoints not found response
func (o *GetServiceServiceIDEndpointsNotFound) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetServiceServiceIDEndpointsNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetServiceServiceIDEndpointsUnprocessableEntityCode is the HTTP code returned for type GetServiceServiceIDEndpointsUnprocessableEntity
const GetServiceServiceIDEndpointsUnprocessableEntityCode int = 422

/*
GetServiceServiceIDEndpointsUnprocessableEntity Unprocessable Content

swagger:response getServiceServiceIdEndpointsUnprocessableEntity
*/
type GetServiceServiceIDEndpointsUnprocessableEntity struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetServiceServiceIDEndpointsUnprocessableEntity creates GetServiceServiceIDEndpointsUnprocessableEntity with default headers values
func NewGetServiceServiceIDEndpointsUnprocessableEntity() *GetServiceServiceIDEndpointsUnprocessableEntity {

	return &GetServiceServiceIDEndpointsUnprocessableEntity{}
}

// WithPayload adds the payload to the get service service Id endpoints unprocessable entity response
func (o *GetServiceServiceIDEndpointsUnprocessableEntity) WithPayload(payload *models.Error) *GetServiceServiceIDEndpointsUnprocessableEntity {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get service service Id endpoints unprocessable entity response
func (o *GetServiceServiceIDEndpointsUnprocessableEntity) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetServiceServiceIDEndpointsUnprocessableEntity) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(422)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
