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
	"github.com/go-openapi/loads"

	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/internal/neutron"
)

type Controller struct {
	spec    *loads.Document
	pool    db.PgxIface
	neutron *neutron.NeutronClient
}

func NewController(pool db.PgxIface, spec *loads.Document, client *neutron.NeutronClient) *Controller {
	return &Controller{pool: pool, spec: spec, neutron: client}
}
