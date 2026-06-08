// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package neutron

import (
	"context"
	"fmt"
	"net/url"
	"runtime"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsbinding"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	log "github.com/sirupsen/logrus"

	aErrors "github.com/sapcc/archer/v2/internal/errors"
	"github.com/sapcc/archer/v2/models"
)

// PortListOpts builds a ports.ListOptsBuilder that filters by a set of port IDs (multiple id=
// query parameters), useful for fetching a known batch in one round trip.
type PortListOpts struct {
	IDs []string
}

// fixedIPCreateOpts is the request shape Neutron expects inside CreateOpts.FixedIPs (which
// is typed `any` upstream). gophercloud's own ports.FixedIPOpts has no JSON tags, so
// json.Marshal emits "SubnetID" / "IPAddress" — Neutron silently ignores those and rejects
// the request with "IP allocation requires subnet_id or ip_address". Use this type instead.
type fixedIPCreateOpts struct {
	SubnetID string `json:"subnet_id"`
}

// callerName returns the short name of the function `skip` frames above the caller, e.g.
// "EnsureNeutronSelfIPs". Used to tag log output without making each caller pass a string.
// Falls back to "?" if the runtime can't resolve the frame (shouldn't happen in practice).
func callerName(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "?"
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "?"
	}
	name := fn.Name()
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	return name
}

func (opts PortListOpts) ToPortListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	params := q.Query()
	for _, id := range opts.IDs {
		params.Add("id", id)
	}
	q = &url.URL{RawQuery: params.Encode()}
	return q.String(), err
}

// PortListOptsExt adds the PortBinding options to the base port ListOpts.
type PortListOptsExt struct {
	ports.ListOptsBuilder

	// The ID of the host where the port is allocated
	HostID string
}

// ToPortListQuery adds the PortBinding options to the base port list options.
func (opts PortListOptsExt) ToPortListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts.ListOptsBuilder)
	if err != nil {
		return "", err
	}

	params := q.Query()

	// From ListOpts.FixedIPs
	for _, _fixedIP := range opts.ListOptsBuilder.(ports.ListOpts).FixedIPs {
		if _fixedIP.IPAddress != "" {
			params.Add("fixed_ips", fmt.Sprintf("ip_address=%s", _fixedIP.IPAddress))
		}
		if _fixedIP.IPAddressSubstr != "" {
			params.Add("fixed_ips", fmt.Sprintf("ip_address_substr=%s", _fixedIP.IPAddressSubstr))
		}
		if _fixedIP.SubnetID != "" {
			params.Add("fixed_ips", fmt.Sprintf("subnet_id=%s", _fixedIP.SubnetID))
		}
	}

	if opts.HostID != "" {
		params.Add("binding:host_id", opts.HostID)
	}

	q = &url.URL{RawQuery: params.Encode()}
	return q.String(), err
}

// PortWithBinding pairs a Neutron port with its binding extension. Use it (with
// ports.ExtractPortsInto) to read binding:host_id alongside the rest of the port — needed when
// callers want to skip a no-op rebind.
type PortWithBinding struct {
	ports.Port
	portsbinding.PortsBindingExt
}

// UpdatePortBinding updates the binding:host_id of a Neutron port. Used after a service has
// been migrated to a different host so the port follows the workload.
func (n *NeutronClient) UpdatePortBinding(ctx context.Context, portID, hostID string) error {
	updateOpts := portsbinding.UpdateOptsExt{
		UpdateOptsBuilder: ports.UpdateOpts{},
		HostID:            &hostID,
	}
	if _, err := ports.Update(ctx, n.ServiceClient, portID, updateOpts).Extract(); err != nil {
		return err
	}
	n.invalidatePortCache()
	return nil
}

// invalidatePortCache purges every cached port listing. Called whenever a port is created,
// updated, or deleted — coarse but correct, since a single mutation can affect multiple
// queries (different filters, host_ids, etc.) and the working set is small. Nil-safe so
// tests that skip InitCache don't panic.
func (n *NeutronClient) invalidatePortCache() {
	if n.portCache != nil {
		n.portCache.Purge()
	}
}

