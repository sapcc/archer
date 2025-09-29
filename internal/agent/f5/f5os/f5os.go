// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package f5os

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/sethvargo/go-retry"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/config"
)

// token represents a JWT token used for authentication with F5OS.
type token string

// Valid checks if the token is valid by decoding it and checking the expiry time.
func (t token) Valid() bool {
	jwt := strings.Split(string(t), ".")
	if len(jwt) != 3 {
		return false
	}

	decoded, err := base64.RawURLEncoding.DecodeString(jwt[1])
	if err != nil {
		log.Errorf("error decoding JWT payload: %v", err)
		return false
	}

	var payload map[string]any
	if err = json.Unmarshal(decoded, &payload); err != nil {
		log.Errorf("error unmarshalling JWT payload: %v", err)
		return false
	}

	exp, ok := payload["exp"].(float64)
	if !ok {
		log.Error("JWT payload does not contain 'exp' field")
		return false
	}

	// Convert the expiry time from seconds since epoch to time.Time
	expiry := time.Unix(int64(exp), 0)
	return time.Now().Before(expiry)
}

// F5OS represents a session with an F5OS device.
type F5OS struct {
	client   *http.Client
	user     string
	password string
	token    token
	uri      *url.URL
}

func (f *F5OS) newRequest(method, path string, body any) *http.Request {
	buf := new(bytes.Buffer)
	// If body is provided, encode it to JSON
	if body != nil {
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			log.Errorf("error encoding request body for %s %s: %v", method, path, err)
			return nil
		}
	}

	// Create a new request with the provided method, path, and body
	req, err := http.NewRequest(method, f.uri.ResolveReference(&url.URL{Path: path}).String(), buf)
	if err != nil {
		log.Errorf("error creating request for %s %s: %v", method, path, err)
		return nil
	}

	// Set the default headers
	req.Header.Set("Content-Type", "application/yang-data+json")
	req.Header.Set("Accept", "application/yang-data+json")
	return req
}

// apiCall performs an API call to the F5OS device using the provided request.
func (f *F5OS) apiCall(req *http.Request, v any) error {
	// use configured retry backoff for all API calls
	backoff := retry.WithMaxDuration(config.Global.Agent.MaxDuration,
		retry.WithMaxRetries(config.Global.Agent.MaxRetries,
			retry.NewFibonacci(2*time.Second)))

	return retry.Do(req.Context(), backoff, func(ctx context.Context) error {
		// Set the Authorization header
		if f.token.Valid() {
			req.Header.Set("X-Auth-Token", string(f.token))
			req.Header.Set("Authorization", string("Bearer "+f.token))
		} else {
			req.SetBasicAuth(f.user, f.password)
		}

		var resp *http.Response
		var err error
		resp, err = f.client.Do(req)
		if err != nil {
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled for %s: %w", req.URL.Redacted(), ctx.Err())
			default:
				return retry.RetryableError(err)
			}
		}
		defer func() { _ = resp.Body.Close() }()

		log.WithFields(log.Fields{
			"method": req.Method,
			"url":    req.URL.Redacted(),
			"status": resp.StatusCode,
		}).Debug("F5OS API call")

		if resp.StatusCode == http.StatusUnauthorized {
			log.Warningf("unauthorized request to %s, token may be expired", req.URL.Redacted())
			// If the token is invalid, we need to re-authenticate
			f.token = "" // Clear the token to force re-authentication
			return retry.RetryableError(fmt.Errorf("unauthorized request to %s", req.URL.Redacted()))
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return retry.RetryableError(fmt.Errorf("unexpected status code for %s %d: %s",
				req.URL, resp.StatusCode, body))
		}

		// Update the token if the response contains a new one
		if tokenHeader := resp.Header.Get("X-Auth-Token"); tokenHeader != "" && string(f.token) != tokenHeader {
			f.token = token(tokenHeader)
			if !f.token.Valid() {
				log.Warningf("received expired token from %s", req.URL.Redacted())
				return retry.RetryableError(fmt.Errorf("received expired token from %s", req.URL.Redacted()))
			}
		}

		if err = json.NewDecoder(resp.Body).Decode(v); err != nil {
			if errors.Is(err, io.EOF) {
				// If the body is empty, we can return nil
				return nil
			}
			return fmt.Errorf("error decoding response for %s: %w", req.URL.Redacted(), err)
		}
		return nil
	})
}

