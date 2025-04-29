// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package as3

import "github.com/f5devcentral/go-bigip"

//go:generate mockery --name BigIPIface
type BigIPIface interface {
	PostAs3Bigip(as3NewJson, tenantFilter, queryParam string) (error, string, string)
	GetDevices() ([]bigip.Device, error)
	APICall(options *bigip.APIRequest) ([]byte, error)
	SelfIP(selfip string) (*bigip.SelfIP, error)
	SelfIPs() (*bigip.SelfIPs, error)
	DeleteSelfIP(name string) error
	DeleteRouteDomain(name string) error
	CreateSelfIP(config *bigip.SelfIP) error
	UpdateVcmpGuest(name string, config *bigip.VcmpGuest) error
	Vlans() (*bigip.Vlans, error)
	CreateVlan(config *bigip.Vlan) error
	ModifyVlan(name string, config *bigip.Vlan) error
	GetVlanInterfaces(vlan string) (*bigip.VlanInterfaces, error)
	AddInterfaceToVlan(vlan, iface string, tagged bool) error
	DeleteVlan(name string) error
	TMPartitions() (*bigip.TMPartitions, error)
}