// ListPorts returns ports matching opts (with hostID applied as binding:host_id when
// non-empty) along with their binding extension, served from a 10-minute LRU when possible.
// The cache key includes the full set of filters; mutations elsewhere call invalidatePortCache
// to drop stale entries. When InitCache hasn't been called the cache is bypassed.
func (n *NeutronClient) ListPorts(ctx context.Context, opts ports.ListOpts, hostID string) ([]PortWithBinding, error) {
	listOpts := PortListOptsExt{ListOptsBuilder: opts, HostID: hostID}
	key, err := listOpts.ToPortListQuery()
	if err != nil {
		return nil, err
	}
	if n.portCache != nil {
		if cached, ok := n.portCache.Get(key); ok {
			return cached, nil
		}
	}

	pages, err := ports.List(n.ServiceClient, listOpts).AllPages(ctx)
	if err != nil {
		return nil, err
	}
	var got []PortWithBinding
	if err := ports.ExtractPortsInto(pages, &got); err != nil {
		return nil, err
	}
	if n.portCache != nil {
		n.portCache.Add(key, got)
	}
	return got, nil
}

func (n *NeutronClient) GetPort(ctx context.Context, portId string) (*ports.Port, error) {
	return ports.Get(ctx, n.ServiceClient, portId).Extract()
}

func (n *NeutronClient) DeletePort(ctx context.Context, portId string) error {
	err := ports.Delete(ctx, n.ServiceClient, portId).ExtractErr()
	if err == nil {
		n.invalidatePortCache()
	}
	return err
}

func (n *NeutronClient) AllocateNeutronEndpointPort(ctx context.Context, target *models.EndpointTarget, endpoint *models.Endpoint,
	projectID string, host string, client *gophercloud.ServiceClient) (*ports.Port, error) {

	if target.Port != nil {
		port, err := ports.Get(ctx, client, target.Port.String()).Extract()
		if err != nil {
			return nil, err
		}

		if port.ProjectID != projectID {
			return nil, aErrors.ErrProjectMismatch
		}

		if len(port.FixedIPs) < 1 {
			return nil, aErrors.ErrMissingIPAddress
		}

		return port, nil
	}

	var fixedIPs []fixedIPCreateOpts
	if target.Network == nil {
		subnet, err := subnets.Get(ctx, client, target.Subnet.String()).Extract()
		if err != nil {
			return nil, err
		}

		fixedIPs = append(fixedIPs, fixedIPCreateOpts{SubnetID: subnet.ID})
		target.Network = new(strfmt.UUID(subnet.NetworkID))
	} else {
		network, err := n.GetNetwork(ctx, target.Network.String())
		if err != nil {
			return nil, err
		}
		if len(network.Subnets) == 0 {
			return nil, aErrors.ErrMissingSubnets
		}

		fixedIPs = append(fixedIPs, fixedIPCreateOpts{SubnetID: network.Subnets[0]})
	}

	// allocate neutron port
	port := portsbinding.CreateOptsExt{
		CreateOptsBuilder: ports.CreateOpts{
			Name:        fmt.Sprintf("endpoint-%s", endpoint.ID),
			DeviceOwner: "network:archer",
			DeviceID:    endpoint.ID.String(),
			NetworkID:   target.Network.String(),
			TenantID:    projectID,
			FixedIPs:    fixedIPs,
		},
		HostID: host,
	}

	res, err := ports.Create(ctx, n.ServiceClient, port).Extract()
	if err != nil {
		return nil, err
	}
	n.invalidatePortCache()
	// to ensure fresh segment cache
	n.RemoveFromCache(res.NetworkID)
	return res, nil
}

// ensurePortsOpts controls how ensurePorts reconciles a desired set of Neutron ports.
type ensurePortsOpts struct {
	// SubnetID is the subnet that newly-created ports get a fixed_ip on. Required.
	SubnetID string

	// Desired maps a caller-defined result key (e.g. "snat-0", "<deviceID>") to the
	// template port. Templates' Name + DeviceOwner identify existing ports; NetworkID,
	// TenantID, FixedIPs are derived from SubnetID at create time.
	Desired map[string]ports.Port

	// HostID, when non-empty, is set as binding:host_id on newly-created ports.
	HostID string

	// DeviceIDFilter narrows the existing-port lookup to ports with this device_id.
	// Required: both call sites (SelfIP scoped by subnet ID, SNAT scoped by service ID)
	// have a natural device_id, and using it lets the lookup span all hosts so a port
	// that migrated to a new host surfaces for rebinding instead of being duplicated.
	DeviceIDFilter string

	// RebindOnHostMismatch, when true, calls UpdatePortBinding on any matched existing
	// port whose binding:host_id != HostID. Used for SNAT to follow the service.
	RebindOnHostMismatch bool

	// DeleteExtras, when true, deletes existing ports owned by any DeviceOwner found
	// in Desired that don't match any template Name.
	DeleteExtras bool

	// DryRun skips all mutations; matched existing ports are still returned.
	DryRun bool
}

