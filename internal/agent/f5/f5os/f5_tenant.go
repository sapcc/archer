// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package f5os

type F5TenantVlans struct {
	F5TenantsVlans []int `json:"f5-tenants:vlans"`
}

type F5TenantsTenant struct {
	Name  string `json:"name"`
	State struct {
		Vlans []int `json:"vlans"`
	} `json:"state"`
}

type F5Tenants struct {
	F5TenantsTenant []F5TenantsTenant `json:"f5-tenants:tenant"`
}