func (f *F5OS) PostAS3(_ *as3.AS3, _ string) error {
	panic("not supported for F5OS")
}

func (f *F5OS) GetDeviceType() string {
	return "f5os"
}

func (f *F5OS) GetHostname() string {
	return f.uri.Hostname()
}

func (f *F5OS) GetFailoverState() string {
	return "unknown" // F5OS does not have a failover state like BIG-IP
}

func (f *F5OS) GetPartitions() ([]string, error) {
	panic("not supported for F5OS")
}

func (f *F5OS) GetVLANs() ([]string, error) {
	req := f.newRequest("GET", "api/data/openconfig-vlan:vlans", nil)
	var vlans OpenConfigVlans
	if err := f.apiCall(req, &vlans); err != nil {
		return nil, fmt.Errorf("error fetching VLANs from %s: %w", f.uri.Host, err)
	}

	var vlanNames []string
	for _, vlan := range vlans.OpenconfigVlanVlans.Vlan {
		if vlan.Config.Name != "" {
			vlanNames = append(vlanNames, vlan.Config.Name)
		}
	}

	return vlanNames, nil
}

func (f *F5OS) GetRouteDomains() ([]string, error) {
	panic("not supported for F5OS")
}

func (f *F5OS) GetSelfIPs() ([]string, error) {
	panic("not supported for F5OS")
}

func (f *F5OS) EnsureVLAN(segmentId int, _ int) error {
	newVlan := OpenConfigVlan{
		OpenconfigVlanVlan: []vlan{
			{
				VlanId: segmentId,
				Config: struct {
					VlanId int    `json:"vlan-id"`
					Name   string `json:"name"`
				}{VlanId: segmentId, Name: fmt.Sprintf("vlan-%d", segmentId)},
			},
		},
	}
	path := fmt.Sprintf("api/data/openconfig-vlan:vlans/vlan=%d", segmentId)
	req := f.newRequest("PUT", path, newVlan)
	if err := f.apiCall(req, nil); err != nil {
		return fmt.Errorf("error ensuring VLAN %d on %s: %w", segmentId, f.uri.Host, err)
	}

	return nil
}

func (f *F5OS) EnsureInterfaceVlan(segmentID int) error {
	path := "api/data/openconfig-interfaces:interfaces/interface=" +
		config.Global.Agent.PhysicalInterface +
		"/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans"

	var vlans trunkVlans
	if err := f.apiCall(f.newRequest("GET", path, nil), &vlans); err != nil {
		return fmt.Errorf("error fetching trunk VLANs: %w", err)
	}

	if slices.Contains(vlans.OpenconfigVlanTrunkVlans, segmentID) {
		log.WithFields(log.Fields{
			"host":      f.GetHostname(),
			"segmentID": segmentID,
			"interface": config.Global.Agent.PhysicalInterface,
		}).Debug("Trunk VLAN already exists")
		return nil
	}

	vlans.OpenconfigVlanTrunkVlans = append(vlans.OpenconfigVlanTrunkVlans, segmentID)

	req := f.newRequest("PUT", path, vlans)
	if err := f.apiCall(req, nil); err != nil {
		return fmt.Errorf("error ensuring interface VLAN %d on %s: %w", segmentID, f.uri.Host, err)
	}
	return nil
}

