// Code generated by go-swagger; DO NOT EDIT.

// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/security"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"

	"github.com/sapcc/archer/restapi/operations/endpoint"
	"github.com/sapcc/archer/restapi/operations/quota"
	"github.com/sapcc/archer/restapi/operations/rbac"
	"github.com/sapcc/archer/restapi/operations/service"
	"github.com/sapcc/archer/restapi/operations/version"
)

// NewArcherAPI creates a new Archer instance
func NewArcherAPI(spec *loads.Document) *ArcherAPI {
	return &ArcherAPI{
		handlers:            make(map[string]map[string]http.Handler),
		formats:             strfmt.Default,
		defaultConsumes:     "application/json",
		defaultProduces:     "application/json",
		customConsumers:     make(map[string]runtime.Consumer),
		customProducers:     make(map[string]runtime.Producer),
		PreServerShutdown:   func() {},
		ServerShutdown:      func() {},
		spec:                spec,
		useSwaggerUI:        false,
		ServeError:          errors.ServeError,
		BasicAuthenticator:  security.BasicAuth,
		APIKeyAuthenticator: security.APIKeyAuth,
		BearerAuthenticator: security.BearerAuth,

		JSONConsumer: runtime.JSONConsumer(),

		JSONProducer: runtime.JSONProducer(),

		EndpointDeleteEndpointEndpointIDHandler: endpoint.DeleteEndpointEndpointIDHandlerFunc(func(params endpoint.DeleteEndpointEndpointIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation endpoint.DeleteEndpointEndpointID has not yet been implemented")
		}),
		QuotaDeleteQuotasProjectIDHandler: quota.DeleteQuotasProjectIDHandlerFunc(func(params quota.DeleteQuotasProjectIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation quota.DeleteQuotasProjectID has not yet been implemented")
		}),
		RbacDeleteRbacPoliciesRbacPolicyIDHandler: rbac.DeleteRbacPoliciesRbacPolicyIDHandlerFunc(func(params rbac.DeleteRbacPoliciesRbacPolicyIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation rbac.DeleteRbacPoliciesRbacPolicyID has not yet been implemented")
		}),
		ServiceDeleteServiceServiceIDHandler: service.DeleteServiceServiceIDHandlerFunc(func(params service.DeleteServiceServiceIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.DeleteServiceServiceID has not yet been implemented")
		}),
		VersionGetHandler: version.GetHandlerFunc(func(params version.GetParams) middleware.Responder {
			return middleware.NotImplemented("operation version.Get has not yet been implemented")
		}),
		EndpointGetEndpointHandler: endpoint.GetEndpointHandlerFunc(func(params endpoint.GetEndpointParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation endpoint.GetEndpoint has not yet been implemented")
		}),
		EndpointGetEndpointEndpointIDHandler: endpoint.GetEndpointEndpointIDHandlerFunc(func(params endpoint.GetEndpointEndpointIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation endpoint.GetEndpointEndpointID has not yet been implemented")
		}),
		QuotaGetQuotasHandler: quota.GetQuotasHandlerFunc(func(params quota.GetQuotasParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation quota.GetQuotas has not yet been implemented")
		}),
		QuotaGetQuotasDefaultsHandler: quota.GetQuotasDefaultsHandlerFunc(func(params quota.GetQuotasDefaultsParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation quota.GetQuotasDefaults has not yet been implemented")
		}),
		QuotaGetQuotasProjectIDHandler: quota.GetQuotasProjectIDHandlerFunc(func(params quota.GetQuotasProjectIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation quota.GetQuotasProjectID has not yet been implemented")
		}),
		RbacGetRbacPoliciesHandler: rbac.GetRbacPoliciesHandlerFunc(func(params rbac.GetRbacPoliciesParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation rbac.GetRbacPolicies has not yet been implemented")
		}),
		RbacGetRbacPoliciesRbacPolicyIDHandler: rbac.GetRbacPoliciesRbacPolicyIDHandlerFunc(func(params rbac.GetRbacPoliciesRbacPolicyIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation rbac.GetRbacPoliciesRbacPolicyID has not yet been implemented")
		}),
		ServiceGetServiceHandler: service.GetServiceHandlerFunc(func(params service.GetServiceParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.GetService has not yet been implemented")
		}),
		ServiceGetServiceServiceIDHandler: service.GetServiceServiceIDHandlerFunc(func(params service.GetServiceServiceIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.GetServiceServiceID has not yet been implemented")
		}),
		ServiceGetServiceServiceIDEndpointsHandler: service.GetServiceServiceIDEndpointsHandlerFunc(func(params service.GetServiceServiceIDEndpointsParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.GetServiceServiceIDEndpoints has not yet been implemented")
		}),
		EndpointPostEndpointHandler: endpoint.PostEndpointHandlerFunc(func(params endpoint.PostEndpointParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation endpoint.PostEndpoint has not yet been implemented")
		}),
		RbacPostRbacPoliciesHandler: rbac.PostRbacPoliciesHandlerFunc(func(params rbac.PostRbacPoliciesParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation rbac.PostRbacPolicies has not yet been implemented")
		}),
		ServicePostServiceHandler: service.PostServiceHandlerFunc(func(params service.PostServiceParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.PostService has not yet been implemented")
		}),
		EndpointPutEndpointEndpointIDHandler: endpoint.PutEndpointEndpointIDHandlerFunc(func(params endpoint.PutEndpointEndpointIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation endpoint.PutEndpointEndpointID has not yet been implemented")
		}),
		QuotaPutQuotasProjectIDHandler: quota.PutQuotasProjectIDHandlerFunc(func(params quota.PutQuotasProjectIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation quota.PutQuotasProjectID has not yet been implemented")
		}),
		RbacPutRbacPoliciesRbacPolicyIDHandler: rbac.PutRbacPoliciesRbacPolicyIDHandlerFunc(func(params rbac.PutRbacPoliciesRbacPolicyIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation rbac.PutRbacPoliciesRbacPolicyID has not yet been implemented")
		}),
		ServicePutServiceServiceIDHandler: service.PutServiceServiceIDHandlerFunc(func(params service.PutServiceServiceIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.PutServiceServiceID has not yet been implemented")
		}),
		ServicePutServiceServiceIDAcceptEndpointsHandler: service.PutServiceServiceIDAcceptEndpointsHandlerFunc(func(params service.PutServiceServiceIDAcceptEndpointsParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.PutServiceServiceIDAcceptEndpoints has not yet been implemented")
		}),
		ServicePutServiceServiceIDRejectEndpointsHandler: service.PutServiceServiceIDRejectEndpointsHandlerFunc(func(params service.PutServiceServiceIDRejectEndpointsParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.PutServiceServiceIDRejectEndpoints has not yet been implemented")
		}),

		// Applies when the "X-Auth-Token" header is set
		XAuthTokenAuth: func(token string) (interface{}, error) {
			return nil, errors.NotImplemented("api key auth (X-Auth-Token) X-Auth-Token from header param [X-Auth-Token] has not yet been implemented")
		},
		// default authorizer is authorized meaning no requests are blocked
		APIAuthorizer: security.Authorized(),
	}
}

