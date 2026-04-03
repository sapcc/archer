// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"time"

	"github.com/sapcc/archer/client/agent"
)

var AgentOptions struct {
	AgentList `command:"list" description:"List Agents"`
	AgentShow `command:"show" description:"Show Agent details"`
}

type AgentList struct{}

func (*AgentList) Execute(_ []string) error {
	params := agent.NewGetAgentsParams()
	resp, err := ArcherClient.Agent.GetAgents(params, nil)
	if err != nil {
		return err
	}

	DefaultColumns = []string{"host", "availability_zone", "provider", "enabled", "services", "last_heartbeat"}

	type agentRow struct {
		Host             string `json:"host"`
		AvailabilityZone string `json:"availability_zone"`
		Provider         string `json:"provider"`
		Enabled          bool   `json:"enabled"`
		Physnet          string `json:"physnet"`
		Services         int64  `json:"services"`
		LastHeartbeat    string `json:"last_heartbeat"`
	}

	rows := make([]agentRow, 0, len(resp.Payload.Items))
	for _, a := range resp.Payload.Items {
		var az, physnet string
		if a.AvailabilityZone != nil {
			az = *a.AvailabilityZone
		}
		if a.Physnet != nil {
			physnet = *a.Physnet
		}
		var enabled bool
		if a.Enabled != nil {
			enabled = *a.Enabled
		}
		elapsed := time.Since(a.HeartbeatAt)
		rows = append(rows, agentRow{
			Host:             a.Host,
			AvailabilityZone: az,
			Provider:         a.Provider,
			Enabled:          enabled,
			Physnet:          physnet,
			Services:         a.Services,
			LastHeartbeat:    elapsed.Truncate(time.Second).String(),
		})
	}

	return WriteTable(rows)
}

type AgentShow struct {
	Positional struct {
		Host string `positional-arg-name:"host" description:"Agent hostname" required:"true"`
	} `positional-args:"true" required:"true"`
}

func (a *AgentShow) Execute(_ []string) error {
	params := agent.NewGetAgentsAgentHostParams().WithAgentHost(a.Positional.Host)
	resp, err := ArcherClient.Agent.GetAgentsAgentHost(params, nil)
	if err != nil {
		return err
	}

	return WriteTable(resp.GetPayload())
}

func init() {
	if _, err := Parser.AddCommand("agent", "Agents",
		"Agent Commands.", &AgentOptions); err != nil {
		panic(err)
	}
}
