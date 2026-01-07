// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package netlink

import (
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	log "github.com/sirupsen/logrus"
)

type FakeNetlink struct {
	name string
}

func NewFakeNetlink() *FakeNetlink {
	return &FakeNetlink{}
}

func (ns *FakeNetlink) EnableNetworkNamespace() error {
	log.Infof("FakeNetlink: enabling network namespace: %s", ns.name)
	return nil
}

func (ns *FakeNetlink) DisableNetworkNamespace() error {
	log.Infof("FakeNetlink: disabling network namespace '%s'", ns.name)
	return nil
}

func (ns *FakeNetlink) Close() error {
	log.Infof("FakeNetlink: closing network namespace %s", ns.name)
	return nil
}

func (ns *FakeNetlink) EnsureNetworkNamespace(port *ports.Port, _ *gophercloud.ServiceClient) error {
	// Fake network namespace implementation for debugging
	ns.name = fmt.Sprintf("qinjector-%s", port.NetworkID)
	log.Infof("FakeNetlink: ensuring network namespace '%s'", ns.name)
	return nil
}

func (ns *FakeNetlink) DeleteNetworkNamespace() error {
	log.Infof("FakeNetlink: deleting network namespace '%s'", ns.name)
	return nil
}
