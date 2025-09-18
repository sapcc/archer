// This file is safe to edit. Once it exists it will not be overwritten

// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jessevdk/go-flags"
	"github.com/sapcc/go-api-declarations/bininfo"
	log "github.com/sirupsen/logrus"
)

var (
	Version   = bininfo.Version()
	BuildTime = bininfo.BuildDate()
	Global    Archer
)

type Archer struct {
	Version func() `short:"v" long:"version" description:"Show application version"`

	ConfigFile  []string    `long:"config-file" description:"Use config file"`
	Default     Default     `group:"DEFAULT"`
	Database    Database    `group:"database"`
	ApiSettings ApiSettings `group:"api_settings"`
	ServiceAuth AuthInfo    `group:"service_auth"`
	Quota       Quota       `group:"quota"`
	Audit       Audit       `group:"audit_middleware_notifications"`
	Agent       Agent       `group:"agent"`
}

type Default struct {
	Debug            bool   `short:"d" long:"debug" description:"Show debug information"`
	AvailabilityZone string `long:"availability-zone" ini-name:"availability_zone" description:"Availability zone of this node." env:"AVAILABILITY_ZONE"`
	Host             string `long:"hostname" ini-name:"host" description:"Hostname used by the server/agent. Defaults to auto-discovery."`
	Prometheus       bool   `long:"prometheus" description:"Enable prometheus exporter."`
	PrometheusListen string `long:"prometheus-listen" ini-name:"prometheus_listen" default:"127.0.0.1:9090" description:"Prometheus listen TCP network address."`
	Sentry           bool   `long:"sentry" ini-name:"sentry" description:"Enable Sentry"`
	SentryDSN        string `long:"sentry-dsn" ini-name:"sentry_dsn" description:"Sentry Data Source Name."`
	EndpointType     string `ini-name:"endpoint_type" default:"public"`
}

type ApiSettings struct {
	PolicyFile                string  `long:"policy-file" ini-name:"policy_file" description:"Use policy file" default:"policy.ini"`
	AuthStrategy              string  `long:"auth-strategy" ini-name:"auth_strategy" description:"The auth strategy for API requests, currently supported: [keystone, none]" default:"none"`
	PolicyEngine              string  `long:"policy-engine" ini-name:"policy_engine" description:"Policy engine to use, currently supported: [goslo, noop]"`
	DisablePagination         bool    `long:"disable-pagination" ini-name:"disable_pagination" description:"Disable the usage of pagination"`
	DisableSorting            bool    `long:"disable-sorting" ini-name:"disable_sorting" description:"Disable the usage of sorting"`
	PaginationMaxLimit        int64   `long:"pagination-max-limit" ini-name:"pagination_max_limit" default:"1000" description:"The maximum number of items returned in a single response."`
	RateLimit                 float64 `long:"rate-limit" ini-name:"rate_limit" default:"100" description:"Maximum number of requests to limit per second."`
	DisableCors               bool    `long:"disable-cors" ini-name:"disable_cors" description:"Stops sending Access-Control-Allow-Origin Header to allow cross-origin requests."`
	EnableProxyHeadersParsing bool    `long:"enable-proxy-headers-parsing" ini-name:"enable_proxy_headers_parsing" description:"Try parsing proxy headers for http scheme and base url."`
}

type Quota struct {
	Enabled              bool  `long:"enable-quota" ini-name:"enabled" description:"Enable quotas."`
	DefaultQuotaService  int64 `long:"default-quota-service" ini-name:"service" default:"0" description:"Default quota of services per project."`
	DefaultQuotaEndpoint int64 `long:"default-quota-endpoint" ini-name:"endpoint" default:"0" description:"Default quota of endpoints per project."`
}

type Database struct {
	Connection string `long:"database-connection" ini-name:"connection" description:"Connection string to use to connect to the database."`
	Trace      bool   `long:"database-trace" ini-name:"trace" description:"Enable tracing of SQL queries"`
}

type Audit struct {
	Enabled      bool   `long:"enable-audit" ini-name:"enabled" description:"Enables message notification bus."`
	TransportURL string `long:"transport-url" ini-name:"transport_url" env:"AUDIT_TRANSPORT_URL" description:"The network address and optional user credentials for connecting to the messaging backend."`
	QueueName    string `long:"queue-name" ini-name:"queue_name" description:"RabbitMQ queue name"`
}

