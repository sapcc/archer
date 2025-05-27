// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package neutron

import (
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
)

type PortListOpts struct {
	IDs []string
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
