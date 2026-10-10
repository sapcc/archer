// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ni

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/agent/ni/haproxy"
	"github.com/sapcc/archer/v2/internal/agent/ni/models"
	"github.com/sapcc/archer/v2/internal/agent/ni/netlink"
	"github.com/sapcc/archer/v2/internal/agent/ni/proxy"
	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/internal/neutron"
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
	serviceClient, err := openstack.NewNetworkV2(providerClient, eo)
	if err != nil {
		return err
	}
	// Set timeout to 10 secs
	serviceClient.HTTPClient.Timeout = time.Second * 10
	a.neutron = &neutron.NeutronClient{ServiceClient: serviceClient}
	a.neutron.InitCache()
	return nil
}

func (a *Agent) EnableInjection(ctx context.Context, si *models.ServiceInjection) error {
	injectorPort, err := ports.Get(ctx, a.neutron.ServiceClient, si.PortId.String()).Extract()
	if err != nil {
		return fmt.Errorf("failed to get port %s: %w", si.PortId, err)
	}

	// Create network namespace with ip/mac, keyed by endpoint ID.
	ns := netlink.NewNetworkNamespace()
	defer func() { _ = ns.Close() }()

	if err = ns.EnsureNetworkNamespace(ctx, injectorPort, a.neutron.ServiceClient, si.ID.String()); err != nil {
		return fmt.Errorf("failed to ensure network namespace: %w", err)
	}

	// Ensure the per-endpoint directory exists (holds the proxy socket + HAProxy
	// files) and start the unprivileged proxy for this endpoint's service.
	endpointID := si.ID.String()
	if err = os.MkdirAll(proxy.GetNetworkDir(endpointID), 0o777); err != nil {
		return fmt.Errorf("failed to create endpoint dir: %w", err)
	}
	ports := make([]int32, len(si.ServicePorts))
	for i, p := range si.ServicePorts {
		ports[i] = int32(p)
	}
	a.proxyManager.StartProxy(si.ID, si.ServiceIPAddress, ports)

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
		log.Errorf("Error enabling haproxy: %s, dumping conf", err)
		haproxy.Dump(haproxy.GetConfigFilePath(endpointID))
		haproxy.TryRemoveFile(haproxy.GetConfigFilePath(endpointID))
		return fmt.Errorf("failed to add haproxy instance: %w", err)
	}

	return nil
}

func (a *Agent) DisableInjection(si *models.ServiceInjection) error {
	endpointID := si.ID.String()

	if a.haproxy.IsRunning(endpointID) {
		if err := a.haproxy.RemoveInstance(endpointID); err != nil {
			return fmt.Errorf("failed to remove haproxy instance: %w", err)
		}
	}
	a.proxyManager.StopProxy(si.ID)
	return nil
}

func (a *Agent) CollectStats() {
	a.haproxy.CollectStats()
}

// deleteOwnedPort deletes the Neutron port for si if it was created by Archer (Owned=true).
// A 404 is tolerated — the port may already be gone.
func (a *Agent) deleteOwnedPort(ctx context.Context, si *models.ServiceInjection) error {
	if !si.Owned {
		return nil
	}
	if err := a.neutron.DeletePort(ctx, si.PortId.String()); err != nil {
		if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return fmt.Errorf("failed to delete endpoint port %s: %w", si.PortId, err)
		}
	}
	return nil
}
