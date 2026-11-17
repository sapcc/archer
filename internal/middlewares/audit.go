// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	stdlog "log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/go-api-declarations/cadf"
	"github.com/sapcc/go-bits/audittools"
	"github.com/sapcc/go-bits/gopherpolicy"
	"github.com/sapcc/go-bits/logg"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/internal/policy"
)

// maxAuditBodySize caps the amount of request body we copy into the audit
// event to protect the audit backend from oversized payloads.
const maxAuditBodySize = 64 * 1024

// auditEventsTotal counts audit events by outcome. Dropped events never
// reach the message bus, so the audittools_* submission counters alone
// cannot reveal them (e.g. a deployment without Keystone auth drops every
// event and the bus counters stay at zero either way).
var auditEventsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "archer_audit_events_total",
	Help: "Total number of audit events by outcome.",
}, []string{"outcome"})

var registerAuditMetrics = sync.OnceFunc(func() {
	prometheus.MustRegister(auditEventsTotal)
})

// logrusLogWriter adapts stdlib log output (used by go-bits' logg package
// for audit publish/connection errors) to logrus, so those lines keep the
// same structured format as the rest of Archer's logs.
type logrusLogWriter struct{}

func (logrusLogWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\n")
	level := log.InfoLevel
	switch {
	case strings.HasPrefix(msg, "ERROR: "):
		level = log.ErrorLevel
		msg = strings.TrimPrefix(msg, "ERROR: ")
	case strings.HasPrefix(msg, "FATAL: "):
		// logg.Fatal() terminates the process after logging; ErrorLevel
		// avoids triggering logrus's own Fatal handling on top of that.
		level = log.ErrorLevel
		msg = strings.TrimPrefix(msg, "FATAL: ")
	case strings.HasPrefix(msg, "DEBUG: "):
		level = log.DebugLevel
		msg = strings.TrimPrefix(msg, "DEBUG: ")
	case strings.HasPrefix(msg, "INFO: "):
		msg = strings.TrimPrefix(msg, "INFO: ")
	}
	log.StandardLogger().Logf(level, "%s", msg)
	return len(p), nil
}

type AuditController struct {
	Auditor audittools.Auditor
}

func NewAuditController() (*AuditController, error) {
	// audittools reports publish and connection problems via go-bits' logg
	// package, which by default writes unstructured stdlib log lines that
	// log pipelines expecting logrus format miss. Route them through
	// logrus so audit errors are visible like any other archer error.
	logg.SetLogger(stdlog.New(logrusLogWriter{}, "", 0))

	registerAuditMetrics()

	auditor, err := audittools.NewAuditor(context.Background(), audittools.AuditorOpts{
		Observer: audittools.Observer{
			TypeURI: "service/injector",
			Name:    "Archer",
			ID:      audittools.GenerateUUID(),
		},
		QueueName:     config.Global.Audit.QueueName,
		ConnectionURL: config.Global.Audit.TransportURL,
	})
	if err != nil {
		// Return the error rather than panicking so cmd/archer-server can
		// decide how to handle audit-backend misconfiguration at boot.
		return nil, fmt.Errorf("initializing audit controller: %w", err)
	}

	return &AuditController{auditor}, nil
}

// AuditResponseWriter is a wrapper of regular ResponseWriter
type AuditResponseWriter struct {
	http.ResponseWriter
	controller  *AuditController
	request     *http.Request
	requestBody []byte
	written     atomic.Bool
}

