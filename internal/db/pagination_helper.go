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
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/models"
)

var (
	sortDirKeyRegex  = regexp.MustCompile("^[a-z0-9_]+$")
	defaultSortKeys  = []string{"id", "created_at"}
	ErrInvalidMarker = errors.New("invalid marker")
)

type Pagination struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*Sets the page size.
	  In: query
	*/
	Limit *int64
	/*Pagination ID of the last item in the previous list.
	  In: query
	*/
	Marker *strfmt.UUID
	/*Filter for resources not having tags, multiple not-tags are considered as logical AND.
	Should be provided in a comma separated list.

	  In: query
	*/
	NotTags []string
	/*Filter for resources not having tags, multiple tags are considered as logical OR.
	Should be provided in a comma separated list.

	  In: query
	*/
	NotTagsAny []string
	/*Sets the page direction.
	  In: query
	*/
	PageReverse *bool
	/*Comma-separated list of sort keys, optionally prefix with - to reverse sort order.
	  In: query
	*/
	Sort *string
	/*Filter for tags, multiple tags are considered as logical AND.
	Should be provided in a comma separated list.

	  In: query
	*/
	Tags []string
	/*Filter for tags, multiple tags are considered as logical OR.
	Should be provided in a comma separated list.

	  In: query
	*/
	TagsAny []string
}

func stripDesc(sortDirKey string) (string, bool) {
	sortKey := strings.TrimPrefix(sortDirKey, "-")
	return sortKey, sortKey != sortDirKey
}

// Query pagination helper that also includes policy query filter
func (p *Pagination) Query(db pgxscan.Querier, query string, filter map[string]any) (pgx.Rows, error) {
	var sortDirKeys []string
	var whereClauses []string
	var orderBy string
	var pageReverse bool

	// add filter
	for key := range filter {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = @%s", key, key))
	}

	// tags Filter
	if p.Tags != nil {
		whereClauses = append(whereClauses, "tags @> @tags")
		filter["tags"] = pgtype.FlatArray[string](p.Tags)
	}
	if p.TagsAny != nil {
		whereClauses = append(whereClauses, "tags && @tags_any")
		filter["tags_any"] = pgtype.FlatArray[string](p.TagsAny)
	}
	if p.NotTags != nil {
		whereClauses = append(whereClauses, "NOT ( tags @> @not_tags )")
		filter["not_tags"] = pgtype.FlatArray[string](p.NotTags)
	}
	if p.NotTagsAny != nil {
		whereClauses = append(whereClauses, "NOT ( tags && @not_tags_any )")
		filter["not_tags_any"] = pgtype.FlatArray[string](p.NotTagsAny)
	}

	// page reverse
	if p.PageReverse != nil {
		pageReverse = *p.PageReverse
	}

	// add sorting
	if !config.Global.ApiSettings.DisableSorting && p.Sort != nil {
		sortDirKeys = strings.Split(*p.Sort, ",")

		// add default sort keys (if not existing)
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
		// creates a copy
		sortDirKeys = append(sortDirKeys, defaultSortKeys...)
	}

	// always order to ensure stable result
	orderBy += " ORDER BY "
	for i, sortDirKey := range sortDirKeys {
		// Input sanitation
		if !sortDirKeyRegex.MatchString(sortDirKey) {
			continue
		}

		sortKey, desc := stripDesc(sortDirKey)
		orderBy += sortKey
		if (desc && !pageReverse) || (!desc && pageReverse) {
			orderBy += " DESC"
		} else {
			orderBy += " ASC"
		}

		if i < len(sortDirKeys)-1 {
			orderBy += ", "
		}
	}

	// paginate
	if !config.Global.ApiSettings.DisablePagination && p.Marker != nil {
		sql := fmt.Sprintf(`%s WHERE id = $1`, query)
		if err := pgxscan.Get(context.Background(), db, &filter, sql, p.Marker); err != nil {
			return nil, err
		}

		if len(filter) == 0 {
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

			sortKey := strings.TrimPrefix(sortDirKey, "-")
			if (sortKey != sortDirKey) && !pageReverse || (sortKey == sortDirKey) && pageReverse {
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

	// add WHERE
	if len(whereClauses) > 0 {
		query += " WHERE ( " + strings.Join(whereClauses, " ) AND ( ") + " )"
	}

	// add ORDER BY
	query += orderBy

	// maximum limit
	var maxLimit = config.Global.ApiSettings.PaginationMaxLimit
	if p.Limit == nil || (p.Limit != nil && *p.Limit > maxLimit) {
		p.Limit = &maxLimit
	}
	query += fmt.Sprint(" LIMIT ", *p.Limit)

	return db.Query(context.Background(), query, pgx.NamedArgs(filter))
}

func (p *Pagination) GetLinks(modelList any) []*models.Link {
	var links []*models.Link
	if reflect.TypeOf(modelList).Kind() != reflect.Slice {
		return nil
	}

	s := reflect.ValueOf(modelList)
	if s.Len() > 0 {
		var prevAttr, nextAttr []string
		first := s.Index(0).Elem().FieldByName("ID").String()
		last := s.Index(s.Len() - 1).Elem().FieldByName("ID").String()

		for key, val := range p.HTTPRequest.URL.Query() {
			if key == "marker" || key == "page_reverse" {
				continue
			}
			prevAttr = append(prevAttr, fmt.Sprint(key, "=", val[0]))
		}

		// Make a shallow copy
		nextAttr = append(prevAttr[:0:0], prevAttr...)

		// Previous link of marker supplied
		if p.Marker != nil {
			prevAttr = append(prevAttr, fmt.Sprintf("marker=%s", first), "page_reverse=True")
			prevUrl := fmt.Sprint(config.GetApiBaseUrl(p.HTTPRequest), p.HTTPRequest.URL.Path,
				"?", strings.Join(prevAttr, "&"))

			links = append(links, &models.Link{
				Href: prevUrl,
				Rel:  "previous",
			})
		}

		// Next link of limit < size(fetched items)
		if p.Limit != nil && int64(s.Len()) >= *p.Limit {
			nextAttr = append(nextAttr, fmt.Sprintf("marker=%s", last))
			nextUrl := fmt.Sprint(config.GetApiBaseUrl(p.HTTPRequest), p.HTTPRequest.URL.Path,
				"?", strings.Join(nextAttr, "&"))
			links = append(links, &models.Link{
				Href: nextUrl,
				Rel:  "next",
			})
		}
	}
	return links
}
