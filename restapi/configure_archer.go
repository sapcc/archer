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
	"github.com/sapcc/archer/internal/agent/neutron"
	"net/http"
	"time"

	"github.com/IBM/pgxpoolprometheus"
	"github.com/didip/tollbooth/v7"
	"github.com/dre1080/recovr"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/swag"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/auth"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/controller"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/internal/middlewares"
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

	config.ResolveHost()
	if config.Global.Default.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              config.Global.Default.SentryDSN,
			AttachStacktrace: true,
			Release:          "TODO Version",
		}); err != nil {
			logg.Fatal("Sentry initialization failed: %v", err)
		}

		logg.Info("Sentry is enabled")
	}

	connConfig, err := pgxpool.ParseConfig(config.Global.Database.Connection)
	if err != nil {
		logg.Fatal(err.Error())
	}
	if config.Global.Database.Trace {
		logger := tracelog.TraceLog{
			Logger:   db.NewLogger(),
			LogLevel: tracelog.LogLevelDebug,
		}
		connConfig.ConnConfig.Tracer = &logger
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), connConfig)
	if err != nil {
		logg.Fatal(err.Error())
	}

	if config.Global.Default.Prometheus {
		http.Handle("/metrics", promhttp.Handler())
		go prometheusListenerThread()

		collector := pgxpoolprometheus.NewCollector(pool, map[string]string{"db_name": connConfig.ConnConfig.Database})
		prometheus.MustRegister(collector)
	}

	// Keystone authentication
	authInfo := clientconfig.AuthInfo(config.Global.ServiceAuth)
	providerClient, err := clientconfig.AuthenticatedClient(&clientconfig.ClientOpts{
		AuthInfo: &authInfo})
	if err != nil {
		logg.Fatal(err.Error())
	}

	neutronClient, err := neutron.ConnectToNeutron(providerClient)
	if err != nil {
		logg.Fatal("While connecting to Neutron: %s", err.Error())
	}
	logg.Info("Connected to Neutron %s", neutronClient.Endpoint)

	var keystone *auth.Keystone
	if config.Global.ApiSettings.AuthStrategy == "keystone" {
		keystone, err = auth.InitializeKeystone(providerClient)
		if err != nil {
			logg.Fatal(err.Error())
		}
	}

	// Applies when the "X-Auth-Token" header is set
	api.XAuthTokenAuth = func(token string) (interface{}, error) {
		if keystone != nil {
			return keystone.AuthenticateToken(token)
		}

		return "", nil
	}

	c := controller.NewController(pool, SwaggerSpec, neutronClient)

	api.VersionGetHandler = version.GetHandlerFunc(c.GetVersionHandler)

	api.ServiceGetServiceHandler = service.GetServiceHandlerFunc(c.GetServiceHandler)
	api.ServicePostServiceHandler = service.PostServiceHandlerFunc(c.PostServiceHandler)
	api.ServiceDeleteServiceServiceIDHandler = service.DeleteServiceServiceIDHandlerFunc(c.DeleteServiceServiceIDHandler)
	api.ServiceGetServiceServiceIDHandler = service.GetServiceServiceIDHandlerFunc(c.GetServiceServiceIDHandler)
	api.ServicePutServiceServiceIDHandler = service.PutServiceServiceIDHandlerFunc(c.PutServiceServiceIDHandler)
	api.ServiceGetServiceServiceIDEndpointsHandler = service.GetServiceServiceIDEndpointsHandlerFunc(c.GetServiceServiceIDEndpointsHandler)
	api.ServicePutServiceServiceIDAcceptEndpointsHandler = service.PutServiceServiceIDAcceptEndpointsHandlerFunc(c.PutServiceServiceIDAcceptEndpointsHandler)
	api.ServicePutServiceServiceIDRejectEndpointsHandler = service.PutServiceServiceIDRejectEndpointsHandlerFunc(c.PutServiceServiceIDRejectEndpointsHandler)

	api.EndpointGetEndpointHandler = endpoint.GetEndpointHandlerFunc(c.GetEndpointHandler)
	api.EndpointPostEndpointHandler = endpoint.PostEndpointHandlerFunc(c.PostEndpointHandler)
	api.EndpointPutEndpointEndpointIDHandler = endpoint.PutEndpointEndpointIDHandlerFunc(c.PutEndpointEndpointIDHandler)
	api.EndpointDeleteEndpointEndpointIDHandler = endpoint.DeleteEndpointEndpointIDHandlerFunc(c.DeleteEndpointEndpointIDHandler)
	api.EndpointGetEndpointEndpointIDHandler = endpoint.GetEndpointEndpointIDHandlerFunc(c.GetEndpointEndpointIDHandler)

	api.QuotaGetQuotasHandler = quota.GetQuotasHandlerFunc(c.GetQuotasHandler)
	api.QuotaGetQuotasDefaultsHandler = quota.GetQuotasDefaultsHandlerFunc(c.GetQuotasDefaultsHandler)
	api.QuotaGetQuotasProjectIDHandler = quota.GetQuotasProjectIDHandlerFunc(c.GetQuotasProjectIDHandler)
	api.QuotaPutQuotasProjectIDHandler = quota.PutQuotasProjectIDHandlerFunc(c.PutQuotasProjectIDHandler)
	api.QuotaDeleteQuotasProjectIDHandler = quota.DeleteQuotasProjectIDHandlerFunc(c.DeleteQuotasProjectIDHandler)

	api.RbacGetRbacPoliciesHandler = rbac.GetRbacPoliciesHandlerFunc(c.GetRbacPoliciesHandler)
	api.RbacPostRbacPoliciesHandler = rbac.PostRbacPoliciesHandlerFunc(c.PostRbacPoliciesHandler)
	api.RbacGetRbacPoliciesRbacPolicyIDHandler = rbac.GetRbacPoliciesRbacPolicyIDHandlerFunc(c.GetRbacPoliciesRbacPolicyIDHandler)
	api.RbacPutRbacPoliciesRbacPolicyIDHandler = rbac.PutRbacPoliciesRbacPolicyIDHandlerFunc(c.PutRbacPoliciesRbacPolicyIDHandler)
	api.RbacDeleteRbacPoliciesRbacPolicyIDHandler = rbac.DeleteRbacPoliciesRbacPolicyIDHandlerFunc(c.DeleteRbacPoliciesRbacPolicyIDHandler)

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {
		pool.Close()
		sentry.Flush(5 * time.Second)
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
	if rl := config.Global.ApiSettings.RateLimit; rl > .0 {
		limiter := tollbooth.NewLimiter(rl, nil)
		limiter.SetHeader("X-Auth-Token", nil)
		limiter.SetMethods([]string{"GET", "POST", "PUT", "DELETE"})
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
	// health check middleware
	handler = middlewares.HealthCheckMiddleware(handler)

	if !config.Global.ApiSettings.DisableCors {
		logg.Info("Initializing CORS middleware")
		handler = cors.New(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"HEAD", "GET", "POST", "PUT", "DELETE"},
			AllowedHeaders: []string{"Content-Type", "User-Agent", "X-Auth-Token"},
		}).Handler(handler)
	}

	// Pass via sentry handler
	handler = sentryhttp.New(sentryhttp.Options{
		Repanic: true,
	}).Handle(handler)

	// recover with recovr
	return recovr.New()(handler)
}

func prometheusListenerThread() {
	logg.Info("Serving prometheus metrics at http://%s/metrics", config.Global.Default.PrometheusListen)
	if err := http.ListenAndServe(config.Global.Default.PrometheusListen, nil); err != nil {
		logg.Fatal(err.Error())
	}
}
