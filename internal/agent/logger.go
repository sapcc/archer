// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

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
	logWithFields := log.WithFields(log.Fields{
		"id":       id,
		"tags":     tags,
		"status":   status,
		"duration": endTime.Sub(startTime),
	})

	if err != nil {
		logWithFields.WithError(err).Errorf("Job %s", name)
		sentry.CaptureException(err)
		return
	}

	logWithFields.Debugf("Job %s", name)
}
