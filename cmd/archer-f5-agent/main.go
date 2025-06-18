// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/sapcc/archer/internal/agent/f5"
	"github.com/sapcc/archer/internal/config"
)

func main() {
	parser := flags.NewParser(&config.Global, flags.Default)
	parser.ShortDescription = "Archer F5 Agent"

	if _, err := parser.Parse(); err != nil {
		code := 1
		var fe *flags.Error
		if errors.As(err, &fe) {
			if fe.Type == flags.ErrHelp {
				code = 0
			}
		}
		os.Exit(code)
	}

	config.ParseConfig(parser)
	config.InitSentry()

	a := f5.NewAgent()
	a.Run()
}
