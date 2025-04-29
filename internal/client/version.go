// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	"github.com/sapcc/archer/client/version"
	"github.com/sapcc/archer/internal/config"
)

type VersionOptions struct{}

func (*VersionOptions) Execute(_ []string) error {
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
