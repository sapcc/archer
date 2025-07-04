// Code generated by go-swagger; DO NOT EDIT.

// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package quota

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// DeleteQuotasProjectIDHandlerFunc turns a function with the right signature into a delete quotas project ID handler
type DeleteQuotasProjectIDHandlerFunc func(DeleteQuotasProjectIDParams, interface{}) middleware.Responder

// Handle executing the request and returning a response
func (fn DeleteQuotasProjectIDHandlerFunc) Handle(params DeleteQuotasProjectIDParams, principal interface{}) middleware.Responder {
	return fn(params, principal)
}

// DeleteQuotasProjectIDHandler interface for that can handle valid delete quotas project ID params
type DeleteQuotasProjectIDHandler interface {
	Handle(DeleteQuotasProjectIDParams, interface{}) middleware.Responder
}

// NewDeleteQuotasProjectID creates a new http.Handler for the delete quotas project ID operation
func NewDeleteQuotasProjectID(ctx *middleware.Context, handler DeleteQuotasProjectIDHandler) *DeleteQuotasProjectID {
	return &DeleteQuotasProjectID{Context: ctx, Handler: handler}
}

/*
	DeleteQuotasProjectID swagger:route DELETE /quotas/{project_id} Quota deleteQuotasProjectId

Reset all Quota of a project
*/
type DeleteQuotasProjectID struct {
	Context *middleware.Context
	Handler DeleteQuotasProjectIDHandler
}

func (o *DeleteQuotasProjectID) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewDeleteQuotasProjectIDParams()
	uprinc, aCtx, err := o.Context.Authorize(r, route)
	if err != nil {
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}
	if aCtx != nil {
		*r = *aCtx
	}
	var principal interface{}
	if uprinc != nil {
		principal = uprinc.(interface{}) // this is really a interface{}, I promise
	}

	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params, principal) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
