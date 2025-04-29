// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/strfmt"
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
	/*Filter for resources belonging or accessible by a specific project.

	  Max Length: 32
	  Min Length: 32
	  In: query
	*/
	ProjectID *string
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

func (p *Pagination) Query(db pgxscan.Querier, q sq.SelectBuilder) (string, []any, error) {
	var sortDirKeys []string
	var pageReverse bool

	if p.ProjectID != nil {
		q = q.Where("project_id = ?", p.ProjectID)
	}

	// tags Filter
	if p.Tags != nil {
		q = q.Where("tags @> ?", pgtype.FlatArray[string](p.Tags))
	}
	if p.TagsAny != nil {
		q = q.Where("tags && ?", pgtype.FlatArray[string](p.TagsAny))
	}
	if p.NotTags != nil {
		q = q.Where("NOT ( tags @> ? )", pgtype.FlatArray[string](p.NotTags))
	}
	if p.NotTagsAny != nil {
		q = q.Where("NOT ( tags && ? )", pgtype.FlatArray[string](p.NotTagsAny))
	}

	// page reverse
	if p.PageReverse != nil {
		pageReverse = *p.PageReverse
	}

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

	// paginate
	if !config.Global.ApiSettings.DisablePagination && p.Marker != nil {
		filter := make(map[string]any)
		filter["id"] = strfmt.UUID("")
		sql, args := q.Where("id = ?", p.Marker).MustSql()
		if err := pgxscan.Get(context.Background(), db, &filter, sql, args...); err != nil {
			return "", nil, err
		}

		if len(filter) == 0 {
			return "", nil, ErrInvalidMarker
		}

		// workaround so squirrel doesn't complain about arrays from uuids
		for key, val := range filter {
			if v, ok := val.([16]uint8); ok {
				filter[key] = pgtype.UUID{Bytes: v, Valid: true}
			}
		}

		// Craft WHERE ... conditions
		var sortWhereClauses sq.Or
		for i, sortDirKey := range sortDirKeys {
			var critAttrs sq.And
			for j := range sortDirKeys[:i] {
				sortKey := strings.TrimPrefix(sortDirKeys[j], "-")
				critAttrs = append(critAttrs, sq.Eq{sortKey: filter[sortKey]})
			}

			sortKey := strings.TrimPrefix(sortDirKey, "-")
			if (sortKey != sortDirKey) && !pageReverse || (sortKey == sortDirKey) && pageReverse {
				critAttrs = append(critAttrs, sq.Lt{sortKey: filter[sortKey]})
			} else {
				critAttrs = append(critAttrs, sq.Gt{sortKey: filter[sortKey]})
			}

			sortWhereClauses = append(sortWhereClauses, critAttrs)
		}
		q = q.Where(sortWhereClauses)
	}

	// always order to ensure stable result
	for _, sortDirKey := range sortDirKeys {
		// Input sanitation
		if !sortDirKeyRegex.MatchString(sortDirKey) {
			continue
		}

		sortKey, desc := stripDesc(sortDirKey)

		if (desc && !pageReverse) || (!desc && pageReverse) {
			q = q.OrderBy(fmt.Sprintf("%s DESC", sortKey))
		} else {
			q = q.OrderBy(fmt.Sprintf("%s ASC", sortKey))
		}
	}

	var maxLimit = config.Global.ApiSettings.PaginationMaxLimit
	if p.Limit == nil || (p.Limit != nil && *p.Limit > maxLimit) {
		p.Limit = &maxLimit
	}
	q = q.Limit(uint64(*p.Limit))

	return q.ToSql()
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
