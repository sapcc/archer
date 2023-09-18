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

package as3

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/f5devcentral/go-bigip"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/sethvargo/go-retry"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/errors"
	"github.com/sapcc/archer/internal/neutron"
	"github.com/sapcc/archer/models"
)

func GetServiceSnatPoolName(Id strfmt.UUID) string {
	return fmt.Sprintf("snatpool-%s", Id)
}

func GetServicePoolName(Id strfmt.UUID) string {
	return fmt.Sprintf("pool-%s", Id)
}

func GetServiceName(Id strfmt.UUID) string {
	return fmt.Sprintf("service-%s", Id)
}

func GetEndpointTenantName(networkId strfmt.UUID) string {
	return fmt.Sprintf("net-%s", networkId)
}

func GetAS3Declaration(tenants map[string]Tenant) AS3 {
	return AS3{
		Class:   "AS3",
		Action:  "deploy",
		Persist: false,
		Declaration: ADC{
			Class:         "ADC",
			SchemaVersion: "3.36.0",
			UpdateMode:    "selective",
			Id:            "urn:uuid:07649173-4AF7-48DF-963F-84000C70F0DD",
			Tenants:       tenants,
		},
	}
}

func GetServiceTenants(endpointServices []*ExtendedService) Tenant {
	services := make(map[string]any, len(endpointServices)*2)

	for _, service := range endpointServices {
		if service.Status == "PENDING_DELETE" {
			// Skip services in pending deletion
			continue
		}

		var snatAddresses []string
		for _, snatPort := range service.SnatPorts {
			for _, fixedIP := range snatPort.FixedIPs {
				snatAddresses = append(snatAddresses,
					fmt.Sprintf("%s%%%d", fixedIP.IPAddress, service.SegmentId))
			}
		}

		services[GetServiceSnatPoolName(service.ID)] = SnatPool{
			Class:         "SNAT_Pool",
			Label:         GetServiceName(service.ID),
			SnatAddresses: snatAddresses,
		}

		var serverAddresses []string
		for _, ipAddress := range service.IPAddresses {
			ip, _, _ := net.ParseCIDR(ipAddress.String())
			serverAddresses = append(serverAddresses, ip.String())
		}

		adminState := "enable"
		if service.Enabled != nil && !*service.Enabled {
			adminState = "disable"
		}

		services[GetServicePoolName(service.ID)] = Pool{
			Class: "Pool",
			Label: GetServiceName(service.ID),
			Members: []PoolMember{{
				AdminState:      adminState,
				RouteDomain:     service.SegmentId,
				ServicePort:     service.Port,
				ServerAddresses: serverAddresses,
				Remark:          GetServiceName(service.ID),
			}},
			Monitors: []Pointer{
				{BigIP: "/Common/cc_gwicmp_monitor"},
			},
		}
	}

	return Tenant{
		Class: "Tenant",
		Applications: map[string]Application{
			"Shared": {
				Class:    "Application",
				Template: "shared",
				Services: services,
			},
		},
	}
}

func GetEndpointTenants(endpoints []*ExtendedEndpoint) Tenant {
	services := make(map[string]any, len(endpoints))

	for _, endpoint := range endpoints {
		// Skip pending delete endpoints
		if endpoint.Status == models.EndpointStatusPENDINGDELETE {
			continue
		}

		endpointName := fmt.Sprintf("endpoint-%s", endpoint.ID)
		iRuleName := fmt.Sprintf("irule-%s", endpoint.ID)
		pool := fmt.Sprintf("/Common/Shared/%s", GetServicePoolName(endpoint.ServiceID))
		snat := fmt.Sprintf("/Common/Shared/%s", GetServiceSnatPoolName(endpoint.ServiceID))
		var virtualAddresses []string
		for _, fixedIP := range endpoint.Port.FixedIPs {
			virtualAddresses = append(virtualAddresses,
				fmt.Sprintf("%s%%%d", fixedIP.IPAddress, endpoint.SegmentId),
			)
		}
		iRules := make([]Pointer, 0)
		var class string
		var l4profile *Pointer
		if endpoint.ProxyProtocol {
			// Add iRule for proxy protocol v2
			class = "Service_TCP"
			services[iRuleName] = IRule{
				Label: fmt.Sprint("irule-", endpointName),
				Class: "iRule",
				IRule: IRuleBase64{pp2},
			}
			iRules = append(iRules, Pointer{
				Use: iRuleName,
			})
		} else {
			class = "Service_L4"
			l4profile = &Pointer{BigIP: "/Common/cc_fastL4_profile"}
		}

		services[endpointName] = Service{
			Label:               endpointName,
			Class:               class,
			IRules:              iRules,
			Mirroring:           "L4",
			PersistanceMethods:  []string{},
			Pool:                Pointer{BigIP: pool},
			ProfileL4:           l4profile,
			Snat:                Pointer{BigIP: snat},
			TranslateServerPort: true,
			VirtualPort:         endpoint.ServicePortNr,
			AllowVlans: []string{
				fmt.Sprintf("/Common/vlan-%d", endpoint.SegmentId),
			},
			VirtualAddresses: virtualAddresses,
		}
	}

	if len(services) == 0 {
		return Tenant{
			Class: "Tenant",
		}
	}

	return Tenant{
		Class: "Tenant",
		Applications: map[string]Application{
			"si-endpoints": {
				Class:    "Application",
				Template: "generic",
				Services: services,
			},
		},
	}
}

