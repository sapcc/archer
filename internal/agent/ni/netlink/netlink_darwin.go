// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package netlink

func NewNetworkNamespace() Netlink {
	return NewFakeNetlink()
}
