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

package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/sapcc/go-bits/logg"
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	message := fmt.Sprintf("%s: %v", msg, data)
	switch level {
	case tracelog.LogLevelDebug:
		logg.Debug(message)
	case tracelog.LogLevelInfo:
		logg.Info(message)
	case tracelog.LogLevelError:
		logg.Error(message)
	default:
		logg.Other(level.String(), message)
	}
}
