// This file is safe to edit. Once it exists it will not be overwritten

// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package restapi

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/IBM/pgxpoolprometheus"
	"github.com/didip/tollbooth/v8"
	"github.com/dre1080/recovr"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/swag/cmdutils"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/sapcc/go-bits/gopherpolicy"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/auth"
	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/internal/controller"
	"github.com/sapcc/archer/v2/internal/db"
	"github.com/sapcc/archer/v2/internal/middlewares"
	"github.com/sapcc/archer/v2/internal/neutron"
	"github.com/sapcc/archer/v2/internal/policy"
	"github.com/sapcc/archer/v2/internal/scheduler"
	"github.com/sapcc/archer/v2/restapi/operations"
	"github.com/sapcc/archer/v2/restapi/operations/agent"
	"github.com/sapcc/archer/v2/restapi/operations/endpoint"
	"github.com/sapcc/archer/v2/restapi/operations/quota"
	"github.com/sapcc/archer/v2/restapi/operations/rbac"
	"github.com/sapcc/archer/v2/restapi/operations/service"
	"github.com/sapcc/archer/v2/restapi/operations/version"
)

//go:generate swagger generate server --target ../../archer --name Archer --spec ../swagger.yaml --principal interface{}

var (
	// SwaggerSpec make parsed swaggerspec available globally
	SwaggerSpec *loads.Document

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "code"})

	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "code"})

	httpResponseSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_response_size_bytes",
		Help:    "Size of HTTP responses in bytes.",
		Buckets: prometheus.ExponentialBuckets(100, 10, 7),
	}, []string{"method", "code"})
)

