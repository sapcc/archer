// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package notifier

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
)

type CampfireRequest struct {
	ProjectID  string   `json:"project_id,omitempty"`
	Recipients []string `json:"recipients,omitempty"`
	Subject    string   `json:"subject"`
	MimeType   string   `json:"mime_type"`
	MailText   string   `json:"mail_text"`
}

type CampfireClient struct {
	url            string
	providerClient *gophercloud.ProviderClient
}

func NewCampfireClient(url string, providerClient *gophercloud.ProviderClient) *CampfireClient {
	return &CampfireClient{
		url:            url,
		providerClient: providerClient,
	}
}

func (c *CampfireClient) SendEmail(ctx context.Context, req *CampfireRequest) error {
	opts := &gophercloud.RequestOpts{
		JSONBody: req,
		OkCodes:  []int{200},
	}

	resp, err := c.providerClient.Request(ctx, "POST", c.url, opts)
	if err != nil {
		return fmt.Errorf("sending campfire request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}
