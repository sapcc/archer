// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package ni

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type NetworkNamespace struct {
	newns   netns.NsHandle
	origin  netns.NsHandle
	enabled bool
}

func (ns *NetworkNamespace) EnableNetworkNamespace() error {
	log.Debugf("enabling network namespace '%s'", ns.newns.String())
	if ns.enabled {
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
	ns.enabled = true
	return nil
}

func (ns *NetworkNamespace) DisableNetworkNamespace() error {
	log.Debugf("disabling network namespace '%s'", ns.newns.String())
	if !ns.enabled {
		return fmt.Errorf("network namespace '%s' not enabled", ns.newns.String())
	}

	// Disable
	if err := netns.Set(ns.origin); err != nil {
		return err
	}
	ns.enabled = false
	return nil
}

func (ns *NetworkNamespace) Close() error {
	return ns.newns.Close()
}

func EnsureNetworkNamespace(port *ports.Port, client *gophercloud.ServiceClient) (*NetworkNamespace, error) {
	name := fmt.Sprintf("qinjector-%s", port.NetworkID)
	// namespace already exists?
	if ns, err := netns.GetFromName(name); err == nil {
		return &NetworkNamespace{ns, -1, false}, err
	}

	// TODO: check if namespace exists but veth pair is not valid

	// create veth pair
	mac, err := net.ParseMAC(port.MACAddress)
	if err != nil {
		return nil, err
	}
	veth := netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:         "veth0",
			HardwareAddr: mac,
		},
		// Magic name tap<port-id> is detected by linuxbridge agent
		PeerName: fmt.Sprintf("tap%s", port.ID[:11]),
	}
	if err := netlink.LinkAdd(&veth); err != nil {
		return nil, err
	}

	// create network namespace and associate handle
	newns, err := createNamespace(name)
	if err != nil {
		return nil, err
	}
	handle, err := netlink.NewHandleAt(newns)
	if err != nil {
		return nil, err
	}

	// bring up loopback device
	link, err := handle.LinkByName("lo")
	if err != nil {
		return nil, err
	}
	if err := handle.LinkSetUp(link); err != nil {
		return nil, err
	}

	// Put veth0 into network namespace and set ip addresses
	if err := netlink.LinkSetNsFd(&veth, int(newns)); err != nil {
		return nil, err
	}
	for _, fixedIP := range port.FixedIPs {
		ip := net.ParseIP(fixedIP.IPAddress)
		if ip == nil {
			return nil, fmt.Errorf("failed parsing ip address '%s'", fixedIP.IPAddress)
		}

		subnet, err := subnets.Get(context.Background(), client, fixedIP.SubnetID).Extract()
		if err != nil {
			return nil, err
		}

		prefix := subnet.CIDR[strings.Index(subnet.CIDR, "/"):]
		ipaddress := fmt.Sprintf("%s%s", ip.String(), prefix)
		addr, err := netlink.ParseAddr(ipaddress)
		if err != nil {
			return nil, err
		}

		if err := handle.AddrAdd(&veth, addr); err != nil {
			return nil, err
		}
	}

	// set veth0 up
	if err := handle.LinkSetUp(&veth); err != nil {
		return nil, err
	}

	return &NetworkNamespace{newns, -1, false}, err
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

func DeleteNetworkNamespace(networkID string) error {
	name := fmt.Sprintf("qinjector-%s", networkID)
	log.Debugf("deleting network namespace '%s'", name)

	// namespace already exists?
	ns, err := netns.GetFromName(name)
	if err != nil {
		return err
	}

	if err := ns.Close(); err != nil {
		return err
	}

	if err := netns.DeleteNamed(name); err != nil {
		return err
	}

	return nil
}
