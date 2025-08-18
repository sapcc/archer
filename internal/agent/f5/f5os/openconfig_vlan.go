// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package f5os

type member struct {
	State struct {
		Interface string `json:"interface"`
	} `json:"state"`
}

type vlan struct {
	VlanId int `json:"vlan-id"`
	Config struct {
		VlanId int    `json:"vlan-id"`
		Name   string `json:"name"`
	} `json:"config"`
	Members struct {
		Member []member `json:"member,omitempty"`
	} `json:"members"`
}

type trunkVlans struct {
	OpenconfigVlanTrunkVlans []int `json:"openconfig-vlan:trunk-vlans"`
}

type OpenConfigVlan struct {
	OpenconfigVlanVlan []vlan `json:"openconfig-vlan:vlan"`
}
type OpenConfigVlans struct {
	OpenconfigVlanVlans struct {
		Vlan []vlan `json:"vlan"`
	} `json:"openconfig-vlan:vlans"`
}
