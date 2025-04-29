// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"github.com/Masterminds/squirrel"
)

// This is a basic postgresql query generator that allows building select stmts with ORM feeling.

var (
	StmtBuilder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	Select      = StmtBuilder.Select
	Update      = StmtBuilder.Update
	Insert      = StmtBuilder.Insert
	Delete      = StmtBuilder.Delete
)
