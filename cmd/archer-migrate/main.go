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
