// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package bigip

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/f5devcentral/go-bigip"
)

type routeDomain struct {
	bigip.RouteDomain
	Parent string `json:"parent,omitempty"`
}

type RouteDomains struct {
	RouteDomains []routeDomain `json:"items"`
}

func getRouteDomain(b *BigIP, name string) (*routeDomain, error) {
	var rd routeDomain
	req := &bigip.APIRequest{
		Method:      "get",
		URL:         fmt.Sprint("net/route-domain/", name),
		ContentType: "application/json",
	}

	resp, err := (*bigip.BigIP)(b).APICall(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, &rd)
	if err != nil {
		return nil, err
	}

	return &rd, nil
}

func (rd *routeDomain) Update(b *BigIP) error {
	m, err := json.Marshal(rd)
	if err != nil {
		return err
	}

	req := &bigip.APIRequest{
		Method:      "post",
		URL:         "net/route-domain",
		Body:        strings.TrimRight(string(m), "\n"),
		ContentType: "application/json",
	}

	if _, err := getRouteDomain(b, rd.Name); err == nil {
		// Modify instead
		req.Method = "put"
		req.URL = fmt.Sprint(req.URL, "/", rd.Name)
	}

	if _, err = (*bigip.BigIP)(b).APICall(req); err != nil {
		return err
	}
	return nil
}

func (b *BigIP) RouteDomains() (*RouteDomains, error) {
	var rds RouteDomains
	req := &bigip.APIRequest{
		Method:      "get",
		URL:         "net/route-domain",
		ContentType: "application/json",
	}

	resp, err := (*bigip.BigIP)(b).APICall(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, &rds)
	if err != nil {
		return nil, err
	}

	return &rds, nil
}
