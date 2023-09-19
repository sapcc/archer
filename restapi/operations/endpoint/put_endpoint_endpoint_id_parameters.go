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
	"io"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
)

// NewPutEndpointEndpointIDParams creates a new PutEndpointEndpointIDParams object
//
// There are no default values defined in the spec.
func NewPutEndpointEndpointIDParams() PutEndpointEndpointIDParams {

	return PutEndpointEndpointIDParams{}
}

// PutEndpointEndpointIDParams contains all the bound params for the put endpoint endpoint ID operation
// typically these are obtained from a http.Request
//
// swagger:parameters PutEndpointEndpointID
type PutEndpointEndpointIDParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*Endpoint object that needs to be updated
	  Required: true
	  In: body
	*/
	Body PutEndpointEndpointIDBody
	/*The UUID of the endpoint
	  Required: true
	  In: path
	*/
	EndpointID strfmt.UUID
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewPutEndpointEndpointIDParams() beforehand.
func (o *PutEndpointEndpointIDParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	if runtime.HasBody(r) {
		defer r.Body.Close()
		var body PutEndpointEndpointIDBody
		if err := route.Consumer.Consume(r.Body, &body); err != nil {
			if err == io.EOF {
				res = append(res, errors.Required("body", "body", ""))
			} else {
				res = append(res, errors.NewParseError("body", "body", "", err))
			}
		} else {
			// validate body object
			if err := body.Validate(route.Formats); err != nil {
				res = append(res, err)
			}

			ctx := validate.WithOperationRequest(r.Context())
			if err := body.ContextValidate(ctx, route.Formats); err != nil {
				res = append(res, err)
			}

			if len(res) == 0 {
				o.Body = body
			}
		}
	} else {
		res = append(res, errors.Required("body", "body", ""))
	}

	rEndpointID, rhkEndpointID, _ := route.Params.GetOK("endpoint_id")
	if err := o.bindEndpointID(rEndpointID, rhkEndpointID, route.Formats); err != nil {
		res = append(res, err)
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindEndpointID binds and validates parameter EndpointID from path.
func (o *PutEndpointEndpointIDParams) bindEndpointID(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// Parameter is provided by construction from the route

	// Format: uuid
	value, err := formats.Parse("uuid", raw)
	if err != nil {
		return errors.InvalidType("endpoint_id", "path", "strfmt.UUID", raw)
	}
	o.EndpointID = *(value.(*strfmt.UUID))

	if err := o.validateEndpointID(formats); err != nil {
		return err
	}

	return nil
}

// validateEndpointID carries on validations for parameter EndpointID
func (o *PutEndpointEndpointIDParams) validateEndpointID(formats strfmt.Registry) error {

	if err := validate.FormatOf("endpoint_id", "path", "uuid", o.EndpointID.String(), formats); err != nil {
		return err
	}
	return nil
}