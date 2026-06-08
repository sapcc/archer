// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package netlink

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
)

type Netlink interface {
	EnsureNetworkNamespace(ctx context.Context, port *ports.Port, client *gophercloud.ServiceClient) error
	EnableNetworkNamespace() error
	DisableNetworkNamespace() error
	DeleteNetworkNamespace() error
	Close() error
}
