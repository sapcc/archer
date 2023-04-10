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
	"encoding/json"
	"fmt"
	"github.com/sapcc/archer/models"
	"net"
	"net/url"

	"github.com/f5devcentral/go-bigip"
	"github.com/go-openapi/strfmt"

	"github.com/sapcc/archer/internal/config"
)

func GetBigIPSession() (*bigip.BigIP, error) {
	parsedURL, err := url.Parse(config.Global.Agent.Host)
	if err != nil {
		return nil, err
	}

	// check for password
	pw, ok := parsedURL.User.Password()
	if !ok {
		return nil, fmt.Errorf("password required for host '%s'", parsedURL.Hostname())
	}

	session, err := bigip.NewTokenSession(&bigip.Config{
		Address:           parsedURL.Host,
		Username:          parsedURL.User.Username(),
		Password:          pw,
		LoginReference:    "tmos",
		CertVerifyDisable: !config.Global.Agent.ValidateCert,
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

func PostBigIP(bigip *bigip.BigIP, as3 *AS3, tenant string) error {
	data, err := json.MarshalIndent(as3, "", "  ")
	if err != nil {
		return err
	}

	if config.IsDebug() {
		fmt.Printf("-------------------> %s\n%s\n-------------------\n", bigip.Host, data)
	}

	err, _, _ = bigip.PostAs3Bigip(string(data), tenant)
	if err != nil {
		return err
	}

	return nil
}

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
		for _, fixedIP := range service.SnatPort.FixedIPs {
			snatAddresses = append(snatAddresses,
				fmt.Sprintf("%s%%%d", fixedIP.IPAddress, service.SegmentId))
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

		services[GetServicePoolName(service.ID)] = Pool{
			Class: "Pool",
			Label: GetServiceName(service.ID),
			Members: []PoolMember{{
				Enable:          service.Enabled,
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
		pool := fmt.Sprintf("/Common/Shared/%s", GetServicePoolName(endpoint.ServiceID))
		snat := fmt.Sprintf("/Common/Shared/%s", GetServiceSnatPoolName(endpoint.ServiceID))
		virtualAddresses := []string{fmt.Sprintf("0.0.0.0%%%d/0", endpoint.SegmentId)}
		for _, fixedIP := range endpoint.Port.FixedIPs {
			virtualAddresses = append(virtualAddresses,
				fmt.Sprintf("%s%%%d", fixedIP.IPAddress, endpoint.SegmentId),
			)
		}

		services[endpointName] = ServiceL4{
			Label:               endpointName,
			Class:               "Service_L4",
			Mirroring:           "L4",
			PersistanceMethods:  []string{},
			Pool:                Pointer{BigIP: pool},
			ProfileL4:           Pointer{BigIP: "/Common/cc_fastL4_profile"},
			Snat:                Pointer{BigIP: snat},
			TranslateServerPort: false,
			VirtualPort:         endpoint.ServicePortNr,
			AllowVlans: []string{
				fmt.Sprintf("/Common/vlan-%d", endpoint.SegmentId),
			},
			VirtualAddresses: virtualAddresses,
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

func EnsureRouteDomain(big *bigip.BigIP, segmentId int) error {
	routeDomains, err := big.RouteDomains()
	if err != nil {
		return err
	}

	var found bool
	for _, routeDomain := range routeDomains.RouteDomains {
		if routeDomain.ID == segmentId {
			found = true
			break
		}
	}

	if found {
		return nil
	}

	// Create route domain
	if err := big.CreateRouteDomain(
		fmt.Sprintf("vlan-%d", segmentId), segmentId,
		true, "/Common/vlan-%d"); err != nil {
		return err
	}

	return nil
}

func EnsureVLAN(big *bigip.BigIP, segmentId int) error {
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
	if err := big.CreateVlan(&vlan); err != nil {
		return err
	}

	//a.bigip.AddInterfaceToVlan()

	return nil
}
