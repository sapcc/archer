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

package main

import (
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/agent/f5"
	"github.com/sapcc/archer/internal/config"
)

func main() {
	parser := flags.NewParser(&config.Global, flags.Default)
	parser.ShortDescription = "Archer F5 Agent"

	if _, err := parser.Parse(); err != nil {
		code := 1
		if fe, ok := err.(*flags.Error); ok {
			if fe.Type == flags.ErrHelp {
				code = 0
			} else {
				logg.Fatal(fe.Error())
			}
		}
		os.Exit(code)
	}

	// parse config file
	if config.Global.ConfigFile != "" {
		ini := flags.NewIniParser(parser)
		if err := ini.ParseFile(config.Global.ConfigFile); err != nil {
			logg.Fatal(err.Error())
		}
	}

	logg.ShowDebug = config.Global.Default.Debug
	a := f5.NewAgent()
	if err := a.Run(); err != nil {
		logg.Fatal(err.Error())
	}
}
