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

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
)

// NewGetQuotasParams creates a new GetQuotasParams object
//
// There are no default values defined in the spec.
func NewGetQuotasParams() GetQuotasParams {

	return GetQuotasParams{}
}

// GetQuotasParams contains all the bound params for the get quotas operation
// typically these are obtained from a http.Request
//
// swagger:parameters GetQuotas
type GetQuotasParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*The ID of the project to query.
	  Max Length: 32
	  Min Length: 32
	  In: query
	*/
	ProjectID *string
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewGetQuotasParams() beforehand.
func (o *GetQuotasParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	qs := runtime.Values(r.URL.Query())

	qProjectID, qhkProjectID, _ := qs.GetOK("project_id")
	if err := o.bindProjectID(qProjectID, qhkProjectID, route.Formats); err != nil {
		res = append(res, err)
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindProjectID binds and validates parameter ProjectID from query.
func (o *GetQuotasParams) bindProjectID(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: false
	// AllowEmptyValue: false

	if raw == "" { // empty values pass all other validations
		return nil
	}
	o.ProjectID = &raw

	if err := o.validateProjectID(formats); err != nil {
		return err
	}

	return nil
}

// validateProjectID carries on validations for parameter ProjectID
func (o *GetQuotasParams) validateProjectID(formats strfmt.Registry) error {

	if err := validate.MinLength("project_id", "query", *o.ProjectID, 32); err != nil {
		return err
	}

	if err := validate.MaxLength("project_id", "query", *o.ProjectID, 32); err != nil {
		return err
	}

	return nil
}