/*
ArcherAPI # Documentation
Archer is an API service that can privately connect services from one private [OpenStack Network](https://docs.openstack.org/neutron/latest/admin/intro-os-networking.html) to another. Consumers can select a *service* from a service catalog and **inject** it to their network, which means making this *service* available via a private ip address.

Archer implements an *OpenStack* like API and integrates with *OpenStack Keystone* and *OpenStack Neutron*.

### Architecture
There are two types of resources: **services** and **endpoints**

* **Services** are private or public services that are manually configured in *Archer*. They can be accessed by creating an endpoint.
* **Service endpoints**, or short **endpoints**, are IP endpoints in a local network used to transparently access services residing in different private networks.

### Features
* Multi-tenant capable via OpenStack Identity service
* OpenStack `policy.json` access policy support
* Prometheus Exporter
* Rate limiting

### Supported Backends
* F5 BigIP

### Requirements
* PostgreSQL Database

## API properties
This section describes properties of the Archer API. It uses a ReSTful HTTP API.

#### Request format
The Archer API only accepts requests with the JSON data serialization format. The Content-Type header for POST requests is always expected to be `application/json`.

#### Response format
The Archer API always response with JSON data serialization format. The Content-Type header is always `Content-Type: application/json`.

#### Authentication and authorization
The **Archer API** uses the OpenStack Identity service as the default authentication service. When Keystone is enabled, users that submit requests to the OpenStack Networking service must provide an authentication token in `X-Auth-Token` request header.
You obtain the token by authenticating to the Keystone endpoint.

When Keystone is enabled, the `project_id` attribute is not required in create requests because the project ID is derived from the authentication token.

#### Pagination
To reduce load on the service, list operations will return a maximum number of items at a time. To navigate the collection, the parameters limit, marker and page_reverse can be set in the URI. For example:

```
?limit=100&marker=1234&page_reverse=False
```

The `marker` parameter is the ID of the last item in the previous list. The `limit` parameter sets the page size. The `page_reverse` parameter sets the page direction.
These parameters are optional.
If the client requests a limit beyond the maximum limit configured by the deployment, the server returns the maximum limit number of items.

For convenience, list responses contain atom **next** links and **previous** links. The last page in the list requested with `page_reverse=False` will not contain **next** link, and the last page in the list requested with `page_reverse=True` will not contain **previous** link.

To determine if pagination is supported, a user can check whether the `pagination` capability is available through the Archer API detail endpoint.

#### Sorting
You can use the `sort` parameter to sort the results of list operations.
The sort parameter contains a comma-separated list of sort keys, in order of the sort priority. Each sort key can be optionally prepended with a minus **-** character to reverse default sort direction (ascending).

For example:

```
?sort=key1,-key2,key3
```

**key1** is the first key (ascending order), **key2** is the second key (descending order) and **key3** is the third key in ascending order.

To determine if sorting is supported, a user can check whether the `sort` capability is available through the Archer API detail endpoint.

#### Filtering by tags
Most resources (e.g. service and endpoint) support adding tags to the resource attributes. Archer supports advanced filtering using these tags for list operations. The following tag filters are supported by the Archer API:

* `tags` - Return the list of entities that have this tag or tags.
* `tags-any` - Return the list of entities that have one or more of the given tags.
* `not-tags` - Return the list of entities that do not have one or more of the given tags.
* `not-tags-any` - Return the list of entities that do not have at least one of the given tags.

Each tag supports a maximum amount of 64 characters.

For example to get a list of resources having both, **red** and **blue** tags:

```
?tags=red,blue
```

To get a list of resourcing having either, **red** or **blue** tags:

```
?tags-any=red,blue
```

Tag filters can also be combined in the same request:

```
?tags=red,blue&tags-any=green,orange
```

#### Response Codes (Faults)

| Code  | Description       |
| ----- | ----------------- |
| 400   | Validation Error |
| 401   | Unauthorized |
| 403   | Policy does not allow current user to do this <br> The project is over quota for the request |
| 404   | Not Found <br> Resource not found |
| 409   | Conflict |
| 422   | Unprocessable Entity |
| 429   | You have reached maximum request limit |
| 500   | Internal server error |

## Endpoint identification

Archer supports the Proxy Protocol v2 for endpoint identification.

The Proxy Protocol is a widely used protocol for passing client connection information through a load balancer to the backend server. It is used to identify the original client IP address and port number. The Proxy Protocol v2 is a binary protocol that is more efficient than the original text-based Proxy Protocol v1.

The proxy protocol header also includes the ID of the endpoint. This information is encoded using a custom Type-Length-Value (TLV) vector as follows.

| Field | Length (Octets) | Description                                                    |
| ----- | --------------- | -------------------------------------------------------------- |
| Type  | 1               | PP2_TYPE_SAPCC (0xEC)                                          |
| Length| 2               | Length of the value (UUIDv4 is always 36 byte as ASCII string) |
| Value | 36              | ASCII UUID of the endpoint                                     |
*/
type ArcherAPI struct {
	spec            *loads.Document
	context         *middleware.Context
	handlers        map[string]map[string]http.Handler
	formats         strfmt.Registry
	customConsumers map[string]runtime.Consumer
	customProducers map[string]runtime.Producer
	defaultConsumes string
	defaultProduces string
	Middleware      func(middleware.Builder) http.Handler
	useSwaggerUI    bool

	// BasicAuthenticator generates a runtime.Authenticator from the supplied basic auth function.
	// It has a default implementation in the security package, however you can replace it for your particular usage.
	BasicAuthenticator func(security.UserPassAuthentication) runtime.Authenticator

	// APIKeyAuthenticator generates a runtime.Authenticator from the supplied token auth function.
	// It has a default implementation in the security package, however you can replace it for your particular usage.
	APIKeyAuthenticator func(string, string, security.TokenAuthentication) runtime.Authenticator

	// BearerAuthenticator generates a runtime.Authenticator from the supplied bearer token auth function.
	// It has a default implementation in the security package, however you can replace it for your particular usage.
	BearerAuthenticator func(string, security.ScopedTokenAuthentication) runtime.Authenticator

	// JSONConsumer registers a consumer for the following mime types:
	//   - application/json
	JSONConsumer runtime.Consumer

	// JSONProducer registers a producer for the following mime types:
	//   - application/json
	JSONProducer runtime.Producer

	// XAuthTokenAuth registers a function that takes a token and returns a principal
	// it performs authentication based on an api key X-Auth-Token provided in the header
	XAuthTokenAuth func(string) (interface{}, error)

	// APIAuthorizer provides access control (ACL/RBAC/ABAC) by providing access to the request and authenticated principal
	APIAuthorizer runtime.Authorizer

	// EndpointDeleteEndpointEndpointIDHandler sets the operation handler for the delete endpoint endpoint ID operation
	EndpointDeleteEndpointEndpointIDHandler endpoint.DeleteEndpointEndpointIDHandler
	// QuotaDeleteQuotasProjectIDHandler sets the operation handler for the delete quotas project ID operation
	QuotaDeleteQuotasProjectIDHandler quota.DeleteQuotasProjectIDHandler
	// RbacDeleteRbacPoliciesRbacPolicyIDHandler sets the operation handler for the delete rbac policies rbac policy ID operation
	RbacDeleteRbacPoliciesRbacPolicyIDHandler rbac.DeleteRbacPoliciesRbacPolicyIDHandler
	// ServiceDeleteServiceServiceIDHandler sets the operation handler for the delete service service ID operation
	ServiceDeleteServiceServiceIDHandler service.DeleteServiceServiceIDHandler
	// VersionGetHandler sets the operation handler for the get operation
	VersionGetHandler version.GetHandler
	// EndpointGetEndpointHandler sets the operation handler for the get endpoint operation
	EndpointGetEndpointHandler endpoint.GetEndpointHandler
	// EndpointGetEndpointEndpointIDHandler sets the operation handler for the get endpoint endpoint ID operation
	EndpointGetEndpointEndpointIDHandler endpoint.GetEndpointEndpointIDHandler
	// QuotaGetQuotasHandler sets the operation handler for the get quotas operation
	QuotaGetQuotasHandler quota.GetQuotasHandler
	// QuotaGetQuotasDefaultsHandler sets the operation handler for the get quotas defaults operation
	QuotaGetQuotasDefaultsHandler quota.GetQuotasDefaultsHandler
	// QuotaGetQuotasProjectIDHandler sets the operation handler for the get quotas project ID operation
	QuotaGetQuotasProjectIDHandler quota.GetQuotasProjectIDHandler
	// RbacGetRbacPoliciesHandler sets the operation handler for the get rbac policies operation
	RbacGetRbacPoliciesHandler rbac.GetRbacPoliciesHandler
	// RbacGetRbacPoliciesRbacPolicyIDHandler sets the operation handler for the get rbac policies rbac policy ID operation
	RbacGetRbacPoliciesRbacPolicyIDHandler rbac.GetRbacPoliciesRbacPolicyIDHandler
	// ServiceGetServiceHandler sets the operation handler for the get service operation
	ServiceGetServiceHandler service.GetServiceHandler
	// ServiceGetServiceServiceIDHandler sets the operation handler for the get service service ID operation
	ServiceGetServiceServiceIDHandler service.GetServiceServiceIDHandler
	// ServiceGetServiceServiceIDEndpointsHandler sets the operation handler for the get service service ID endpoints operation
	ServiceGetServiceServiceIDEndpointsHandler service.GetServiceServiceIDEndpointsHandler
	// EndpointPostEndpointHandler sets the operation handler for the post endpoint operation
	EndpointPostEndpointHandler endpoint.PostEndpointHandler
	// RbacPostRbacPoliciesHandler sets the operation handler for the post rbac policies operation
	RbacPostRbacPoliciesHandler rbac.PostRbacPoliciesHandler
	// ServicePostServiceHandler sets the operation handler for the post service operation
	ServicePostServiceHandler service.PostServiceHandler
	// EndpointPutEndpointEndpointIDHandler sets the operation handler for the put endpoint endpoint ID operation
	EndpointPutEndpointEndpointIDHandler endpoint.PutEndpointEndpointIDHandler
	// QuotaPutQuotasProjectIDHandler sets the operation handler for the put quotas project ID operation
	QuotaPutQuotasProjectIDHandler quota.PutQuotasProjectIDHandler
	// RbacPutRbacPoliciesRbacPolicyIDHandler sets the operation handler for the put rbac policies rbac policy ID operation
	RbacPutRbacPoliciesRbacPolicyIDHandler rbac.PutRbacPoliciesRbacPolicyIDHandler
	// ServicePutServiceServiceIDHandler sets the operation handler for the put service service ID operation
	ServicePutServiceServiceIDHandler service.PutServiceServiceIDHandler
	// ServicePutServiceServiceIDAcceptEndpointsHandler sets the operation handler for the put service service ID accept endpoints operation
	ServicePutServiceServiceIDAcceptEndpointsHandler service.PutServiceServiceIDAcceptEndpointsHandler
	// ServicePutServiceServiceIDRejectEndpointsHandler sets the operation handler for the put service service ID reject endpoints operation
	ServicePutServiceServiceIDRejectEndpointsHandler service.PutServiceServiceIDRejectEndpointsHandler

	// ServeError is called when an error is received, there is a default handler
	// but you can set your own with this
	ServeError func(http.ResponseWriter, *http.Request, error)

	// PreServerShutdown is called before the HTTP(S) server is shutdown
	// This allows for custom functions to get executed before the HTTP(S) server stops accepting traffic
	PreServerShutdown func()

	// ServerShutdown is called when the HTTP(S) server is shut down and done
	// handling all active connections and does not accept connections any more
	ServerShutdown func()

	// Custom command line argument groups with their descriptions
	CommandLineOptionsGroups []swag.CommandLineOptionsGroup

	// User defined logger function.
	Logger func(string, ...interface{})
}

