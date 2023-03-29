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

	"github.com/jedib0t/go-pretty/table"
)

func formatValue(v reflect.Value) string {
	switch kind := v.Kind(); kind {
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	case reflect.Ptr:
		if v.IsNil() {
			return "Null"
		}
		return formatValue(v.Elem())
	default:
		return fmt.Sprintf("%v", v)
	}
}

func getRow(row reflect.Value, iMap []int) table.Row {
	if row.Kind() == reflect.Ptr {
		row = row.Elem()
	}

	r := make([]interface{}, 0)
	for i := 0; i < len(iMap); i++ {
		r = append(r, formatValue(row.Field(iMap[i])))
	}
	return r
}

func addSortedHeader(v reflect.Value) ([]int, error) {
	type IndexMap struct {
		Header string
		Index  int
	}

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	header := make([]interface{}, 0)
	var indexes []int
	if len(opts.Formatters.Columns) > 0 {
		// Filter columns
		for _, column := range opts.Formatters.Columns {
			header = append(header, column)
		}
		for column, index := range Mapper.TraversalsByName(v.Type(), opts.Formatters.Columns) {
			if len(index) == 0 {
				err := fmt.Errorf("column '%s' is not a valid column filter", opts.Formatters.Columns[column])
				return nil, err
			}
			indexes = append(indexes, index[0])
		}
	} else {
		var indexMap []IndexMap

		// Get all columns
		tm := Mapper.TypeMap(v.Type())
		for tagName, fi := range tm.Names {
			indexMap = append(indexMap, IndexMap{tagName, fi.Index[0]})
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
				return indexMap[i].Index < indexMap[j].Index
			}
		})

		for _, v := range indexMap {
			header = append(header, v.Header)
			indexes = append(indexes, v.Index)
		}
	}

	if opts.Formatters.Format != "value" {
		Table.AppendHeader(header)
	}
	return indexes, nil
}

// WriteTableFromStruct scans a struct and prints content via Table writer
func WriteTable(data interface{}) error {
	v := reflect.ValueOf(data)

	if v.Kind() == reflect.Ptr {
		// dereference: v = *v
		v = v.Elem()
	}

	if v.Kind() == reflect.Slice && v.Len() > 0 {
		indexMap, err := addSortedHeader(v.Index(0))
		if err != nil {
			return err
		}
		for i := 0; i < v.Len(); i++ {
			Table.AppendRow(getRow(v.Index(i), indexMap))
		}
	}

	if v.Kind() == reflect.Struct {
		// For struct, we transpose the key-value to rows
		for key, field := range Mapper.FieldMap(v) {
			if opts.Formatters.Columns == nil {
				Table.AppendRow(table.Row{key, formatValue(field)})
				continue
			}

			for _, column := range opts.Formatters.Columns {
				if column == key {
					Table.AppendRow(table.Row{key, formatValue(field)})
					continue
				}
			}
		}
		Table.SortBy([]table.SortBy{{Number: 1, Mode: table.Asc}})
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
		Table.SetStyle(table.StyleLight)
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