func configureFlags(api *operations.ArcherAPI) {
	api.CommandLineOptionsGroups = []cmdutils.CommandLineOptionsGroup{
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
	api.Logger = log.Infof
	api.UseRedoc()
	api.JSONConsumer = runtime.JSONConsumer()
	api.JSONProducer = runtime.JSONProducer()

	config.ResolveHost()
	config.InitSentry()
	connConfig, err := pgxpool.ParseConfig(config.Global.Database.Connection)
	if err != nil {
		log.Fatal(err.Error())
	}
	connConfig.ConnConfig.Tracer = db.GetTracer()
	connConfig.ConnConfig.RuntimeParams["application_name"] = "archer-api"
	pool, err := pgxpool.NewWithConfig(context.Background(), connConfig)
	if err != nil {
		log.Fatal(err.Error())
	}

	if config.Global.Default.Prometheus {
		http.Handle("/metrics", promhttp.Handler())
		go prometheusListenerThread()

		collector := pgxpoolprometheus.NewCollector(pool, map[string]string{"db_name": connConfig.ConnConfig.Database})
		prometheus.MustRegister(collector)
		prometheus.MustRegister(httpRequestDuration, httpRequestsTotal, httpResponseSize)
	}

	// Keystone authentication
	authInfo := clientconfig.AuthInfo(config.Global.ServiceAuth)
	providerClient, err := clientconfig.AuthenticatedClient(context.Background(), &clientconfig.ClientOpts{
		AuthInfo: &authInfo})
	if err != nil {
		log.Fatal(err.Error())
	}

	neutronClient, err := neutron.ConnectToNeutron(providerClient)
	if err != nil {
		log.Fatalf("While connecting to Neutron: %s", err.Error())
	}
	log.Infof("Connected to Neutron %s", neutronClient.Endpoint)
	neutronClient.InitCache()

	var keystone *auth.Keystone
	if config.Global.ApiSettings.AuthStrategy == "keystone" {
		keystone, err = auth.InitializeKeystone(providerClient)
		if err != nil {
			log.Fatal(err.Error())
		}
	} else {
		log.Info("Warning: authentication disabled (noop)")
	}

	if keystone != nil {
		// Applies when the "X-Auth-Token" header is set
		api.XAuthTokenAuth = func(token string) (interface{}, error) {
			return keystone.AuthenticateToken(token)
		}

		api.APIAuthorizer = runtime.AuthorizerFunc(func(r *http.Request, p interface{}) error {
			if t, ok := p.(*gopherpolicy.Token); ok {
				rule := policy.RuleFromHTTPRequest(r)
				if t.Check(rule + "-global") {
					return nil
				}
				if t.Check(rule) {
					r.Header.Set("X-Project-Id", t.ProjectScopeUUID())
					return nil
				}
			}
			return errors.New(401, "Unauthorized")
		})
	} else {
		api.XAuthTokenAuth = func(token string) (interface{}, error) { return nil, nil }
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
	api.ServicePostServiceServiceIDMigrateHandler = service.PostServiceServiceIDMigrateHandlerFunc(c.PostServiceServiceIDMigrateHandler)

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

	api.AgentGetAgentsHandler = agent.GetAgentsHandlerFunc(c.GetAgentsHandler)
	api.AgentGetAgentsAgentHostHandler = agent.GetAgentsAgentHostHandlerFunc(c.GetAgentsAgentHostHandler)

	// Start background scheduler for agent rescheduling and rebalancing
	// Uses PostgreSQL advisory locks for distributed leader election across multiple API instances
	schedulerCfg := scheduler.Config{
		StaleTimeout:           config.Global.Agent.AgentStaleTimeout,
		CheckInterval:          config.Global.Agent.RescheduleCheckInterval,
		RebalanceDelay:         config.Global.Agent.RebalanceDelay,
		RebalanceThreshold:     config.Global.Agent.RebalanceThreshold,
		RebalanceMaxMigrations: config.Global.Agent.RebalanceMaxMigrations,
	}
	notifyFunc := func(host string) { db.NotifyService(pool, host) }
	serviceScheduler := scheduler.NewServiceScheduler(pool, schedulerCfg, notifyFunc)
	bgScheduler, err := scheduler.NewBackgroundScheduler(serviceScheduler, pool, schedulerCfg.CheckInterval, schedulerCfg.RebalanceDelay)
	if err != nil {
		log.Fatalf("Failed to create background scheduler: %v", err)
	}
	if err := bgScheduler.Start(context.Background()); err != nil {
		log.Fatalf("Failed to start background scheduler: %v", err)
	}

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {
		if err := bgScheduler.Stop(); err != nil {
			log.WithError(err).Error("Failed to stop background scheduler")
		}
		pool.Close()
		sentry.Flush(5 * time.Second)
	}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(_ *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(_ *http.Server, scheme, addr string) {
	log.Infof("Server configured to listen on %s://%s", scheme, addr)
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	if rl := config.Global.ApiSettings.RateLimit; rl > .0 {
		log.Info("Initializing rate limit middleware")
		limiter := tollbooth.NewLimiter(rl, nil)
		limiter.SetHeader("X-Auth-Token", nil)
		limiter.SetMethods([]string{"GET", "POST", "PUT", "DELETE"})
		handler = tollbooth.LimitHandler(limiter, handler)
	}

	if config.Global.Audit.Enabled {
		log.Info("Initializing audit middleware")
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
		log.Info("Initializing CORS middleware")
		handler = cors.New(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"HEAD", "GET", "POST", "PUT", "DELETE"},
			AllowedHeaders: []string{"Content-Type", "User-Agent", "X-Auth-Token"},
		}).Handler(handler)
	}

	// Prometheus HTTP metrics instrumentation
	if config.Global.Default.Prometheus {
		handler = promhttp.InstrumentHandlerDuration(httpRequestDuration,
			promhttp.InstrumentHandlerCounter(httpRequestsTotal,
				promhttp.InstrumentHandlerResponseSize(httpResponseSize, handler)))
	}

	// Pass via sentry handler
	handler = sentryhttp.New(sentryhttp.Options{
		Repanic: true,
	}).Handle(handler)

	// recover with recovr
	return recovr.New()(handler)
}

func prometheusListenerThread() {
	log.Infof("Serving prometheus metrics at http://%s/metrics", config.Global.Default.PrometheusListen)
	if err := http.ListenAndServe(config.Global.Default.PrometheusListen, nil); err != nil {
		log.Fatal(err.Error())
	}
}