// UseRedoc for documentation at /docs
func (o *ArcherAPI) UseRedoc() {
	o.useSwaggerUI = false
}

// UseSwaggerUI for documentation at /docs
func (o *ArcherAPI) UseSwaggerUI() {
	o.useSwaggerUI = true
}

// SetDefaultProduces sets the default produces media type
func (o *ArcherAPI) SetDefaultProduces(mediaType string) {
	o.defaultProduces = mediaType
}

// SetDefaultConsumes returns the default consumes media type
func (o *ArcherAPI) SetDefaultConsumes(mediaType string) {
	o.defaultConsumes = mediaType
}

// SetSpec sets a spec that will be served for the clients.
func (o *ArcherAPI) SetSpec(spec *loads.Document) {
	o.spec = spec
}

// DefaultProduces returns the default produces media type
func (o *ArcherAPI) DefaultProduces() string {
	return o.defaultProduces
}

// DefaultConsumes returns the default consumes media type
func (o *ArcherAPI) DefaultConsumes() string {
	return o.defaultConsumes
}

// Formats returns the registered string formats
func (o *ArcherAPI) Formats() strfmt.Registry {
	return o.formats
}

// RegisterFormat registers a custom format validator
func (o *ArcherAPI) RegisterFormat(name string, format strfmt.Format, validator strfmt.Validator) {
	o.formats.Add(name, format, validator)
}

