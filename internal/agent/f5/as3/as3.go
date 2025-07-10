// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package as3

import (
	"fmt"
	"net"

	"github.com/go-openapi/strfmt"

	"github.com/sapcc/archer/internal/config"
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
		for _, snatPort := range service.NeutronPorts {
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
				Enable:          true,
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
		if endpoint.Status == models.EndpointStatusPENDINGDELETE || endpoint.Status == models.EndpointStatusPENDINGREJECTED {
			continue
		}

		endpointName := fmt.Sprintf("endpoint-%s", endpoint.ID)
		iRuleName := fmt.Sprintf("irule-%s", endpoint.ID)
		pool := fmt.Sprintf("/Common/Shared/%s", GetServicePoolName(endpoint.ServiceID))
		snat := fmt.Sprintf("/Common/Shared/%s", GetServiceSnatPoolName(endpoint.ServiceID))
		var virtualAddresses []string
		for _, fixedIP := range endpoint.Port.FixedIPs {
			virtualAddresses = append(virtualAddresses,
				fmt.Sprintf("%s%%%d", fixedIP.IPAddress, *endpoint.SegmentId),
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
			l4profile = &Pointer{BigIP: config.Global.Agent.L4Profile}
		}

		services[endpointName] = Service{
			Label:               endpointName,
			Class:               class,
			IRules:              iRules,
			Mirroring:           "L4",
			PersistanceMethods:  []string{},
			Pool:                Pointer{BigIP: pool},
			ProfileL4:           l4profile,
			ProfileTCP:          &Pointer{BigIP: config.Global.Agent.TCPProfile},
			Snat:                Pointer{BigIP: snat},
			TranslateServerPort: true,
			VirtualPort:         endpoint.ServicePortNr,
			AllowVlans: []string{
				fmt.Sprintf("/Common/vlan-%d", *endpoint.SegmentId),
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
