// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/sapcc/archer/v2/internal/config"
)

// --------------------------------------------------------------------------
// L2 (VLAN, Route Domain, Guest VLAN)
// --------------------------------------------------------------------------

// EnsureL2 ensures that L2 configuration exists on BIG-IP Guest(s) and Host(s) for the given segmentID.
func (a *Agent) EnsureL2(ctx context.Context, segmentID int, parentSegmentID *int, mtu int) error {
	printSegmentID := "nil"
	if parentSegmentID != nil {
		printSegmentID = fmt.Sprint(*parentSegmentID)
	}
	log.WithFields(log.Fields{"segmentID": segmentID, "parentSegmentID": printSegmentID}).Debug("EnsureL2")

	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		for _, host := range a.hosts {
			if err := host.EnsureVLAN(segmentID, mtu); err != nil {
				return fmt.Errorf("EnsureVLAN: %s", err.Error())
			}
			if err := host.EnsureInterfaceVlan(segmentID); err != nil {
				return fmt.Errorf("EnsureInterfaceVlan: %s", err.Error())
			}
			if err := host.EnsureGuestVlan(segmentID); err != nil {
				return fmt.Errorf("EnsureGuestVlan: %s", err.Error())
			}
		}
		return nil
	})

	// Guest configuration
	g.Go(func() error {
		// Ensure VLAN and Route Domain
		for _, bigip := range a.devices {
			if err := bigip.EnsureVLAN(segmentID, mtu); err != nil {
				return fmt.Errorf("EnsureVLAN: %s", err.Error())
			}
			if err := bigip.EnsureRouteDomain(segmentID, parentSegmentID); err != nil {
				return fmt.Errorf("EnsureRouteDomain: %s", err.Error())
			}
		}
		return nil
	})

	// Wait for L2 configuration done
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

// CleanupL2 cleans up L2 configuration on BIG-IP(s) and VCMP(s) for the given segmentID.
func (a *Agent) CleanupL2(ctx context.Context, segmentID int) error {
	logger := log.WithField("segmentID", segmentID)
	logger.Debug("CleanupL2")

	g, _ := errgroup.WithContext(ctx)

	// Cleanup VCMP
	g.Go(func() error {
		for _, vcmp := range a.hosts {
			logger.WithField("vcmp", vcmp.GetHostname()).Debug("CleanupL2: cleaning up VCMP L2 configuration")
			if err := vcmp.DeleteGuestVLAN(segmentID); err != nil {
				return fmt.Errorf("CleanupGuestVlan: %s", err.Error())
			}
			if err := vcmp.DeleteVLAN(segmentID); err != nil {
				return fmt.Errorf("CleanupVLAN: %s", err.Error())
			}
		}
		return nil
	})

	// Cleanup Guest
	g.Go(func() error {
		for _, bigip := range a.devices {
			logger.WithField("bigip", bigip.GetHostname()).Debug("CleanupL2: cleaning up Guest L2 configuration")
			if err := bigip.DeleteRouteDomain(segmentID); err != nil {
				return fmt.Errorf("CleanupRouteDomain: device=%s %s", bigip.GetHostname(), err.Error())
			}
			if err := bigip.DeleteVLAN(segmentID); err != nil {
				return fmt.Errorf("CleanupVLAN: %s", err.Error())
			}
		}
		return nil
	})

	// Wait for L2 configuration done
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

// --------------------------------------------------------------------------
// SelfIPs
// --------------------------------------------------------------------------

// EnsureSelfIPs ensures that a SelfIPs exists on the BIG-IP(s) for a given subnet.
func (a *Agent) EnsureSelfIPs(ctx context.Context, subnetID string, dryRun bool) error {
	log.WithFields(log.Fields{"subnetID": subnetID, "dryRun": dryRun}).Debug("EnsureSelfIPs")

	neutronPorts, err := a.neutron.EnsureNeutronSelfIPs(ctx, a.getDeviceIDs(), subnetID, dryRun)
	if err != nil {
		return err
	}

	segmentID, err := a.neutron.GetSubnetSegment(ctx, subnetID, config.Global.Agent.PhysicalNetwork)
	if err != nil {
		return err
	}

	mask, err := a.neutron.GetMask(ctx, subnetID)
	if err != nil {
		return err
	}

	for _, big := range a.devices {
		port, ok := neutronPorts[big.GetHostname()]
		if !ok || port == nil {
			// In dryRun mode EnsureNeutronSelfIPs does not create missing ports, so a device
			// may legitimately be absent from the map on the first pass. Skip it — the next
			// non-dryRun invocation (from the endpoint path) will materialize the port.
			log.WithFields(log.Fields{"device": big.GetHostname(), "subnetID": subnetID, "dryRun": dryRun}).
				Debug("EnsureSelfIPs: no neutron port for device, skipping")
			continue
		}
		if len(port.FixedIPs) == 0 {
			return fmt.Errorf("EnsureSelfIPs: no fixedIPs on neutron port %s for device %s", port.ID, big.GetHostname())
		}

		name := fmt.Sprint("selfip-", port.ID)
		ip := port.FixedIPs[0].IPAddress
		address := fmt.Sprint(ip, "%", segmentID, "/", mask)
		if err := big.EnsureBigIPSelfIP(name, address, segmentID); err != nil {
			return err
		}
	}
	return nil
}

func (a *Agent) CleanupSelfIPs(ctx context.Context, subnetID string) error {
	log.WithField("subnetID", subnetID).Debug("Running CleanupSelfIPs")

	// don't create new neutron selfip ports, just return existing ones
	neutronPorts, err := a.neutron.EnsureNeutronSelfIPs(ctx, a.getDeviceIDs(), subnetID, true)
	if err != nil {
		return err
	}

	// delete from device
	for _, big := range a.devices {
		if port, ok := neutronPorts[big.GetHostname()]; ok {
			name := fmt.Sprint("selfip-", port.ID)
			logger := log.WithFields(log.Fields{"name": name, "device": big.GetHostname()})
			logger.Debug("CleanupSelfIPs: deleting SelfIP on device")
			if err = big.DeleteSelfIP(name); err != nil {
				if !strings.Contains(err.Error(), "was not found") {
					return err
				}
				logger.Warning("BigIP SelfIP cleanup: selfip not found, skipping")
			}
		}
	}

	// finally delete from neutron
	for _, port := range neutronPorts {
		log.WithField("id", port.ID).Debug("CleanupSelfIPs: deleting neutron port")
		if err := a.neutron.DeletePort(ctx, port.ID); err != nil {
			if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
				return err
			}
			log.WithField("id", port.ID).Warning("CleanupSelfIPs: neutron port not found, skipping")
		}
	}
	return nil
}

// --------------------------------------------------------------------------
// Support functions
// --------------------------------------------------------------------------

// getDeviceIDs returns a list of device IDs for all BIG-IPs
func (a *Agent) getDeviceIDs() []string {
	var deviceIDs []string
	for _, big := range a.devices {
		deviceIDs = append(deviceIDs, big.GetHostname())
	}
	return deviceIDs
}
