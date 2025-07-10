// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"net/url"

	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/agent/f5/bigip"
	"github.com/sapcc/archer/internal/agent/f5/f5os"
)

// F5Device defines the interface for interacting with F5 appliances.
type F5Device interface {
	// PostAS3 posts AS3 configuration to the device.
	PostAS3(as3 *as3.AS3, tenant string) error

	// GetDeviceType returns the type of the F5 device (e.g., "bigip", "f5os").
	GetDeviceType() string

	// GetHostname returns the hostname of the F5 device.
	GetHostname() string

	// GetFailoverState returns the failover state of the device.
	// It should return a string indicating the state, such as "active", "standby", or "unknown".
	GetFailoverState() string

	// GetPartitions returns a list of partitions on the F5 device.
	GetPartitions() ([]string, error)

	// GetVLANs returns a list of VLANs on the F5 device.
	GetVLANs() ([]string, error)

	// GetRouteDomains returns a list of route domains on the F5 device.
	GetRouteDomains() ([]string, error)

	// GetSelfIPs returns a list of self IPs on the F5 device.
	GetSelfIPs() ([]string, error)

	// EnsureVLAN ensures that a VLAN with the given segment ID and MTU exists on the device.
	EnsureVLAN(segmentId int, mtu int) error

	// EnsureInterfaceVlan ensures that the interface VLAN for the given segment ID exists on the device.
	EnsureInterfaceVlan(segmentId int) error

	// EnsureGuestVlan ensures that the guest VLAN for the given segment ID exists on the device.
	EnsureGuestVlan(segmentId int) error

	// EnsureRouteDomain ensures that a route domain with the given segment ID and
	// optional parent segment ID exists on the device.
	EnsureRouteDomain(segmentId int, parentSegmentID *int) error

	// EnsureBigIPSelfIP ensures that a self IP with the given name, address and segment ID exists on the device.
	EnsureBigIPSelfIP(name, address string, segmentId int) error

	// SyncGuestVLANs syncs the guest VLANs based on the provided used segments.
	SyncGuestVLANs(usedSegments map[int]string) error

	// DeleteVLAN deletes a VLAN with the given segment id from the F5 device.
	DeleteVLAN(segmentId int) error

	// DeleteSelfIP cleans up the self IP with the given name on the device.
	DeleteSelfIP(name string) error

	// DeleteGuestVLAN cleans up the guest VLAN for the given segment ID.
	DeleteGuestVLAN(segmentId int) error

	// DeleteRouteDomain cleans up the route domain for the given segment ID.
	DeleteRouteDomain(segmentId int) error
}

func GetF5DeviceSession(rawURL string) (F5Device, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	// Try initializing a BigIP session first
	b, err := bigip.NewSession(parsedURL)
	if err == nil {
		return b, nil
	}

	// If that fails, try initializing an F5OS session
	return f5os.NewSession(parsedURL)
}
