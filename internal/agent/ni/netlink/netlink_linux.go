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
	name     string
	newns    netns.NsHandle
	origin   netns.NsHandle
	isLocked bool
}

func NewLinuxNetworkNamespace() *LinuxNetworkNamespace {
	return &LinuxNetworkNamespace{
		newns:    -1,
		origin:   -1,
		isLocked: false,
	}
}

// lockThread locks the OS thread and updates the isLocked flag
func (ns *LinuxNetworkNamespace) lockThread() {
	runtime.LockOSThread()
	ns.isLocked = true
}

// unlockThread unlocks the OS thread and updates the isLocked flag
func (ns *LinuxNetworkNamespace) unlockThread() {
	if ns.isLocked {
		runtime.UnlockOSThread()
		ns.isLocked = false
	}
}

// EnableNetworkNamespace switches the current thread to the network namespace
func (ns *LinuxNetworkNamespace) EnableNetworkNamespace() error {
	log.Debugf("enabling network namespace '%s'", ns.newns.String())
	if !ns.Valid() {
		return fmt.Errorf("network namespace is not valid")
	}
	if ns.origin != -1 {
		return fmt.Errorf("network namespace '%s' already enabled", ns.newns.String())
	}

	ns.lockThread()

	var err error
	ns.origin, err = netns.Get()
	if err != nil {
		ns.unlockThread()
		return fmt.Errorf("failed to get current namespace: %w", err)
	}

	// Enable
	if err = netns.Set(ns.newns); err != nil {
		ns.unlockThread()
		_ = ns.origin.Close()
		ns.origin = -1
		return fmt.Errorf("failed to switch to namespace: %w", err)
	}
	return nil
}

// DisableNetworkNamespace switches back to the original network namespace
func (ns *LinuxNetworkNamespace) DisableNetworkNamespace() error {
	log.Debugf("disabling network namespace '%s'", ns.newns.String())
	if ns.origin == -1 {
		return fmt.Errorf("network namespace '%s' not enabled", ns.newns.String())
	}
	if !ns.isLocked {
		return fmt.Errorf("thread not locked for namespace '%s'", ns.newns.String())
	}

	// Disable
	if err := netns.Set(ns.origin); err != nil {
		return fmt.Errorf("failed to restore original namespace: %w", err)
	}

	// Close the origin handle
	if err := ns.origin.Close(); err != nil {
		log.Warnf("failed to close origin namespace handle: %v", err)
	}
	ns.origin = -1

	// Unlock the thread
	ns.unlockThread()
	return nil
}

// Close closes the network namespace handle
func (ns *LinuxNetworkNamespace) Close() error {
	if ns.newns == -1 {
		return nil // already closed
	}
	err := ns.newns.Close()
	if err == nil {
		ns.newns = -1
	}
	return err
}

