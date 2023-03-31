// This file is safe to edit. Once it exists it will not be overwritten

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

package restapi

import (
	"crypto/tls"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/swag"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/restapi/operations"
	"github.com/sapcc/archer/restapi/operations/endpoint"
	"github.com/sapcc/archer/restapi/operations/quota"
	"github.com/sapcc/archer/restapi/operations/rbac"
	"github.com/sapcc/archer/restapi/operations/service"
	"github.com/sapcc/archer/restapi/operations/version"
)

//go:generate swagger generate server --target ../../archer --name Archer --spec ../swagger.yaml --principal interface{}

func configureFlags(api *operations.ArcherAPI) {
	configFlags := config.Archer{}
	api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{
		{
			ShortDescription: "Archer Flags",
			LongDescription:  "Archer specific flags",
			Options:          &configFlags,
		},
	}
}

func configureAPI(api *operations.ArcherAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	// Applies when the "X-Auth-Token" header is set
	if api.XAuthTokenAuth == nil {
		api.XAuthTokenAuth = func(token string) (interface{}, error) {
			return nil, errors.NotImplemented("api key auth (X-Auth-Token) X-Auth-Token from header param [X-Auth-Token] has not yet been implemented")
		}
	}

	// Set your custom authorizer if needed. Default one is security.Authorized()
	// Expected interface runtime.Authorizer
	//
	// Example:
	// api.APIAuthorizer = security.Authorized()

	if api.EndpointDeleteEndpointEndpointIDHandler == nil {
		api.EndpointDeleteEndpointEndpointIDHandler = endpoint.DeleteEndpointEndpointIDHandlerFunc(func(params endpoint.DeleteEndpointEndpointIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation endpoint.DeleteEndpointEndpointID has not yet been implemented")
		})
	}
	if api.QuotaDeleteQuotasProjectIDHandler == nil {
		api.QuotaDeleteQuotasProjectIDHandler = quota.DeleteQuotasProjectIDHandlerFunc(func(params quota.DeleteQuotasProjectIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation quota.DeleteQuotasProjectID has not yet been implemented")
		})
	}
	if api.RbacDeleteRbacPoliciesRbacPolicyIDHandler == nil {
		api.RbacDeleteRbacPoliciesRbacPolicyIDHandler = rbac.DeleteRbacPoliciesRbacPolicyIDHandlerFunc(func(params rbac.DeleteRbacPoliciesRbacPolicyIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation rbac.DeleteRbacPoliciesRbacPolicyID has not yet been implemented")
		})
	}
	if api.ServiceDeleteServiceServiceIDHandler == nil {
		api.ServiceDeleteServiceServiceIDHandler = service.DeleteServiceServiceIDHandlerFunc(func(params service.DeleteServiceServiceIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.DeleteServiceServiceID has not yet been implemented")
		})
	}
	if api.VersionGetHandler == nil {
		api.VersionGetHandler = version.GetHandlerFunc(func(params version.GetParams) middleware.Responder {
			return middleware.NotImplemented("operation version.Get has not yet been implemented")
		})
	}
	if api.EndpointGetEndpointHandler == nil {
		api.EndpointGetEndpointHandler = endpoint.GetEndpointHandlerFunc(func(params endpoint.GetEndpointParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation endpoint.GetEndpoint has not yet been implemented")
		})
	}
	if api.EndpointGetEndpointEndpointIDHandler == nil {
		api.EndpointGetEndpointEndpointIDHandler = endpoint.GetEndpointEndpointIDHandlerFunc(func(params endpoint.GetEndpointEndpointIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation endpoint.GetEndpointEndpointID has not yet been implemented")
		})
	}
	if api.QuotaGetQuotasHandler == nil {
		api.QuotaGetQuotasHandler = quota.GetQuotasHandlerFunc(func(params quota.GetQuotasParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation quota.GetQuotas has not yet been implemented")
		})
	}
	if api.QuotaGetQuotasDefaultsHandler == nil {
		api.QuotaGetQuotasDefaultsHandler = quota.GetQuotasDefaultsHandlerFunc(func(params quota.GetQuotasDefaultsParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation quota.GetQuotasDefaults has not yet been implemented")
		})
	}
	if api.QuotaGetQuotasProjectIDHandler == nil {
		api.QuotaGetQuotasProjectIDHandler = quota.GetQuotasProjectIDHandlerFunc(func(params quota.GetQuotasProjectIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation quota.GetQuotasProjectID has not yet been implemented")
		})
	}
	if api.RbacGetRbacPoliciesHandler == nil {
		api.RbacGetRbacPoliciesHandler = rbac.GetRbacPoliciesHandlerFunc(func(params rbac.GetRbacPoliciesParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation rbac.GetRbacPolicies has not yet been implemented")
		})
	}
	if api.RbacGetRbacPoliciesRbacPolicyIDHandler == nil {
		api.RbacGetRbacPoliciesRbacPolicyIDHandler = rbac.GetRbacPoliciesRbacPolicyIDHandlerFunc(func(params rbac.GetRbacPoliciesRbacPolicyIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation rbac.GetRbacPoliciesRbacPolicyID has not yet been implemented")
		})
	}
	if api.ServiceGetServiceHandler == nil {
		api.ServiceGetServiceHandler = service.GetServiceHandlerFunc(func(params service.GetServiceParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.GetService has not yet been implemented")
		})
	}
	if api.ServiceGetServiceServiceIDHandler == nil {
		api.ServiceGetServiceServiceIDHandler = service.GetServiceServiceIDHandlerFunc(func(params service.GetServiceServiceIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.GetServiceServiceID has not yet been implemented")
		})
	}
	if api.ServiceGetServiceServiceIDEndpointsHandler == nil {
		api.ServiceGetServiceServiceIDEndpointsHandler = service.GetServiceServiceIDEndpointsHandlerFunc(func(params service.GetServiceServiceIDEndpointsParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.GetServiceServiceIDEndpoints has not yet been implemented")
		})
	}
	if api.EndpointPostEndpointHandler == nil {
		api.EndpointPostEndpointHandler = endpoint.PostEndpointHandlerFunc(func(params endpoint.PostEndpointParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation endpoint.PostEndpoint has not yet been implemented")
		})
	}
	if api.RbacPostRbacPoliciesHandler == nil {
		api.RbacPostRbacPoliciesHandler = rbac.PostRbacPoliciesHandlerFunc(func(params rbac.PostRbacPoliciesParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation rbac.PostRbacPolicies has not yet been implemented")
		})
	}
	if api.ServicePostServiceHandler == nil {
		api.ServicePostServiceHandler = service.PostServiceHandlerFunc(func(params service.PostServiceParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.PostService has not yet been implemented")
		})
	}
	if api.QuotaPutQuotasProjectIDHandler == nil {
		api.QuotaPutQuotasProjectIDHandler = quota.PutQuotasProjectIDHandlerFunc(func(params quota.PutQuotasProjectIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation quota.PutQuotasProjectID has not yet been implemented")
		})
	}
	if api.RbacPutRbacPoliciesRbacPolicyIDHandler == nil {
		api.RbacPutRbacPoliciesRbacPolicyIDHandler = rbac.PutRbacPoliciesRbacPolicyIDHandlerFunc(func(params rbac.PutRbacPoliciesRbacPolicyIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation rbac.PutRbacPoliciesRbacPolicyID has not yet been implemented")
		})
	}
	if api.ServicePutServiceServiceIDHandler == nil {
		api.ServicePutServiceServiceIDHandler = service.PutServiceServiceIDHandlerFunc(func(params service.PutServiceServiceIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.PutServiceServiceID has not yet been implemented")
		})
	}
	if api.ServicePutServiceServiceIDAcceptEndpointsHandler == nil {
		api.ServicePutServiceServiceIDAcceptEndpointsHandler = service.PutServiceServiceIDAcceptEndpointsHandlerFunc(func(params service.PutServiceServiceIDAcceptEndpointsParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.PutServiceServiceIDAcceptEndpoints has not yet been implemented")
		})
	}
	if api.ServicePutServiceServiceIDRejectEndpointsHandler == nil {
		api.ServicePutServiceServiceIDRejectEndpointsHandler = service.PutServiceServiceIDRejectEndpointsHandlerFunc(func(params service.PutServiceServiceIDRejectEndpointsParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation service.PutServiceServiceIDRejectEndpoints has not yet been implemented")
		})
	}

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