func (f *F5OS) EnsureGuestVlan(segmentId int) error {
	tenant, err := f.getTenant()
	if err != nil {
		return fmt.Errorf("error getTenant: %w", err)
	}

	if slices.Contains(tenant.State.Vlans, segmentId) {
		log.WithField("host", f.GetHostname()).
			Debugf("Guest VLAN %d already exists on tenant %s", segmentId, tenant.Name)
		return nil
	}

	path := fmt.Sprintf("api/data/f5-tenants:tenants/tenant=%s/config/vlans", tenant.Name)
	existingVlans := F5TenantVlans{
		F5TenantsVlans: tenant.State.Vlans,
	}

	if slices.Contains(existingVlans.F5TenantsVlans, segmentId) {
		return nil
	}

	existingVlans.F5TenantsVlans = append(existingVlans.F5TenantsVlans, segmentId)
	req := f.newRequest("PUT", path, existingVlans)
	if err = f.apiCall(req, nil); err != nil {
		return fmt.Errorf("error ensuring guest VLAN %d on tenant %s: %w", segmentId, tenant.Name, err)
	}
	return nil
}

func (f *F5OS) EnsureRouteDomain(_ int, _ *int) error {
	panic("not supported for F5OS")
}

func (f *F5OS) EnsureBigIPSelfIP(_, _ string, _ int) error {
	panic("not supported for F5OS")
}

func (f *F5OS) getTenant() (*F5TenantsTenant, error) {
	req := f.newRequest("GET", "api/data/f5-tenants:tenants/tenant", nil)
	var tenants F5Tenants
	if err := f.apiCall(req, &tenants); err != nil {
		return nil, fmt.Errorf("error fetching tenants from %s: %w", f.uri.Host, err)
	}

	for _, tenant := range tenants.F5TenantsTenant {
		for _, deviceHost := range config.Global.Agent.Devices {
			if strings.Contains(deviceHost, tenant.Name) {
				return &tenant, nil
			}
		}
	}

	return nil, fmt.Errorf("no tenant found for host %s", f.GetHostname())
}

func (f *F5OS) SyncGuestVLANs(usedSegments map[int]string) error {
	// Fetch all vlans to distinguish between management and guest vlans
	VlanNames, err := f.GetVLANs()
	if err != nil {
		return fmt.Errorf("error fetching VLANs from %s: %w", f.uri.Host, err)
	}

	var updatedVlans []int
	tenant, err := f.getTenant()
	if err != nil {
		return fmt.Errorf("error getTenant: %w", err)
	}

	for _, vid := range tenant.State.Vlans {
		if !slices.Contains(VlanNames, fmt.Sprintf("vlan-%d", vid)) {
			// it is a management VLAN, keep it
			updatedVlans = append(updatedVlans, vid)
			continue
		}

		// is a guest VLAN, check if it is used
		if _, ok := usedSegments[vid]; ok {
			// VLAN is still used, keep it
			updatedVlans = append(updatedVlans, vid)
		} else {
			// VLAN is not used anymore, remove it
			log.WithField("host", f.GetHostname()).Debugf("Removing unused guest VLAN %d from tenant %s",
				vid, tenant.Name)
		}
	}

	log.WithField("host", f.GetHostname()).Debugf("Syncing guest VLANs on tenant %s: old=%d, new=%d",
		tenant.Name, len(tenant.State.Vlans), len(updatedVlans))

	path := fmt.Sprintf("api/data/f5-tenants:tenants/tenant=%s/config/vlans", tenant.Name)
	vlans := F5TenantVlans{updatedVlans}
	req := f.newRequest("PUT", path, vlans)
	if err = f.apiCall(req, nil); err != nil {
		return fmt.Errorf("error syncing guest VLANs on tenant %s: %w", tenant.Name, err)
	}
	return nil
}

func (f *F5OS) DeleteVLAN(segmentId int) error {
	// We need to remove the interface VLAN first if it exists
	if err := f.DeleteInterfaceVlan(segmentId); err != nil {
		return fmt.Errorf("error deleting interface VLAN %d on %s: %w", segmentId, f.uri.Host, err)
	}

	path := fmt.Sprintf("api/data/openconfig-vlan:vlans/vlan=%d", segmentId)
	req := f.newRequest("DELETE", path, nil)
	if err := f.apiCall(req, nil); err != nil {
		return fmt.Errorf("error deleting VLAN %d on %s: %w", segmentId, f.uri.Host, err)
	}
	return nil
}

