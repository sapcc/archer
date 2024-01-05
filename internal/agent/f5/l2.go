// Copyright 2023 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package f5

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// --------------------------------------------------------------------------
// L2 (VLAN, Route Domain, Guest VLAN)
// --------------------------------------------------------------------------

// EnsureL2 ensures that L2 configuration exists on BIG-IP(s) and VCMP(s) for the given segmentID.
func (a *Agent) EnsureL2(ctx context.Context, segmentID int, parentSegmentID *int) error {
	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		for _, vcmp := range a.vcmps {
			if err := vcmp.EnsureVLAN(segmentID); err != nil {
				return fmt.Errorf("EnsureVLAN: %s", err.Error())
			}
			if err := vcmp.EnsureInterfaceVlan(segmentID); err != nil {
				return fmt.Errorf("EnsureInterfaceVlan: %s", err.Error())
			}
			if err := vcmp.EnsureGuestVlan(segmentID); err != nil {
				return fmt.Errorf("EnsureGuestVlan: %s", err.Error())
			}
		}
		return nil
	})

	// Guest configuration
	g.Go(func() error {
		// Ensure VLAN and Route Domain
		for _, bigip := range a.bigips {
			if err := bigip.EnsureVLAN(segmentID); err != nil {
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
	g, _ := errgroup.WithContext(ctx)
	// Cleanup VCMP
	g.Go(func() error {
		for _, vcmp := range a.vcmps {
			if err := vcmp.CleanupGuestVlan(segmentID); err != nil {
				return fmt.Errorf("CleanupGuestVlan: %s", err.Error())
			}
			if err := vcmp.CleanupVLAN(segmentID); err != nil {
				return fmt.Errorf("CleanupVLAN: %s", err.Error())
			}
		}
		return nil
	})

	// Cleanup Guest
	g.Go(func() error {
		for _, bigip := range a.bigips {
			if err := bigip.CleanupRouteDomain(segmentID); err != nil {
				return fmt.Errorf("CleanupRouteDomain: device=%s %s", bigip.GetHostname(), err.Error())
			}
			if err := bigip.CleanupVLAN(segmentID); err != nil {
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

// EnsureSelfIPs ensures that a SelfIPs exists on the BIG-IP(s) for the given endpoint port.
func (a *Agent) EnsureSelfIPs(segmentID int, subnetID string, dryRun bool) error {
	neutronPorts, err := a.neutron.EnsureNeutronSelfIPs(a.getDeviceIDs(), subnetID, dryRun)
	if err != nil {
		return err
	}

	for _, big := range a.bigips {
		big := big

		// Fetch netmask
		mask, err := a.neutron.GetMask(subnetID)
		if err != nil {
			return err
		}

		name := fmt.Sprint("selfip-", neutronPorts[big.GetHostname()].ID)
		ip := neutronPorts[big.GetHostname()].FixedIPs[0].IPAddress
		address := fmt.Sprint(ip, "%", segmentID, "/", mask)
		if err := big.EnsureBigIPSelfIP(name, address, segmentID); err != nil {
			return err
		}
	}
	return nil
}

func (a *Agent) CleanupSelfIPs(subnetID string) error {
	// don't create new neutron selfip ports, just return existing ones
	neutronPorts, err := a.neutron.EnsureNeutronSelfIPs(a.getDeviceIDs(), subnetID, true)
	if err != nil {
		return err
	}

	// delete from device
	for _, big := range a.bigips {
		big := big

		if port, ok := neutronPorts[big.GetHostname()]; ok {
			name := fmt.Sprint("selfip-", port.ID)
			if err := big.CleanupSelfIP(name); err != nil {
				if !strings.Contains(err.Error(), "was not found") {
					return err
				}
				log.WithField("name", name).Warning("BigIP SelfIP cleanup: selfip not found, skipping")
			}
		}
	}

	// finally delete from neutron
	for _, port := range neutronPorts {
		if err := a.neutron.DeletePort(port.ID); err != nil {
			var errDefault404 gophercloud.ErrDefault404
			if !errors.As(err, &errDefault404) {
				return err
			}
			log.WithField("id", port.ID).Warning("CleanupSelfIPs: neutron port not found, skipping")
		}
	}
	return nil
}

func (a *Agent) CleanupSNATPorts(networkID string) error {
	log.WithField("networkID", networkID).Debug("Running CleanupSNATPorts")
	// Fetch SNAT Ports
	ports, err := a.neutron.FetchSNATPorts(networkID)
	if err != nil {
		return err
	}

	// delete from device
	for _, big := range a.bigips {
		big := big

		if port, ok := ports[big.GetHostname()]; ok {
			name := fmt.Sprint("snat-", port.ID)
			if err := big.CleanupSelfIP(name); err != nil {
				if !strings.Contains(err.Error(), "was not found") {
					return err
				}
				log.WithField("name", name).Warning("CleanupSNATPorts: SelfIP not found, skipping")
			}
		}
	}

	// finally delete from neutron
	for _, port := range ports {
		if err := a.neutron.DeletePort(port.ID); err != nil {
			var errDefault404 gophercloud.ErrDefault404
			if !errors.As(err, &errDefault404) {
				return err
			}
			log.WithField("id", port.ID).Warning("CleanupSNATPorts: neutron port not found, skipping")
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
	for _, big := range a.bigips {
		deviceIDs = append(deviceIDs, big.GetHostname())
	}
	return deviceIDs
}
