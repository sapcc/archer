// SPDX-FileCopyrightText: Copyright 2026 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"encoding/csv"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sapcc/archer/v2/models"
)

type tableWriterTestRow struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Target    struct {
		Value string `json:"value"`
	} `json:"target"`
}

func setupTableWriterTest(t *testing.T) *bytes.Buffer {
	t.Helper()

	originalFormatters := opts.Formatters
	originalDefaultColumns := DefaultColumns
	originalTable := Table
	output := &bytes.Buffer{}
	Table = table.NewWriter()
	Table.SetOutputMirror(output)
	t.Cleanup(func() {
		opts.Formatters = originalFormatters
		DefaultColumns = originalDefaultColumns
		Table = originalTable
	})

	return output
}

func TestFormatValue(t *testing.T) {
	value := 42
	timestamp := time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)

	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "boolean", value: true, want: "true"},
		{name: "nil pointer", value: (*int)(nil), want: "Null"},
		{name: "pointer", value: &value, want: "42"},
		{name: "timestamp", value: timestamp, want: timestamp.In(time.Local).Format(time.RFC850)},
		{name: "signed ranges", value: []int{5, 2, 3, 1, 7}, want: "[1-3, 5, 7]"},
		{name: "unsigned ranges", value: []uint{4, 2, 3}, want: "[2-4]"},
		{name: "string slice", value: []string{"one", "two"}, want: "[one two]"},
		{name: "string", value: "value", want: "value"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, formatValue(reflect.ValueOf(test.value)))
		})
	}
}

func TestGetRow(t *testing.T) {
	setupTableWriterTest(t)
	row := &tableWriterTestRow{ID: "endpoint-id", Name: "endpoint"}
	indexes := [][]int{{0}, {1}}

	assert.Equal(t, table.Row{"endpoint-id", "endpoint"}, getRow(reflect.ValueOf(row), indexes, []any{"id", "name"}))
}

func TestGetIndexMap(t *testing.T) {
	t.Run("default columns", func(t *testing.T) {
		setupTableWriterTest(t)
		DefaultColumns = []string{"name", "id"}

		indexes, header, err := getIndexMap(reflect.ValueOf(&tableWriterTestRow{}))
		require.NoError(t, err)
		assert.Equal(t, []any{"name", "id"}, header)
		assert.Equal(t, table.Row{"endpoint", "endpoint-id"}, getRow(
			reflect.ValueOf(tableWriterTestRow{ID: "endpoint-id", Name: "endpoint"}), indexes, header))
	})

	t.Run("explicit columns override defaults", func(t *testing.T) {
		setupTableWriterTest(t)
		DefaultColumns = []string{"id"}
		opts.Formatters.Columns = []string{"status", "name"}

		_, header, err := getIndexMap(reflect.ValueOf(tableWriterTestRow{}))
		require.NoError(t, err)
		assert.Equal(t, []any{"status", "name"}, header)
	})

	t.Run("invalid column", func(t *testing.T) {
		setupTableWriterTest(t)
		opts.Formatters.Columns = []string{"missing"}

		_, _, err := getIndexMap(reflect.ValueOf(tableWriterTestRow{}))
		require.Error(t, err)
		assert.ErrorContains(t, err, "column 'missing' is not a valid column filter")
	})
}

func TestWriteTable(t *testing.T) {
	createdAt := time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)
	data := []tableWriterTestRow{{ID: "one", Name: "first", Status: "available", CreatedAt: createdAt}}

	tests := []struct {
		name   string
		format string
		data   any
		want   []string
	}{
		{name: "table slice", format: "table", data: data, want: []string{"ID", "NAME", "one", "first"}},
		{name: "csv slice", format: "csv", data: data, want: []string{"id,name", "one,first"}},
		{name: "markdown slice", format: "markdown", data: data, want: []string{"| id", "one", "first"}},
		{name: "html slice", format: "html", data: data, want: []string{"<table", "<th>id</th>", "one", "first"}},
		{name: "value struct", format: "value", data: data[0], want: []string{"one", "first"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := setupTableWriterTest(t)
			DefaultColumns = []string{"id", "name"}
			opts.Formatters = outputFormatters{Format: test.format, NoColor: true}

			require.NoError(t, WriteTable(test.data))
			for _, want := range test.want {
				assert.Contains(t, output.String(), want)
			}
		})
	}

	t.Run("custom sorting", func(t *testing.T) {
		output := setupTableWriterTest(t)
		DefaultColumns = []string{"id", "name"}
		opts.Formatters = outputFormatters{Format: "csv", NoColor: true, SortColumn: []string{"name"}}

		require.NoError(t, WriteTable([]tableWriterTestRow{{ID: "two", Name: "second"}, {ID: "one", Name: "first"}}))
		assert.Contains(t, output.String(), "one,first")
		assert.Contains(t, output.String(), "two,second")
		assert.Less(t, strings.Index(output.String(), "one,first"), strings.Index(output.String(), "two,second"))
	})

	t.Run("empty input", func(t *testing.T) {
		setupTableWriterTest(t)
		opts.Formatters = outputFormatters{Format: "csv", NoColor: true}
		require.NoError(t, WriteTable([]tableWriterTestRow{}))
	})

	t.Run("struct column", func(t *testing.T) {
		output := setupTableWriterTest(t)
		opts.Formatters = outputFormatters{Format: "csv", NoColor: true, Columns: []string{"target"}}
		row := tableWriterTestRow{}
		row.Target.Value = "value"

		require.NotPanics(t, func() {
			require.NoError(t, WriteTable(row))
		})
		records, err := csv.NewReader(strings.NewReader(output.String())).ReadAll()
		require.NoError(t, err)
		assert.Equal(t, [][]string{{"target", `{"value":"value"}`}}, records)
	})

	t.Run("unsupported format", func(t *testing.T) {
		setupTableWriterTest(t)
		opts.Formatters = outputFormatters{Format: "json"}
		err := WriteTable(data)
		require.Error(t, err)
		assert.ErrorContains(t, err, "format option json is not supported")
	})
}

func TestGetIndexMapLongEndpointColumnsAreDeterministic(t *testing.T) {
	originalFormatters := opts.Formatters
	t.Cleanup(func() { opts.Formatters = originalFormatters })
	opts.Formatters = outputFormatters{Long: true}

	want := []any{
		"id", "name", "connection_mirroring", "description", "ip_address",
		"project_id", "service_id", "status", "tags",
		"target.network", "target.port", "target.subnet",
		"created_at", "updated_at",
	}

	for range 100 {
		_, header, err := getIndexMap(reflect.ValueOf(models.Endpoint{}))
		require.NoError(t, err)
		assert.Equal(t, want, header)
	}
}

func TestGetIndexMapPreservesRequestedColumnOrder(t *testing.T) {
	originalFormatters := opts.Formatters
	t.Cleanup(func() { opts.Formatters = originalFormatters })
	opts.Formatters = outputFormatters{Columns: []string{"target.subnet", "id", "target.port"}}

	_, header, err := getIndexMap(reflect.ValueOf(models.Endpoint{}))
	require.NoError(t, err)
	assert.Equal(t, []any{"target.subnet", "id", "target.port"}, header)
}
