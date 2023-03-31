/*
 *   Copyright 2020 SAP SE
 *
 *   Licensed under the Apache License, Version 2.0 (the "License");
 *   you may not use this file except in compliance with the License.
 *   You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 */

package config

import (
	"github.com/gophercloud/utils/openstack/clientconfig"
)

var (
	Global Archer
)

type Archer struct {
	Verbose     bool                  `short:"v" long:"verbose" description:"Show verbose debug information"`
	Database    Database              `json:"database" group:"Database Options"`
	ApiSettings ApiSettings           `json:"api_settings" group:"API Settings"`
	ServiceAuth clientconfig.AuthInfo `json:"service_auth" group:"Service Authentication"`
	Quota       Quota                 `json:"quota" group:"Quota Options"`
	Audit       Audit                 `json:"audit_middleware_notifications" group:"Audit Middleware Options"`
}

type ApiSettings struct {
	PolicyFile         string  `long:"policy-file" json:"policy-file" description:"Use policy file" default:"policy.json"`
	AuthStrategy       string  `long:"auth-strategy" json:"auth_strategy" description:"The auth strategy for API requests, currently supported: [keystone, none]"`
	PolicyEngine       string  `long:"policy-engine" json:"policy_engine" description:"Policy engine to use, currently supported: [goslo, noop]"`
	DisablePagination  bool    `long:"disable-pagination" json:"disable_pagination" description:"Disable the usage of pagination"`
	DisableSorting     bool    `long:"disable-sorting" json:"disable_sorting" description:"Disable the usage of sorting"`
	PaginationMaxLimit int64   `long:"pagination-max-limit" json:"pagination_max_limit" default:"1000" description:"The maximum number of items returned in a single response."`
	RateLimit          float64 `long:"rate-limit" json:"rate_limit" default:"100" description:"Maximum number of requests to limit per second."`
	DisableCors        bool    `long:"disable-cors" json:"disable_cors" description:"Stops sending Access-Control-Allow-Origin Header to allow cross-origin requests."`
}

type Quota struct {
	Enabled              bool  `long:"enable-quota" json:"enabled" description:"Enable quotas."`
	DefaultQuotaService  int64 `long:"default-quota-service" json:"service" default:"0" description:"Default quota of services per project."`
	DefaultQuotaEndpoint int64 `long:"default-quota-endpoint" json:"endpoint" default:"0" description:"Default quota of endpoints per project."`
}

type Database struct {
	Connection string `long:"database-connection" json:"connection" description:"Connection string to use to connect to the database."`
}

type Audit struct {
	Enabled      bool   `long:"enable-audit" json:"enabled" description:"Enables message notification bus."`
	TransportURL string `long:"transport-url" json:"transport_url" description:"The network address and optional user credentials for connecting to the messaging backend."`
	QueueName    string `long:"queue-name" json:"queue_name" description:"RabbitMQ queue name"`
}
