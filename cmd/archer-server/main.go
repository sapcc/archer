// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

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
