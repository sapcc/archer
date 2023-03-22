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
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
)

// for a valid return value.
func TestPaginationGeneric(t *testing.T) {
	config.Global.ApiSettings.PaginationMaxLimit = 10

	p := Pagination{
		HTTPRequest: &http.Request{URL: &url.URL{RawQuery: ""}},
	}

	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()
	sql, args, err := p.Query(dbMock, Select("*").From("example"))
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, "SELECT * FROM example ORDER BY id ASC, created_at ASC LIMIT 10", sql)
	assert.Nil(t, args)

	links := p.GetLinks([]*struct{}{})
	assert.Empty(t, links)
}

func TestPaginationLimit(t *testing.T) {
	config.Global.ApiSettings.PaginationMaxLimit = 1000

	two := int64(2)
	p := Pagination{
		HTTPRequest: &http.Request{URL: &url.URL{RawQuery: "limit=2"}},
		Limit:       &two,
	}

	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()
	sql, args, err := p.Query(dbMock, Select("*").From("example"))
	if err != nil {
		assert.Error(t, err)
	}

	type example struct {
		ID string
	}

	assert.Equal(t, "SELECT * FROM example ORDER BY id ASC, created_at ASC LIMIT 2", sql)
	assert.Nil(t, args)

	var items = []*example{
		{"00000000-0000-0000-0000-000000000000"},
		{"00000000-0000-0000-0000-000000000001"},
	}
	links := p.GetLinks(items)
	assert.NotEmpty(t, links)
	assert.Equal(t, links[0].Rel, "next")
	assert.Contains(t, links[0].Href, "marker=00000000-0000-0000-0000-000000000001")
}

func TestPaginationMarker(t *testing.T) {
	config.Global.ApiSettings.PaginationMaxLimit = 1000

	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	marker := strfmt.UUID("00000000-0000-0000-0000-000000000001")
	now := time.Now()

	dbMock.ExpectQuery(`SELECT * FROM example WHERE id = $1`).WithArgs(&marker).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).AddRow(marker, now))

	p := Pagination{
		HTTPRequest: &http.Request{URL: &url.URL{RawQuery: ""}},
		Marker:      &marker,
	}

	sql, args, err := p.Query(dbMock, Select("*").From("example"))
	assert.Nil(t, err)
	assert.Equal(t, `SELECT * FROM example WHERE ((id > $1) OR (id = $2 AND created_at > $3)) ORDER BY id ASC, created_at ASC LIMIT 1000`,
		sql)
	assert.Len(t, args, 3)
	assert.Equal(t, args[0], marker.String())
	assert.Equal(t, args[1], marker.String())
	assert.Equal(t, args[2], now)
	if err := dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	type example struct {
		ID string
	}
	var data = []*example{
		{ID: "00000000-0000-0000-0000-000000000002"},
		{ID: "00000000-0000-0000-0000-000000000003"},
	}
	links := p.GetLinks(data)
	assert.Equal(t, "previous", links[0].Rel)
	assert.Equal(t, "http:?marker=00000000-0000-0000-0000-000000000002&page_reverse=True",
		links[0].Href)
}

func TestPageReverse(t *testing.T) {
	config.Global.ApiSettings.PaginationMaxLimit = 2
	dbMock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer dbMock.Close()

	pageReverse := true
	p := Pagination{
		HTTPRequest: &http.Request{URL: &url.URL{RawQuery: ""}},
		PageReverse: &pageReverse,
	}

	sql, args, err := p.Query(dbMock, Select("*").From("example"))
	assert.Nil(t, err)
	assert.Equal(t, sql, "SELECT * FROM example ORDER BY id DESC, created_at DESC LIMIT 2")
	assert.Nil(t, args)
	if err := dbMock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}

	type example struct {
		ID string
	}
	var data = []*example{
		{ID: "00000000-0000-0000-0000-000000000003"},
		{ID: "00000000-0000-0000-0000-000000000002"},
	}
	links := p.GetLinks(data)
	assert.Equal(t, "next", links[0].Rel)
	assert.Equal(t, "http:?marker=00000000-0000-0000-0000-000000000002",
		links[0].Href)
}
