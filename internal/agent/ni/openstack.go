// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ni

import (
	"context"
	"os"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	"github.com/sapcc/archer/internal/agent/ni/haproxy"
	"github.com/sapcc/archer/internal/agent/ni/models"
	"github.com/sapcc/archer/internal/agent/ni/netlink"
	log "github.com/sirupsen/logrus"

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
		return err
	}

	//if injectorPort == nil {
	//	log.Printf("Creating port for network %s (%s)", network.Name, network.ID)
	//	port := dns.PortCreateOptsExt{
	//		CreateOptsBuilder: portsbinding.CreateOptsExt{
	//			CreateOptsBuilder: ports.CreateOpts{
	//				/*Name:        config.NetworkTag + " injection port",*/
	//				DeviceOwner: GetDeviceOwner(),
	//				DeviceID:    "network-injector",
	//				NetworkID:   network.ID,
	//				TenantID:    network.TenantID,
	//			},
	//			/*HostID: config.Hostname,*/
	//		},
	//		/*DNSName: config.InjectorDNS,*/
	//	}
	//
	//	var err error
	//	if injectorPort, err = ports.Create(o.neutron, port).Extract(); err != nil {
	//		return err
	//	}
	//	log.Printf("Port '%s' created", injectorPort.ID)
	//}

	// Create network namespace with ip/mac
	ns := netlink.NewNetworkNamespace()
	if err = ns.EnsureNetworkNamespace(injectorPort, a.neutron); err != nil {
		return err
	}

	if a.haproxy.IsRunning(injectorPort.NetworkID) {
		// Nothing to do
		return nil
	}

	// Run haproxy inside network namespace
	if err = ns.EnableNetworkNamespace(); err != nil {
		return err
	}
	defer func() { _ = ns.Close() }()
	if err = a.haproxy.AddInstance(si); err != nil {
		log.Errorf("Error enabling haproxy: %s, dumping conf/log", err)
		haproxy.Dump(haproxy.GetLogFilePath(si.Network.String()))
		haproxy.Dump(haproxy.GetConfigFilePath(si.Network.String()))
		haproxy.TryRemoveFile(haproxy.GetConfigFilePath(si.Network.String()))
		haproxy.TryRemoveFile(haproxy.GetLogFilePath(si.Network.String()))
	}
	if err := ns.DisableNetworkNamespace(); err != nil {
		return err
	}

	return nil
}

func (a *Agent) DisableInjection(si *models.ServiceInjection) error {
	log.Debugf("DisableInjection(si='%+v')", si)
	injectorPort, err := ports.Get(context.Background(), a.neutron, si.PortId.String()).Extract()
	if err != nil {
		return err
	}

	if a.haproxy.IsRunning(injectorPort.NetworkID) {
		if err := a.haproxy.RemoveInstance(injectorPort.NetworkID); err != nil {
			return err
		}
	}

	ns := netlink.NewNetworkNamespace()
	if err = ns.EnsureNetworkNamespace(injectorPort, a.neutron); err != nil {
		return err
	}

	if err = ns.DeleteNetworkNamespace(); err != nil {
		// namespace does not exist
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func (a *Agent) CollectStats() {
	a.haproxy.CollectStats()
}
