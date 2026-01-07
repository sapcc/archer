// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package netlink

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type LinuxNetworkNamespace struct {
	name   string
	newns  netns.NsHandle
	origin netns.NsHandle
}

func NewLinuxNetworkNamespace() *LinuxNetworkNamespace {
	return &LinuxNetworkNamespace{
		origin: -1,
	}
}

// EnableNetworkNamespace switches the current thread to the network namespace
func (ns *LinuxNetworkNamespace) EnableNetworkNamespace() error {
	log.Debugf("enabling network namespace '%s'", ns.newns.String())
	if ns.origin != -1 {
		return fmt.Errorf("network namespace '%s' already enabled", ns.newns.String())
	}
	runtime.LockOSThread()

	var err error
	ns.origin, err = netns.Get()
	if err != nil {
		return err
	}

	// Enable
	if err = netns.Set(ns.newns); err != nil {
		return err
	}
	return nil
}

// DisableNetworkNamespace switches back to the original network namespace
func (ns *LinuxNetworkNamespace) DisableNetworkNamespace() error {
	log.Debugf("disabling network namespace '%s'", ns.newns.String())
	if ns.origin == -1 {
		return fmt.Errorf("network namespace '%s' not enabled", ns.newns.String())
	}

	// Disable
	if err := netns.Set(ns.origin); err != nil {
		return err
	}
	ns.origin = -1
	return nil
}

// Close closes the network namespace handle
func (ns *LinuxNetworkNamespace) Close() error {
	return ns.newns.Close()
}

// EnsureNetworkNamespace ensures that a network namespace for the given port exists
func (ns *LinuxNetworkNamespace) EnsureNetworkNamespace(port *ports.Port, client *gophercloud.ServiceClient) error {
	name := fmt.Sprintf("qinjector-%s", port.NetworkID)
	// namespace already exists?
	if existingNS, err := netns.GetFromName(name); err == nil {
		if ns.Valid() && existingNS != ns.newns {
			return fmt.Errorf("existing Namespace (%d) associated to other Namespace (%d)", existingNS, ns.newns)
		}
		ns.name = name
		ns.newns = existingNS
		return nil
	}

	// TODO: check if namespace exists but veth pair is not valid

	// create veth pair
	mac, err := net.ParseMAC(port.MACAddress)
	if err != nil {
		return err
	}
	veth := netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:         "veth0",
			HardwareAddr: mac,
		},
		// Magic name tap<port-id> is detected by linuxbridge agent
		PeerName: fmt.Sprintf("tap%s", port.ID[:11]),
	}
	if err = netlink.LinkAdd(&veth); err != nil {
		return err
	}

	// create network namespace and associate handle
	newns, err := createNamespace(name)
	if err != nil {
		return err
	}
	handle, err := netlink.NewHandleAt(newns)
	if err != nil {
		return err
	}

	// bring up loopback device
	link, err := handle.LinkByName("lo")
	if err != nil {
		return err
	}
	if err = handle.LinkSetUp(link); err != nil {
		return err
	}

	// Put veth0 into network namespace and set ip addresses
	if err = netlink.LinkSetNsFd(&veth, int(newns)); err != nil {
		return err
	}
	for _, fixedIP := range port.FixedIPs {
		ip := net.ParseIP(fixedIP.IPAddress)
		if ip == nil {
			return fmt.Errorf("failed parsing ip address '%s'", fixedIP.IPAddress)
		}

		subnet, err := subnets.Get(context.Background(), client, fixedIP.SubnetID).Extract()
		if err != nil {
			return err
		}

		prefix := subnet.CIDR[strings.Index(subnet.CIDR, "/"):]
		ipaddress := fmt.Sprintf("%s%s", ip.String(), prefix)
		addr, err := netlink.ParseAddr(ipaddress)
		if err != nil {
			return err
		}

		if err = handle.AddrAdd(&veth, addr); err != nil {
			return err
		}
	}

	// set veth0 up
	if err := handle.LinkSetUp(&veth); err != nil {
		return err
	}

	ns.newns = newns
	return nil
}

func createNamespace(name string) (netns.NsHandle, error) {
	log.Debugf("creating network namespace '%s'", name)

	// Lock the OS Thread, so we don't accidentally switch namespaces
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Save the current network namespace
	origns, err := netns.Get()
	if err != nil {
		return -1, err
	}
	defer func() { _ = origns.Close() }()

	// Create a new network namespace
	newns, err := netns.NewNamed(name)
	if err != nil {
		return -1, err
	}

	// Switch back to the original namespace
	if err = netns.Set(origns); err != nil {
		return -1, err
	}

	return newns, nil
}

// DeleteNetworkNamespace deletes the network namespace for the given network ID
func (ns *LinuxNetworkNamespace) DeleteNetworkNamespace() error {
	log.Debugf("deleting network namespace '%s'", ns.name)

	if _, err := netns.GetFromName(ns.name); err != nil {
		return fmt.Errorf("network namespace '%s' not found", ns.name)
	}

	if !ns.Valid() {
		return fmt.Errorf("network namespace '%s' is not valid", ns.name)
	}

	if err := ns.Close(); err != nil {
		return err
	}

	if err := netns.DeleteNamed(ns.name); err != nil {
		return err
	}

	return nil
}

func (ns *LinuxNetworkNamespace) Valid() bool {
	return ns.newns != -1
}

func NewNetworkNamespace() Netlink {
	if testing.Testing() {
		return NewFakeNetlink()
	}
	return NewLinuxNetworkNamespace()
}
