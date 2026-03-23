// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/jmoiron/sqlx/reflectx"
)

var DefaultColumns []string

// colorizeValue applies color to specific values when color mode is enabled
func colorizeValue(value string, columnName string) string {
	// Only colorize for table format
	if opts.Formatters.Format != "table" || opts.Formatters.NoColor {
		return value
	}

	// Color based on common patterns
	switch strings.ToLower(value) {
	case "true", "enabled", "active", "available", "success", "accepted":
		return text.FgGreen.Sprint(value)
	case "false", "disabled", "inactive", "error", "failed", "rejected":
		return text.FgRed.Sprint(value)
	case "pending", "processing", "waiting":
		return text.FgYellow.Sprint(value)
	case "null":
		return text.FgHiBlack.Sprint(value)
	}

	// Color based on column name
	columnLower := strings.ToLower(columnName)
	switch columnLower {
	case "status":
		// Status-specific colors
		if strings.Contains(strings.ToUpper(value), "DOWN") || strings.Contains(strings.ToUpper(value), "ERROR") {
			return text.FgRed.Sprint(value)
		}
		return text.FgCyan.Sprint(value)
	case "id", "project_id":
		return text.FgHiBlack.Sprint(value)
	case "name":
		return text.FgCyan.Sprint(value)
	}

	return value
}

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
	case reflect.Slice:
		// check if slice elements are integers, if so, we can group consecutive integers into ranges
		signedTypes := []reflect.Kind{reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64}
		unsignedTypes := []reflect.Kind{reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64}
		elemKind := v.Type().Elem().Kind()
		if slices.Contains(signedTypes, elemKind) || slices.Contains(unsignedTypes, elemKind) {
			ints := make([]int, v.Len())
			for i := 0; i < v.Len(); i++ {
				if slices.Contains(unsignedTypes, elemKind) {
					ints[i] = int(v.Index(i).Uint())
				} else {
					ints[i] = int(v.Index(i).Int())
				}
			}
			sort.Ints(ints)

			// Group consecutive integers into ranges
			var parts []string
			i := 0
			for i < len(ints) {
				start := ints[i]
				end := start
				for i+1 < len(ints) && ints[i+1] == end+1 {
					i++
					end = ints[i]
				}
				if start == end {
					parts = append(parts, fmt.Sprintf("%d", start))
				} else {
					parts = append(parts, fmt.Sprintf("%d-%d", start, end))
				}
				i++
			}
			return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
		}
		fallthrough
	default:
		return fmt.Sprintf("%+v", v)
	}
}

func getRow(row reflect.Value, iMap [][]int, header []any) table.Row {
	if row.Kind() == reflect.Ptr {
		row = row.Elem()
	}

	r := make([]any, 0)
	for i := range iMap {
		value := formatValue(reflectx.FieldByIndexes(row, iMap[i]))
		columnName := ""
		if i < len(header) {
			columnName = fmt.Sprintf("%v", header[i])
		}
		r = append(r, colorizeValue(value, columnName))
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
				err := fmt.Errorf("column '%s' is not a valid column filter, possible filters: %+v",
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
			if !opts.Formatters.NoColor && opts.Formatters.Format == "table" {
				// Colorize headers
				coloredHeader := make(table.Row, len(header))
				for i, h := range header {
					coloredHeader[i] = text.FgHiCyan.Sprint(h)
				}
				Table.AppendHeader(coloredHeader)
			} else {
				Table.AppendHeader(header)
			}
		}
		for i := 0; i < v.Len(); i++ {
			Table.AppendRow(getRow(v.Index(i), indexMap, header))
		}
	}

	if v.Kind() == reflect.Struct {
		// For struct, we transpose the key-value to rows
		indexMap, header, err := getIndexMap(v)
		if err != nil {
			return err
		}

		for i, index := range indexMap {
			columnName := fmt.Sprintf("%v", header[i])
			value := formatValue(reflectx.FieldByIndexes(v, index))
			if opts.Formatters.Format == "value" {
				Table.AppendRow(table.Row{value})
			} else {
				if !opts.Formatters.NoColor && opts.Formatters.Format == "table" {
					Table.AppendRow(table.Row{
						text.FgHiCyan.Sprint(columnName),
						colorizeValue(value, columnName),
					})
				} else {
					Table.AppendRow(table.Row{columnName, value})
				}
			}
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
		return fmt.Errorf("format option %s is not supported", opts.Formatters.Format)
	}

	return nil
}
