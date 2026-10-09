// SPDX-FileCopyrightText: Copyright 2026 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package haproxy

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/agent/ni/models"
)

type FakeHaproxy struct {
	Running                bool
	AddInstanceReturnError error
}

func NewFakeHaproxy() *FakeHaproxy {
	return &FakeHaproxy{}
}

func (h *FakeHaproxy) CollectStats() {
	log.Debug("collecting haproxy stats (fake)")
}

func (h *FakeHaproxy) IsRunning(endpointID string) bool {
	log.Debugf("checking if haproxy is running for endpoint %s", endpointID)
	return h.Running
}

func (h *FakeHaproxy) AddInstance(injection *models.ServiceInjection) error {
	log.Debugf("adding instance %s", injection.Name)
	return h.AddInstanceReturnError
}

func (h *FakeHaproxy) RemoveInstance(endpointID string) error {
	log.Debugf("removing instance %s", endpointID)
	return nil
}

func (h *FakeHaproxy) Run(ctx context.Context) {
	log.Debug("running haproxy (fake)")
	<-ctx.Done()
	log.Debug("stopping haproxy (fake)")
}
