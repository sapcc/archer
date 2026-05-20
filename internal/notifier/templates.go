// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package notifier

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"text/template"
	"time"

	"github.com/sapcc/archer/v2/models"
)

//go:embed templates/subject.tmpl templates/notification.tmpl
var embeddedTemplates embed.FS

const (
	subjectName = "subject.tmpl"
	bodyName    = "notification.tmpl"
)

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
	tmpl *template.Template
}

var funcMap = template.FuncMap{
	"since": func(t time.Time) string {
		return time.Since(t).Round(time.Minute).String()
	},
}

func LoadTemplates(overridePath string) (*Templates, error) {
	root := template.New("notifier").Funcs(funcMap)

	var (
		parsed *template.Template
		err    error
	)
	if overridePath != "" {
		parsed, err = root.ParseFiles(
			filepath.Join(overridePath, subjectName),
			filepath.Join(overridePath, bodyName),
		)
	} else {
		parsed, err = root.ParseFS(embeddedTemplates,
			"templates/"+subjectName,
			"templates/"+bodyName,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("parsing notification templates: %w", err)
	}
	return &Templates{tmpl: parsed}, nil
}

func (t *Templates) render(name string, data NotificationData) (string, error) {
	var buf bytes.Buffer
	if err := t.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("rendering %s: %w", name, err)
	}
	return buf.String(), nil
}

func (t *Templates) RenderSubject(data NotificationData) (string, error) {
	return t.render(subjectName, data)
}

func (t *Templates) RenderBody(data NotificationData) (string, error) {
	return t.render(bodyName, data)
}
