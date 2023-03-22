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
	"fmt"

	"golang.org/x/sync/errgroup"
)

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
				return fmt.Errorf("CleanupRouteDomain: %s", err.Error())
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