type Agent struct {
	Devices                []string      `long:"device" ini-name:"device[]" description:"F5 Hostnames"`
	VCMPs                  []string      `long:"vcmp" ini-name:"vcmp[]" description:"F5 Platform Hostnames"`
	ValidateCert           bool          `long:"validate-certificates" ini-name:"validate_certificates" description:"Validate HTTPS Certificate."`
	PhysicalNetwork        string        `long:"physical-network" ini-name:"physical_network" description:"Physical Network"`
	PhysicalInterface      string        `long:"physical-interface" ini-name:"physical_interface" description:"Physical Interface" default:"portchannel1"`
	PendingSyncInterval    time.Duration `long:"pending-sync-interval" ini-name:"sync-interval" default:"120s" description:"Interval for pending sync scans, supports suffix (e.g. 10s)."`
	CreateService          bool          `long:"create-service" ini-name:"create_service" description:"Auto-create Service for network injection agent."`
	ServiceName            string        `long:"service-name" ini-name:"service_name" description:"Service name for auto-created service."`
	ServicePort            int           `long:"service-port" ini-name:"service_port" description:"Service port for auto-created service."`
	ServiceRequireApproval bool          `long:"service-require-approval" ini-name:"service_require_approval" description:"Service requires approval."`
	ServiceUpstreamHost    string        `long:"service-upstream-host" ini-name:"service_upstream_host" description:"Service upstream host."`
	ServiceProxyPath       string        `long:"service-proxy-path" ini-name:"service_proxy_path" description:"Service proxy path." default:"/var/run/socat-proxy/proxy.sock"`
	ServiceProtocol        string        `long:"service-protocol" ini-name:"service_protocol" description:"Service protocol."`
	L4Profile              string        `long:"l4-profile" ini-name:"l4_profile" description:"L4 profile to use for F5 endpoint service." default:"/Common/fastL4"`
	TCPProfile             string        `long:"tcp-profile" ini-name:"tcp_profile" description:"TCP profile to use for F5 endpoint service." default:"/Common/tcp"`
	MaxRetries             uint64        `long:"max-retries" ini-name:"max_retries" description:"Maximum number of retries for F5 operations." default:"5"`
	MaxDuration            time.Duration `long:"max-duration" ini-name:"max_duration" description:"Maximum duration for F5 operations, supports suffix (e.g. 10s)." default:"30s"`
}

type AuthInfo struct {
	AuthURL                     string `ini-name:"auth_url"`
	Token                       string `ini-name:"token"`
	Username                    string `ini-name:"username"`
	UserID                      string `ini-name:"user_id" `
	Password                    string `ini-name:"password" env:"OS_PASSWORD"`
	ApplicationCredentialID     string `ini-name:"application_credential_id"`
	ApplicationCredentialName   string `ini-name:"application_credential_name" `
	ApplicationCredentialSecret string `ini-name:"application_credential_secret" `
	SystemScope                 string `ini-name:"system_scope" `
	ProjectName                 string `ini-name:"project_name"`
	ProjectID                   string `ini-name:"project_id" `
	UserDomainName              string `ini-name:"user_domain_name"`
	UserDomainID                string `ini-name:"user_domain_id"`
	ProjectDomainName           string `ini-name:"project_domain_name" `
	ProjectDomainID             string `ini-name:"project_domain_id" `
	DomainName                  string `ini-name:"domain_name"`
	DomainID                    string `ini-name:"domain_id"`
	DefaultDomain               string `ini-name:"default_domain"`
	AllowReauth                 bool   `ini-name:"allow_reauth"`
}

func init() {
	Global.Version = func() {
		fmt.Printf("%s version %s\n", bininfo.Component(), bininfo.Version())
		os.Exit(0)
	}
}

func IsDebug() bool {
	return Global.Default.Debug
}

func ResolveHost() {
	if Global.Default.Host == "" {
		if hostname, err := os.Hostname(); err != nil {
			log.Fatal(err.Error())
		} else {
			Global.Default.Host = hostname
		}
	}
}

func ParseConfig(parser *flags.Parser) {
	log.SetLevel(log.InfoLevel)

	// parse config file
	for _, file := range Global.ConfigFile {
		ini := flags.NewIniParser(parser)
		if err := ini.ParseFile(file); err != nil {
			log.Fatal(err.Error())
		}
	}

	if Global.Default.Debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Infof("Running Archer @%s (%s)", Version, BuildTime)
}

func InitSentry() {
	if Global.Default.Sentry {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              Global.Default.SentryDSN,
			AttachStacktrace: true,
			Release:          Version,
			ServerName:       Global.Default.Host,
		}); err != nil {
			log.Fatalf("Sentry initialization failed: %v", err)
		}

		log.Info("Sentry is enabled")
	}
}

func GetApiBaseUrl(r *http.Request) string {
	var baseUrl url.URL

	baseUrl.Scheme = "http"
	if r.TLS != nil {
		baseUrl.Scheme = "https"
	}
	baseUrl.Host = Global.Default.Host
	if Global.ApiSettings.EnableProxyHeadersParsing {
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			baseUrl.Scheme = proto
		}
		if host := r.Header.Get("X-Forwarded-Host"); host != "" {
			baseUrl.Host = host
		}
	}

	return baseUrl.String()
}
