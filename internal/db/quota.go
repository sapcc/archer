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

package db

import (
	"net/http"

	sq "github.com/Masterminds/squirrel"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/auth"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/errors"
)

func CheckQuota(pool PgxIface, r *http.Request, resource string) error {
	if !config.Global.Quota.Enabled {
		return nil
	}

	// Get project scope
	project := auth.GetProjectID(r)

	// Check for quota
	var quotaAvailable, quotaUsed int

	// Insert quota if not exists
	sql, args, err := Insert("quota").
		Columns("project_id", "service", "endpoint").
		Values(
			project,
			config.Global.Quota.DefaultQuotaService,
			config.Global.Quota.DefaultQuotaEndpoint).
		Suffix("ON CONFLICT (project_id) DO NOTHING").
		ToSql()
	if err != nil {
		panic(err)
	}
	if _, err = pool.Exec(r.Context(), sql, args...); err != nil {
		panic(err)
	}

	sql, args, err = Select(resource).
		Column(sq.Alias(
			Select("COUNT(id)").
				From(resource).
				Where("project_id = quota.project_id"), "use")).
		From("quota").
		Where(sq.Eq{"project_id": project}).
		ToSql()
	if err != nil {
		panic(err)
	}

	if err := pool.QueryRow(r.Context(), sql, args...).
		Scan(&quotaAvailable, &quotaUsed); err != nil {
		panic(err)
	}

	log.Debugf("Quota %s of project %s is %d of %d", resource, project, quotaUsed, quotaAvailable)
	if quotaAvailable-quotaUsed < 1 {
		return errors.ErrQuotaExceeded
	}
	return nil
}