// Validate validates the registrations in the ArcherAPI
func (o *ArcherAPI) Validate() error {
	var unregistered []string

	if o.JSONConsumer == nil {
		unregistered = append(unregistered, "JSONConsumer")
	}

	if o.JSONProducer == nil {
		unregistered = append(unregistered, "JSONProducer")
	}

	if o.XAuthTokenAuth == nil {
		unregistered = append(unregistered, "XAuthTokenAuth")
	}

	if o.EndpointDeleteEndpointEndpointIDHandler == nil {
		unregistered = append(unregistered, "endpoint.DeleteEndpointEndpointIDHandler")
	}
	if o.QuotaDeleteQuotasProjectIDHandler == nil {
		unregistered = append(unregistered, "quota.DeleteQuotasProjectIDHandler")
	}
	if o.RbacDeleteRbacPoliciesRbacPolicyIDHandler == nil {
		unregistered = append(unregistered, "rbac.DeleteRbacPoliciesRbacPolicyIDHandler")
	}
	if o.ServiceDeleteServiceServiceIDHandler == nil {
		unregistered = append(unregistered, "service.DeleteServiceServiceIDHandler")
	}
	if o.VersionGetHandler == nil {
		unregistered = append(unregistered, "version.GetHandler")
	}
	if o.EndpointGetEndpointHandler == nil {
		unregistered = append(unregistered, "endpoint.GetEndpointHandler")
	}
	if o.EndpointGetEndpointEndpointIDHandler == nil {
		unregistered = append(unregistered, "endpoint.GetEndpointEndpointIDHandler")
	}
	if o.QuotaGetQuotasHandler == nil {
		unregistered = append(unregistered, "quota.GetQuotasHandler")
	}
	if o.QuotaGetQuotasDefaultsHandler == nil {
		unregistered = append(unregistered, "quota.GetQuotasDefaultsHandler")
	}
	if o.QuotaGetQuotasProjectIDHandler == nil {
		unregistered = append(unregistered, "quota.GetQuotasProjectIDHandler")
	}
	if o.RbacGetRbacPoliciesHandler == nil {
		unregistered = append(unregistered, "rbac.GetRbacPoliciesHandler")
	}
	if o.RbacGetRbacPoliciesRbacPolicyIDHandler == nil {
		unregistered = append(unregistered, "rbac.GetRbacPoliciesRbacPolicyIDHandler")
	}
	if o.ServiceGetServiceHandler == nil {
		unregistered = append(unregistered, "service.GetServiceHandler")
	}
	if o.ServiceGetServiceServiceIDHandler == nil {
		unregistered = append(unregistered, "service.GetServiceServiceIDHandler")
	}
	if o.ServiceGetServiceServiceIDEndpointsHandler == nil {
		unregistered = append(unregistered, "service.GetServiceServiceIDEndpointsHandler")
	}
	if o.EndpointPostEndpointHandler == nil {
		unregistered = append(unregistered, "endpoint.PostEndpointHandler")
	}
	if o.RbacPostRbacPoliciesHandler == nil {
		unregistered = append(unregistered, "rbac.PostRbacPoliciesHandler")
	}
	if o.ServicePostServiceHandler == nil {
		unregistered = append(unregistered, "service.PostServiceHandler")
	}
	if o.EndpointPutEndpointEndpointIDHandler == nil {
		unregistered = append(unregistered, "endpoint.PutEndpointEndpointIDHandler")
	}
	if o.QuotaPutQuotasProjectIDHandler == nil {
		unregistered = append(unregistered, "quota.PutQuotasProjectIDHandler")
	}
	if o.RbacPutRbacPoliciesRbacPolicyIDHandler == nil {
		unregistered = append(unregistered, "rbac.PutRbacPoliciesRbacPolicyIDHandler")
	}
	if o.ServicePutServiceServiceIDHandler == nil {
		unregistered = append(unregistered, "service.PutServiceServiceIDHandler")
	}
	if o.ServicePutServiceServiceIDAcceptEndpointsHandler == nil {
		unregistered = append(unregistered, "service.PutServiceServiceIDAcceptEndpointsHandler")
	}
	if o.ServicePutServiceServiceIDRejectEndpointsHandler == nil {
		unregistered = append(unregistered, "service.PutServiceServiceIDRejectEndpointsHandler")
	}

	if len(unregistered) > 0 {
		return fmt.Errorf("missing registration: %s", strings.Join(unregistered, ", "))
	}

	return nil
}

