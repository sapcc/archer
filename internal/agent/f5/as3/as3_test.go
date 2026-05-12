// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package as3

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag/conv"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/models"
)

func TestGetEndpointTenants(t *testing.T) {
	config.Global.Agent.L4Profile = "test-l4-profile"
	config.Global.Agent.TCPProfile = "test-tcp-profile"
	endpoints := []*ExtendedEndpoint{
		{
			Endpoint: models.Endpoint{
				CreatedAt:   time.Time{},
				Description: "test-description",
				ID:          "3ad9b1f0-4e5a-44c3-ada6-71696925ae64",
				IPAddress:   "1.2.3.4",
				Name:        "test-name",
				ProjectID:   "test-project",
				ServiceID:   strfmt.UUID("4e50bf87-e597-41f2-9ce0-83d3e24dedf3"),
				Target:      models.EndpointTarget{},
				UpdatedAt:   time.Time{},
			},
			ProxyProtocol:       true,
			ConnectionMirroring: true,
			Port: &ports.Port{
				FixedIPs: []ports.IP{{IPAddress: "1.2.3.4"}},
			},
			SegmentId:    conv.Pointer(1),
			ServicePorts: []int32{0},
		},
	}
	tenant := GetEndpointTenants(endpoints)
	assert.NotNil(t, tenant)
	json, err := tenant.MarshalJSON()
	assert.Nil(t, err)

	expectedJSON := fmt.Sprintf(`{"class":"Tenant","si-endpoints":{"class":"Application","endpoint-0-3ad9b1f0-4e5a-44c3-ada6-71696925ae64":{"label":"endpoint-0-3ad9b1f0-4e5a-44c3-ada6-71696925ae64","class":"Service_TCP","allowVlans":["/Common/vlan-1"],"iRules":[{"use":"irule-3ad9b1f0-4e5a-44c3-ada6-71696925ae64"}],"mirroring":"L4","persistenceMethods":[],"pool":{"bigip":"/Common/Shared/pool-4e50bf87-e597-41f2-9ce0-83d3e24dedf3-0"},"profileTCP":{"bigip":"test-tcp-profile"},"snat":{"bigip":"/Common/Shared/snatpool-4e50bf87-e597-41f2-9ce0-83d3e24dedf3"},"virtualAddresses":["1.2.3.4%%1"],"translateServerPort":true,"virtualPort":0},"irule-3ad9b1f0-4e5a-44c3-ada6-71696925ae64":{"label":"irule-endpoint-3ad9b1f0-4e5a-44c3-ada6-71696925ae64","class":"iRule","iRule":{"base64":"%s"}},"template":"generic"}}`, base64.StdEncoding.EncodeToString([]byte(pp2)))
	assert.JSONEq(t, expectedJSON, string(json), "Tenant JSON should be equal")
}

func TestGetEndpointWithoutTenants(t *testing.T) {
	expected := Tenant{
		Class:        "Tenant",
		Label:        "",
		Remark:       "",
		Applications: nil,
	}
	assert.Equal(t, expected, GetEndpointTenants([]*ExtendedEndpoint{}))
}

func TestGetEndpointTenantsMirroringDisabled(t *testing.T) {
	config.Global.Agent.L4Profile = "test-l4-profile"
	config.Global.Agent.TCPProfile = "test-tcp-profile"
	endpoints := []*ExtendedEndpoint{
		{
			Endpoint: models.Endpoint{
				ID:        "3ad9b1f0-4e5a-44c3-ada6-71696925ae64",
				ServiceID: strfmt.UUID("4e50bf87-e597-41f2-9ce0-83d3e24dedf3"),
			},
			ProxyProtocol:       false,
			ConnectionMirroring: false,
			Port: &ports.Port{
				FixedIPs: []ports.IP{{IPAddress: "1.2.3.4"}},
			},
			SegmentId:    conv.Pointer(1),
			ServicePorts: []int32{80},
		},
	}
	tenant := GetEndpointTenants(endpoints)
	json, err := tenant.MarshalJSON()
	assert.Nil(t, err)
	assert.Contains(t, string(json), `"mirroring":"none"`)
}

