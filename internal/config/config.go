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

package config

import (
	"net/http"
	"net/url"
	"time"
)

var (
	Global Archer
)

type Archer struct {
	ConfigFile  string      `long:"config-file" description:"Use config file"`
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
	AvailabilityZone string `long:"availability-zone" ini-name:"availability_zone" description:"Availability zone of this node."`
	Host             string `long:"hostname" ini-name:"host" description:"Hostname used by the server/agent. Defaults to auto-discovery."`
	Prometheus       bool   `long:"prometheus" description:"Enable prometheus exporter."`
	PrometheusListen string `long:"prometheus-listen" ini-name:"prometheus_listen" default:"127.0.0.1:9090" description:"Prometheus listen TCP network address."`
	SentryDSN        string `long:"sentry-dsn" ini-name:"sentry_dsn" description:"Sentry Data Source Name."`
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
	TransportURL string `long:"transport-url" ini-name:"transport_url" description:"The network address and optional user credentials for connecting to the messaging backend."`
	QueueName    string `long:"queue-name" ini-name:"queue_name" description:"RabbitMQ queue name"`
}

type Agent struct {
	Host                string        `long:"bigip-host" ini-name:"host" description:"F5 BigIP Hostname"`
	ValidateCert        bool          `long:"validate-certificates" ini-name:"validate_certificates" description:"Validate HTTPS Certificate."`
	PhysicalNetwork     string        `long:"physical-network" ini-name:"physical_network" description:"Physical Network"`
	PendingSyncInterval time.Duration `long:"pending-sync-interval" ini-name:"sync-interval" default:"120s" description:"Interval for pending sync scans, supports suffix (e.g. 10s)."`
}

type AuthInfo struct {
	AuthURL                     string `ini-name:"auth_url"`
	Token                       string `ini-name:"token"`
	Username                    string `ini-name:"username"`
	UserID                      string `ini-name:"user_id" `
	Password                    string `ini-name:"password" `
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

func IsDebug() bool {
	return Global.Default.Debug
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