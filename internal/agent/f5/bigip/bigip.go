// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package bigip

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/f5devcentral/go-bigip"
	"github.com/sethvargo/go-retry"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal"
	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/errors"
)

type BigIP bigip.BigIP

func (b *BigIP) GetDeviceType() string {
	return "bigip"
}

func (b *BigIP) GetFailoverState() string {
	// Return device failover state.

	device, err := b.getDevice()
	if err == nil {
		return device.FailoverState
	}

	return "unknown"
}

func (b *BigIP) GetPartitions() ([]string, error) {
	// Return list of partitions on the device.
	partitions, err := (*bigip.BigIP)(b).TMPartitions()
	if err != nil {
		return nil, fmt.Errorf("GetPartitions: %w", err)
	}

	var partitionNames []string
	for _, partition := range partitions.TMPartitions {
		partitionNames = append(partitionNames, partition.Name)
	}

	return partitionNames, nil
}

func (b *BigIP) GetVLANs() ([]string, error) {
	// Return list of VLANs on the device.
	vlans, err := (*bigip.BigIP)(b).Vlans()
	if err != nil {
		return nil, fmt.Errorf("GetVLANs: %w", err)
	}

	var vlanNames []string
	for _, vlan := range vlans.Vlans {
		vlanNames = append(vlanNames, vlan.Name)
	}

	return vlanNames, nil
}

func (b *BigIP) GetRouteDomains() ([]string, error) {
	// Return list of route domains on the device.
	routeDomains, err := (*bigip.BigIP)(b).RouteDomains()
	if err != nil {
		return nil, fmt.Errorf("GetRouteDomains: %w", err)
	}

	var routeDomainNames []string
	for _, rd := range routeDomains.RouteDomains {
		routeDomainNames = append(routeDomainNames, rd.Name)
	}

	return routeDomainNames, nil
}

func (b *BigIP) GetSelfIPs() ([]string, error) {
	// Return list of self IPs on the device.
	selfIPs, err := (*bigip.BigIP)(b).SelfIPs()
	if err != nil {
		return nil, fmt.Errorf("GetSelfIPs: %w", err)
	}

	var selfIPNames []string
	for _, selfIP := range selfIPs.SelfIPs {
		selfIPNames = append(selfIPNames, selfIP.Name)
	}

	return selfIPNames, nil
}

func (b *BigIP) GetHostname() string {
	deviceURL, err := url.Parse(b.Host)
	if err != nil {
		panic(err)
	}
	if deviceURL.Hostname() != "" {
		return deviceURL.Hostname()
	}
	return b.Host
}

func (b *BigIP) PostAS3(as3 *as3.AS3, tenant string) error {
	data, err := json.MarshalIndent(as3, "", "  ")
	if err != nil {
		return err
	}

	if config.IsDebug() {
		fmt.Printf("-------------------> %s\n%s\n-------------------\n", b.Host, data)
	}

	r := retry.WithMaxDuration(config.Global.Agent.MaxDuration,
		retry.WithMaxRetries(config.Global.Agent.MaxRetries,
			retry.NewExponential(5*time.Second)))
	err = retry.Do(context.Background(), r, func(ctx context.Context) error {
		err, _, _ = (*bigip.BigIP)(b).PostAs3Bigip(string(data), tenant, "")
		return retry.RetryableError(err)
	})
	return err
}

type VcmpGuests struct {
	Guests []bigip.VcmpGuest `json:"items,omitempty"`
}

func (b *BigIP) getVCMPGuests() (*VcmpGuests, error) {
	var guests VcmpGuests

	req := &bigip.APIRequest{
		Method:      "get",
		URL:         "vcmp/guest",
		ContentType: "application/json",
	}
	resp, err := (*bigip.BigIP)(b).APICall(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, &guests)
	if err != nil {
		return nil, err
	}

	return &guests, nil
}