// BigIP Wrapper
type BigIP struct {
	*bigip.BigIP
}

func (b *BigIP) GetHostname() string {
	deviceURL, err := url.Parse(b.Host)
	if err != nil {
		panic(err)
	}

	return deviceURL.Hostname()
}

func GetBigIPSession(rawURL string) (*BigIP, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	// check for password
	pw, ok := parsedURL.User.Password()
	if !ok {
		return nil, fmt.Errorf("password required for host '%s'", parsedURL.Hostname())
	}

	session := bigip.NewSession(&bigip.Config{
		Address:           parsedURL.Host,
		Username:          parsedURL.User.Username(),
		Password:          pw,
		LoginReference:    "tmos",
		CertVerifyDisable: !config.Global.Agent.ValidateCert,
	})
	return &BigIP{session}, nil
}

func (big *BigIP) PostBigIP(as3 *AS3, tenant string) error {
	data, err := json.MarshalIndent(as3, "", "  ")
	if err != nil {
		return err
	}

	if config.IsDebug() {
		fmt.Printf("-------------------> %s\n%s\n-------------------\n", big.Host, data)
	}

	r := retry.WithMaxRetries(3, retry.NewExponential(3*time.Second))
	err = retry.Do(context.Background(), r, func(ctx context.Context) error {
		err, _, _ = big.PostAs3Bigip(string(data), tenant)
		return retry.RetryableError(err)
	})
	return err
}

func (big *BigIP) GetBigIPDevice(hostname string) *bigip.Device {
	devices, err := big.GetDevices()
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, device := range devices {
		if strings.HasSuffix(hostname, device.Hostname) {
			log.Infof("Connected to %s, %s (%s %s), %s", device.MarketingName, device.Name, device.Version,
				device.Edition, device.FailoverState)
			return &device
		}
	}
	return nil
}

type VcmpGuests struct {
	Guests []bigip.VcmpGuest `json:"items,omitempty"`
}

func (big *BigIP) GetVCMPGuests() (*VcmpGuests, error) {
	var guests VcmpGuests

	req := &bigip.APIRequest{
		Method:      "get",
		URL:         "vcmp/guest",
		ContentType: "application/json",
	}
	resp, err := big.APICall(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, &guests)
	if err != nil {
		return nil, err
	}

	return &guests, nil
}

func (big *BigIP) EnsureSelfIP(neutron *neutron.NeutronClient, service *ExtendedService) error {
	port, ok := service.SnatPorts[big.GetHostname()]
	if !ok {
		return fmt.Errorf("EnsureSelfIP: no port for service '%s' found on bigip '%s'", service.ID, big.Host)
	}

	name := fmt.Sprint("selfip-", port.ID)
	selfIPs, err := big.SelfIPs()
	if err != nil {
		return err
	}
	for _, selfIP := range selfIPs.SelfIPs {
		if selfIP.Name == name {
			return nil
		}
	}

	// Fetch netmask
	subnet, err := subnets.Get(neutron.ServiceClient, port.FixedIPs[0].SubnetID).Extract()
	if err != nil {
		return err
	}
	_, ipNet, err := net.ParseCIDR(subnet.CIDR)
	if err != nil {
		return err
	}
	mask, _ := ipNet.Mask.Size()
	selfIP := bigip.SelfIP{
		Name:    name,
		Address: fmt.Sprint(port.FixedIPs[0].IPAddress, "%", service.SegmentId, "/", mask),
		Vlan:    fmt.Sprint("/Common/vlan-", service.SegmentId),
	}
	if err := big.CreateSelfIP(&selfIP); err != nil {
		return err
	}
	return nil
}

