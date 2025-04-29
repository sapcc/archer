// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"testing"

	"github.com/pashagolub/pgxmock/v4"

	"github.com/sapcc/archer/internal/config"
)

func TestRegisterAgent(t *testing.T) {
	config.Global.Default.Host = "test-host"
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	var nilString *string
	dbMock.
		ExpectExec("INSERT INTO agents (host,availability_zone,provider) VALUES ($1,$2,$3) ON CONFLICT (host) DO UPDATE SET availability_zone = $4, updated_at = now()").
		WithArgs(config.Global.Default.Host, nilString, "test", nilString).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	RegisterAgent(dbMock, "test")
}

func TestRegisterAgentWithAZ(t *testing.T) {
	config.Global.Default.Host = "test-host"
	config.Global.Default.AvailabilityZone = "test-az"
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dbMock.Close()
	}()

	dbMock.
		ExpectExec("INSERT INTO agents (host,availability_zone,provider) VALUES ($1,$2,$3) ON CONFLICT (host) DO UPDATE SET availability_zone = $4, updated_at = now()").
		WithArgs(config.Global.Default.Host, &config.Global.Default.AvailabilityZone, "test", &config.Global.Default.AvailabilityZone).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	RegisterAgent(dbMock, "test")
}