func (b *BigIP) EnsureBigIPSelfIP(name string, address string, segmentId int) error {
	log.WithFields(log.Fields{
		"hostname":  b.GetHostname(),
		"name":      name,
		"address":   address,
		"segmentId": segmentId}).
		Debug("EnsureBigIPSelfIP")

	selfIP, err := (*bigip.BigIP)(b).SelfIP(name)
	if err != nil && !strings.Contains(err.Error(), "was not found") {
		return err
	}

	if selfIP != nil {
		return nil
		// nothing to do
	}

	newSelfIP := bigip.SelfIP{
		Name:    name,
		Address: address, //fmt.Sprint(address, "%", segmentId, "/", mask),
		Vlan:    fmt.Sprint("/Common/vlan-", segmentId),
	}
	return (*bigip.BigIP)(b).CreateSelfIP(&newSelfIP)
}

func (b *BigIP) DeleteSelfIP(name string) error {
	_, err := (*bigip.BigIP)(b).SelfIP(name)
	if err != nil {
		return fmt.Errorf("DeleteSelfIP: SelfIP %s not found: %w", name, err)
	}

	return (*bigip.BigIP)(b).DeleteSelfIP(name)
}

func (b *BigIP) EnsureRouteDomain(segmentId int, parent *int) error {
	routeDomains, err := b.RouteDomains()
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

	return c.Update(b)
}

func (b *BigIP) DeleteRouteDomain(segmentId int) error {
	return (*bigip.BigIP)(b).DeleteRouteDomain(fmt.Sprintf("vlan-%d", segmentId))
}

func (b *BigIP) EnsureGuestVlan(segmentId int) error {
	guests, err := b.getVCMPGuests()
	if err != nil {
		return err
	}

	for _, guest := range guests.Guests {
		for _, deviceHost := range config.Global.Agent.Devices {
			if strings.HasSuffix(deviceHost, guest.Hostname) {
				vlanName := fmt.Sprintf("/Common/vlan-%d", segmentId)
				if slices.Contains(guest.Vlans, vlanName) {
					// found, nothing to do
					return nil
				}
				newGuest := bigip.VcmpGuest{Vlans: internal.Unique(append(guest.Vlans, vlanName))}
				return (*bigip.BigIP)(b).UpdateVcmpGuest(guest.Name, &newGuest)
			}
		}
	}
	return errors.ErrNoVCMPFound
}

func (b *BigIP) DeleteGuestVLAN(segmentId int) error {
	guests, err := b.getVCMPGuests()
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
				return (*bigip.BigIP)(b).UpdateVcmpGuest(guest.Name, &newGuest)
			}
		}
	}
	return errors.ErrNoVCMPFound
}

func (b *BigIP) SyncGuestVLANs(usedSegments map[int]string) error {
	guests, err := b.getVCMPGuests()
	if err != nil {
		return err
	}

	for _, guest := range guests.Guests {
		for _, deviceHost := range config.Global.Agent.Devices {
			if strings.HasSuffix(deviceHost, guest.Hostname) {
				var vlans []string
				for _, vlan := range guest.Vlans {
					var segmentId int
					if n, err := fmt.Sscanf(vlan, "/Common/vlan-%d", &segmentId); err != nil || n != 1 {
						// not a provisioned vlan
						vlans = append(vlans, vlan)
					} else if _, ok := usedSegments[segmentId]; ok {
						// vlan is still in use
						vlans = append(vlans, vlan)
					} else {
						log.WithFields(log.Fields{
							"host":  b.GetHostname(),
							"guest": guest.Hostname,
							"vlan":  vlan,
						}).Info("Found orphan vcmp guest vlan, deleting")
					}
				}
				newGuest := bigip.VcmpGuest{Vlans: vlans}
				return (*bigip.BigIP)(b).UpdateVcmpGuest(guest.Name, &newGuest)
			}
		}
	}

	return errors.ErrNoVCMPFound
}

