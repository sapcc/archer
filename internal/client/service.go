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
	"github.com/jedib0t/go-pretty/table"
)

var ServiceOptions struct {
	ServiceList `command:"list" description:"List Services"`
}

type ServiceList struct{}

func (*ServiceList) Execute(_ []string) error {
	resp, err := ArcherClient.Service.GetService(nil, nil)
	if err != nil {
		return err
	}

	Table.AppendHeader(table.Row{"ID", "Name"})
	for _, service := range resp.Payload {
		Table.AppendRow(table.Row{service.ID, service.Name})
	}
	Table.Render()
	return nil
}

func init() {
	_, _ = Parser.AddCommand("service", "Services", "Service Commands.", &ServiceOptions)
}
