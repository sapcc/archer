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
	"net/url"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
)

// for a valid return value.
func TestPaginationGeneric(t *testing.T) {
	config.Global.ApiSettings.PaginationMaxLimit = 10

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT \* FROM example .*`).
		WithArgs(pgx.NamedArgs(nil)).
		WillReturnRows()
	p := Pagination{
		HTTPRequest: &http.Request{URL: &url.URL{RawQuery: ""}},
	}

	_, err = p.Query(mock, "SELECT * FROM example", nil)
	if err != nil {
		t.Error(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	links := p.GetLinks([]*struct{}{})
	assert.Empty(t, links)
}

func TestPaginationLimit(t *testing.T) {
	config.Global.ApiSettings.PaginationMaxLimit = 1000

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	exampleRows := mock.
		NewRows([]string{"id"}).
		AddRow("00000000-0000-0000-0000-000000000000").
		AddRow("00000000-0000-0000-0000-000000000001")

	mock.ExpectQuery(`SELECT \* FROM example .* LIMIT 2`).
		WithArgs(pgx.NamedArgs(nil)).
		WillReturnRows(exampleRows)
	two := int64(2)
	p := Pagination{
		HTTPRequest: &http.Request{URL: &url.URL{RawQuery: "limit=2"}},
		Limit:       &two,
	}

	rows, err := p.Query(mock, "SELECT * FROM example", nil)
	if err != nil {
		assert.Error(t, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		assert.Errorf(t, err, "there were unfulfilled expectations")
	}

	type example struct {
		ID string
	}
	var items []*example
	for rows.Next() {
		var item example
		_ = rows.Scan(&item.ID)
		items = append(items, &item)
	}

	links := p.GetLinks(items)
	assert.NotEmpty(t, links)
	assert.Equal(t, links[0].Rel, "next")
	assert.Contains(t, links[0].Href, "marker=00000000-0000-0000-0000-000000000001")
}

func TestPaginationMarker(t *testing.T) {
	config.Global.ApiSettings.PaginationMaxLimit = 1000

	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	marker := strfmt.UUID("00000000-0000-0000-0000-000000000001")
	values := [][]any{
		{marker},
		{strfmt.UUID("00000000-0000-0000-0000-000000000002")},
		{strfmt.UUID("00000000-0000-0000-0000-000000000003")},
	}

	mock.ExpectQuery(`SELECT id FROM example WHERE id = $1`).WithArgs(&marker).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(marker))
	mock.ExpectQuery(`SELECT id FROM example WHERE ( ( id > @id ) OR ( id = @id AND created_at > @created_at ) ) ORDER BY id ASC, created_at ASC LIMIT 1000`).
		WithArgs(pgx.NamedArgs{"id": marker}).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRows(values[1:]...))

	p := Pagination{
		HTTPRequest: &http.Request{URL: &url.URL{RawQuery: ""}},
		Marker:      &marker,
	}

	if _, err := p.Query(mock, "SELECT id FROM example", nil); err != nil {
		t.Error(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	type example struct {
		ID string
	}
	var data = []*example{
		{ID: values[1][0].(strfmt.UUID).String()},
		{ID: values[2][0].(strfmt.UUID).String()},
	}
	links := p.GetLinks(data)
	assert.Equal(t, "previous", links[0].Rel)
	assert.Equal(t, "http:?marker=00000000-0000-0000-0000-000000000002&page_reverse=True",
		links[0].Href)
}

func TestPageReverse(t *testing.T) {
	config.Global.ApiSettings.PaginationMaxLimit = 2
	mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	values := [][]any{
		{strfmt.UUID("00000000-0000-0000-0000-000000000002")},
		{strfmt.UUID("00000000-0000-0000-0000-000000000003")},
	}

	mock.ExpectQuery("SELECT id FROM example ORDER BY id DESC, created_at DESC LIMIT 2").
		WithArgs(pgx.NamedArgs(nil)).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRows(values...))

	pageReverse := true
	p := Pagination{
		HTTPRequest: &http.Request{URL: &url.URL{RawQuery: ""}},
		PageReverse: &pageReverse,
	}

	if _, err := p.Query(mock, "SELECT id FROM example", nil); err != nil {
		t.Error(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	type example struct {
		ID string
	}
	var data = []*example{
		{ID: values[1][0].(strfmt.UUID).String()},
		{ID: values[0][0].(strfmt.UUID).String()},
	}
	links := p.GetLinks(data)
	assert.Equal(t, "next", links[0].Rel)
	assert.Equal(t, "http:?marker=00000000-0000-0000-0000-000000000002",
		links[0].Href)
}
