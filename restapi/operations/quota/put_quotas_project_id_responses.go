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

package quota

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/sapcc/archer/models"
)

// PutQuotasProjectIDAcceptedCode is the HTTP code returned for type PutQuotasProjectIDAccepted
const PutQuotasProjectIDAcceptedCode int = 202

/*
PutQuotasProjectIDAccepted Updated quota for a project.

swagger:response putQuotasProjectIdAccepted
*/
type PutQuotasProjectIDAccepted struct {

	/*
	  In: Body
	*/
	Payload *PutQuotasProjectIDAcceptedBody `json:"body,omitempty"`
}

// NewPutQuotasProjectIDAccepted creates PutQuotasProjectIDAccepted with default headers values
func NewPutQuotasProjectIDAccepted() *PutQuotasProjectIDAccepted {

	return &PutQuotasProjectIDAccepted{}
}

// WithPayload adds the payload to the put quotas project Id accepted response
func (o *PutQuotasProjectIDAccepted) WithPayload(payload *PutQuotasProjectIDAcceptedBody) *PutQuotasProjectIDAccepted {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the put quotas project Id accepted response
func (o *PutQuotasProjectIDAccepted) SetPayload(payload *PutQuotasProjectIDAcceptedBody) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PutQuotasProjectIDAccepted) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(202)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// PutQuotasProjectIDForbiddenCode is the HTTP code returned for type PutQuotasProjectIDForbidden
const PutQuotasProjectIDForbiddenCode int = 403

/*
PutQuotasProjectIDForbidden Forbidden

swagger:response putQuotasProjectIdForbidden
*/
type PutQuotasProjectIDForbidden struct {
}

// NewPutQuotasProjectIDForbidden creates PutQuotasProjectIDForbidden with default headers values
func NewPutQuotasProjectIDForbidden() *PutQuotasProjectIDForbidden {

	return &PutQuotasProjectIDForbidden{}
}

// WriteResponse to the client
func (o *PutQuotasProjectIDForbidden) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(403)
}

// PutQuotasProjectIDNotFoundCode is the HTTP code returned for type PutQuotasProjectIDNotFound
const PutQuotasProjectIDNotFoundCode int = 404

/*
PutQuotasProjectIDNotFound Not found

swagger:response putQuotasProjectIdNotFound
*/
type PutQuotasProjectIDNotFound struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewPutQuotasProjectIDNotFound creates PutQuotasProjectIDNotFound with default headers values
func NewPutQuotasProjectIDNotFound() *PutQuotasProjectIDNotFound {

	return &PutQuotasProjectIDNotFound{}
}

// WithPayload adds the payload to the put quotas project Id not found response
func (o *PutQuotasProjectIDNotFound) WithPayload(payload *models.Error) *PutQuotasProjectIDNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the put quotas project Id not found response
func (o *PutQuotasProjectIDNotFound) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PutQuotasProjectIDNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}