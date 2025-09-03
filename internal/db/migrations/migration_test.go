// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package migrations

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sapcc/go-bits/easypg"
	"github.com/sapcc/go-bits/osext"
	"github.com/z0ne-dev/mgx/v2"
)

func connectToDatabase(t *testing.T) *pgx.Conn {
	t.Helper()

	// create db connection
	url := osext.GetenvOrDefault(
		"DB_URL", "postgres://postgres:postgres@127.0.0.1:54320/postgres?sslmode=disable")
	if url == "" {
		t.Fatal("DB_URL env variable is not set")
	}

	db, err := pgx.Connect(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(context.Background(), "DROP SCHEMA public CASCADE"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(context.Background(), "CREATE SCHEMA public"); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMain(m *testing.M) {
	easypg.WithTestDB(m, func() int { return m.Run() })
}

func TestMigrate(t *testing.T) {
	db := connectToDatabase(t)
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
