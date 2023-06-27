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

package as3

import (
	"strings"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/sapcc/archer/internal/neutron"
	"github.com/sapcc/archer/models"
)

// ExtendedService is a service with additional fields for snatpool ports etc.
type ExtendedService struct {
	models.Service
	SnatPorts   map[string]*ports.Port
	TXAllocated bool
	SegmentId   int
}

func (es *ExtendedService) ProcessVCMP(vcmp *BigIP) error {
	return nil
}

func (es *ExtendedService) GetSNATPort(device string) *ports.Port {
	for _, port := range es.SnatPorts {
		if strings.HasSuffix(port.Name, device) {
			return port
		}
	}
	return nil
}

func (es *ExtendedService) EnsureSNATPort(bigip *BigIP, client *neutron.NeutronClient) error {
	if err := bigip.EnsureVLAN(es.SegmentId); err != nil {
		return err
	}
	if err := bigip.EnsureRouteDomain(es.SegmentId, nil); err != nil {
		return err
	}
	return bigip.EnsureSelfIP(client, es)
}
