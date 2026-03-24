// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package netlink

import (
	"fmt"
	"time"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// TestNamespaceCreateDelete tests creating and deleting a network namespace.
func (s *NetlinkSuite) TestNamespaceCreateDelete() {
	name := fmt.Sprintf("test-ns-%d", time.Now().UnixNano())

	// Create namespace
	newns, err := createNamespace(name)
	s.Require().NoError(err, "Failed to create namespace")
	s.Require().NotEqual(netns.NsHandle(-1), newns, "Invalid namespace handle")

	// Verify namespace exists by getting another handle
	existingNS, err := netns.GetFromName(name)
	s.Require().NoError(err, "Namespace should exist after creation")
	// Note: handles are different file descriptors but reference same namespace
	s.NotEqual(netns.NsHandle(-1), existingNS, "Should get valid handle")
	_ = existingNS.Close()

	// Delete namespace
	err = netns.DeleteNamed(name)
	s.Require().NoError(err, "Failed to delete namespace")

	// Verify namespace is deleted
	_, err = netns.GetFromName(name)
	s.Error(err, "Namespace should not exist after deletion")

	_ = newns.Close()
}

// TestEnableDisableCycle tests the enable/disable cycle of a network namespace.
func (s *NetlinkSuite) TestEnableDisableCycle() {
	name := fmt.Sprintf("test-ns-%d", time.Now().UnixNano())
	defer func() { _ = netns.DeleteNamed(name) }()

	newns, err := createNamespace(name)
	s.Require().NoError(err, "Failed to create namespace")

	ns := &LinuxNetworkNamespace{
		name:     name,
		newns:    newns,
		origin:   -1,
		isLocked: false,
	}

	// Test Enable
	err = ns.EnableNetworkNamespace()
	s.Require().NoError(err, "EnableNetworkNamespace should succeed")
	s.True(ns.isLocked, "Thread should be locked after Enable")
	s.NotEqual(netns.NsHandle(-1), ns.origin, "Origin namespace should be set")

	// Test Disable
	err = ns.DisableNetworkNamespace()
	s.Require().NoError(err, "DisableNetworkNamespace should succeed")
	s.False(ns.isLocked, "Thread should be unlocked after Disable")
	s.Equal(netns.NsHandle(-1), ns.origin, "Origin namespace should be reset to -1")
}

// TestVethPairCreation tests creating a veth pair and moving one end to a namespace.
func (s *NetlinkSuite) TestVethPairCreation() {
	name := fmt.Sprintf("test-ns-%d", time.Now().UnixNano())
	// Create a unique peer name (max 15 chars for interface names)
	peerName := fmt.Sprintf("tap%d", time.Now().UnixNano()%100000000)

	// Create veth pair
	veth := netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: "veth0",
		},
		PeerName: peerName,
	}

	err := netlink.LinkAdd(&veth)
	s.Require().NoError(err, "Failed to create veth pair")
	defer func() { _ = netlink.LinkDel(&veth) }()

	// Verify peer exists in host namespace
	link, err := netlink.LinkByName(peerName)
	s.Require().NoError(err, "Peer should exist in host namespace")
	s.NotNil(link)

	// Create namespace and move veth0 into it
	newns, err := createNamespace(name)
	s.Require().NoError(err, "Failed to create namespace")
	defer func() {
		_ = netns.DeleteNamed(name)
		_ = newns.Close()
	}()

	err = netlink.LinkSetNsFd(&veth, int(newns))
	s.Require().NoError(err, "Failed to move veth to namespace")

	// Verify veth0 is now in the namespace
	handle, err := netlink.NewHandleAt(newns)
	s.Require().NoError(err, "Failed to get handle for namespace")
	defer handle.Close()

	link, err = handle.LinkByName("veth0")
	s.Require().NoError(err, "veth0 should exist in target namespace")
	s.NotNil(link)

	// Verify veth0 is NOT in host namespace anymore
	_, err = netlink.LinkByName("veth0")
	s.Error(err, "veth0 should not exist in host namespace after move")
}

// TestCleanupOnFailure tests that cleanupFailedNamespace properly cleans up resources.
func (s *NetlinkSuite) TestCleanupOnFailure() {
	name := fmt.Sprintf("test-ns-%d", time.Now().UnixNano())

	// Create namespace
	newns, err := createNamespace(name)
	s.Require().NoError(err, "Failed to create namespace")

	// Create veth pair
	veth := netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{Name: "veth0"},
		PeerName:  "tapcleanup",
	}
	err = netlink.LinkAdd(&veth)
	s.Require().NoError(err, "Failed to create veth pair")

	// Call cleanup helper
	ns := NewLinuxNetworkNamespace()
	ns.cleanupFailedNamespace(name, newns, &veth)

	// Verify namespace is deleted
	_, err = netns.GetFromName(name)
	s.Error(err, "Namespace should be deleted after cleanup")

	// Verify veth is deleted
	_, err = netlink.LinkByName("tapcleanup")
	s.Error(err, "Veth peer should be deleted after cleanup")
}

