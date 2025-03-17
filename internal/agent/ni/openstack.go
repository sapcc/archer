/*
Copyright 2022 SAP SE.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ni

import (
	"context"
	"os"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
)

type ServiceInjection struct {
	Id        strfmt.UUID
	Status    string
	PortId    strfmt.UUID
	Network   strfmt.UUID
	IpAddress strfmt.IPv4
	Port      int
	Protocol  string
}

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

	a.haproxy = NewHAProxyController()
	return nil
}

func (a *Agent) EnableInjection(si *ServiceInjection) error {
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
	ns, err := EnsureNetworkNamespace(injectorPort, a.neutron)
	if err != nil {
		return err
	}

	if a.haproxy.isRunning(injectorPort.NetworkID) {
		// Nothing to do
		return nil
	}

	// Run haproxy inside network namespace
	if err = ns.EnableNetworkNamespace(); err != nil {
		return err
	}
	defer func() { _ = ns.Close() }()
	if _, err = a.haproxy.addInstance(injectorPort.NetworkID, si.Protocol); err != nil {
		log.Errorf("Error enabling haproxy: %s, dumping log", err)
		a.haproxy.dumpLog(injectorPort.NetworkID)
	}
	if err := ns.DisableNetworkNamespace(); err != nil {
		return err
	}

	return nil
}

func (a *Agent) DisableInjection(si *ServiceInjection) error {
	log.Debugf("DisableInjection(si='%+v')", si)
	injectorPort, err := ports.Get(context.Background(), a.neutron, si.PortId.String()).Extract()
	if err != nil {
		return err
	}

	if a.haproxy.isRunning(injectorPort.NetworkID) {
		if err := a.haproxy.removeInstance(injectorPort.NetworkID); err != nil {
			return err
		}
	}

	if err := DeleteNetworkNamespace(injectorPort.NetworkID); err != nil {
		// namespace does not exist
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func (a *Agent) CollectStats() {
	a.haproxy.collectStats()
}