// AuditResource is an audittools.EventRenderer.
type AuditResource struct {
	project     string
	domain      string
	resource    string
	routeParams middleware.RouteParams
	id          string
	requestBody []byte
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
	if len(a.requestBody) > 0 {
		// Feed the raw bytes as json.RawMessage so NewJSONAttachment emits
		// them verbatim (typeURI "mime:application/json") when valid JSON,
		// rather than re-marshaling and double-encoding the payload.
		if att, err := cadf.NewJSONAttachment("request_body", json.RawMessage(a.requestBody)); err == nil {
			attachments = append(attachments, att)
		} else {
			log.WithError(err).Warn("Audit Middleware: failed to build request_body attachment")
		}
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
	if !arw.written.CompareAndSwap(false, true) {
		return
	}
	arw.ResponseWriter.WriteHeader(code)

	// 4xx responses represent requests that never took effect — auth
	// rejected them, the resource was missing, or validation failed.
	// They're not state changes and shouldn't clutter the audit trail.
	// 2xx/3xx succeed and 5xx indicate an attempted state change that
	// errored — both are worth recording.
	if code >= 400 && code < 500 {
		auditEventsTotal.WithLabelValues("skipped_4xx").Inc()
		return
	}

	arw.recordAuditEvent(code, middleware.SecurityPrincipalFrom(arw.request))
}

// recordAuditEvent submits one audit event for a non-4xx response, or logs
// and counts the reason the event was dropped.
func (arw *AuditResponseWriter) recordAuditEvent(code int, principal any) {
	if principal == nil {
		// No authenticated principal (e.g. request rejected before the
		// auth middleware attached a token). Nothing to audit.
		log.Error("Audit Middleware WriteHeader: missing security principal")
		auditEventsTotal.WithLabelValues("dropped_no_principal").Inc()
		return
	}
	user, ok := principal.(audittools.UserInfo)
	if !ok {
		log.Errorf("Audit Middleware WriteHeader: principal is not audittools.UserInfo (got %T)", principal)
		auditEventsTotal.WithLabelValues("dropped_invalid_principal").Inc()
		return
	}
	// ProjectID/DomainID are only available from a *gopherpolicy.Token; other
	// UserInfo implementations (e.g. mocks, service tokens) still get audited,
	// just without scope IDs in the target resource.
	var projectID, domainID string
	if token, ok := principal.(*gopherpolicy.Token); ok {
		projectID = token.ProjectScopeUUID()
		domainID = token.DomainScopeUUID()
	}

	mr := middleware.MatchedRouteFrom(arw.request)
	var routeParams middleware.RouteParams
	if mr != nil {
		routeParams = mr.Params
	}
	resource := strings.Split(policy.RuleFromHTTPRequest(arw.request), ":")[0]

	arw.controller.Auditor.Record(audittools.EventParameters{
		Time:       time.Now(),
		Request:    arw.request,
		User:       user,
		ReasonCode: code,
		Action:     cadf.GetAction(arw.request.Method),
		Target: AuditResource{
			project:     projectID,
			domain:      domainID,
			resource:    resource,
			routeParams: routeParams,
			id:          arw.Header().Get("X-Target-Id"),
			requestBody: arw.requestBody,
		},
	})
	auditEventsTotal.WithLabelValues("recorded").Inc()
}

// Write implements http.ResponseWriter. The stdlib will invoke WriteHeader(200)
// on the first Write if the handler hasn't called it explicitly — we need to
// funnel that through our wrapper so the implicit 200 also records an audit
// event.
func (arw *AuditResponseWriter) Write(b []byte) (int, error) {
	if !arw.written.Load() {
		arw.WriteHeader(http.StatusOK)
	}
	return arw.ResponseWriter.Write(b)
}

func (ac *AuditController) NewAuditResponseWriter(w http.ResponseWriter, r *http.Request, body []byte) *AuditResponseWriter {
	return &AuditResponseWriter{
		ResponseWriter: w,
		controller:     ac,
		request:        r,
		requestBody:    body,
	}
}

// AuditHandler provides the audit handling.
func (ac *AuditController) AuditHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		body := captureRequestBody(r)
		qrw := ac.NewAuditResponseWriter(w, r, body)
		next.ServeHTTP(qrw, r)
	})
}

// captureRequestBody reads up to maxAuditBodySize bytes from a POST/PUT
// request body so it can be attached to the audit event, then restores the
// request body so downstream handlers can read it unchanged. Only JSON
// payloads are captured; other content types return nil to avoid dumping
// binary uploads into the audit trail.
func captureRequestBody(r *http.Request) []byte {
	if r.Body == nil || http.NoBody == r.Body {
		return nil
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		return nil
	}
	// Only capture JSON payloads. Use go-openapi's ContentType parser so we
	// handle charset params and malformed headers consistently with the
	// rest of the server. A missing Content-Type is treated as JSON since
	// that's what the Archer API expects for its POST/PUT routes.
	if r.Header.Get(runtime.HeaderContentType) != "" {
		mt, _, err := runtime.ContentType(r.Header)
		if err != nil || (mt != runtime.JSONMime && !strings.HasSuffix(mt, "+json")) {
			return nil
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).Warn("Audit Middleware: failed to read request body")
		return nil
	}
	_ = r.Body.Close()

	// Always restore the FULL body — the handler must see what the client
	// sent, regardless of how much of it we're willing to put in the audit
	// trail. Truncation only applies to the audit attachment below.
	r.Body = io.NopCloser(bytes.NewReader(body))

	if len(body) > maxAuditBodySize {
		log.WithFields(log.Fields{
			"path":       r.URL.Path,
			"body_size":  len(body),
			"audit_size": maxAuditBodySize,
		}).Warn("Audit Middleware: request body truncated for audit trail")
		return body[:maxAuditBodySize]
	}
	return body
}
