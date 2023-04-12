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

package main

import (
	"os"

	"github.com/go-openapi/loads"
	"github.com/jessevdk/go-flags"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/restapi"
	"github.com/sapcc/archer/restapi/operations"
)

func main() {
	var err error
	restapi.SwaggerSpec, err = loads.Embedded(restapi.SwaggerJSON, restapi.FlatSwaggerJSON)
	if err != nil {
		logg.Fatal(err.Error())
	}

	api := operations.NewArcherAPI(restapi.SwaggerSpec)
	server := restapi.NewServer(api)
	defer server.Shutdown()

	parser := flags.NewParser(server, flags.Default)
	parser.ShortDescription = "🏹 Archer"
	parser.LongDescription = "Archer is an API service that can privately connect services from one to another."
	server.ConfigureFlags()
	for _, optsGroup := range api.CommandLineOptionsGroups {
		_, err := parser.AddGroup(optsGroup.ShortDescription, optsGroup.LongDescription, optsGroup.Options)
		if err != nil {
			logg.Fatal(err.Error())
		}
	}

	if _, err := parser.Parse(); err != nil {
		code := 1
		if fe, ok := err.(*flags.Error); ok {
			if fe.Type == flags.ErrHelp {
				code = 0
			}
		}
		os.Exit(code)
	}

	// parse config file
	if config.Global.ConfigFile != "" {
		ini := flags.NewIniParser(parser)
		if err := ini.ParseFile(config.Global.ConfigFile); err != nil {
			logg.Fatal(err.Error())
		}
	}

	logg.ShowDebug = config.Global.Default.Debug
	server.ConfigureAPI()

	if err := server.Serve(); err != nil {
		logg.Fatal(err.Error())
	}

}