// EnsureNetworkNamespace ensures that a network namespace for the given port exists
func (ns *LinuxNetworkNamespace) EnsureNetworkNamespace(port *ports.Port, client *gophercloud.ServiceClient) error {
	name := fmt.Sprintf("qinjector-%s", port.NetworkID)

	// Check if namespace already exists and matches
	if existingNS, err := netns.GetFromName(name); err == nil {
		// Namespace exists
		if ns.Valid() && ns.newns != existingNS {
			// We already have a different namespace handle
			_ = existingNS.Close()
			return fmt.Errorf("namespace '%s' exists but conflicts with current handle", name)
		}

		// Use existing namespace
		ns.name = name
		if !ns.Valid() {
			ns.newns = existingNS
		} else if ns.newns != existingNS {
			_ = existingNS.Close()
		}

		// Validate veth pair exists
		handle, err := netlink.NewHandleAt(ns.newns)
		if err != nil {
			return fmt.Errorf("failed to get handle for namespace '%s': %w", name, err)
		}
		defer handle.Close()

		peerName := fmt.Sprintf("tap%s", port.ID[:11])
		if _, err := netlink.LinkByName(peerName); err != nil {
			log.Warnf("namespace '%s' exists but veth pair '%s' not found, will recreate", name, peerName)
			// Namespace exists but veth is missing, delete and recreate
			if err := ns.deleteNamespace(name); err != nil {
				return fmt.Errorf("failed to delete broken namespace: %w", err)
			}
			ns.newns = -1
			// Fall through to create new namespace
		} else {
			return nil
		}
	}

	// Create veth pair
	mac, err := net.ParseMAC(port.MACAddress)
	if err != nil {
		return fmt.Errorf("failed to parse MAC address: %w", err)
	}

	peerName := fmt.Sprintf("tap%s", port.ID[:11])
	veth := netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:         "veth0",
			HardwareAddr: mac,
		},
		// Magic name tap<port-id> is detected by linuxbridge agent
		PeerName: peerName,
	}

	if err = netlink.LinkAdd(&veth); err != nil {
		return fmt.Errorf("failed to create veth pair: %w", err)
	}

	// Create network namespace and associate handle
	newns, err := createNamespace(name)
	if err != nil {
		// Clean up veth pair
		if delErr := netlink.LinkDel(&veth); delErr != nil {
			log.Warnf("failed to clean up veth pair after namespace creation failure: %v", delErr)
		}
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	handle, err := netlink.NewHandleAt(newns)
	if err != nil {
		// Clean up namespace and veth
		_ = netns.DeleteNamed(name)
		_ = newns.Close()
		_ = netlink.LinkDel(&veth)
		return fmt.Errorf("failed to get handle for new namespace: %w", err)
	}
	defer handle.Close()

	// bring up loopback device
	link, err := handle.LinkByName("lo")
	if err != nil {
		ns.cleanupFailedNamespace(name, newns, &veth)
		return fmt.Errorf("failed to find loopback device: %w", err)
	}
	if err = handle.LinkSetUp(link); err != nil {
		ns.cleanupFailedNamespace(name, newns, &veth)
		return fmt.Errorf("failed to bring up loopback: %w", err)
	}

	// Put veth0 into network namespace and set ip addresses
	if err = netlink.LinkSetNsFd(&veth, int(newns)); err != nil {
		ns.cleanupFailedNamespace(name, newns, &veth)
		return fmt.Errorf("failed to move veth to namespace: %w", err)
	}

	for _, fixedIP := range port.FixedIPs {
		ip := net.ParseIP(fixedIP.IPAddress)
		if ip == nil {
			ns.cleanupFailedNamespace(name, newns, nil)
			return fmt.Errorf("failed parsing ip address '%s'", fixedIP.IPAddress)
		}

		subnet, err := subnets.Get(context.Background(), client, fixedIP.SubnetID).Extract()
		if err != nil {
			ns.cleanupFailedNamespace(name, newns, nil)
			return fmt.Errorf("failed to get subnet %s: %w", fixedIP.SubnetID, err)
		}

		prefix := subnet.CIDR[strings.Index(subnet.CIDR, "/"):]
		ipaddress := fmt.Sprintf("%s%s", ip.String(), prefix)
		addr, err := netlink.ParseAddr(ipaddress)
		if err != nil {
			ns.cleanupFailedNamespace(name, newns, nil)
			return fmt.Errorf("failed to parse address %s: %w", ipaddress, err)
		}

		if err = handle.AddrAdd(&veth, addr); err != nil {
			ns.cleanupFailedNamespace(name, newns, nil)
			return fmt.Errorf("failed to add address: %w", err)
		}
	}

	// set veth0 up
	if err := handle.LinkSetUp(&veth); err != nil {
		ns.cleanupFailedNamespace(name, newns, nil)
		return fmt.Errorf("failed to bring up veth: %w", err)
	}

	ns.name = name
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

	if !ns.Valid() {
		return fmt.Errorf("network namespace is not valid")
	}

	if ns.origin != -1 {
		return fmt.Errorf("cannot delete namespace while it is enabled")
	}

	return ns.deleteNamespace(ns.name)
}

// deleteNamespace is a helper to delete a namespace by name
func (ns *LinuxNetworkNamespace) deleteNamespace(name string) error {
	// Check if namespace exists
	existingNS, err := netns.GetFromName(name)
	if err != nil {
		// Namespace doesn't exist, nothing to do
		return nil
	}
	defer func() { _ = existingNS.Close() }()

	// Close our handle if it matches
	if ns.newns == existingNS {
		if err := ns.Close(); err != nil {
			log.Warnf("failed to close namespace handle: %v", err)
		}
	}

	// Delete the namespace
	if err := netns.DeleteNamed(name); err != nil {
		return fmt.Errorf("failed to delete namespace '%s': %w", name, err)
	}

	return nil
}

// cleanupFailedNamespace cleans up resources after a failed namespace setup
func (ns *LinuxNetworkNamespace) cleanupFailedNamespace(name string, handle netns.NsHandle, veth *netlink.Veth) {
	if err := netns.DeleteNamed(name); err != nil {
		log.Warnf("failed to delete namespace '%s' during cleanup: %v", name, err)
	}
	if err := handle.Close(); err != nil {
		log.Warnf("failed to close namespace handle during cleanup: %v", err)
	}
	if veth != nil {
		if err := netlink.LinkDel(veth); err != nil {
			log.Warnf("failed to delete veth pair during cleanup: %v", err)
		}
	}
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
