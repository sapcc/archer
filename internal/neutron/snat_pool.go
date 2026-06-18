// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package neutron

import (
	"context"
	"fmt"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"

	"github.com/sapcc/archer/v2/internal/config"
)

// SnatPortDeviceOwner is the device_owner used for service-scoped SNAT pool ports. Distinct from
// SelfIPDeviceOwner so listing/cleanup never crosses the two.
const SnatPortDeviceOwner = "network:f5snat"

// EnsureServiceSnatPorts ensures `count` service-scoped SNAT ports exist on subnetID for the
// given service. Existing ports are matched by name (scoped via device_id) regardless of their
// current binding:host_id, so a service migrated to this agent has its ports rebound rather
// than duplicated. Stale ports outside the desired set are deleted. Returns ports keyed by
// "snat-<index>". dryRun=true returns the matching subset without mutating Neutron.
func (n *NeutronClient) EnsureServiceSnatPorts(
	ctx context.Context,
	serviceID strfmt.UUID,
	subnetID string,
	count int,
	dryRun bool,
) (map[string]*ports.Port, error) {
	desired := make(map[string]ports.Port, count)
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("snat-%d", i)
		desired[key] = ports.Port{
			Name:        fmt.Sprintf("snat-%s-%d", serviceID.String(), i),
			Description: fmt.Sprintf("Archer SNAT port for service %s", serviceID),
			DeviceOwner: SnatPortDeviceOwner,
			DeviceID:    serviceID.String(),
		}
	}
	return n.ensurePorts(ctx, ensurePortsOpts{
		SubnetID:             subnetID,
		Desired:              desired,
		HostID:               config.Global.Default.Host,
		DeviceIDFilter:       serviceID.String(),
		RebindOnHostMismatch: true,
		DeleteExtras:         true,
		DryRun:               dryRun,
	})
}

// CleanupServiceSnatPorts deletes every SNAT port owned by the given service. Idempotent.
// Used by the agent's PENDING_DELETE cleanup branch.
func (n *NeutronClient) CleanupServiceSnatPorts(ctx context.Context, serviceID strfmt.UUID) error {
	owned, err := n.ListPorts(ctx, ports.ListOpts{
		DeviceOwner: SnatPortDeviceOwner,
		DeviceID:    serviceID.String(),
	}, "")
	if err != nil {
		return err
	}
	for _, p := range owned {
		if err := n.DeletePort(ctx, p.ID); err != nil {
			return err
		}
	}
	return nil
}

// FetchSnatPorts lists every SNAT pool port bound to this agent's host. Returns
// a flat slice (not grouped) because the orphan signal for SNAT ports is
// per-service (device_id) rather than per-segment.
func (n *NeutronClient) FetchSnatPorts(ctx context.Context) ([]*ports.Port, error) {
	got, err := n.ListPorts(ctx,
		ports.ListOpts{DeviceOwner: SnatPortDeviceOwner},
		config.Global.Default.Host,
	)
	if err != nil {
		return nil, err
	}
	out := make([]*ports.Port, 0, len(got))
	for i := range got {
		p := got[i].Port
		out = append(out, &p)
	}
	return out, nil
}
