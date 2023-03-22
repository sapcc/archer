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

	"github.com/z0ne-dev/mgx/v2"
)

var Migrations = mgx.Migrations(
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
					id                UUID           DEFAULT gen_random_uuid() PRIMARY KEY,
					enabled           BOOLEAN        DEFAULT true NOT NULL,
					name              VARCHAR(64)    NOT NULL,
					description       VARCHAR(255)   NOT NULL,
					network_id        UUID           NOT NULL,
					ip_addresses      INET[]         NOT NULL,
                    port              INTEGER        NOT NULL,
					status            VARCHAR(14)    DEFAULT 'PENDING_CREATE' NOT NULL,
					require_approval  BOOLEAN        NOT NULL,
					visibility        VARCHAR(7)     NOT NULL,
					availability_zone VARCHAR(64)    NULL,
					host              VARCHAR(64)    NULL,
					proxy_protocol    BOOLEAN        NOT NULL,
					created_at        TIMESTAMP      NOT NULL DEFAULT now(),
					updated_at        TIMESTAMP      NOT NULL DEFAULT now(),
					project_id        VARCHAR(36)    NOT NULL,
					tags              VARCHAR(64)[]  NOT NULL DEFAULT '{}',
					CONSTRAINT visibility CHECK (visibility IN ('private', 'public')),
					CONSTRAINT status FOREIGN KEY (status) REFERENCES service_status(name),
					UNIQUE (network_id, ip_addresses, availability_zone)
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
					CONSTRAINT fk_service FOREIGN KEY(service_id) REFERENCES service(id) ON DELETE CASCADE
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
					('PENDING_REJECTED'),
					('REJECTED'),
					('FAILED')
				;`,
		); err != nil {
			return err
		}

		if _, err := commands.Exec(ctx, `
				CREATE TABLE endpoint
				(
					id                UUID           DEFAULT gen_random_uuid() PRIMARY KEY,
					service_id        UUID           NOT NULL,
					status            VARCHAR(18)    NOT NULL DEFAULT 'PENDING_CREATE',
					created_at        TIMESTAMP      NOT NULL DEFAULT now(),
					updated_at        TIMESTAMP      NOT NULL DEFAULT now(),
					project_id        VARCHAR(36)    NOT NULL,
					tags              VARCHAR(64)[]  NOT NULL DEFAULT '{}',
					CONSTRAINT fk_service FOREIGN KEY(service_id) REFERENCES service(id),
					CONSTRAINT fk_status FOREIGN KEY (status) REFERENCES endpoint_status(name)
				);`,
		); err != nil {
			return err
		}

		if _, err := commands.Exec(ctx, `
				CREATE TABLE endpoint_port
				(
				    endpoint_id UUID NOT NULL PRIMARY KEY,
					port_id	    UUID NOT NULL,
					subnet      UUID NOT NULL,
					network     UUID NOT NULL,
					ip_address  INET NOT NULL,
					UNIQUE(endpoint_id),
					CONSTRAINT fk_port FOREIGN KEY(endpoint_id) REFERENCES endpoint(id) ON DELETE CASCADE
				);`,
		); err != nil {
			return err
		}

		if _, err := commands.Exec(ctx, `
				CREATE TABLE agents
				(
					host                 VARCHAR(255) NOT NULL,
					availability_zone    VARCHAR(64),
					UNIQUE(host)
				);`,
		); err != nil {
			return err
		}

		if _, err := commands.Exec(ctx, `
				CREATE TABLE rbac
				(
					id                UUID         DEFAULT gen_random_uuid() PRIMARY KEY,
					target_project    VARCHAR(36)  NOT NULL,
					service_id        UUID         NOT NULL,
					project_id        VARCHAR(36)  NOT NULL,
					created_at        TIMESTAMP    NOT NULL DEFAULT now(),
					updated_at        TIMESTAMP    NOT NULL DEFAULT now(),
					CONSTRAINT fk_service FOREIGN KEY(service_id) REFERENCES service(id) ON DELETE CASCADE,
					UNIQUE(target_project, service_id)
				);`); err != nil {
			return err
		}

		if _, err := commands.Exec(ctx, `
				CREATE TABLE quota
				(
					project_id  VARCHAR(36)  NOT NULL PRIMARY KEY,
					service     BIGINT       NOT NULL,
					endpoint    BIGINT       NOT NULL
				);`); err != nil {
			return err
		}

		return nil
	}),
	mgx.NewMigration("add_provider", func(ctx context.Context, commands mgx.Commands) error {
		if _, err := commands.Exec(ctx, `
			ALTER TABLE service
    			ADD COLUMN provider VARCHAR(64) DEFAULT 'tenant' CONSTRAINT provider CHECK (provider IN ('tenant', 'cp'));
		`); err != nil {
			return err
		}

		return nil
	}),
	mgx.NewMigration("add_agents", func(ctx context.Context, commands mgx.Commands) error {
		if _, err := commands.Exec(ctx, `
			DROP TABLE IF EXISTS agents;
			CREATE TABLE agents
			(
				host              VARCHAR(64)  NOT NULL PRIMARY KEY,
				availability_zone VARCHAR(64)  NOT NULL,
				created_at        TIMESTAMP    NOT NULL DEFAULT now(),
				updated_at        TIMESTAMP    NOT NULL DEFAULT now(),
				enabled           BOOLEAN      DEFAULT true NOT NULL,
				provider          VARCHAR(64)  DEFAULT 'tenant' NOT NULL
			);
		`); err != nil {
			return err
		}

		return nil
	}),
	mgx.NewMigration("add_quota_error", func(ctx context.Context, commands mgx.Commands) error {
		_, err := commands.Exec(ctx, "INSERT INTO service_status(name) VALUES ('ERROR_QUOTA');")
		return err
	}),
	mgx.NewMigration("adapt_constraint", func(ctx context.Context, commands mgx.Commands) error {
		_, err := commands.Exec(ctx, `
			ALTER TABLE service DROP CONSTRAINT service_network_id_ip_addresses_availability_zone_key;
			ALTER TABLE service ADD CONSTRAINT service_const UNIQUE (host, network_id, ip_addresses, availability_zone);
		`)
		return err
	}),
	mgx.NewMigration("add_endpoint_name", func(ctx context.Context, commands mgx.Commands) error {
		_, err := commands.Exec(ctx, `
			ALTER TABLE endpoint ADD COLUMN name VARCHAR(64) NOT NULL DEFAULT '';
			ALTER TABLE endpoint ADD COLUMN description VARCHAR(255) NOT NULL DEFAULT '';
		`)
		return err
	}),
	mgx.NewMigration("unique_endpoint_port", func(ctx context.Context, commands mgx.Commands) error {
		_, err := commands.Exec(ctx, `
			ALTER TABLE endpoint_port ADD CONSTRAINT endpoint_port_uniq UNIQUE (port_id);
		`)
		return err
	}),
	mgx.NewMigration("add_endpoint_port_ownership", func(ctx context.Context, commands mgx.Commands) error {
		_, err := commands.Exec(ctx, `
			ALTER TABLE endpoint_port ADD COLUMN owned BOOLEAN NOT NULL DEFAULT true;
		`)
		return err
	}),
)
