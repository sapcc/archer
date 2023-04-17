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
	"fmt"
	"strings"
)

// This is a basic postgresql query generator that allows building select stmts with ORM feeling.

/////////////////////////
// SELECT
/////////////////////////

type SelectBuilder struct {
	columns      []string
	from         string
	limit        int64
	whereClauses []string
	args         []any
}

func Select(columns ...string) *SelectBuilder {
	return &SelectBuilder{columns: columns}
}

func (b *SelectBuilder) Select(columns ...string) *SelectBuilder {
	b.columns = columns
	return b
}

// From creates from part of SQL
func (b *SelectBuilder) From(from string) *SelectBuilder {
	b.from = from
	return b
}

func (b *SelectBuilder) Limit(n uint64) *SelectBuilder {
	b.limit = int64(n)
	return b
}

func (b *SelectBuilder) Where(pred string, args ...any) *SelectBuilder {
	// interpolating ? ? => $1 $2 ...
	for i := len(b.args) + 1; i <= len(b.args)+len(args); i++ {
		pred = strings.Replace(pred, "?", fmt.Sprintf("$%d", i), 1)
	}

	b.whereClauses = append(b.whereClauses, pred)
	b.args = append(b.args, args...)
	return b
}

func (b *SelectBuilder) ToSQL() (string, *[]any) {
	var sb strings.Builder

	// SELECT ...
	sb.WriteString("SELECT ")

	for i, column := range b.columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(column)
	}

	// FROM ...
	sb.WriteString(fmt.Sprint(" FROM ", b.from))

	// WHERE ...
	sb.WriteString(" WHERE ")
	for i, whereClause := range b.whereClauses {
		if i > 0 {
			sb.WriteString(" AND ")
		}
		sb.WriteString(whereClause)
	}

	// LIMIT ...
	if b.limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", b.limit))
	}
	return sb.String(), &b.args
}

/////////////////////////
// INSERT
/////////////////////////

type InsertBuilder struct {
	columns   []string
	into      string
	values    []any
	returning []string
}

func Insert(into string) *InsertBuilder {
	return &InsertBuilder{into: into}
}

func (b *InsertBuilder) Insert(into string) *InsertBuilder {
	b.into = into
	return b
}

func (b *InsertBuilder) Columns(columns ...string) *InsertBuilder {
	b.columns = columns
	return b
}

func (b *InsertBuilder) Values(values ...any) *InsertBuilder {
	b.values = values
	return b
}

func (b *InsertBuilder) Returning(returning ...string) *InsertBuilder {
	b.returning = returning
	return b
}

func (b *InsertBuilder) ToSQL() (string, *[]any) {
	var sb strings.Builder

	// SELECT ...
	sb.WriteString(fmt.Sprint("INSERT INTO ", b.into, " ("))

	for i, column := range b.columns {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(column)
	}
	sb.WriteString(") VALUES (")

	for i := range b.values {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("$%d", i+1))
	}
	sb.WriteString(")")

	// RETURNING ...
	if len(b.returning) > 0 {
		sb.WriteString(" RETURNING ")
		sb.WriteString(strings.Join(b.returning, ", "))
	}
	return sb.String(), &b.values
}

/////////////////////////
// DELETE
/////////////////////////

type DeleteBuilder struct {
	from         string
	whereClauses []string
	args         []any
}

func Delete(from string) *DeleteBuilder {
	return &DeleteBuilder{from: from}
}

func (b *DeleteBuilder) Delete(from string) *DeleteBuilder {
	b.from = from
	return b
}

func (b *DeleteBuilder) Where(pred string, args ...any) *DeleteBuilder {
	// interpolating ? ? => $1 $2 ...
	for i := len(b.args) + 1; i <= len(b.args)+len(args); i++ {
		pred = strings.Replace(pred, "?", fmt.Sprintf("$%d", i), 1)
	}

	b.whereClauses = append(b.whereClauses, pred)
	b.args = append(b.args, args...)
	return b
}

func (b *DeleteBuilder) ToSQL() (string, *[]any) {
	var sb strings.Builder

	// FROM ...
	sb.WriteString(fmt.Sprint("DELETE FROM ", b.from))

	// WHERE ...
	sb.WriteString(" WHERE ")
	for i, whereClause := range b.whereClauses {
		if i > 0 {
			sb.WriteString(" AND ")
		}
		sb.WriteString(whereClause)
	}

	return sb.String(), &b.args
}