func (b *BigIP) EnsureVLAN(segmentId int, mtu int) error {
	vlans, err := (*bigip.BigIP)(b).Vlans()
	if err != nil {
		return err
	}

	var existingVLAN *bigip.Vlan
	for _, vlan := range vlans.Vlans {
		if vlan.Tag == segmentId {
			existingVLAN = &vlan
			break
		}
	}

	if existingVLAN == nil {
		// Create vlan
		vlan := bigip.Vlan{
			Name: fmt.Sprintf("vlan-%d", segmentId),
			Tag:  segmentId,
			MTU:  mtu,
		}
		return (*bigip.BigIP)(b).CreateVlan(&vlan)
	}

	if existingVLAN.MTU != mtu {
		// update mtu
		log.WithFields(log.Fields{
			"host": b.GetHostname(),
			"vlan": existingVLAN.Name,
			"mtu":  mtu,
		}).Debug("Updating VLAN MTU")

		existingVLAN.MTU = mtu
		newVlan := &bigip.Vlan{
			MTU: mtu,
			Tag: existingVLAN.Tag,
		}
		if err = (*bigip.BigIP)(b).ModifyVlan(existingVLAN.Name, newVlan); err != nil {
			// bug in bigip, ignore
			if strings.Contains(err.Error(), "DAG adjustment is not supported on this platform") {
				return nil
			}
			return err
		}
	}

	return nil
}

func (b *BigIP) EnsureInterfaceVlan(segmentId int) error {
	name := fmt.Sprintf("vlan-%d", segmentId)

	vlanInterfaces, err := (*bigip.BigIP)(b).GetVlanInterfaces(name)
	if err != nil {
		return err
	}

	for _, iface := range vlanInterfaces.VlanInterfaces {
		if iface.Name == config.Global.Agent.PhysicalInterface {
			// found, nothing to do
			return nil
		}
	}

	return (*bigip.BigIP)(b).AddInterfaceToVlan(name, config.Global.Agent.PhysicalInterface, true)
}

func (b *BigIP) DeleteVLAN(segmentId int) error {
	return (*bigip.BigIP)(b).DeleteVlan(fmt.Sprintf("vlan-%d", segmentId))
}

func (b *BigIP) getDevice() (*bigip.Device, error) {
	devices, err := (*bigip.BigIP)(b).GetDevices()
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		if strings.HasSuffix(b.GetHostname(), device.Hostname) {
			return &device, nil
		}
	}
	return nil, fmt.Errorf("device %s not found", b.GetHostname())
}

func NewSession(uri *url.URL) (*BigIP, error) {
	// check for user
	user := uri.User.Username()
	if user == "" {
		var ok bool
		user, ok = os.LookupEnv("BIGIP_USER")
		if !ok {
			return nil, fmt.Errorf("BIGIP_USER required for host '%s'", uri.Hostname())
		}
	}

	// check for password
	password, ok := uri.User.Password()
	if !ok {
		password, ok = os.LookupEnv("BIGIP_PASSWORD")
		if !ok {
			return nil, fmt.Errorf("BIGIP_PASSWORD required for host '%s'", uri.Hostname())
		}
	}

	b := (*BigIP)(bigip.NewSession(&bigip.Config{
		Address:           uri.Hostname(),
		Username:          user,
		Password:          password,
		LoginReference:    "tmos",
		CertVerifyDisable: !config.Global.Agent.ValidateCert,
		ConfigOptions: &bigip.ConfigOptions{
			APICallTimeout: 60 * time.Second,
			TokenTimeout:   1200 * time.Second,
			APICallRetries: int(config.Global.Agent.MaxRetries),
		},
	}))

	device, err := b.getDevice()
	if err != nil {
		return nil, fmt.Errorf("failed to get device information: %w", err)
	}

	log.Infof("Connected to %s, %s (%s %s), %s", device.MarketingName, device.Name, device.Version,
		device.Edition, device.FailoverState)
	return b, nil
}
