// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package as3

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

func getRouteDomain(big *BigIP, name string) (*routeDomain, error) {
	var rd routeDomain
	req := &bigip.APIRequest{
		Method:      "get",
		URL:         fmt.Sprint("net/route-domain/", name),
		ContentType: "application/json",
	}

	resp, err := big.APICall(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, &rd)
	if err != nil {
		return nil, err
	}

	return &rd, nil
}

func (rd *routeDomain) Update(big *BigIP) error {
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

	if _, err := getRouteDomain(big, rd.Name); err == nil {
		// Modify instead
		req.Method = "put"
		req.URL = fmt.Sprint(req.URL, "/", rd.Name)
	}

	if _, err = big.APICall(req); err != nil {
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

	resp, err := b.APICall(req)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(resp, &rds)
	if err != nil {
		return nil, err
	}

	return &rds, nil
}
