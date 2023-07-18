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

package client

import (
	"fmt"

	"github.com/sapcc/archer/client/version"
	"github.com/sapcc/archer/internal/config"
)

type VersionOptions struct{}

func (*VersionOptions) Execute(args []string) error {
	fmt.Printf("CLI Version: %s (%s)\n", config.Version, config.BuildTime)
	res, err := ArcherClient.Version.Get(version.NewGetParams())
	if err != nil {
		return err
	}
	fmt.Printf("Server Version: %s (%s) %+v\n", res.Payload.Version, res.Payload.Updated, res.Payload.Capabilities)
	return nil
}

func init() {
	if _, err := Parser.AddCommand("version", "Version",
		"Show Version.", &VersionOptions{}); err != nil {
		panic(err)
	}
}
