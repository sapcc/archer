// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnique(t *testing.T) {
	// nil input must return nil to preserve COALESCE semantics in partial PUTs
	assert.Nil(t, Unique(nil))

	// empty slice stays empty
	assert.Equal(t, []string{}, Unique([]string{}))

	// duplicates are removed, order is preserved
	assert.Equal(t, []string{"a", "b", "c"}, Unique([]string{"a", "b", "a", "c", "b"}))

	// single element
	assert.Equal(t, []string{"x"}, Unique([]string{"x"}))
}
