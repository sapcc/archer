// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sapcc/go-bits/osext"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/z0ne-dev/mgx/v2"
)

func TestMigrate(t *testing.T) {
	// this flag is set in CI jobs where running rootless Docker is not possible
	if osext.GetenvBool("CHECK_SKIPS_FUNCTIONAL_TEST") {
		t.Skip("Skipping migration test as CHECK_SKIPS_FUNCTIONAL_TEST is set")
	}

	// start postgres container
	pgContainer, err := postgres.Run(t.Context(),
		"postgres:16-alpine",
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatal("Failed starting postgres container", err)
	}
	defer func(pgContainer *postgres.PostgresContainer, ctx context.Context) {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatal("Failed terminating postgres container", err)
		}
	}(pgContainer, t.Context())

	// connect to database
	url := pgContainer.MustConnectionString(t.Context(), "sslmode=disable")
	db, err := pgx.Connect(context.Background(), url)
	if err != nil {
		t.Fatal("Failed connecting to database", err)
	}
	defer func(db *pgx.Conn, ctx context.Context) {
		_ = db.Close(ctx)
	}(db, context.Background())

	// create migrator
	migrator, err := mgx.New(Migrations)
	if err != nil {
		t.Fatal(err)
	}

	// run migrator
	err = migrator.Migrate(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
}
