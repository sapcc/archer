// Code generated by go-swagger; DO NOT EDIT.

// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
)

// NewDeleteServiceServiceIDParams creates a new DeleteServiceServiceIDParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeleteServiceServiceIDParams() *DeleteServiceServiceIDParams {
	return &DeleteServiceServiceIDParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeleteServiceServiceIDParamsWithTimeout creates a new DeleteServiceServiceIDParams object
// with the ability to set a timeout on a request.
func NewDeleteServiceServiceIDParamsWithTimeout(timeout time.Duration) *DeleteServiceServiceIDParams {
	return &DeleteServiceServiceIDParams{
		timeout: timeout,
	}
}

// NewDeleteServiceServiceIDParamsWithContext creates a new DeleteServiceServiceIDParams object
// with the ability to set a context for a request.
func NewDeleteServiceServiceIDParamsWithContext(ctx context.Context) *DeleteServiceServiceIDParams {
	return &DeleteServiceServiceIDParams{
		Context: ctx,
	}
}

// NewDeleteServiceServiceIDParamsWithHTTPClient creates a new DeleteServiceServiceIDParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeleteServiceServiceIDParamsWithHTTPClient(client *http.Client) *DeleteServiceServiceIDParams {
	return &DeleteServiceServiceIDParams{
		HTTPClient: client,
	}
}

/*
DeleteServiceServiceIDParams contains all the parameters to send to the API endpoint

	for the delete service service ID operation.

	Typically these are written to a http.Request.
*/
type DeleteServiceServiceIDParams struct {

	/* ServiceID.

	   The UUID of the service

	   Format: uuid
	*/
	ServiceID strfmt.UUID

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete service service ID params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteServiceServiceIDParams) WithDefaults() *DeleteServiceServiceIDParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete service service ID params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeleteServiceServiceIDParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete service service ID params
func (o *DeleteServiceServiceIDParams) WithTimeout(timeout time.Duration) *DeleteServiceServiceIDParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete service service ID params
func (o *DeleteServiceServiceIDParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete service service ID params
func (o *DeleteServiceServiceIDParams) WithContext(ctx context.Context) *DeleteServiceServiceIDParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete service service ID params
func (o *DeleteServiceServiceIDParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete service service ID params
func (o *DeleteServiceServiceIDParams) WithHTTPClient(client *http.Client) *DeleteServiceServiceIDParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete service service ID params
func (o *DeleteServiceServiceIDParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithServiceID adds the serviceID to the delete service service ID params
func (o *DeleteServiceServiceIDParams) WithServiceID(serviceID strfmt.UUID) *DeleteServiceServiceIDParams {
	o.SetServiceID(serviceID)
	return o
}

// SetServiceID adds the serviceId to the delete service service ID params
func (o *DeleteServiceServiceIDParams) SetServiceID(serviceID strfmt.UUID) {
	o.ServiceID = serviceID
}

// WriteToRequest writes these params to a swagger request
func (o *DeleteServiceServiceIDParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param service_id
	if err := r.SetPathParam("service_id", o.ServiceID.String()); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
