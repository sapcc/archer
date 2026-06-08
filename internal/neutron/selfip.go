// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package neutron

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"

	"github.com/sapcc/archer/v2/internal/config"
)

// SelfIPDeviceOwner is the device_owner used for default-mode per-F5-device SelfIP/SNAT ports.
const SelfIPDeviceOwner = "network:f5selfip"

// EnsureNeutronSelfIPs ensures one SelfIP port exists per deviceID on the given subnet.
// dryRun=true skips creation but still returns the matching subset. Existing extras are not
// deleted (legacy behavior — cleanup happens through CleanupSelfIPs).
func (n *NeutronClient) EnsureNeutronSelfIPs(ctx context.Context, deviceIDs []string, subnetID string, dryRun bool) (map[string]*ports.Port, error) {
	desired := make(map[string]ports.Port, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		desired[deviceID] = ports.Port{
			Name:        fmt.Sprintf("local-%s", deviceID),
			Description: fmt.Sprintf("Archer SelfIP for device %s", deviceID),
			DeviceOwner: SelfIPDeviceOwner,
			DeviceID:    subnetID,
		}
	}
	return n.ensurePorts(ctx, ensurePortsOpts{
		SubnetID:       subnetID,
		Desired:        desired,
		HostID:         config.Global.Default.Host,
		DeviceIDFilter: subnetID,
		DryRun:         dryRun,
	})
}

// FetchSelfIPPorts lists every SelfIP port bound to this agent's host, grouped by network ID.
func (n *NeutronClient) FetchSelfIPPorts(ctx context.Context) (map[string][]*ports.Port, error) {
	got, err := n.ListPorts(ctx,
		ports.ListOpts{DeviceOwner: SelfIPDeviceOwner},
		config.Global.Default.Host,
	)
	if err != nil {
		return nil, err
	}
	portMap := make(map[string][]*ports.Port)
	for i := range got {
		p := got[i].Port
		portMap[p.NetworkID] = append(portMap[p.NetworkID], &p)
	}
	return portMap, nil
}
