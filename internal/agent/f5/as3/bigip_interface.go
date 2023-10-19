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

import "github.com/f5devcentral/go-bigip"

//go:generate mockery --name BigIPIface
type BigIPIface interface {
	PostAs3Bigip(as3NewJson string, tenantFilter string) (error, string, string)
	GetDevices() ([]bigip.Device, error)
	APICall(options *bigip.APIRequest) ([]byte, error)
	SelfIPs() (*bigip.SelfIPs, error)
	DeleteSelfIP(name string) error
	DeleteRouteDomain(name string) error
	CreateSelfIP(config *bigip.SelfIP) error
	UpdateVcmpGuest(name string, config *bigip.VcmpGuest) error
	Vlans() (*bigip.Vlans, error)
	CreateVlan(config *bigip.Vlan) error
	GetVlanInterfaces(vlan string) (*bigip.VlanInterfaces, error)
	AddInterfaceToVlan(vlan, iface string, tagged bool) error
	DeleteVlan(name string) error
}
