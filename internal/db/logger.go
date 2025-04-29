// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	logrus "github.com/jackc/pgx-logrus"
	"github.com/jackc/pgx/v5/tracelog"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
)

func GetTracer() *tracelog.TraceLog {
	logLevel := tracelog.LogLevelError
	if config.Global.Database.Trace {
		logLevel = tracelog.LogLevelDebug
	}
	return &tracelog.TraceLog{
		Logger:   logrus.NewLogger(log.StandardLogger()),
		LogLevel: logLevel,
	}
}