// ensurePorts reconciles a set of Neutron ports on opts.SubnetID against opts.Desired
// templates, matching existing ports by Name. The result map preserves the keys from
// opts.Desired.
//
// Behavior is governed entirely by ensurePortsOpts; see its field docs.
func (n *NeutronClient) ensurePorts(ctx context.Context, opts ensurePortsOpts) (map[string]*ports.Port, error) {
	logger := log.WithField("caller", callerName(2))
	logger.WithFields(log.Fields{
		"subnet":        opts.SubnetID,
		"desired":       len(opts.Desired),
		"device_id":     opts.DeviceIDFilter,
		"rebind":        opts.RebindOnHostMismatch,
		"delete_extras": opts.DeleteExtras,
		"dry_run":       opts.DryRun,
	}).Debug("EnsurePorts")

	// Collect the set of device_owners covered by templates so the list can be done in one
	// pass per owner. Today every caller passes a single owner.
	owners := map[string]struct{}{}
	for _, p := range opts.Desired {
		owners[p.DeviceOwner] = struct{}{}
	}

	var existing []PortWithBinding
	for owner := range owners {
		got, err := n.ListPorts(ctx, ports.ListOpts{
			DeviceOwner: owner,
			DeviceID:    opts.DeviceIDFilter,
		}, "")
		if err != nil {
			return nil, err
		}
		existing = append(existing, got...)
	}

	byName := make(map[string]PortWithBinding, len(existing))
	for i := range existing {
		byName[existing[i].Name] = existing[i]
	}

	result := make(map[string]*ports.Port, len(opts.Desired))
	matched := make(map[string]struct{}, len(opts.Desired))
	for key, tmpl := range opts.Desired {
		p, ok := byName[tmpl.Name]
		if !ok {
			continue
		}
		matched[tmpl.Name] = struct{}{}
		if opts.RebindOnHostMismatch && !opts.DryRun && p.HostID != opts.HostID {
			logger.WithFields(log.Fields{
				"port": p.ID, "from": p.HostID, "to": opts.HostID,
			}).Info("EnsurePorts: rebinding port to current host")
			if err := n.UpdatePortBinding(ctx, p.ID, opts.HostID); err != nil {
				return result, fmt.Errorf("UpdatePortBinding(%s): %w", p.ID, err)
			}
		}
		result[key] = new(p.Port)
	}

	// Create any missing ports. Subnet lookup is lazy so a delete-only call (count=0,
	// subnetID="") doesn't need a valid SubnetID.
	var subnet *subnets.Subnet
	for key, tmpl := range opts.Desired {
		if _, ok := matched[tmpl.Name]; ok {
			continue
		}
		if opts.DryRun {
			continue
		}
		if subnet == nil {
			s, err := n.GetSubnet(ctx, opts.SubnetID)
			if err != nil {
				return result, err
			}
			subnet = s
		}
		logger.WithFields(log.Fields{
			"network": subnet.NetworkID,
			"subnet":  opts.SubnetID,
			"name":    tmpl.Name,
		}).Info("EnsurePorts: allocating new port")
		createOpts := portsbinding.CreateOptsExt{
			CreateOptsBuilder: ports.CreateOpts{
				Name:        tmpl.Name,
				Description: tmpl.Description,
				DeviceOwner: tmpl.DeviceOwner,
				DeviceID:    tmpl.DeviceID,
				NetworkID:   subnet.NetworkID,
				TenantID:    subnet.TenantID,
				FixedIPs:    []fixedIPCreateOpts{{SubnetID: opts.SubnetID}},
			},
			HostID: opts.HostID,
		}
		p, err := ports.Create(ctx, n.ServiceClient, createOpts).Extract()
		if err != nil {
			return result, err
		}
		n.invalidatePortCache()
		result[key] = p
	}

	if opts.DeleteExtras && !opts.DryRun {
		for _, p := range existing {
			if _, ok := matched[p.Name]; ok {
				continue
			}
			logger.WithFields(log.Fields{
				"port": p.ID,
				"name": p.Name,
			}).Debug("EnsurePorts: deleting port")
			if err := n.DeletePort(ctx, p.ID); err != nil {
				return result, err
			}
		}
	}

	if subnet != nil && len(result) != len(byName) {
		// Membership changed; refresh the network cache so segment lookups see the new ports.
		n.RemoveFromCache(subnet.NetworkID)
	}
	return result, nil
}
