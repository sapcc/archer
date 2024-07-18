/*
 *   Copyright 2022 SAP SE
 *
 *   Licensed under the Apache License, Version 2.0 (the "License");
 *   you may not use this file except in compliance with the License.
 *   You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 */

package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/go-api-declarations/cadf"
	"github.com/sapcc/go-bits/audittools"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/policy"
)

type AuditController struct {
	EventSink    chan<- cadf.Event
	observerUUID string
}

func NewAuditController() *AuditController {
	s := make(chan cadf.Event, 20)
	q := AuditController{
		EventSink:    s,
		observerUUID: audittools.GenerateUUID(),
	}
	rabbitmqQueueName := config.Global.Audit.QueueName
	transportURL, err := url.Parse(config.Global.Audit.TransportURL)
	if err != nil {
		panic(err)
	}

	go audittools.AuditTrail{
		EventSink: s,
		OnSuccessfulPublish: func() {
			log.Debug("Notification sent")
		},
		OnFailedPublish: func() {
			log.Debug("Notification failed")
		},
	}.Commit(context.Background(), *transportURL, rabbitmqQueueName)
	return &q
}

// AuditResponseWriter is a wrapper of regular ResponseWriter
type AuditResponseWriter struct {
	http.ResponseWriter
	controller *AuditController
	request    *http.Request
}

// AuditResource is an audittools.EventRenderer.
type AuditResource struct {
	project     string
	domain      string
	resource    string
	routeParams middleware.RouteParams
	id          string
}

// Render implements the audittools.EventRenderer interface.
func (a AuditResource) Render() cadf.Resource {
	id := a.id
	var attachments []cadf.Attachment
	for _, routeParam := range a.routeParams {
		attachments = append(attachments, cadf.Attachment{
			Name:    routeParam.Name,
			Content: routeParam.Value,
		})
		// Last route param is our target id
		id = routeParam.Value
	}
	res := cadf.Resource{
		TypeURI:     fmt.Sprintf("injector/%s", a.resource),
		ID:          id,
		ProjectID:   a.project,
		DomainID:    a.domain,
		Attachments: attachments,
	}

	return res
}

func (arw *AuditResponseWriter) WriteHeader(code int) {
	arw.ResponseWriter.WriteHeader(code)

	mr := middleware.MatchedRouteFrom(arw.request)
	resource := strings.Split(policy.RuleFromHTTPRequest(arw.request), ":")[0]
	uprinc := middleware.SecurityPrincipalFrom(arw.request)
	user := uprinc.(audittools.UserInfo)
	if user == nil {
		log.Error("Audit Middleware WriteHeader: missing token")
		return
	}

	p := audittools.EventParameters{
		Time:       time.Now(),
		Request:    arw.request,
		User:       user,
		ReasonCode: code,
		Action:     cadf.GetAction(arw.request.Method),
		Target: AuditResource{
			user.ProjectScopeUUID(),
			user.DomainScopeUUID(),
			resource,
			mr.Params,
			arw.Header().Get("X-Target-Id"),
		},
	}
	p.Observer.TypeURI = "service/injector"
	p.Observer.Name = "Archer"
	p.Observer.ID = arw.controller.observerUUID
	arw.controller.EventSink <- audittools.NewEvent(p)
}

func (ac *AuditController) NewAuditResponseWriter(w http.ResponseWriter, r *http.Request) *AuditResponseWriter {
	return &AuditResponseWriter{w, ac, r}
}

// AuditHandler provides the audit handling.
func (ac *AuditController) AuditHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			next.ServeHTTP(w, r)
			return
		}

		qrw := ac.NewAuditResponseWriter(w, r)
		next.ServeHTTP(qrw, r)
	})
}
