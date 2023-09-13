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
