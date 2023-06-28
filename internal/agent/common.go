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
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
)

func RegisterAgent(pool *pgxpool.Pool, provider string) {
	sql, args := db.Insert("agents").
		Columns("host", "availability_zone", "provider").
		Values(config.Global.Default.Host, config.Global.Default.AvailabilityZone, provider).
		Suffix("ON CONFLICT (host) DO UPDATE SET").
		SuffixExpr(sq.Expr("availability_zone = ?,", config.Global.Default.AvailabilityZone)).
		Suffix("updated_at = now()").
		MustSql()

	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		panic(err)
	}
}
