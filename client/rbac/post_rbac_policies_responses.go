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
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/sapcc/archer/models"
)

// PostRbacPoliciesReader is a Reader for the PostRbacPolicies structure.
type PostRbacPoliciesReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PostRbacPoliciesReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewPostRbacPoliciesOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 403:
		result := NewPostRbacPoliciesForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 409:
		result := NewPostRbacPoliciesConflict()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewPostRbacPoliciesOK creates a PostRbacPoliciesOK with default headers values
func NewPostRbacPoliciesOK() *PostRbacPoliciesOK {
	return &PostRbacPoliciesOK{}
}

/*
PostRbacPoliciesOK describes a response with status code 200, with default header values.

RBAC policy
*/
type PostRbacPoliciesOK struct {
	Payload *models.Rbacpolicy
}

// IsSuccess returns true when this post rbac policies o k response has a 2xx status code
func (o *PostRbacPoliciesOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this post rbac policies o k response has a 3xx status code
func (o *PostRbacPoliciesOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this post rbac policies o k response has a 4xx status code
func (o *PostRbacPoliciesOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this post rbac policies o k response has a 5xx status code
func (o *PostRbacPoliciesOK) IsServerError() bool {
	return false
}

// IsCode returns true when this post rbac policies o k response a status code equal to that given
func (o *PostRbacPoliciesOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the post rbac policies o k response
func (o *PostRbacPoliciesOK) Code() int {
	return 200
}

func (o *PostRbacPoliciesOK) Error() string {
	return fmt.Sprintf("[POST /rbac-policies][%d] postRbacPoliciesOK  %+v", 200, o.Payload)
}

func (o *PostRbacPoliciesOK) String() string {
	return fmt.Sprintf("[POST /rbac-policies][%d] postRbacPoliciesOK  %+v", 200, o.Payload)
}

func (o *PostRbacPoliciesOK) GetPayload() *models.Rbacpolicy {
	return o.Payload
}

func (o *PostRbacPoliciesOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Rbacpolicy)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPostRbacPoliciesForbidden creates a PostRbacPoliciesForbidden with default headers values
func NewPostRbacPoliciesForbidden() *PostRbacPoliciesForbidden {
	return &PostRbacPoliciesForbidden{}
}

/*
PostRbacPoliciesForbidden describes a response with status code 403, with default header values.

Forbidden
*/
type PostRbacPoliciesForbidden struct {
}

// IsSuccess returns true when this post rbac policies forbidden response has a 2xx status code
func (o *PostRbacPoliciesForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this post rbac policies forbidden response has a 3xx status code
func (o *PostRbacPoliciesForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this post rbac policies forbidden response has a 4xx status code
func (o *PostRbacPoliciesForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this post rbac policies forbidden response has a 5xx status code
func (o *PostRbacPoliciesForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this post rbac policies forbidden response a status code equal to that given
func (o *PostRbacPoliciesForbidden) IsCode(code int) bool {
	return code == 403
}

// Code gets the status code for the post rbac policies forbidden response
func (o *PostRbacPoliciesForbidden) Code() int {
	return 403
}

func (o *PostRbacPoliciesForbidden) Error() string {
	return fmt.Sprintf("[POST /rbac-policies][%d] postRbacPoliciesForbidden ", 403)
}

func (o *PostRbacPoliciesForbidden) String() string {
	return fmt.Sprintf("[POST /rbac-policies][%d] postRbacPoliciesForbidden ", 403)
}

func (o *PostRbacPoliciesForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPostRbacPoliciesConflict creates a PostRbacPoliciesConflict with default headers values
func NewPostRbacPoliciesConflict() *PostRbacPoliciesConflict {
	return &PostRbacPoliciesConflict{}
}

/*
PostRbacPoliciesConflict describes a response with status code 409, with default header values.

Exists
*/
type PostRbacPoliciesConflict struct {
	Payload *models.Error
}

// IsSuccess returns true when this post rbac policies conflict response has a 2xx status code
func (o *PostRbacPoliciesConflict) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this post rbac policies conflict response has a 3xx status code
func (o *PostRbacPoliciesConflict) IsRedirect() bool {
	return false
}

// IsClientError returns true when this post rbac policies conflict response has a 4xx status code
func (o *PostRbacPoliciesConflict) IsClientError() bool {
	return true
}

// IsServerError returns true when this post rbac policies conflict response has a 5xx status code
func (o *PostRbacPoliciesConflict) IsServerError() bool {
	return false
}

// IsCode returns true when this post rbac policies conflict response a status code equal to that given
func (o *PostRbacPoliciesConflict) IsCode(code int) bool {
	return code == 409
}

// Code gets the status code for the post rbac policies conflict response
func (o *PostRbacPoliciesConflict) Code() int {
	return 409
}

func (o *PostRbacPoliciesConflict) Error() string {
	return fmt.Sprintf("[POST /rbac-policies][%d] postRbacPoliciesConflict  %+v", 409, o.Payload)
}

func (o *PostRbacPoliciesConflict) String() string {
	return fmt.Sprintf("[POST /rbac-policies][%d] postRbacPoliciesConflict  %+v", 409, o.Payload)
}

func (o *PostRbacPoliciesConflict) GetPayload() *models.Error {
	return o.Payload
}

func (o *PostRbacPoliciesConflict) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}