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
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/sapcc/archer/models"
)

// PutQuotasProjectIDReader is a Reader for the PutQuotasProjectID structure.
type PutQuotasProjectIDReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PutQuotasProjectIDReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewPutQuotasProjectIDOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewPutQuotasProjectIDUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewPutQuotasProjectIDForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewPutQuotasProjectIDNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 422:
		result := NewPutQuotasProjectIDUnprocessableEntity()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewPutQuotasProjectIDOK creates a PutQuotasProjectIDOK with default headers values
func NewPutQuotasProjectIDOK() *PutQuotasProjectIDOK {
	return &PutQuotasProjectIDOK{}
}

/*
PutQuotasProjectIDOK describes a response with status code 200, with default header values.

Updated quota for a project.
*/
type PutQuotasProjectIDOK struct {
	Payload *models.Quota
}

// IsSuccess returns true when this put quotas project Id o k response has a 2xx status code
func (o *PutQuotasProjectIDOK) IsSuccess() bool {
	return true
}

// IsRedirect returns true when this put quotas project Id o k response has a 3xx status code
func (o *PutQuotasProjectIDOK) IsRedirect() bool {
	return false
}

// IsClientError returns true when this put quotas project Id o k response has a 4xx status code
func (o *PutQuotasProjectIDOK) IsClientError() bool {
	return false
}

// IsServerError returns true when this put quotas project Id o k response has a 5xx status code
func (o *PutQuotasProjectIDOK) IsServerError() bool {
	return false
}

// IsCode returns true when this put quotas project Id o k response a status code equal to that given
func (o *PutQuotasProjectIDOK) IsCode(code int) bool {
	return code == 200
}

// Code gets the status code for the put quotas project Id o k response
func (o *PutQuotasProjectIDOK) Code() int {
	return 200
}

func (o *PutQuotasProjectIDOK) Error() string {
	return fmt.Sprintf("[PUT /quotas/{project_id}][%d] putQuotasProjectIdOK  %+v", 200, o.Payload)
}

func (o *PutQuotasProjectIDOK) String() string {
	return fmt.Sprintf("[PUT /quotas/{project_id}][%d] putQuotasProjectIdOK  %+v", 200, o.Payload)
}

func (o *PutQuotasProjectIDOK) GetPayload() *models.Quota {
	return o.Payload
}

func (o *PutQuotasProjectIDOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Quota)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPutQuotasProjectIDUnauthorized creates a PutQuotasProjectIDUnauthorized with default headers values
func NewPutQuotasProjectIDUnauthorized() *PutQuotasProjectIDUnauthorized {
	return &PutQuotasProjectIDUnauthorized{}
}

/*
PutQuotasProjectIDUnauthorized describes a response with status code 401, with default header values.

Unauthorized
*/
type PutQuotasProjectIDUnauthorized struct {
	Payload *models.Error
}

// IsSuccess returns true when this put quotas project Id unauthorized response has a 2xx status code
func (o *PutQuotasProjectIDUnauthorized) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this put quotas project Id unauthorized response has a 3xx status code
func (o *PutQuotasProjectIDUnauthorized) IsRedirect() bool {
	return false
}

// IsClientError returns true when this put quotas project Id unauthorized response has a 4xx status code
func (o *PutQuotasProjectIDUnauthorized) IsClientError() bool {
	return true
}

// IsServerError returns true when this put quotas project Id unauthorized response has a 5xx status code
func (o *PutQuotasProjectIDUnauthorized) IsServerError() bool {
	return false
}

// IsCode returns true when this put quotas project Id unauthorized response a status code equal to that given
func (o *PutQuotasProjectIDUnauthorized) IsCode(code int) bool {
	return code == 401
}

// Code gets the status code for the put quotas project Id unauthorized response
func (o *PutQuotasProjectIDUnauthorized) Code() int {
	return 401
}

func (o *PutQuotasProjectIDUnauthorized) Error() string {
	return fmt.Sprintf("[PUT /quotas/{project_id}][%d] putQuotasProjectIdUnauthorized  %+v", 401, o.Payload)
}

func (o *PutQuotasProjectIDUnauthorized) String() string {
	return fmt.Sprintf("[PUT /quotas/{project_id}][%d] putQuotasProjectIdUnauthorized  %+v", 401, o.Payload)
}

func (o *PutQuotasProjectIDUnauthorized) GetPayload() *models.Error {
	return o.Payload
}