func TestGetEndpointTenantsMirroringEnabled(t *testing.T) {
	config.Global.Agent.L4Profile = "test-l4-profile"
	config.Global.Agent.TCPProfile = "test-tcp-profile"
	endpoints := []*ExtendedEndpoint{
		{
			Endpoint: models.Endpoint{
				ID:        "3ad9b1f0-4e5a-44c3-ada6-71696925ae64",
				ServiceID: strfmt.UUID("4e50bf87-e597-41f2-9ce0-83d3e24dedf3"),
			},
			ProxyProtocol:       false,
			ConnectionMirroring: true,
			Port: &ports.Port{
				FixedIPs: []ports.IP{{IPAddress: "1.2.3.4"}},
			},
			SegmentId:    conv.Pointer(1),
			ServicePorts: []int32{80},
		},
	}
	tenant := GetEndpointTenants(endpoints)
	json, err := tenant.MarshalJSON()
	assert.Nil(t, err)
	assert.Contains(t, string(json), `"mirroring":"L4"`)
}

func TestGetServiceName(t *testing.T) {
	id := strfmt.UUID("4e50bf87-e597-41f2-9ce0-83d3e24dedf3")
	assert.Equal(t, "service-4e50bf87-e597-41f2-9ce0-83d3e24dedf3",
		GetServiceName(id), "Service name should match expected format")
}

func TestGetEndpointTenantName(t *testing.T) {
	assert.Equal(t, "net-4e50bf87-e597-41f2-9ce0-83d3e24dedf3",
		GetEndpointTenantName(strfmt.UUID("4e50bf87-e597-41f2-9ce0-83d3e24dedf3")),
		"Endpoint tenant name should match expected format")
}

