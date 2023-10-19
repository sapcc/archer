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

package agent

import (
	"testing"

	"github.com/pashagolub/pgxmock/v3"

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