func (o *PutQuotasProjectIDUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPutQuotasProjectIDForbidden creates a PutQuotasProjectIDForbidden with default headers values
func NewPutQuotasProjectIDForbidden() *PutQuotasProjectIDForbidden {
	return &PutQuotasProjectIDForbidden{}
}

/*
PutQuotasProjectIDForbidden describes a response with status code 403, with default header values.

Forbidden
*/
type PutQuotasProjectIDForbidden struct {
}

// IsSuccess returns true when this put quotas project Id forbidden response has a 2xx status code
func (o *PutQuotasProjectIDForbidden) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this put quotas project Id forbidden response has a 3xx status code
func (o *PutQuotasProjectIDForbidden) IsRedirect() bool {
	return false
}

// IsClientError returns true when this put quotas project Id forbidden response has a 4xx status code
func (o *PutQuotasProjectIDForbidden) IsClientError() bool {
	return true
}

// IsServerError returns true when this put quotas project Id forbidden response has a 5xx status code
func (o *PutQuotasProjectIDForbidden) IsServerError() bool {
	return false
}

// IsCode returns true when this put quotas project Id forbidden response a status code equal to that given
func (o *PutQuotasProjectIDForbidden) IsCode(code int) bool {
	return code == 403
}

// Code gets the status code for the put quotas project Id forbidden response
func (o *PutQuotasProjectIDForbidden) Code() int {
	return 403
}

func (o *PutQuotasProjectIDForbidden) Error() string {
	return fmt.Sprintf("[PUT /quotas/{project_id}][%d] putQuotasProjectIdForbidden ", 403)
}

func (o *PutQuotasProjectIDForbidden) String() string {
	return fmt.Sprintf("[PUT /quotas/{project_id}][%d] putQuotasProjectIdForbidden ", 403)
}

func (o *PutQuotasProjectIDForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewPutQuotasProjectIDNotFound creates a PutQuotasProjectIDNotFound with default headers values
func NewPutQuotasProjectIDNotFound() *PutQuotasProjectIDNotFound {
	return &PutQuotasProjectIDNotFound{}
}

/*
PutQuotasProjectIDNotFound describes a response with status code 404, with default header values.

Not found
*/
type PutQuotasProjectIDNotFound struct {
	Payload *models.Error
}

// IsSuccess returns true when this put quotas project Id not found response has a 2xx status code
func (o *PutQuotasProjectIDNotFound) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this put quotas project Id not found response has a 3xx status code
func (o *PutQuotasProjectIDNotFound) IsRedirect() bool {
	return false
}

// IsClientError returns true when this put quotas project Id not found response has a 4xx status code
func (o *PutQuotasProjectIDNotFound) IsClientError() bool {
	return true
}

// IsServerError returns true when this put quotas project Id not found response has a 5xx status code
func (o *PutQuotasProjectIDNotFound) IsServerError() bool {
	return false
}

// IsCode returns true when this put quotas project Id not found response a status code equal to that given
func (o *PutQuotasProjectIDNotFound) IsCode(code int) bool {
	return code == 404
}

// Code gets the status code for the put quotas project Id not found response
func (o *PutQuotasProjectIDNotFound) Code() int {
	return 404
}

func (o *PutQuotasProjectIDNotFound) Error() string {
	return fmt.Sprintf("[PUT /quotas/{project_id}][%d] putQuotasProjectIdNotFound  %+v", 404, o.Payload)
}

func (o *PutQuotasProjectIDNotFound) String() string {
	return fmt.Sprintf("[PUT /quotas/{project_id}][%d] putQuotasProjectIdNotFound  %+v", 404, o.Payload)
}

func (o *PutQuotasProjectIDNotFound) GetPayload() *models.Error {
	return o.Payload
}

func (o *PutQuotasProjectIDNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPutQuotasProjectIDUnprocessableEntity creates a PutQuotasProjectIDUnprocessableEntity with default headers values
func NewPutQuotasProjectIDUnprocessableEntity() *PutQuotasProjectIDUnprocessableEntity {
	return &PutQuotasProjectIDUnprocessableEntity{}
}

/*
PutQuotasProjectIDUnprocessableEntity describes a response with status code 422, with default header values.

Unprocessable Content
*/
type PutQuotasProjectIDUnprocessableEntity struct {
	Payload *models.Error
}

// IsSuccess returns true when this put quotas project Id unprocessable entity response has a 2xx status code
func (o *PutQuotasProjectIDUnprocessableEntity) IsSuccess() bool {
	return false
}

// IsRedirect returns true when this put quotas project Id unprocessable entity response has a 3xx status code
func (o *PutQuotasProjectIDUnprocessableEntity) IsRedirect() bool {
	return false
}

// IsClientError returns true when this put quotas project Id unprocessable entity response has a 4xx status code
func (o *PutQuotasProjectIDUnprocessableEntity) IsClientError() bool {
	return true
}

// IsServerError returns true when this put quotas project Id unprocessable entity response has a 5xx status code
func (o *PutQuotasProjectIDUnprocessableEntity) IsServerError() bool {
	return false
}

// IsCode returns true when this put quotas project Id unprocessable entity response a status code equal to that given
func (o *PutQuotasProjectIDUnprocessableEntity) IsCode(code int) bool {
	return code == 422
}

// Code gets the status code for the put quotas project Id unprocessable entity response
func (o *PutQuotasProjectIDUnprocessableEntity) Code() int {
	return 422
}

func (o *PutQuotasProjectIDUnprocessableEntity) Error() string {
	return fmt.Sprintf("[PUT /quotas/{project_id}][%d] putQuotasProjectIdUnprocessableEntity  %+v", 422, o.Payload)
}

func (o *PutQuotasProjectIDUnprocessableEntity) String() string {
	return fmt.Sprintf("[PUT /quotas/{project_id}][%d] putQuotasProjectIdUnprocessableEntity  %+v", 422, o.Payload)
}

func (o *PutQuotasProjectIDUnprocessableEntity) GetPayload() *models.Error {
	return o.Payload
}

func (o *PutQuotasProjectIDUnprocessableEntity) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}