/*
 * Copyright (c) 2023. Lorem ipsum dolor sit amet, consectetur adipiscing elit.
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //    http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 *
 */

package agent

import (
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/sapcc/archer/models"
)

// ExtendedService is a service with additional fields for snatpool ports etc.
type ExtendedService struct {
	models.Service
	SnatPortId  *strfmt.UUID
	SnatPort    *ports.Port
	TXAllocated bool
	SegmentId   int
}

// ExtendedEndpoint is an endpoint with additional fields...
type ExtendedEndpoint struct {
	models.Endpoint
	Port          *ports.Port
	ServicePortNr int32
	SegmentId     int
}
