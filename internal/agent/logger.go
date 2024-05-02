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
	"github.com/go-co-op/gocron/v2"
	"github.com/sirupsen/logrus"
)

type logger struct{}

func (l *logger) Debug(msg string, args ...any) {
	logrus.Debugf(msg, args...)
}

func (l *logger) Error(msg string, args ...any) {
	logrus.Errorf(msg, args...)
}

func (l *logger) Info(msg string, args ...any) {
	logrus.Infof(msg, args...)
}

func (l *logger) Warn(msg string, args ...any) {
	logrus.Warnf(msg, args...)
}

func NewGoCronLogger() gocron.Logger {
	return &logger{}
}
