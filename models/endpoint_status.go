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

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"encoding/json"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
)

// EndpointStatus Status of the endpoint
//
// ### Status can be one of
// | Status             | Description                           |
// | ------------------ | ------------------------------------- |
// | AVAILABLE          | Endpoint is active for consumption    |
// | PENDING_APPROVAL   | Endpoint is waiting for approval      |
// | PENDING_CREATE     | Endpoint is being set up              |
// | PENDING_REJECTED   | Endpoint is being rejected            |
// | PENDING_DELETE     | Endpoint is being deleted             |
// | REJECTED           | Endpoint was rejected                 |
// | FAILED             | Endpoint setup failed                 |
//
// swagger:model EndpointStatus
type EndpointStatus string

func NewEndpointStatus(value EndpointStatus) *EndpointStatus {
	return &value
}

// Pointer returns a pointer to a freshly-allocated EndpointStatus.
func (m EndpointStatus) Pointer() *EndpointStatus {
	return &m
}

const (

	// EndpointStatusACTIVE captures enum value "ACTIVE"
	EndpointStatusACTIVE EndpointStatus = "ACTIVE"

	// EndpointStatusPENDINGAPPROVAL captures enum value "PENDING_APPROVAL"
	EndpointStatusPENDINGAPPROVAL EndpointStatus = "PENDING_APPROVAL"

	// EndpointStatusPENDINGCREATE captures enum value "PENDING_CREATE"
	EndpointStatusPENDINGCREATE EndpointStatus = "PENDING_CREATE"

	// EndpointStatusPENDINGREJECTED captures enum value "PENDING_REJECTED"
	EndpointStatusPENDINGREJECTED EndpointStatus = "PENDING_REJECTED"

	// EndpointStatusPENDINGDELETE captures enum value "PENDING_DELETE"
	EndpointStatusPENDINGDELETE EndpointStatus = "PENDING_DELETE"

	// EndpointStatusREJECTED captures enum value "REJECTED"
	EndpointStatusREJECTED EndpointStatus = "REJECTED"

	// EndpointStatusFAILED captures enum value "FAILED"
	EndpointStatusFAILED EndpointStatus = "FAILED"
)

// for schema
var endpointStatusEnum []interface{}

func init() {
	var res []EndpointStatus
	if err := json.Unmarshal([]byte(`["ACTIVE","PENDING_APPROVAL","PENDING_CREATE","PENDING_REJECTED","PENDING_DELETE","REJECTED","FAILED"]`), &res); err != nil {
		panic(err)
	}
	for _, v := range res {
		endpointStatusEnum = append(endpointStatusEnum, v)
	}
}

func (m EndpointStatus) validateEndpointStatusEnum(path, location string, value EndpointStatus) error {
	if err := validate.EnumCase(path, location, value, endpointStatusEnum, true); err != nil {
		return err
	}
	return nil
}

// Validate validates this endpoint status
func (m EndpointStatus) Validate(formats strfmt.Registry) error {
	var res []error

	// value enum
	if err := m.validateEndpointStatusEnum("", "body", m); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// ContextValidate validate this endpoint status based on the context it is used
func (m EndpointStatus) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	var res []error

	if err := validate.ReadOnly(ctx, "", "body", EndpointStatus(m)); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}