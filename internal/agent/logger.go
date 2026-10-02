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

// argsToFields converts gocron's key-value args pairs into logrus fields.
func argsToFields(args []any) log.Fields {
	fields := make(log.Fields, len(args)/2)
	for i := 0; i+1 < len(args); i += 2 {
		if key, ok := args[i].(string); ok {
			fields[key] = args[i+1]
		}
	}
	return fields
}

func (l *logger) Debug(msg string, args ...any) {
	log.WithFields(argsToFields(args)).Debug(msg)
}

func (l *logger) Error(msg string, args ...any) {
	log.WithFields(argsToFields(args)).Error(msg)
}

func (l *logger) Info(msg string, args ...any) {
	log.WithFields(argsToFields(args)).Info(msg)
}

func (l *logger) Warn(msg string, args ...any) {
	log.WithFields(argsToFields(args)).Warn(msg)
}

func NewGoCronLogger() gocron.Logger {
	return &logger{}
}

type DebugMonitor struct{}

func (d *DebugMonitor) IncrementJob(id uuid.UUID, name string, tags []string, status gocron.JobStatus) {
	log.Debugf("Job %s status changed: id=%s, tags=%v, status=%s", name, id, tags, status)
}

func (d *DebugMonitor) RecordJobTiming(startTime, endTime time.Time, id uuid.UUID, name string, tags []string) {
}

func (d *DebugMonitor) RecordJobTimingWithStatus(startTime, endTime time.Time, id uuid.UUID, name string, tags []string, status gocron.JobStatus, err error) {
	logWithFields := log.WithFields(log.Fields{
		"job_id":       id,
		"endpoint_ids": tags,
		"status":       status,
		"duration":     endTime.Sub(startTime),
	})

	if err != nil {
		logWithFields.WithError(err).Errorf("Job %s", name)
		sentry.CaptureException(err)
		return
	}

	logWithFields.Debugf("Job %s", name)
}
