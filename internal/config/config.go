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
	"errors"
	"os"
	"strings"

	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/mcuadros/go-defaults"
	"github.com/urfave/cli/v2"
)

var (
	Global Archer
)

// ParseArgsAndRun Evaluate --config-file flags and remove them from env
func ParseArgsAndRun(name string, usage string, action cli.ActionFunc, flags ...cli.Flag) {
	// Append --config-file
	flags = append(flags, &cli.StringSliceFlag{
		Name:  "config-file",
		Usage: "Use config file",
	})

	app := &cli.App{
		Name:   name,
		Usage:  usage,
		Flags:  flags,
		Action: action,
		Before: func(c *cli.Context) error {
			if !c.IsSet("config-file") {
				return errors.New("No config files specified")
			}

			// Set defaults
			defaults.SetDefaults(&Global)

			// Parse config flags
			if err := parseConfigFlags(c.StringSlice("config-file")); err != nil {
				return err
			}

			// Ugly workaround, remove flags from osArgs
			// because micro wants to parse them and fails miserably
			i := 0
			newOsArgs := []string{}
			for {
				if i >= len(os.Args) {
					break
				}

				reaped := false

				for _, flagName := range c.FlagNames() {
					if strings.HasSuffix(os.Args[i], flagName) {
						// Reap the flag from osArgs
						reaped = true
						i++
						break
					}
				}
				if !reaped {
					newOsArgs = append(newOsArgs, os.Args[i])
				}
				i++
			}
			os.Args = newOsArgs
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		currentErr := err
		for errors.Unwrap(currentErr) != nil {
			logger.Fatal(currentErr)
			currentErr = errors.Unwrap(currentErr)
		}
		logger.Fatal(currentErr)
	}
}

func parseConfigFlags(flags []string) error {
	// new yaml encoder
	enc := yaml.NewEncoder()

	// new config
	conf, _ := config.NewConfig(
		config.WithReader(
			json.NewReader( // json reader for internal config merge
				reader.WithEncoder(enc),
			),
		),
	)

	logger.Infof("Config files: %s", flags)
	for _, path := range flags {
		if err := conf.Load(file.NewSource(
			file.WithPath(path),
			source.WithEncoder(enc),
		)); err != nil {
			return err
		}
	}

	if err := conf.Scan(&Global); err != nil {
		return err
	}
	return nil
}

type Archer struct {
	Verbose      bool                  `short:"v" long:"verbose" description:"Show verbose debug information"`
	Default      Default               `json:"DEFAULT"`
	Database     Database              `json:"database"`
	ApiSettings  ApiSettings           `json:"api_settings"`
	ServiceAuth  clientconfig.AuthInfo `json:"service_auth"`
	Quota        Quota                 `json:"quota"`
	F5Config     F5Config              `json:"f5"`
	AkamaiConfig AkamaiConfig          `json:"akamai"`
	Audit        Audit                 `json:"audit_middleware_notifications"`
}

type ApiSettings struct {
	PolicyFile         string  `short:"p" json:"policy-file" description:"Use policy file" default:"policy.json"`
	AuthStrategy       string  `json:"auth_strategy" description:"The auth strategy for API requests, currently supported: [keystone, none]"`
	PolicyEngine       string  `json:"policy_engine" description:"Policy engine to use, currently supported: [goslo, noop]"`
	DisablePagination  bool    `json:"disable_pagination" description:"Disable the usage of pagination"`
	DisableSorting     bool    `json:"disable_sorting" description:"Disable the usage of sorting"`
	PaginationMaxLimit int64   `json:"pagination_max_limit" default:"1000" description:"The maximum number of items returned in a single response."`
	RateLimit          float64 `json:"rate_limit" default:"100" description:"Maximum number of requests to limit per second."`
	DisableCors        bool    `json:"disable_cors" description:"Stops sending Access-Control-Allow-Origin Header to allow cross-origin requests."`
}

type Quota struct {
	Enabled                bool  `json:"enabled" description:"Enable quotas."`
	DefaultQuotaDomain     int64 `json:"domains" default:"0" description:"Default quota of domain per project."`
	DefaultQuotaPool       int64 `json:"pools" default:"0" description:"Default quota of pool per project."`
	DefaultQuotaMember     int64 `json:"members" default:"0" description:"Default quota of member per project."`
	DefaultQuotaMonitor    int64 `json:"monitors" default:"0" description:"Default quota of monitor per project."`
	DefaultQuotaDatacenter int64 `json:"datacenters" default:"0" description:"Default quota of datacenter per project."`
}

type Default struct {
	ApiBaseURL   string `json:"api_base_uri" description:"Base URI for the API for use in pagination links. This will be autodetected from the request if not overridden here."`
	TransportURL string `json:"transport_url" description:"The network address and optional user credentials for connecting to the messaging backend."`
}

type Database struct {
	Connection string `json:"connection" description:"Connection string to use to connect to the database."`
}

type F5Config struct {
	Host             string `json:"host"`
	DNSServerAddress string `json:"dns_server_address"`
	ValidateCert     bool   `json:"validate_certificates"`
}

type AkamaiConfig struct {
	EdgeRC       string `json:"edgerc" description:"Path to akamai edgerc file, else sourced from AKAMAI_EDGE_RC env variable."`
	Domain       string `json:"domain" description:"Traffic Management Domain to use (e.g. production.akadns.net)."`
	DomainType   string `json:"domain_type" description:"Indicates the type of domain available based on your contract, defaults to autodetect. Either failover-only, static, weighted, basic, or full."`
	ContractId   string `json:"contract_id" description:"Indicated the contract id to use, autodetects if only one contract is associated."`
	SyncInterval int64  `json:"sync_interval" default:"30" description:"Sync interval for checking for pending updates" `
}

type Audit struct {
	Enabled      bool   `json:"enabled" description:"Enables message notification bus."`
	TransportURL string `json:"transport_url" description:"The network address and optional user credentials for connecting to the messaging backend."`
	QueueName    string `json:"queue_name" description:"RabbitMQ queue name"`
}
