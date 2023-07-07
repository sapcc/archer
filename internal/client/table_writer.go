/*
 *   Copyright 2021 SAP SE
 *
 *   Licensed under the Apache License, Version 2.0 (the "License");
 *   you may not use this file except in compliance with the License.
 *   You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 */

package client

import (
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jmoiron/sqlx/reflectx"
)

var DefaultColumns []string

func formatValue(v reflect.Value) string {
	switch kind := v.Kind(); kind {
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Ptr:
		if v.IsNil() {
			return "Null"
		}
		return formatValue(v.Elem())
	case reflect.Struct:
		if v.Type().String() == "time.Time" {
			return v.Interface().(time.Time).In(time.Local).Format(time.RFC850)
		}
		fallthrough
	default:
		return fmt.Sprintf("%+v", v)
	}
}

func getRow(row reflect.Value, iMap [][]int) table.Row {
	if row.Kind() == reflect.Ptr {
		row = row.Elem()
	}

	r := make([]any, 0)
	for i := 0; i < len(iMap); i++ {
		r = append(r, formatValue(reflectx.FieldByIndexes(row, iMap[i])))
	}
	return r
}

func getIndexMap(v reflect.Value) ([][]int, []any, error) {
	type IndexMap struct {
		Header string
		Index  []int
	}

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	header := make([]any, 0)
	var indexes [][]int
	var filterColumns []string

	if !opts.Formatters.Long {
		filterColumns = DefaultColumns
	}
	if len(opts.Formatters.Columns) > 0 {
		filterColumns = opts.Formatters.Columns
	}

	if len(filterColumns) > 0 {
		// Filter columns
		for _, column := range filterColumns {
			header = append(header, column)
		}
		for column, index := range Mapper.TraversalsByName(v.Type(), filterColumns) {
			if len(index) == 0 {
				// Get all columns
				var names []string
				for tagName := range Mapper.TypeMap(v.Type()).Names {
					names = append(names, tagName)
				}
				err := fmt.Errorf("column '%s' is not a valid column filter, possbile filters: %+v",
					filterColumns[column], names)
				return nil, nil, err
			}
			indexes = append(indexes, index)
		}
	} else {
		var indexMap []IndexMap

		// Get all columns
		tm := Mapper.TypeMap(v.Type())
		for tagName, fi := range tm.Names {
			if fi.Field.Type.Kind() == reflect.Struct && fi.Field.Type.String() != "time.Time" {
				continue
			}
			indexMap = append(indexMap, IndexMap{tagName, fi.Index})
		}

		// Stable sort
		sort.SliceStable(indexMap, func(i, j int) bool {
			// Always prefer id, name as first columns, created_at and updated_at as last
			if indexMap[i].Header == "id" {
				return true
			} else if indexMap[j].Header == "id" {
				return false
			} else if indexMap[i].Header == "name" {
				return true
			} else if indexMap[j].Header == "name" {
				return false
			} else if indexMap[i].Header == "updated_at" {
				return false
			} else if indexMap[j].Header == "updated_at" {
				return true
			} else if indexMap[i].Header == "created_at" {
				return false
			} else if indexMap[j].Header == "created_at" {
				return true
			} else {
				return indexMap[i].Index[0] < indexMap[j].Index[0]
			}
		})

		for _, v := range indexMap {
			header = append(header, v.Header)
			indexes = append(indexes, v.Index)
		}
	}

	return indexes, header, nil
}

// WriteTable scans a struct and prints content via Table writer
func WriteTable(data any) error {
	v := reflect.ValueOf(data)

	if v.Kind() == reflect.Ptr {
		// dereference: v = *v
		v = v.Elem()
	}

	if v.Kind() == reflect.Slice && v.Len() > 0 {
		indexMap, header, err := getIndexMap(v.Index(0))
		if err != nil {
			return err
		}

		if opts.Formatters.Format != "value" {
			Table.AppendHeader(header)
		}
		for i := 0; i < v.Len(); i++ {
			Table.AppendRow(getRow(v.Index(i), indexMap))
		}
	}

	if v.Kind() == reflect.Struct {
		// For struct, we transpose the key-value to rows
		indexMap, header, err := getIndexMap(v)
		if err != nil {
			return err
		}

		for i, index := range indexMap {
			Table.AppendRow([]any{header[i], formatValue(reflectx.FieldByIndexes(v, index))})
		}
	}

	// Sort Columns
	if len(opts.Formatters.SortColumn) > 0 {
		var tableSorter []table.SortBy
		for _, sortColumn := range opts.Formatters.SortColumn {
			tableSorter = append(tableSorter, table.SortBy{
				Name: sortColumn,
			})
		}
		Table.SortBy(tableSorter)
	} else {
		Table.SortBy([]table.SortBy{{Name: "created_at", Mode: table.Dsc}})
	}

	switch opts.Formatters.Format {
	case "table":
		Table.Render()
	case "csv":
		Table.RenderCSV()
	case "markdown":
		Table.RenderMarkdown()
	case "html":
		Table.RenderHTML()
	case "value":
		Table.SetStyle(table.Style{
			Name: "value",
			Box: table.BoxStyle{
				MiddleHorizontal: " ",
				MiddleVertical:   " ",
			},
			Options: table.OptionsNoBorders,
		})
		Table.Render()
	default:
		return fmt.Errorf("format option %s is not supported.", opts.Formatters.Format)
	}

	return nil
}
