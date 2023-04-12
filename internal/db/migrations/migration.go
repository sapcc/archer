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

package migrations

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/sapcc/go-bits/logg"
	"github.com/z0ne-dev/mgx/v2"

	"github.com/sapcc/archer/internal/config"
)

func Migrate() {
	migrator, _ := mgx.New(mgx.Migrations(
		mgx.NewMigration("initial", func(ctx context.Context, commands mgx.Commands) error {
			if _, err := commands.Exec(ctx, `
				CREATE TABLE service_status
				(
					name VARCHAR(16) PRIMARY KEY
				);`,
			); err != nil {
				return err
			}

			if _, err := commands.Exec(ctx, `
				INSERT INTO
					service_status(name)
				VALUES
					('AVAILABLE'),
					('PENDING_CREATE'),
					('PENDING_UPDATE'),
					('PENDING_DELETE'),
					('UNAVAILABLE')
				;`,
			); err != nil {
				return err
			}

			if _, err := commands.Exec(ctx, `
				CREATE TABLE service
				(
					id                UUID         DEFAULT gen_random_uuid() PRIMARY KEY,
					enabled           BOOLEAN      DEFAULT true NOT NULL,
					name              VARCHAR(64)  NOT NULL,
					description       VARCHAR(255) NOT NULL,
					network_id        UUID         NOT NULL,
					ip_addresses      INET[]       NOT NULL,
                    port              INTEGER      NOT NULL,
					status            VARCHAR(14)  DEFAULT 'PENDING_CREATE' NOT NULL,
					require_approval  BOOLEAN      NOT NULL,
					visibility        VARCHAR(7)   NOT NULL,
					availability_zone VARCHAR(64)  NULL,
					host              VARCHAR(64)  NULL,
					proxy_protocol    BOOLEAN      NOT NULL,
					created_at        TIMESTAMP    NOT NULL DEFAULT now(),
					updated_at        TIMESTAMP    NOT NULL DEFAULT now(),
					project_id        VARCHAR(36)  NOT NULL,
					CONSTRAINT visibility CHECK (visibility IN ('private', 'public')),
					CONSTRAINT status FOREIGN KEY (status) REFERENCES service_status(name),
					UNIQUE (network_id, ip_addresses, availability_zone)
				);`,
			); err != nil {
				return err
			}

			if _, err := commands.Exec(ctx, `
				CREATE TABLE endpoint_status
				(
					name VARCHAR(16) PRIMARY KEY
				);`,
			); err != nil {
				return err
			}

			if _, err := commands.Exec(ctx, `
				INSERT INTO
					endpoint_status(name)
				VALUES
					('AVAILABLE'),
					('PENDING_APPROVAL'),
					('PENDING_CREATE'),
					('PENDING_DELETE'),
					('REJECTED'),
					('FAILED')
				;`,
			); err != nil {
				return err
			}

			if _, err := commands.Exec(ctx, `
				CREATE TABLE endpoint
				(
					id                UUID         DEFAULT gen_random_uuid() PRIMARY KEY,
					service_id        UUID         NOT NULL,
					"target.port"     UUID         NULL,
					"target.network"  UUID         NULL,
					"target.subnet"   UUID         NULL,
					status            VARCHAR(14)  NOT NULL DEFAULT 'PENDING_CREATE',
					created_at        TIMESTAMP    NOT NULL DEFAULT now(),
					updated_at        TIMESTAMP    NOT NULL DEFAULT now(),
					project_id        VARCHAR(36)  NOT NULL,
					CONSTRAINT fk_service FOREIGN KEY(service_id) REFERENCES service(id),
					CONSTRAINT fk_status FOREIGN KEY (status) REFERENCES endpoint_status(name)
				);`,
			); err != nil {
				return err
			}

			if _, err := commands.Exec(ctx, `
				CREATE TABLE service_port
				(
					service_id UUID NOT NULL,
					port_id    UUID NOT NULL,
					UNIQUE(port_id),
					CONSTRAINT fk_service FOREIGN KEY(service_id) REFERENCES service(id)
				);`,
			); err != nil {
				return err
			}

			if _, err := commands.Exec(ctx, `
				CREATE TABLE agents
				(
					host                 UUID NOT NULL,
					availability_zone    VARCHAR(64),
					UNIQUE(host)
				);`,
			); err != nil {
				return err
			}

			return nil
		}),
	))

	logg.ShowDebug = config.IsDebug()
	conn, err := pgx.Connect(context.Background(), config.Global.Database.Connection)
	if err != nil {
		logg.Fatal(err.Error())
	}

	if err := migrator.Migrate(context.Background(), conn); err != nil {
		logg.Fatal(err.Error())
	}
}