func (f *F5OS) DeleteSelfIP(_ string) error {
	panic("not supported for F5OS")
}

func (f *F5OS) DeleteInterfaceVlan(segmentID int) error {
	path := "api/data/openconfig-interfaces:interfaces/interface=" +
		config.Global.Agent.PhysicalInterface +
		"/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans"

	var vlans trunkVlans
	if err := f.apiCall(f.newRequest("GET", path, nil), &vlans); err != nil {
		return fmt.Errorf("error fetching trunk VLANs: %w", err)
	}

	b := make([]int, 0, len(vlans.OpenconfigVlanTrunkVlans)-1)
	for _, v := range vlans.OpenconfigVlanTrunkVlans {
		if v != segmentID {
			b = append(b, v)
		}
	}
	if len(b) == len(vlans.OpenconfigVlanTrunkVlans) {
		return nil // nothing to delete
	}
	vlans.OpenconfigVlanTrunkVlans = b

	req := f.newRequest("PUT", path, vlans)
	if err := f.apiCall(req, nil); err != nil {
		return fmt.Errorf("error deleting interface VLAN %d on %s: %w", segmentID, f.uri.Host, err)
	}
	return nil
}

func (f *F5OS) DeleteGuestVLAN(segmentId int) error {
	tenant, err := f.getTenant()
	if err != nil {
		return fmt.Errorf("error getTenant: %w", err)
	}

	path := fmt.Sprintf("api/data/f5-tenants:tenants/tenant=%s/config/vlans", tenant.Name)
	var existingVlans F5TenantVlans
	req := f.newRequest("GET", path, &existingVlans)
	if err = f.apiCall(req, &existingVlans); err != nil {
		return fmt.Errorf("error fetching existing VLANs for tenant %s: %w", tenant.Name, err)
	}

	for i, v := range existingVlans.F5TenantsVlans {
		if v == segmentId {
			existingVlans.F5TenantsVlans = append(existingVlans.F5TenantsVlans[:i],
				existingVlans.F5TenantsVlans[i+1:]...)
			break
		}
	}

	req = f.newRequest("PUT", path, existingVlans)
	if err = f.apiCall(req, nil); err != nil {
		return fmt.Errorf("error deleting guest VLAN %d on tenant %s: %w", segmentId, tenant.Name, err)
	}
	return nil
}

func (f *F5OS) DeleteRouteDomain(_ int) error {
	panic("not supported for F5OS")
}

func NewSession(uri *url.URL) (*F5OS, error) {
	// Initialize F5OS http client
	client := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !config.Global.Agent.ValidateCert,
			},
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
		},
	}

	user := uri.User.Username()
	if user == "" {
		var ok bool
		user, ok = os.LookupEnv("BIGIP_USER")
		if !ok {
			return nil, fmt.Errorf("BIGIP_USER required for host '%s'", uri.Hostname())
		}
	}

	password, ok := uri.User.Password()
	if !ok {
		password, ok = os.LookupEnv("BIGIP_PASSWORD")
		if !ok {
			return nil, fmt.Errorf("BIGIP_PASSWORD required for host '%s'", uri.Hostname())
		}
	}

	f5os := &F5OS{
		uri:      uri,
		client:   client,
		user:     user,
		password: password,
	}

	// Try to identify the device type
	req := f5os.newRequest("GET",
		"api/data/openconfig-platform:components/component=platform/state", nil)
	var platformState OpenconfigPlatformState
	if err := f5os.apiCall(req, &platformState); err != nil {
		return nil, fmt.Errorf("error fetching platform state from %s: %w", uri.Host, err)
	}

	log.Infof("Connected to F5OS %s, %s (%s)",
		platformState.OpenconfigPlatformState.Description,
		f5os.GetHostname(),
		platformState.OpenconfigPlatformState.PartNo,
	)

	return f5os, nil
}
