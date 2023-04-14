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
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/models"
)

var (
	sortDirKeyRegex  = regexp.MustCompile("^[a-z0-9_]+$")
	defaultSortKeys  = []string{"id", "created_at"}
	ErrInvalidMarker = errors.New("invalid marker")
)

type Pagination struct {
	/*Sets the page size.
	  In: query
	*/
	limit *int64
	/*Pagination ID of the last item in the previous list.
	  In: query
	*/
	marker *strfmt.UUID
	/*Sets the page direction.
	  In: query
	*/
	pageReverse *bool
	/*Comma-separated list of sort keys optionally prefix with - to reverse sort order.
	  In: query
	*/
	sort *string

	table string
}

func NewPagination(Table string, Limit *int64, Marker *strfmt.UUID, Sort *string, pageReverse *bool) *Pagination {
	return &Pagination{
		limit:       Limit,
		marker:      Marker,
		sort:        Sort,
		pageReverse: pageReverse,
		table:       Table,
	}
}

func stripDesc(sortDirKey string) (string, bool) {
	sortKey := strings.TrimPrefix(sortDirKey, "-")
	return sortKey, sortKey != sortDirKey
}

// Query pagination helper that also includes policy query filter
func (p *Pagination) Query(db *pgxpool.Pool, filter map[string]string) (pgx.Rows, error) {
	var sortDirKeys []string
	var whereClauses []string
	var orderBy string
	markerObj := make(map[string]any)

	query := fmt.Sprintf(`SELECT * FROM %s`, p.table)

	//add filter
	for key, val := range filter {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = '%s'", key, val))
	}

	//add sorting
	if !config.Global.ApiSettings.DisableSorting && p.sort != nil {
		sortDirKeys = strings.Split(*p.sort, ",")

		// Add default sort keys (if not existing)
		for _, defaultSortKey := range defaultSortKeys {
			found := false
			for _, paramSortKey := range sortDirKeys {
				sortKey, _ := stripDesc(paramSortKey)
				if sortKey == defaultSortKey {
					found = true
					break
				}
			}

			if !found {
				sortDirKeys = append(sortDirKeys, defaultSortKey)
			}
		}
	} else {
		// Creates a copy
		sortDirKeys = append(sortDirKeys, defaultSortKeys...)
	}

	//always order to ensure stable result
	orderBy += " ORDER BY "
	for i, sortDirKey := range sortDirKeys {
		// Input sanitation
		if !sortDirKeyRegex.MatchString(sortDirKey) {
			continue
		}

		if sortKey, ok := stripDesc(sortDirKey); ok {
			orderBy += fmt.Sprintf("%s DESC", sortKey)
		} else {
			orderBy += sortDirKey
		}

		if i < len(sortDirKeys)-1 {
			orderBy += ", "
		}
	}

	if !config.Global.ApiSettings.DisablePagination && p.marker != nil {
		sql := fmt.Sprintf(`SELECT * FROM %s WHERE id = $1`, p.table)
		if err := pgxscan.Get(context.Background(), db, &markerObj, sql, p.marker); err != nil {
			return nil, err
		}

		if len(markerObj) == 0 {
			return nil, ErrInvalidMarker
		}

		// Craft WHERE ... conditions
		var sortWhereClauses strings.Builder
		for i, sortDirKey := range sortDirKeys {
			var critAttrs []string = nil
			for j := range sortDirKeys[:i] {
				sortKey := strings.TrimPrefix(sortDirKeys[j], "-")
				critAttrs = append(critAttrs, fmt.Sprintf("%s = @%s", sortKey, sortKey))
			}

			if sortKey := strings.TrimPrefix(sortDirKey, "-"); sortKey != sortDirKey {
				critAttrs = append(critAttrs, fmt.Sprintf("%s < @%s", sortKey, sortKey))
			} else {
				critAttrs = append(critAttrs, fmt.Sprintf("%s > @%s", sortKey, sortKey))
			}

			sortWhereClauses.WriteString("( " + strings.Join(critAttrs, " AND ") + " )")

			if i < len(sortDirKeys)-1 {
				sortWhereClauses.WriteString(" OR ")
			}
		}
		whereClauses = append(whereClauses, sortWhereClauses.String())
	}

	//add WHERE
	if len(whereClauses) > 0 {
		query += " WHERE ( " + strings.Join(whereClauses, " ) AND ( ") + " )"
	}

	//add ORDER BY
	query += orderBy

	var limit = config.Global.ApiSettings.PaginationMaxLimit
	if p.limit != nil && *p.limit < config.Global.ApiSettings.PaginationMaxLimit {
		limit = *p.limit
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	return db.Query(context.Background(), query, pgx.NamedArgs(markerObj))
}

func (p *Pagination) GetLinks(modelList any, r *http.Request) []*models.Link {
	var links []*models.Link
	if reflect.TypeOf(modelList).Kind() != reflect.Slice {
		return nil
	}

	s := reflect.ValueOf(modelList)
	if s.Len() > 0 {
		var prevAttr, nextAttr []string
		first := s.Index(0).Elem().FieldByName("ID").String()
		last := s.Index(s.Len() - 1).Elem().FieldByName("ID").String()

		if p.sort != nil {
			prevAttr = append(prevAttr, fmt.Sprintf("sort=%s", *p.sort))
		}
		if p.limit != nil {
			prevAttr = append(prevAttr, fmt.Sprintf("limit=%d", *p.limit))
		}

		// Make a copy
		nextAttr = append(prevAttr[:0:0], prevAttr...)

		// Previous link of marker supplied
		if p.marker != nil {
			prevAttr = append(prevAttr, fmt.Sprintf("marker=%s", first), "page_reverse=True")
			prevUrl := fmt.Sprintf("%s%s?%s", config.Global.ApiSettings.ApiBaseURL,
				r.URL.Path, strings.Join(prevAttr, "&"))

			links = append(links, &models.Link{
				Href: prevUrl,
				Rel:  "previous",
			})
		}

		// Next link of limit < size(fetched items)
		if p.limit != nil && int64(s.Len()) >= *p.limit {
			nextAttr = append(nextAttr, fmt.Sprintf("marker=%s", last))
			nextUrl := fmt.Sprintf("%s%s?%s", config.Global.ApiSettings.ApiBaseURL,
				r.URL.Path, strings.Join(nextAttr, "&"))
			links = append(links, &models.Link{
				Href: nextUrl,
				Rel:  "next",
			})
		}
	}
	return links
}
