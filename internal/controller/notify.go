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

package controller

import (
	"context"
	"fmt"

	"github.com/go-openapi/strfmt"
	"github.com/sapcc/go-bits/logg"
)

func (c *Controller) notifyService(host string) {
	if _, err := c.pool.Exec(context.Background(), "SELECT pg_notify('service', $1)", host); err != nil {
		logg.Error(err.Error())
	}
}

func (c *Controller) notifyEndpoint(host string, id strfmt.UUID) {
	payload := fmt.Sprintf("%s:%s", host, id)
	if _, err := c.pool.Exec(context.Background(), "SELECT pg_notify('endpoint', $1)", payload); err != nil {
		logg.Error(err.Error())
	}
}
