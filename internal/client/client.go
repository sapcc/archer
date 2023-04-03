/*
 *   Copyright 2021 SAP SE
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

package client

import (
	"net/http"
	"net/url"
	"os"

	"github.com/go-openapi/runtime"
	runtimeclient "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	osclient "github.com/gophercloud/utils/client"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jessevdk/go-flags"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/client"
)

var (
	Parser       = flags.NewParser(&opts, flags.Default)
	Table        = table.NewWriter()
	Mapper       = reflectx.NewMapper("json")
	ArcherClient = client.Default
)

type outputFormatters struct {
	Format     string   `short:"f" long:"format" description:"The output format, defaults to table" choice:"table" choice:"csv" choice:"markdown" choice:"html" choice:"value" default:"table"`
	Columns    []string `short:"c" long:"column" description:"specify the column(s) to include, can be repeated to show multiple columns"`
	SortColumn []string `long:"sort-column" description:"specify the column(s) to sort the data (columns specified first have a priority, non-existing columns are ignored), can be repeated"`
}

var opts struct {
	Debug      bool             `long:"debug" description:"Show verbose debug information"`
	Formatters outputFormatters `group:"Output formatters"`

	OSEndpoint          string `long:"os-endpoint" env:"OS_ENDPOINT" description:"The endpoint that will always be used"`
	OSAuthUrl           string `long:"os-auth-url" env:"OS_AUTH_URL" description:"Authentication URL"`
	OSPassword          string `long:"os-password" env:"OS_PASSWORD" description:"User's password to use with"`
	OSUsername          string `long:"os-username" env:"OS_USERNAME" description:"User's username to use with"`
	OSProjectDomainName string `long:"os-project-domain-name" env:"OS_PROJECT_DOMAIN_NAME" description:"Domain name containing project"`
	OSProjectName       string `long:"os-project-name" env:"OS_PROJECT_NAME" description:"Project name to scope to"`
	OSRegionName        string `long:"os-region-name" env:"OS_REGION_NAME" description:"Authentication region name"`
	OSUserDomainName    string `long:"os-user-domain-name" env:"OS_USER_DOMAIN_NAME" description:"User's domain name"`
}

func SetupClient() {
	Table.SetOutputMirror(os.Stdout)

	Parser.CommandHandler = func(command flags.Commander, args []string) error {
		if command == nil {
			return nil
		}

		ao, err := clientconfig.AuthOptions(&clientconfig.ClientOpts{
			RegionName: opts.OSRegionName,
			AuthInfo: &clientconfig.AuthInfo{
				AuthURL:           opts.OSAuthUrl,
				Username:          opts.OSUsername,
				Password:          opts.OSPassword,
				ProjectName:       opts.OSProjectName,
				ProjectDomainName: opts.OSProjectDomainName,
				UserDomainName:    opts.OSUserDomainName,
			},
		})
		if err != nil {
			return err
		}

		provider, err := openstack.NewClient(opts.OSAuthUrl)
		if err != nil {
			return err
		}
		if opts.Debug {
			provider.HTTPClient = http.Client{
				Transport: &osclient.RoundTripper{
					Rt:     &http.Transport{},
					Logger: &osclient.DefaultLogger{},
				},
			}
		}

		err = openstack.Authenticate(provider, *ao)
		if err != nil {
			return err
		}

		endpointOpts := gophercloud.EndpointOpts{
			Region: opts.OSRegionName,
		}
		endpointOpts.ApplyDefaults("gtm")
		endpoint, err := provider.EndpointLocator(endpointOpts)
		if err != nil {
			return err
		}
		// Override endpoint?
		if opts.OSEndpoint != "" {
			endpoint = opts.OSEndpoint
		}

		uri, err := url.Parse(endpoint)
		if err != nil {
			return err
		}

		rt := runtimeclient.New(uri.Host, uri.Path, []string{uri.Scheme})
		rt.SetDebug(opts.Debug)
		rt.DefaultAuthentication = runtime.ClientAuthInfoWriterFunc(func(req runtime.ClientRequest, reg strfmt.Registry) error {
			if err := req.SetHeaderParam("X-Auth-Token", provider.Token()); err != nil {
				return err
			}
			return nil
		})
		ArcherClient.SetTransport(rt)

		return command.Execute(args)
	}

	if _, err := Parser.Parse(); err != nil {
		if fe, ok := err.(*flags.Error); ok && fe.Type == flags.ErrHelp {
			os.Exit(0)
		}
		logg.Fatal(err.Error())
		os.Exit(1)
	}
}
