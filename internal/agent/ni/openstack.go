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
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/config"
)

type ServiceInjection struct {
	Id        strfmt.UUID
	Status    string
	PortId    strfmt.UUID
	Network   strfmt.UUID
	IpAddress strfmt.IPv4
	Port      int
}

func (o *Agent) SetupOpenStack() error {
	authInfo := clientconfig.AuthInfo(config.Global.ServiceAuth)
	// Allow automatically reauthenticate
	authInfo.AllowReauth = true

	providerClient, err := clientconfig.AuthenticatedClient(&clientconfig.ClientOpts{
		AuthInfo: &authInfo})
	if err != nil {
		logg.Fatal(err.Error())
	}

	if o.neutron, err = openstack.NewNetworkV2(providerClient, gophercloud.EndpointOpts{}); err != nil {
		return err
	}
	// Set timeout to 10 secs
	o.neutron.HTTPClient.Timeout = time.Second * 10

	o.haproxy = NewHAProxyController()
	return nil
}

func (o *Agent) EnableInjection(si *ServiceInjection) error {
	injectorPort, err := ports.Get(o.neutron, si.PortId.String()).Extract()
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
	ns, err := EnsureNetworkNamespace(injectorPort, o.neutron)
	if err != nil {
		return err
	}

	if o.haproxy.isRunning(injectorPort.NetworkID) {
		// Nothing to do
		return nil
	}

	// Run haproxy inside network namespace
	if err := ns.EnableNetworkNamespace(); err != nil {
		return err
	}
	defer func() { _ = ns.Close() }()
	if _, err := o.haproxy.addInstance(injectorPort.NetworkID); err != nil {
		return err
	}
	if err := ns.DisableNetworkNamespace(); err != nil {
		return err
	}

	return nil
}

func (o *Agent) DisableInjection(si *ServiceInjection) error {
	logg.Debug("DisableInjection(si='%s')", si)
	injectorPort, err := ports.Get(o.neutron, si.PortId.String()).Extract()
	if err != nil {
		return err
	}

	if o.haproxy.isRunning(injectorPort.NetworkID) {
		if err := o.haproxy.removeInstance(injectorPort.NetworkID); err != nil {
			return err
		}
	}

	if err := DeleteNetworkNamespace(injectorPort.NetworkID); err != nil {
		return err
	}
	return nil
}

func (o *Agent) CollectStats() {
	o.haproxy.collectStats()
}
