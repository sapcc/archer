// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ni

import (
	"context"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/agent/ni/haproxy"
	"github.com/sapcc/archer/internal/agent/ni/models"
	"github.com/sapcc/archer/internal/agent/ni/netlink"

	"github.com/sapcc/archer/internal/config"
)

func (a *Agent) SetupOpenStack() error {
	authInfo := clientconfig.AuthInfo(config.Global.ServiceAuth)
	// Allow automatically reauthenticate
	authInfo.AllowReauth = true

	providerClient, err := clientconfig.AuthenticatedClient(context.Background(), &clientconfig.ClientOpts{
		AuthInfo: &authInfo})
	if err != nil {
		log.Fatal(err.Error())
	}

	var availability gophercloud.Availability
	switch config.Global.Default.EndpointType {
	case "public":
		availability = gophercloud.AvailabilityPublic
	case "internal":
		availability = gophercloud.AvailabilityInternal
	case "admin":
		availability = gophercloud.AvailabilityAdmin
	default:
		log.Fatalf("Invalid endpoint type: %s", config.Global.Default.EndpointType)
	}
	eo := gophercloud.EndpointOpts{Availability: availability}
	if a.neutron, err = openstack.NewNetworkV2(providerClient, eo); err != nil {
		return err
	}
	// Set timeout to 10 secs
	a.neutron.HTTPClient.Timeout = time.Second * 10
	return nil
}

func (a *Agent) EnableInjection(si *models.ServiceInjection) error {
	injectorPort, err := ports.Get(context.Background(), a.neutron, si.PortId.String()).Extract()
	if err != nil {
		return fmt.Errorf("failed to get port %s: %w", si.PortId, err)
	}

	// Create network namespace with ip/mac
	ns := netlink.NewNetworkNamespace()
	defer func() { _ = ns.Close() }()

	if err = ns.EnsureNetworkNamespace(injectorPort, a.neutron); err != nil {
		return fmt.Errorf("failed to ensure network namespace: %w", err)
	}

	if a.haproxy.IsRunning(injectorPort.NetworkID) {
		// Nothing to do
		return nil
	}

	// Run haproxy inside network namespace
	if err = ns.EnableNetworkNamespace(); err != nil {
		return fmt.Errorf("failed to enable network namespace: %w", err)
	}
	defer func() {
		if disableErr := ns.DisableNetworkNamespace(); disableErr != nil {
			log.Errorf("failed to disable network namespace: %v", disableErr)
		}
	}()

	if err = a.haproxy.AddInstance(si); err != nil {
		log.Errorf("Error enabling haproxy: %s, dumping conf/log", err)
		haproxy.Dump(haproxy.GetLogFilePath(si.Network.String()))
		haproxy.Dump(haproxy.GetConfigFilePath(si.Network.String()))
		haproxy.TryRemoveFile(haproxy.GetConfigFilePath(si.Network.String()))
		haproxy.TryRemoveFile(haproxy.GetLogFilePath(si.Network.String()))
		return fmt.Errorf("failed to add haproxy instance: %w", err)
	}

	return nil
}

func (a *Agent) DisableInjection(si *models.ServiceInjection) error {
	// Use the Network field directly instead of fetching from Neutron.
	// This allows cleanup to proceed even if the port was manually deleted.
	networkID := si.Network.String()

	// Only stop haproxy - don't delete the namespace.
	// Other endpoints may still be using this network's namespace.
	// The namespace will be cleaned up when the Neutron port is deleted,
	// or reused if another endpoint is created for this network.
	if a.haproxy.IsRunning(networkID) {
		if err := a.haproxy.RemoveInstance(networkID); err != nil {
			return fmt.Errorf("failed to remove haproxy instance: %w", err)
		}
	}
	return nil
}

func (a *Agent) CollectStats() {
	a.haproxy.CollectStats()
}
