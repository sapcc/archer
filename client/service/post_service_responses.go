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
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/sapcc/archer/models"
)

// PostServiceReader is a Reader for the PostService structure.
type PostServiceReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PostServiceReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewPostServiceOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 403:
		result := NewPostServiceForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 409:
		result := NewPostServiceConflict()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewPostServiceOK creates a PostServiceOK with default headers values
func NewPostServiceOK() *PostServiceOK {
	return &PostServiceOK{}
}

/*
PostServiceOK describes a response with status code 200, with default header values.

Service
*/
type PostServiceOK struct {
	Payload *models.Service
}

// IsSuccess returns true when this post service o k response has a 2xx status code
func (o *PostServiceOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this post service o k response has a 3xx status code
func (o *PostServiceOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this post service o k response has a 4xx status code
func (o *PostServiceOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this post service o k response has a 5xx status code
func (o *PostServiceOK) IsServerError() bool {
	return false
}

// IsCode returns true when this post service o k response a status code equal to that given
func (o *PostServiceOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the post service o k response
func (o *PostServiceOK) Code() int {
	return 200
}

func (o *PostServiceOK) Error() string {
	return fmt.Sprintf("[POST /service][%d] postServiceOK  %+v", 200, o.Payload)
}

func (o *PostServiceOK) String() string {
	return fmt.Sprintf("[POST /service][%d] postServiceOK  %+v", 200, o.Payload)
}

func (o *PostServiceOK) GetPayload() *models.Service {
	return o.Payload
}

func (o *PostServiceOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Service)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPostServiceForbidden creates a PostServiceForbidden with default headers values
func NewPostServiceForbidden() *PostServiceForbidden {
	return &PostServiceForbidden{}
}

/*
PostServiceForbidden describes a response with status code 403, with default header values.

Forbidden
*/
type PostServiceForbidden struct {
}

// IsSuccess returns true when this post service forbidden response has a 2xx status code
func (o *PostServiceForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this post service forbidden response has a 3xx status code
func (o *PostServiceForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this post service forbidden response has a 4xx status code
func (o *PostServiceForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this post service forbidden response has a 5xx status code
func (o *PostServiceForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this post service forbidden response a status code equal to that given
func (o *PostServiceForbidden) IsCode(code int) bool {
	return code == 403
}

// Code gets the status code for the post service forbidden response
func (o *PostServiceForbidden) Code() int {
	return 403
}

func (o *PostServiceForbidden) Error() string {
	return fmt.Sprintf("[POST /service][%d] postServiceForbidden ", 403)
}

func (o *PostServiceForbidden) String() string {
	return fmt.Sprintf("[POST /service][%d] postServiceForbidden ", 403)
}

func (o *PostServiceForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPostServiceConflict creates a PostServiceConflict with default headers values
func NewPostServiceConflict() *PostServiceConflict {
	return &PostServiceConflict{}
}

/*
PostServiceConflict describes a response with status code 409, with default header values.

Duplicate entry
*/
type PostServiceConflict struct {
	Payload *models.Error
}

// IsSuccess returns true when this post service conflict response has a 2xx status code
func (o *PostServiceConflict) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this post service conflict response has a 3xx status code
func (o *PostServiceConflict) IsRedirect() bool {
	return false
}

// IsClientError returns true when this post service conflict response has a 4xx status code
func (o *PostServiceConflict) IsClientError() bool {
	return true
}

// IsServerError returns true when this post service conflict response has a 5xx status code
func (o *PostServiceConflict) IsServerError() bool {
	return false
}

// IsCode returns true when this post service conflict response a status code equal to that given
func (o *PostServiceConflict) IsCode(code int) bool {
	return code == 409
}

// Code gets the status code for the post service conflict response
func (o *PostServiceConflict) Code() int {
	return 409
}

func (o *PostServiceConflict) Error() string {
	return fmt.Sprintf("[POST /service][%d] postServiceConflict  %+v", 409, o.Payload)
}

func (o *PostServiceConflict) String() string {
	return fmt.Sprintf("[POST /service][%d] postServiceConflict  %+v", 409, o.Payload)
}

func (o *PostServiceConflict) GetPayload() *models.Error {
	return o.Payload
}

func (o *PostServiceConflict) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}