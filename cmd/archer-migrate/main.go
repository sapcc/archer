// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
	"github.com/z0ne-dev/mgx/v2"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db/migrations"
)

func main() {
	parser := flags.NewParser(&config.Global, flags.Default)
	parser.ShortDescription = "Archer Migration"

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

	conn, err := pgx.Connect(context.Background(), config.Global.Database.Connection)
	if err != nil {
		log.Fatal(err.Error())
	}
	migrator, err := mgx.New(migrations.Migrations)
	if err != nil {
		log.Fatal(err.Error())
	}

	if err := migrator.Migrate(context.Background(), conn); err != nil {
		log.Fatal(err.Error())
	}
	if err := conn.Close(context.Background()); err != nil {
		log.Fatal(err.Error())
	}
}