// ServeErrorFor gets a error handler for a given operation id
func (o *ArcherAPI) ServeErrorFor(operationID string) func(http.ResponseWriter, *http.Request, error) {
	return o.ServeError
}

// AuthenticatorsFor gets the authenticators for the specified security schemes
func (o *ArcherAPI) AuthenticatorsFor(schemes map[string]spec.SecurityScheme) map[string]runtime.Authenticator {
	result := make(map[string]runtime.Authenticator)
	for name := range schemes {
		switch name {
		case "X-Auth-Token":
			scheme := schemes[name]
			result[name] = o.APIKeyAuthenticator(scheme.Name, scheme.In, o.XAuthTokenAuth)

		}
	}
	return result
}

// Authorizer returns the registered authorizer
func (o *ArcherAPI) Authorizer() runtime.Authorizer {
	return o.APIAuthorizer
}

// ConsumersFor gets the consumers for the specified media types.
// MIME type parameters are ignored here.
func (o *ArcherAPI) ConsumersFor(mediaTypes []string) map[string]runtime.Consumer {
	result := make(map[string]runtime.Consumer, len(mediaTypes))
	for _, mt := range mediaTypes {
		switch mt {
		case "application/json":
			result["application/json"] = o.JSONConsumer
		}

		if c, ok := o.customConsumers[mt]; ok {
			result[mt] = c
		}
	}
	return result
}