func TestGetAS3Declaration(t *testing.T) {
	tenants := map[string]Tenant{}
	as3 := GetAS3Declaration(tenants)
	expected := AS3{
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
	assert.EqualValues(t, expected, as3, "AS3 declaration should match expected structure")
}

func TestGetServiceTenants(t *testing.T) {
	services := []*ExtendedService{
		{
			Service: models.Service{
				AvailabilityZone: conv.Pointer("abc"),
				CreatedAt:        time.Time{},
				Description:      "test",
				ID:               "test-service-id",
				Ports:            []int32{0},
			},
			NeutronPorts: map[string]*ports.Port{
				"snat-port-1": {
					ID: "snat-port-1",
					FixedIPs: []ports.IP{
						{IPAddress: "1.2.3.4", SubnetID: "subnet-1"},
					},
				},
			},
			SubnetID:  "1234",
			SegmentId: 54321,
			MTU:       7890,
		},
	}
	expected := Tenant{
		Class:  "Tenant",
		Label:  "",
		Remark: "",
		Applications: map[string]Application{"Shared": {
			Class:    "Application",
			Label:    "",
			Remark:   "",
			Template: "shared",
			Services: map[string]any{
				"pool-test-service-id-0": Pool{
					Class:  "Pool",
					Label:  "pool-test-service-id-0",
					Remark: "",
					Members: []PoolMember{{
						RouteDomain:     54321,
						ServicePort:     0,
						ServerAddresses: []string(nil),
						Enable:          true,
						AdminState:      "enable",
						Remark:          "service-test-service-id"},
					}, Monitors: []Pointer{{
						Use:   "",
						BigIP: "/Common/cc_gwicmp_monitor"},
					},
				},
				"snatpool-test-service-id": SnatPool{
					Class:         "SNAT_Pool",
					Label:         "service-test-service-id",
					Remark:        "",
					SnatAddresses: []string{"1.2.3.4%54321"},
				},
			},
		}},
	}

	assert.EqualValues(t, expected, GetServiceTenants(services))
}

func TestGetServiceTenantsWithoutServices(t *testing.T) {
	expected := Tenant{Class: "Tenant", Label: "", Remark: "", Applications: map[string]Application{"Shared": {Class: "Application", Label: "", Remark: "", Template: "shared", Services: map[string]any{}}}}
	assert.EqualValues(t, expected, GetServiceTenants([]*ExtendedService{}))
}

func TestGetEndpointTenantsIPv6(t *testing.T) {
	config.Global.Agent.L4Profile = "test-l4-profile"
	config.Global.Agent.TCPProfile = "test-tcp-profile"
	endpoints := []*ExtendedEndpoint{
		{
			Endpoint: models.Endpoint{
				ID:        "3ad9b1f0-4e5a-44c3-ada6-71696925ae64",
				ServiceID: strfmt.UUID("4e50bf87-e597-41f2-9ce0-83d3e24dedf3"),
			},
			ProxyProtocol:       true,
			ConnectionMirroring: false,
			Port: &ports.Port{
				FixedIPs: []ports.IP{{IPAddress: "2001:db8::1"}},
			},
			SegmentId:    conv.Pointer(5),
			ServicePorts: []int32{443},
		},
	}
	tenant := GetEndpointTenants(endpoints)
	assert.NotNil(t, tenant)
	json, err := tenant.MarshalJSON()
	assert.Nil(t, err)

	// IPv6 virtual address should be formatted as ip%routedomain
	assert.Contains(t, string(json), `"virtualAddresses":["2001:db8::1%5"]`)
	assert.Contains(t, string(json), `"mirroring":"none"`)
	assert.Contains(t, string(json), `"class":"Service_TCP"`)
}

func TestGetEndpointTenantsIPv6L4(t *testing.T) {
	config.Global.Agent.L4Profile = "test-l4-profile"
	config.Global.Agent.TCPProfile = "test-tcp-profile"
	endpoints := []*ExtendedEndpoint{
		{
			Endpoint: models.Endpoint{
				ID:        "3ad9b1f0-4e5a-44c3-ada6-71696925ae64",
				ServiceID: strfmt.UUID("4e50bf87-e597-41f2-9ce0-83d3e24dedf3"),
			},
			ProxyProtocol:       false,
			ConnectionMirroring: false,
			Port: &ports.Port{
				FixedIPs: []ports.IP{{IPAddress: "fd00::abcd:1234"}},
			},
			SegmentId:    conv.Pointer(10),
			ServicePorts: []int32{80},
		},
	}
	tenant := GetEndpointTenants(endpoints)
	json, err := tenant.MarshalJSON()
	assert.Nil(t, err)

	assert.Contains(t, string(json), `"virtualAddresses":["fd00::abcd:1234%10"]`)
	assert.Contains(t, string(json), `"class":"Service_L4"`)
}

func TestGetServiceTenantsIPv6(t *testing.T) {
	services := []*ExtendedService{
		{
			Service: models.Service{
				AvailabilityZone: conv.Pointer("abc"),
				Description:      "test-ipv6-service",
				ID:               "test-ipv6-service-id",
				IPAddresses:      []models.InetAddress{"2001:db8::/128"},
				Ports:            []int32{443},
			},
			NeutronPorts: map[string]*ports.Port{
				"snat-port-1": {
					ID: "snat-port-1",
					FixedIPs: []ports.IP{
						{IPAddress: "2001:db8::ff", SubnetID: "subnet-v6"},
					},
				},
			},
			SubnetID:  "subnet-v6",
			SegmentId: 99,
			MTU:       9000,
		},
	}
	tenant := GetServiceTenants(services)
	assert.NotNil(t, tenant)

	app, ok := tenant.Applications["Shared"]
	assert.True(t, ok)

	// Verify SNAT pool has IPv6 address with route domain
	snatPool, ok := app.Services["snatpool-test-ipv6-service-id"].(SnatPool)
	assert.True(t, ok)
	assert.Equal(t, []string{"2001:db8::ff%99"}, snatPool.SnatAddresses)

	// Verify pool has IPv6 server addresses
	pool, ok := app.Services["pool-test-ipv6-service-id-443"].(Pool)
	assert.True(t, ok)
	assert.Equal(t, []string{"2001:db8::"}, pool.Members[0].ServerAddresses)
}

func TestGetServiceTenantsDualStack(t *testing.T) {
	services := []*ExtendedService{
		{
			Service: models.Service{
				AvailabilityZone: conv.Pointer("abc"),
				Description:      "test-dualstack-service",
				ID:               "test-ds-service-id",
				IPAddresses:      []models.InetAddress{"10.0.0.1/32", "2001:db8::1/128"},
				Ports:            []int32{80},
			},
			NeutronPorts: map[string]*ports.Port{
				"snat-port-1": {
					ID: "snat-port-1",
					FixedIPs: []ports.IP{
						{IPAddress: "10.0.0.100", SubnetID: "subnet-v4"},
						{IPAddress: "2001:db8::100", SubnetID: "subnet-v6"},
					},
				},
			},
			SubnetID:  "subnet-v4",
			SegmentId: 42,
			MTU:       1500,
		},
	}
	tenant := GetServiceTenants(services)
	app := tenant.Applications["Shared"]

	// SNAT pool should contain both IPv4 and IPv6 addresses
	snatPool := app.Services["snatpool-test-ds-service-id"].(SnatPool)
	assert.Contains(t, snatPool.SnatAddresses, "10.0.0.100%42")
	assert.Contains(t, snatPool.SnatAddresses, "2001:db8::100%42")

	// Pool should have both server addresses
	pool := app.Services["pool-test-ds-service-id-80"].(Pool)
	assert.Contains(t, pool.Members[0].ServerAddresses, "10.0.0.1")
	assert.Contains(t, pool.Members[0].ServerAddresses, "2001:db8::1")
}
