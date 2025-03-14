// Copyright 2024 SAP SE
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

package agent

import (
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type logger struct{}

func (l *logger) Debug(msg string, args ...any) {
	log.Debugf(msg, args...)
}

func (l *logger) Error(msg string, args ...any) {
	log.Errorf(msg, args...)
}

func (l *logger) Info(msg string, args ...any) {
	log.Infof(msg, args...)
}

func (l *logger) Warn(msg string, args ...any) {
	log.Warnf(msg, args...)
}

func NewGoCronLogger() gocron.Logger {
	return &logger{}
}

type DebugMonitor struct{}

func (d *DebugMonitor) IncrementJob(id uuid.UUID, name string, tags []string, status gocron.JobStatus) {
}

func (d *DebugMonitor) RecordJobTiming(startTime, endTime time.Time, id uuid.UUID, name string, tags []string) {
}

func (d *DebugMonitor) RecordJobTimingWithStatus(startTime, endTime time.Time, id uuid.UUID, name string, tags []string, status gocron.JobStatus, err error) {
	log.WithFields(log.Fields{
		"id":       id,
		"tags":     tags,
		"status":   status,
		"error":    err,
		"duration": endTime.Sub(startTime),
	}).Debugf("Job %s", name)

	if err != nil {
		sentry.CaptureException(err)
	}
}