func (big *BigIP) CleanupSelfIP(port *ports.Port) error {
	name := fmt.Sprint("selfip-", port.ID)
	selfIPs, err := big.SelfIPs()
	if err != nil {
		return err
	}
	for _, selfIP := range selfIPs.SelfIPs {
		if selfIP.Name == name {
			return big.DeleteSelfIP(selfIP.Name)
		}
	}

	return errors.ErrNoSelfIP
}

func (big *BigIP) EnsureRouteDomain(segmentId int, parent *int) error {
	routeDomains, err := big.RouteDomains()
	if err != nil {
		return err
	}

	var found bool
	for _, rd := range routeDomains.RouteDomains {
		if rd.ID == segmentId {
			if parent != nil && rd.Parent != fmt.Sprintf("/Common/vlan-%d", *parent) {
				continue
			}
			found = true
			break
		}
	}

	if found {
		return nil
	}

	c := &routeDomain{
		RouteDomain: bigip.RouteDomain{
			Name:   fmt.Sprintf("vlan-%d", segmentId),
			ID:     segmentId,
			Strict: "enabled",
			Vlans:  []string{fmt.Sprintf("/Common/vlan-%d", segmentId)},
		},
	}
	if parent != nil {
		c.Parent = fmt.Sprintf("vlan-%d", *parent)
	}

	return c.Update(big)
}

func (big *BigIP) CleanupRouteDomain(segmentId int) error {
	return big.DeleteRouteDomain(fmt.Sprintf("vlan-%d", segmentId))
}

func (big *BigIP) EnsureGuestVlan(segmentId int) error {
	guests, err := big.GetVCMPGuests()
	if err != nil {
		return err
	}

	for _, guest := range guests.Guests {
		for _, deviceHost := range config.Global.Agent.Devices {
			if strings.HasSuffix(deviceHost, guest.Hostname) {
				vlanName := fmt.Sprintf("/Common/vlan-%d", segmentId)
				for _, vlan := range guest.Vlans {
					if vlan == vlanName {
						// found, nothing to do
						return nil
					}
				}
				newGuest := bigip.VcmpGuest{Vlans: internal.Unique(append(guest.Vlans, vlanName))}
				return big.UpdateVcmpGuest(guest.Name, &newGuest)
			}
		}
	}
	return errors.ErrNoVCMPFound
}

func (big *BigIP) CleanupGuestVlan(segmentId int) error {
	guests, err := big.GetVCMPGuests()
	if err != nil {
		return err
	}

	for _, guest := range guests.Guests {
		for _, deviceHost := range config.Global.Agent.Devices {
			if strings.HasSuffix(deviceHost, guest.Hostname) {
				var vlans []string
				for _, vlan := range guest.Vlans {
					if vlan != fmt.Sprintf("/Common/vlan-%d", segmentId) {
						vlans = append(vlans, vlan)
					}
				}
				newGuest := bigip.VcmpGuest{Vlans: vlans}
				return big.UpdateVcmpGuest(guest.Name, &newGuest)
			}
		}
	}
	return errors.ErrNoVCMPFound
}

func (big *BigIP) EnsureVLAN(segmentId int) error {
	vlans, err := big.Vlans()
	if err != nil {
		return err
	}

	var found bool
	for _, vlan := range vlans.Vlans {
		if vlan.Tag == segmentId {
			found = true
			break
		}
	}

	if found {
		return nil
	}

	// Create vlan
	vlan := bigip.Vlan{
		Name: fmt.Sprintf("vlan-%d", segmentId),
		Tag:  segmentId,
	}
	return big.CreateVlan(&vlan)
}

func (big *BigIP) EnsureInterfaceVlan(segmentId int) error {
	name := fmt.Sprintf("vlan-%d", segmentId)

	vlanInterfaces, err := big.GetVlanInterfaces(name)
	if err != nil {
		return err
	}

	for _, iface := range vlanInterfaces.VlanInterfaces {
		if iface.Name == config.Global.Agent.PhysicalInterface {
			// found, nothing to do
			return nil
		}
	}

	return big.AddInterfaceToVlan(name, config.Global.Agent.PhysicalInterface, true)
}

func (big *BigIP) CleanupVLAN(segmentId int) error {
	return big.DeleteVlan(fmt.Sprintf("vlan-%d", segmentId))
}
