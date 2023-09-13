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
	"errors"
	"os"

	"github.com/go-openapi/loads"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/restapi"
	"github.com/sapcc/archer/restapi/operations"
)

func main() {
	var err error
	restapi.SwaggerSpec, err = loads.Embedded(restapi.SwaggerJSON, restapi.FlatSwaggerJSON)
	if err != nil {
		log.Fatal(err.Error())
	}

	api := operations.NewArcherAPI(restapi.SwaggerSpec)
	server := restapi.NewServer(api)
	defer func() { _ = server.Shutdown() }()

	parser := flags.NewParser(server, flags.Default)
	parser.ShortDescription = "🏹 Archer"
	parser.LongDescription = "Archer is an API service that can privately connect services from one to another."
	server.ConfigureFlags()
	for _, optsGroup := range api.CommandLineOptionsGroups {
		_, err := parser.AddGroup(optsGroup.ShortDescription, optsGroup.LongDescription, optsGroup.Options)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	if _, err := parser.Parse(); err != nil {
		code := 1
		var fe *flags.Error
		if errors.As(err, &fe) {
			if fe.Type == flags.ErrHelp {
				code = 0
			}
		}
		os.Exit(code)
	}

	config.ParseConfig(parser)
	server.ConfigureAPI()

	if err := server.Serve(); err != nil {
		log.Fatal(err.Error())
	}
}
