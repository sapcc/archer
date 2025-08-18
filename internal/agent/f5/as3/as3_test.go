// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package as3

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/models"
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
			ProxyProtocol: true,
			Port: &ports.Port{
				FixedIPs: []ports.IP{{IPAddress: "1.2.3.4"}},
			},
			SegmentId: swag.Int(1),
		},
	}
	tenant := GetEndpointTenants(endpoints)
	assert.NotNil(t, tenant)
	json, err := tenant.MarshalJSON()
	assert.Nil(t, err)

	expectedJSON := `{"class":"Tenant","si-endpoints":{"class":"Application","endpoint-3ad9b1f0-4e5a-44c3-ada6-71696925ae64":{"label":"endpoint-3ad9b1f0-4e5a-44c3-ada6-71696925ae64","class":"Service_TCP","allowVlans":["/Common/vlan-1"],"iRules":[{"use":"irule-3ad9b1f0-4e5a-44c3-ada6-71696925ae64"}],"mirroring":"L4","persistenceMethods":[],"pool":{"bigip":"/Common/Shared/pool-4e50bf87-e597-41f2-9ce0-83d3e24dedf3"},"profileTCP":{"bigip":"test-tcp-profile"},"snat":{"bigip":"/Common/Shared/snatpool-4e50bf87-e597-41f2-9ce0-83d3e24dedf3"},"virtualAddresses":["1.2.3.4%1"],"translateServerPort":true,"virtualPort":0},"irule-3ad9b1f0-4e5a-44c3-ada6-71696925ae64":{"label":"irule-endpoint-3ad9b1f0-4e5a-44c3-ada6-71696925ae64","class":"iRule","iRule":{"base64":"CiMgc2VydmVyc2lkZSBQUk9YWSBQcm90b2NvbCBWMiBpbXBsZW1lbnRhdGlvbgojIGh0dHBzOi8vZ2l0aHViLmNvbS9oYXByb3h5L2hhcHJveHkvYmxvYi9mZmRmNmEzMmE3NDEzZDViY2Y5MjIzYzM1NTZiNzY1YjVlNDU2YTY5L2RvYy9wcm94eS1wcm90b2NvbC50eHQKCiMgY29udmVydHMgYW4gaXB2NCBpcCBpbiBkb3R0ZWQgbm90YXRpb24gaW50byA0IGJpbmFyeSBzdHJpbmdzIHdpdGggb25lIGJ5dGUgaW4gaGV4IGVhY2gKcHJvYyBpcDJoZXggeyBpcCB9IHsKICAgIHNldCBvY3RldHMgW3NwbGl0IFtnZXRmaWVsZCAkaXAgJSAxXSAuXQogICAgcmV0dXJuIFtiaW5hcnkgZm9ybWF0IGM0ICRvY3RldHNdCn0KCiMgY29udmVydHMgYW4gMkJ5dGUgaW50ZWdlciB0byAyIGJpbmFyeSBzdHJpbmdzIHdpdGggb25lIGJ5dGUgaW4gaGV4IGVhY2gKcHJvYyBwb3J0MmhleCB7IHBvcnQgfSB7CiAgICByZXR1cm4gW2JpbmFyeSBmb3JtYXQgUyAkcG9ydF0KfQoKcHJvYyBwcm94eV9hZGRyIHt9IHsKICAgICMgaHR0cHM6Ly9naXRodWIuY29tL2hhcHJveHkvaGFwcm94eS9ibG9iL2ZmZGY2YTMyYTc0MTNkNWJjZjkyMjNjMzU1NmI3NjViNWU0NTZhNjkvZG9jL3Byb3h5LXByb3RvY29sLnR4dCNMNDcxLUw0ODgKICAgICMgcHJveHlfYWRkcgogICAgY2xpZW50c2lkZSB7CiAgICAgICAgcmV0dXJuIFtjYWxsIGlwMmhleCBbSVA6OnJlbW90ZV9hZGRyXV1bY2FsbCBpcDJoZXggW0lQOjpsb2NhbF9hZGRyXV1bY2FsbCBwb3J0MmhleCBbVENQOjpyZW1vdGVfcG9ydF1dW2NhbGwgcG9ydDJoZXggW1RDUDo6bG9jYWxfcG9ydF1dCiAgICB9Cn0KCnByb2MgdGx2IHtiaW5hcnlfdHlwZSBiaW5hcnlfdmFsdWV9IHsKICAgICMgVExWOiBodHRwczovL2dpdGh1Yi5jb20vaGFwcm94eS9oYXByb3h5L2Jsb2IvZmZkZjZhMzJhNzQxM2Q1YmNmOTIyM2MzNTU2Yjc2NWI1ZTQ1NmE2OS9kb2MvcHJveHktcHJvdG9jb2wudHh0I0w1MjUtTDUzMAogICAgIyBjYWxjdWxhdGUgbGVuZ3RoIG9mIGRhdGEgZm9yIGxlbmd0aF9oaSBhbmQgbGVuZ3RoX2xvICgyIGJ5dGUgZmllbGQpCiAgICBzZXQgdGx2X2xlbmd0aF9oaWxvIFtiaW5hcnkgZm9ybWF0IFMgW3N0cmluZyBsZW5ndGggJGJpbmFyeV92YWx1ZV1dCgogICAgcmV0dXJuICRiaW5hcnlfdHlwZSR0bHZfbGVuZ3RoX2hpbG8kYmluYXJ5X3ZhbHVlCn0KCiNwcm9jIHRsdl90eXBlNSB7fSB7CiMgICAgcmV0dXJuIFtjYWxsIHRsdiBceDA1IFtiaW5hcnkgZm9ybWF0IEgqIFtsaW5kZXggW0FFUzo6a2V5IDEyOF0gMl1dXQojfQoKcHJvYyB0bHZfc2FwY2Mge3V1aWQ0fSB7CiAgICAjIFBQMl9UWVBFX1NBUENDIDB4RUMKICAgICMgcHJlcGFyZSBUTFYKICAgICMgVExWOiBodHRwczovL2dpdGh1Yi5jb20vaGFwcm94eS9oYXByb3h5L2Jsb2IvZmZkZjZhMzJhNzQxM2Q1YmNmOTIyM2MzNTU2Yjc2NWI1ZTQ1NmE2OS9kb2MvcHJveHktcHJvdG9jb2wudHh0I0w1MjUtTDUzMAogICAgcmV0dXJuIFtjYWxsIHRsdiBceGVjIFtiaW5hcnkgZm9ybWF0IEEqICR1dWlkNF1dCn0KCndoZW4gU0VSVkVSX0NPTk5FQ1RFRCBwcmlvcml0eSA5MDAgewogICAgIyBjcmVhdGUgVExWIHR5cGUgNQogICAgI3NldCBwcDJfdGx2IFtjYWxsIHRsdl90eXBlNV0KICAgICNzZXQgcHAyX3RsdiBbY2FsbCB0bHZfc2FwY2MgezQ5N2Y2ZWNhLTYyNzYtNDk5My1iZmViLTUzY2JiYmJhNmYwOH1dCiAgICBzZXQgcHAyX3Rsdl9zdHJsZW4gW3N0cmluZyBsZW5ndGggW3ZpcnR1YWwgbmFtZV1dCiAgICBzZXQgcHAyX3RsdiBbY2FsbCB0bHZfc2FwY2MgW3N0cmluZyByYW5nZSBbdmlydHVhbCBuYW1lXSBbZXhwciB7ICRwcDJfdGx2X3N0cmxlbiAtIDM2IH1dICRwcDJfdGx2X3N0cmxlbl1dCgogICAgIyBodHRwczovL2dpdGh1Yi5jb20vaGFwcm94eS9oYXByb3h5L2Jsb2IvZmZkZjZhMzJhNzQxM2Q1YmNmOTIyM2MzNTU2Yjc2NWI1ZTQ1NmE2OS9kb2MvcHJveHktcHJvdG9jb2wudHh0I0wzMzUKICAgICMgcHJveHkgcHJvdG9jb2wgdmVyc2lvbiAyIHNpZ25hdHVyZSAoMTIgYnl0ZXMpCiAgICBzZXQgcHAyX2hlYWRlcl9zaWduYXR1cmUgXHgwZFx4MGFceDBkXHgwYVx4MDBceDBkXHgwYVx4NTFceDU1XHg0OVx4NTRceDBhCgogICAgIyBodHRwczovL2dpdGh1Yi5jb20vaGFwcm94eS9oYXByb3h5L2Jsb2IvZmZkZjZhMzJhNzQxM2Q1YmNmOTIyM2MzNTU2Yjc2NWI1ZTQ1NmE2OS9kb2MvcHJveHktcHJvdG9jb2wudHh0I0wzNDAtTDM1OAogICAgIyBwcm94eSBwcm90b2NvbCB2ZXJzaW9uIGFuZCBjb21tYW5kIChceDIgLT4gcHJveHkgcHJvdG9jb2wgdmVyc2lvbiBpcyAyOyBceDEgLT4gY29tbWFuZCBpcyBQUk9YWSkKICAgICNzZXQgcHAyX2hlYWRlcl9ieXRlMTMgW2V4cHIgeygweDAyIDw8IDQpICsgMHgwMX1dCiAgICBzZXQgcHAyX2hlYWRlcl9ieXRlMTMgXHgyMQoKICAgICMgaHR0cHM6Ly9naXRodWIuY29tL2hhcHJveHkvaGFwcm94eS9ibG9iL2ZmZGY2YTMyYTc0MTNkNWJjZjkyMjNjMzU1NmI3NjViNWU0NTZhNjkvZG9jL3Byb3h5LXByb3RvY29sLnR4dCNMMzYwLUw0MzMKICAgICMgdHJhbnNwb3J0IHByb3RvY29sIGFuZCBhZGRyZXNzIGZhbWlseSAoXHgxIC0+IHRyYW5zcG9ydCBwcm90b2NvbCBpcyBTVFJFQU0vVENQOyBceDEgLT4gYWRkcmVzcyBmYW1pbHkgaXMgQUZfSU5FVC9pcHY0KQogICAgI3NldCBwcDJfaGVhZGVyX2J5dGUxNCBbZXhwciB7KDB4MDEgPDwgNCkgKyAweDAxfV0KICAgIHNldCBwcDJfaGVhZGVyX2J5dGUxNCBceDExCgogICAgIyBwcm94eSBwcm90b2NvbCBhZGRyCiAgICBzZXQgcHBfcHJveHlfYWRkciBbY2FsbCBwcm94eV9hZGRyXQoKICAgICMgcHJveHkgcHJvdG9jb2wgbGVuZ3RoCiAgICAjIG51bWJlciBvZiBmb2xsb3dpbmcgYnl0ZXMgcGFydCBvZiB0aGUgaGVhZGVyIGluIG5ldHdvcmsgZW5kaWFuIG9yZGVyCiAgICAjIGh0dHBzOi8vZ2l0aHViLmNvbS9oYXByb3h5L2hhcHJveHkvYmxvYi9mZmRmNmEzMmE3NDEzZDViY2Y5MjIzYzM1NTZiNzY1YjVlNDU2YTY5L2RvYy9wcm94eS1wcm90b2NvbC50eHQjTDQ0MS1MNDQ5CiAgICAjICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICRwcF9wcm94eV9hZGRyIGxlbiBpcyAxMgogICAgIyBOT1RFOiBpZiB0aGUgVExWIGxlbmd0aCBpcyBzdGF0aWMsIGl0IGNvdWxkIGJlIHByZS1jYWxjdWxhdGVkIHNpbWlsYXIgdG8gcHBfcHJveHlfYWRkcgogICAgc2V0IHBwMl9oZWFkZXJfYnl0ZTE1MTYgW2JpbmFyeSBmb3JtYXQgUyBbZXhwciB7MTIgKyBbc3RyaW5nIGxlbmd0aCAkcHAyX3Rsdl19XV0KCiAgICAjIGNvbnN0cnVjdCBwcDJfaGVhZGVyCiAgICBzZXQgcHAyX2hlYWRlciAke3BwMl9oZWFkZXJfc2lnbmF0dXJlfSR7cHAyX2hlYWRlcl9ieXRlMTN9JHtwcDJfaGVhZGVyX2J5dGUxNH0kcHAyX2hlYWRlcl9ieXRlMTUxNiRwcF9wcm94eV9hZGRyJHBwMl90bHYKCiAgICAjIGVuc3VyZSBwcCBoZWFkZXIgY29udmVyc2lvbiBvZiBceDAwIHRvIE5VTCgweDAwKSwgbm90IGMwODAKICAgIGJpbmFyeSBzY2FuICRwcDJfaGVhZGVyIEgqIHRtcAoKICAgICMgc2VuZCBwcAogICAgVENQOjpyZXNwb25kICRwcDJfaGVhZGVyCn0="}},"template":"generic"}}`
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
				AvailabilityZone: swag.String("abc"),
				CreatedAt:        time.Time{},
				Description:      "test",
				ID:               "test-service-id",
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
				"pool-test-service-id": Pool{
					Class:  "Pool",
					Label:  "service-test-service-id",
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
