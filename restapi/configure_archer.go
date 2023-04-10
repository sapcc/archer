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
	"context"
	"crypto/tls"
	goerrors "errors"
	"fmt"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/sapcc/archer/internal/db"
	"net/http"
	"os"

	"github.com/didip/tollbooth"
	"github.com/dre1080/recovr"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/swag"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/cors"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/auth"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/middlewares"
	"github.com/sapcc/archer/models"
	"github.com/sapcc/archer/restapi/operations"
	"github.com/sapcc/archer/restapi/operations/endpoint"
	"github.com/sapcc/archer/restapi/operations/quota"
	"github.com/sapcc/archer/restapi/operations/rbac"
	"github.com/sapcc/archer/restapi/operations/service"
	"github.com/sapcc/archer/restapi/operations/version"
)

//go:generate swagger generate server --target ../../archer --name Archer --spec ../swagger.yaml --principal interface{}

var (
	// SwaggerSpec make parsed swaggerspec available globally
	SwaggerSpec *loads.Document
)

func configureFlags(api *operations.ArcherAPI) {
	api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{
		{
			ShortDescription: "Archer Flags",
			LongDescription:  "Archer specific flags",
			Options:          &config.Global,
		},
	}
}

func configureAPI(api *operations.ArcherAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError
	api.Logger = logg.Info
	api.UseRedoc()
	api.JSONConsumer = runtime.JSONConsumer()
	api.JSONProducer = runtime.JSONProducer()

	if config.Global.ApiSettings.ApiBaseURL == "" {
		if hostname, err := os.Hostname(); err != nil {
			logg.Fatal(err.Error())
		} else {
			config.Global.ApiSettings.ApiBaseURL = hostname
		}
	}

	connConfig, err := pgx.ParseConfig(config.Global.Database.Connection)
	if err != nil {
		logg.Fatal(err.Error())
	}
	if config.Global.Database.Trace {
		logger := tracelog.TraceLog{
			Logger:   db.NewLogger(),
			LogLevel: tracelog.LogLevelDebug,
		}
		connConfig.Tracer = &logger
	}
	conn, err := pgx.ConnectConfig(context.Background(), connConfig)
	if err != nil {
		logg.Fatal(err.Error())
	}

	keystone, err := auth.InitializeKeystone()
	if err != nil {
		logg.Info("Keystone disabled: %s", err.Error())
	}

	// Applies when the "X-Auth-Token" header is set
	api.XAuthTokenAuth = func(token string) (interface{}, error) {
		if keystone != nil {
			return keystone.AuthenticateToken(token)
		}

		return "", nil
	}

	// Set your custom authorizer if needed. Default one is security.Authorized()
	// Expected interface runtime.Authorizer
	//
	// Example:
	// api.APIAuthorizer = security.Authorized()

	// Example of the version get handler
	api.VersionGetHandler = version.GetHandlerFunc(func(params version.GetParams) middleware.Responder {
		var capabilities []string
		if !config.Global.ApiSettings.DisablePagination {
			capabilities = append(capabilities, "pagination")
		}
		if !config.Global.ApiSettings.DisableSorting {
			capabilities = append(capabilities, "sorting")
		}
		if !config.Global.ApiSettings.DisableCors {
			capabilities = append(capabilities, "cors")
		}
		if config.Global.ApiSettings.AuthStrategy != "none" {
			capabilities = append(capabilities, config.Global.ApiSettings.AuthStrategy)
		}
		if config.Global.ApiSettings.RateLimit > 0 {
			capabilities = append(capabilities, fmt.Sprintf("ratelimit=%.2f",
				config.Global.ApiSettings.RateLimit))
		}
		if config.Global.ApiSettings.PaginationMaxLimit > 0 {
			capabilities = append(capabilities, fmt.Sprintf("pagination_max=%d",
				config.Global.ApiSettings.PaginationMaxLimit))
		}
		return version.NewGetOK().WithPayload(&models.Version{
			Capabilities: capabilities,
			Links: []*models.VersionLinksItems0{{
				Href: config.Global.ApiSettings.ApiBaseURL,
				Rel:  "self",
				Type: "application/json",
			}},
			Updated: "now", // TODO: build time
			Version: SwaggerSpec.Spec().Info.Version,
		})
	})

	api.ServiceGetServiceHandler = service.GetServiceHandlerFunc(func(params service.GetServiceParams, principal interface{}) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		var servicesResponse []*models.Service
		if err := pgxscan.Select(ctx, conn, &servicesResponse, `SELECT * FROM service`); err != nil {
			panic(err)
		}

		return service.NewGetServiceOK().WithPayload(servicesResponse)
	})

	api.ServicePostServiceHandler = service.PostServiceHandlerFunc(func(params service.PostServiceParams, principal interface{}) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		var serviceResponse models.Service

		// Set default values
		if err := SetModelDefaults(params.Body); err != nil {
			panic(err)
		}

		sql := `
			INSERT INTO service (enabled, 
			                     name, 
			                     description, 
			                     network_id, 
			                     ip_addresses, 
			                     require_approval, 
			                     visibility, 
			                     availability_zone, 
			                     proxy_protocol, 
			                     project_id, 
			                     port)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING *
		`

		var rows pgx.Rows
		rows, err = conn.Query(ctx, sql,
			params.Body.Enabled,
			params.Body.Name,
			params.Body.Description,
			params.Body.NetworkID,
			params.Body.IPAddresses,
			params.Body.RequireApproval,
			params.Body.Visibility,
			params.Body.AvailabilityZone,
			params.Body.ProxyProtocol,
			params.Body.ProjectID,
			params.Body.Port)
		if err != nil {
			panic(err)
		}
		if err := pgxscan.ScanOne(&serviceResponse, rows); err != nil {
			var pe *pgconn.PgError
			if goerrors.As(err, &pe) && pgerrcode.IsIntegrityConstraintViolation(pe.Code) {
				return service.NewPostServiceConflict().WithPayload(&models.Error{
					Code:    409,
					Message: "Entry for network_id, ip_address and availability_zone already exists.",
				})
			}
			panic(err)
		}

		return service.NewPostServiceOK().WithPayload(&serviceResponse)
	})

	api.EndpointGetEndpointHandler = endpoint.GetEndpointHandlerFunc(func(params endpoint.GetEndpointParams, principal interface{}) middleware.Responder {
		ctx := params.HTTPRequest.Context()
		var endpointsResponse []*models.Endpoint
		if err := pgxscan.Select(ctx, conn, &endpointsResponse, `SELECT * FROM endpoint`); err != nil {
			panic(err)
		}

		return endpoint.NewGetEndpointOK().WithPayload(endpointsResponse)
	})

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

	api.ServerShutdown = func() {
		if err := conn.Close(context.Background()); err != nil {
			logg.Fatal(err.Error())
		}
	}

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
	if !config.Global.ApiSettings.DisableCors {
		handler = cors.New(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"HEAD", "GET", "POST", "PUT", "DELETE"},
			AllowedHeaders: []string{"Content-Type", "User-Agent", "X-Auth-Token"},
		}).Handler(handler)
	}

	if rl := config.Global.ApiSettings.RateLimit; rl > .0 {
		limiter := tollbooth.NewLimiter(rl, nil)
		handler = tollbooth.LimitHandler(limiter, handler)
	}

	if config.Global.Audit.Enabled {
		auditMiddleware := middlewares.NewAuditController()
		handler = auditMiddleware.AuditHandler(handler)
	}

	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	handler = middlewares.HealthCheckMiddleware(handler)
	return recovr.New()(handler)
}
