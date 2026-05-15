// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package notifier

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/sapcc/archer/v2/models"
)

//go:embed templates/notification.tmpl
var embeddedTemplates embed.FS

type NotificationData struct {
	Type     string // "immediate" or "digest"
	Services []ServiceInfo
}

func (n NotificationData) TotalEndpoints() int {
	total := 0
	for _, s := range n.Services {
		total += len(s.Endpoints)
	}
	return total
}

type ServiceInfo struct {
	models.Service
	Endpoints []*models.Endpoint
}

type Templates struct {
	notification *template.Template
}

var funcMap = template.FuncMap{
	"since": func(t time.Time) string {
		return time.Since(t).Round(time.Minute).String()
	},
}

func LoadTemplates(overridePath string) (*Templates, error) {
	var tmpl *template.Template
	var err error

	if overridePath != "" {
		path := filepath.Join(overridePath, "notification.tmpl")
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("reading template override %s: %w", path, readErr)
		}
		tmpl, err = template.New("notification.tmpl").Funcs(funcMap).Parse(string(content))
	} else {
		tmpl, err = template.New("notification.tmpl").Funcs(funcMap).ParseFS(embeddedTemplates, "templates/notification.tmpl")
	}

	if err != nil {
		return nil, fmt.Errorf("parsing notification template: %w", err)
	}

	return &Templates{notification: tmpl}, nil
}

func (t *Templates) Render(data NotificationData) (string, error) {
	var buf bytes.Buffer
	if err := t.notification.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering notification template: %w", err)
	}
	return buf.String(), nil
}