// TestDoubleEnableError tests that enabling an already-enabled namespace returns an error.
func (s *NetlinkSuite) TestDoubleEnableError() {
	name := fmt.Sprintf("test-ns-%d", time.Now().UnixNano())
	defer func() { _ = netns.DeleteNamed(name) }()

	newns, err := createNamespace(name)
	s.Require().NoError(err, "Failed to create namespace")

	ns := &LinuxNetworkNamespace{
		name:   name,
		newns:  newns,
		origin: -1,
	}

	// First enable should succeed
	err = ns.EnableNetworkNamespace()
	s.Require().NoError(err, "First EnableNetworkNamespace should succeed")

	// Second enable should fail
	err = ns.EnableNetworkNamespace()
	s.Require().Error(err, "Second EnableNetworkNamespace should fail")
	s.Contains(err.Error(), "already enabled")

	// Cleanup
	_ = ns.DisableNetworkNamespace()
}

// TestDisableWithoutEnable tests that disabling a namespace that was never enabled returns an error.
func (s *NetlinkSuite) TestDisableWithoutEnable() {
	name := fmt.Sprintf("test-ns-%d", time.Now().UnixNano())
	defer func() { _ = netns.DeleteNamed(name) }()

	newns, err := createNamespace(name)
	s.Require().NoError(err, "Failed to create namespace")

	ns := &LinuxNetworkNamespace{
		name:   name,
		newns:  newns,
		origin: -1, // Not enabled
	}

	// Disable without enable should fail
	err = ns.DisableNetworkNamespace()
	s.Require().Error(err, "DisableNetworkNamespace should fail when not enabled")
	s.Contains(err.Error(), "not enabled")
}

// TestInvalidNamespaceOperations tests that operations on invalid namespace handles return errors.
func (s *NetlinkSuite) TestInvalidNamespaceOperations() {
	ns := NewLinuxNetworkNamespace()

	// Enable on invalid namespace should fail
	err := ns.EnableNetworkNamespace()
	s.Require().Error(err, "EnableNetworkNamespace should fail on invalid namespace")
	s.Contains(err.Error(), "not valid")

	// Delete on invalid namespace should fail
	err = ns.DeleteNetworkNamespace()
	s.Require().Error(err, "DeleteNetworkNamespace should fail on invalid namespace")
	s.Contains(err.Error(), "not valid")

	// Valid() should return false
	s.False(ns.Valid(), "Invalid namespace should return false from Valid()")
}

// TestCloseIdempotent tests that closing an already-closed namespace is safe.
func (s *NetlinkSuite) TestCloseIdempotent() {
	name := fmt.Sprintf("test-ns-%d", time.Now().UnixNano())
	defer func() { _ = netns.DeleteNamed(name) }()

	newns, err := createNamespace(name)
	s.Require().NoError(err, "Failed to create namespace")

	ns := &LinuxNetworkNamespace{
		name:  name,
		newns: newns,
	}

	// First close should succeed
	err = ns.Close()
	s.Require().NoError(err, "First Close should succeed")
	s.Equal(netns.NsHandle(-1), ns.newns, "Handle should be -1 after close")

	// Second close should also succeed (idempotent)
	err = ns.Close()
	s.Require().NoError(err, "Second Close should succeed (idempotent)")
}

// TestDeleteWhileEnabled tests that deleting a namespace while enabled returns an error.
func (s *NetlinkSuite) TestDeleteWhileEnabled() {
	name := fmt.Sprintf("test-ns-%d", time.Now().UnixNano())
	defer func() { _ = netns.DeleteNamed(name) }()

	newns, err := createNamespace(name)
	s.Require().NoError(err, "Failed to create namespace")

	ns := &LinuxNetworkNamespace{
		name:   name,
		newns:  newns,
		origin: -1,
	}

	// Enable the namespace
	err = ns.EnableNetworkNamespace()
	s.Require().NoError(err, "EnableNetworkNamespace should succeed")

	// Try to delete while enabled
	err = ns.DeleteNetworkNamespace()
	s.Require().Error(err, "DeleteNetworkNamespace should fail while enabled")
	s.Contains(err.Error(), "while it is enabled")

	// Cleanup
	_ = ns.DisableNetworkNamespace()
}