// ProducersFor gets the producers for the specified media types.
// MIME type parameters are ignored here.
func (o *ArcherAPI) ProducersFor(mediaTypes []string) map[string]runtime.Producer {
	result := make(map[string]runtime.Producer, len(mediaTypes))
	for _, mt := range mediaTypes {
		switch mt {
		case "application/json":
			result["application/json"] = o.JSONProducer
		}

		if p, ok := o.customProducers[mt]; ok {
			result[mt] = p
		}
	}
	return result
}

// HandlerFor gets a http.Handler for the provided operation method and path
func (o *ArcherAPI) HandlerFor(method, path string) (http.Handler, bool) {
	if o.handlers == nil {
		return nil, false
	}
	um := strings.ToUpper(method)
	if _, ok := o.handlers[um]; !ok {
		return nil, false
	}
	if path == "/" {
		path = ""
	}
	h, ok := o.handlers[um][path]
	return h, ok
}

// Context returns the middleware context for the archer API
func (o *ArcherAPI) Context() *middleware.Context {
	if o.context == nil {
		o.context = middleware.NewRoutableContext(o.spec, o, nil)
	}

	return o.context
}

func (o *ArcherAPI) initHandlerCache() {
	o.Context() // don't care about the result, just that the initialization happened
	if o.handlers == nil {
		o.handlers = make(map[string]map[string]http.Handler)
	}

	if o.handlers["DELETE"] == nil {
		o.handlers["DELETE"] = make(map[string]http.Handler)
	}
	o.handlers["DELETE"]["/endpoint/{endpoint_id}"] = endpoint.NewDeleteEndpointEndpointID(o.context, o.EndpointDeleteEndpointEndpointIDHandler)
	if o.handlers["DELETE"] == nil {
		o.handlers["DELETE"] = make(map[string]http.Handler)
	}
	o.handlers["DELETE"]["/quotas/{project_id}"] = quota.NewDeleteQuotasProjectID(o.context, o.QuotaDeleteQuotasProjectIDHandler)
	if o.handlers["DELETE"] == nil {
		o.handlers["DELETE"] = make(map[string]http.Handler)
	}
	o.handlers["DELETE"]["/rbac-policies/{rbac_policy_id}"] = rbac.NewDeleteRbacPoliciesRbacPolicyID(o.context, o.RbacDeleteRbacPoliciesRbacPolicyIDHandler)
	if o.handlers["DELETE"] == nil {
		o.handlers["DELETE"] = make(map[string]http.Handler)
	}
	o.handlers["DELETE"]["/service/{service_id}"] = service.NewDeleteServiceServiceID(o.context, o.ServiceDeleteServiceServiceIDHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"][""] = version.NewGet(o.context, o.VersionGetHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/endpoint"] = endpoint.NewGetEndpoint(o.context, o.EndpointGetEndpointHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/endpoint/{endpoint_id}"] = endpoint.NewGetEndpointEndpointID(o.context, o.EndpointGetEndpointEndpointIDHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/quotas"] = quota.NewGetQuotas(o.context, o.QuotaGetQuotasHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/quotas/defaults"] = quota.NewGetQuotasDefaults(o.context, o.QuotaGetQuotasDefaultsHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/quotas/{project_id}"] = quota.NewGetQuotasProjectID(o.context, o.QuotaGetQuotasProjectIDHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/rbac-policies"] = rbac.NewGetRbacPolicies(o.context, o.RbacGetRbacPoliciesHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/rbac-policies/{rbac_policy_id}"] = rbac.NewGetRbacPoliciesRbacPolicyID(o.context, o.RbacGetRbacPoliciesRbacPolicyIDHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/service"] = service.NewGetService(o.context, o.ServiceGetServiceHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/service/{service_id}"] = service.NewGetServiceServiceID(o.context, o.ServiceGetServiceServiceIDHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/service/{service_id}/endpoints"] = service.NewGetServiceServiceIDEndpoints(o.context, o.ServiceGetServiceServiceIDEndpointsHandler)
	if o.handlers["POST"] == nil {
		o.handlers["POST"] = make(map[string]http.Handler)
	}
	o.handlers["POST"]["/endpoint"] = endpoint.NewPostEndpoint(o.context, o.EndpointPostEndpointHandler)
	if o.handlers["POST"] == nil {
		o.handlers["POST"] = make(map[string]http.Handler)
	}
	o.handlers["POST"]["/rbac-policies"] = rbac.NewPostRbacPolicies(o.context, o.RbacPostRbacPoliciesHandler)
	if o.handlers["POST"] == nil {
		o.handlers["POST"] = make(map[string]http.Handler)
	}
	o.handlers["POST"]["/service"] = service.NewPostService(o.context, o.ServicePostServiceHandler)
	if o.handlers["PUT"] == nil {
		o.handlers["PUT"] = make(map[string]http.Handler)
	}
	o.handlers["PUT"]["/endpoint/{endpoint_id}"] = endpoint.NewPutEndpointEndpointID(o.context, o.EndpointPutEndpointEndpointIDHandler)
	if o.handlers["PUT"] == nil {
		o.handlers["PUT"] = make(map[string]http.Handler)
	}
	o.handlers["PUT"]["/quotas/{project_id}"] = quota.NewPutQuotasProjectID(o.context, o.QuotaPutQuotasProjectIDHandler)
	if o.handlers["PUT"] == nil {
		o.handlers["PUT"] = make(map[string]http.Handler)
	}
	o.handlers["PUT"]["/rbac-policies/{rbac_policy_id}"] = rbac.NewPutRbacPoliciesRbacPolicyID(o.context, o.RbacPutRbacPoliciesRbacPolicyIDHandler)
	if o.handlers["PUT"] == nil {
		o.handlers["PUT"] = make(map[string]http.Handler)
	}
	o.handlers["PUT"]["/service/{service_id}"] = service.NewPutServiceServiceID(o.context, o.ServicePutServiceServiceIDHandler)
	if o.handlers["PUT"] == nil {
		o.handlers["PUT"] = make(map[string]http.Handler)
	}
	o.handlers["PUT"]["/service/{service_id}/accept_endpoints"] = service.NewPutServiceServiceIDAcceptEndpoints(o.context, o.ServicePutServiceServiceIDAcceptEndpointsHandler)
	if o.handlers["PUT"] == nil {
		o.handlers["PUT"] = make(map[string]http.Handler)
	}
	o.handlers["PUT"]["/service/{service_id}/reject_endpoints"] = service.NewPutServiceServiceIDRejectEndpoints(o.context, o.ServicePutServiceServiceIDRejectEndpointsHandler)
}

// Serve creates a http handler to serve the API over HTTP
// can be used directly in http.ListenAndServe(":8000", api.Serve(nil))
func (o *ArcherAPI) Serve(builder middleware.Builder) http.Handler {
	o.Init()

	if o.Middleware != nil {
		return o.Middleware(builder)
	}
	if o.useSwaggerUI {
		return o.context.APIHandlerSwaggerUI(builder)
	}
	return o.context.APIHandler(builder)
}

// Init allows you to just initialize the handler cache, you can then recompose the middleware as you see fit
func (o *ArcherAPI) Init() {
	if len(o.handlers) == 0 {
		o.initHandlerCache()
	}
}

// RegisterConsumer allows you to add (or override) a consumer for a media type.
func (o *ArcherAPI) RegisterConsumer(mediaType string, consumer runtime.Consumer) {
	o.customConsumers[mediaType] = consumer
}

// RegisterProducer allows you to add (or override) a producer for a media type.
func (o *ArcherAPI) RegisterProducer(mediaType string, producer runtime.Producer) {
	o.customProducers[mediaType] = producer
}

// AddMiddlewareFor adds a http middleware to existing handler
func (o *ArcherAPI) AddMiddlewareFor(method, path string, builder middleware.Builder) {
	um := strings.ToUpper(method)
	if path == "/" {
		path = ""
	}
	o.Init()
	if h, ok := o.handlers[um][path]; ok {
		o.handlers[um][path] = builder(h)
	}
}
