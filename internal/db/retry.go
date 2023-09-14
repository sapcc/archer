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
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	log "github.com/sirupsen/logrus"
)

func Retry(fn func() error) (err error) {
	retries := 2
	var pe *pgconn.PgError
	for {
		err = fn()
		if err == nil || retries == 0 || errors.As(err, &pe) && pgerrcode.IsIntegrityConstraintViolation(pe.Code) {
			return err
		}
		log.WithError(err).WithField("retries", retries).Warn("db.Retry")
		retries--
	}
}
