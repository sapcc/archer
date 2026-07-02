// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/sapcc/go-api-declarations/cadf"
	"github.com/sapcc/go-bits/audittools"
	"github.com/sapcc/go-bits/gopherpolicy"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/config"
	"github.com/sapcc/archer/v2/internal/policy"
)

// maxAuditBodySize caps the amount of request body we copy into the audit
// event to protect the audit backend from oversized payloads.
const maxAuditBodySize = 64 * 1024

type AuditController struct {
	Auditor audittools.Auditor
}

func NewAuditController() (*AuditController, error) {
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
	written     bool
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
	if arw.written {
		// Second call would produce a duplicate audit event; the
		// underlying ResponseWriter also warns/no-ops on a second header
		// write, so silently drop the redundant record.
		return
	}
	arw.written = true
	arw.ResponseWriter.WriteHeader(code)

	// 4xx responses represent requests that never took effect — auth
	// rejected them, the resource was missing, or validation failed.
	// They're not state changes and shouldn't clutter the audit trail.
	// 2xx/3xx succeed and 5xx indicate an attempted state change that
	// errored — both are worth recording.
	if code >= 400 && code < 500 {
		return
	}

	uprinc := middleware.SecurityPrincipalFrom(arw.request)
	if uprinc == nil {
		// No authenticated principal (e.g. request rejected before the
		// auth middleware attached a token). Nothing to audit.
		log.Error("Audit Middleware WriteHeader: missing security principal")
		return
	}
	user, ok := uprinc.(audittools.UserInfo)
	if !ok {
		log.Errorf("Audit Middleware WriteHeader: principal is not audittools.UserInfo (got %T)", uprinc)
		return
	}
	token, ok := uprinc.(*gopherpolicy.Token)
	if !ok {
		log.Errorf("Audit Middleware WriteHeader: principal is not *gopherpolicy.Token (got %T)", uprinc)
		return
	}

	mr := middleware.MatchedRouteFrom(arw.request)
	resource := strings.Split(policy.RuleFromHTTPRequest(arw.request), ":")[0]

	arw.controller.Auditor.Record(audittools.EventParameters{
		Time:       time.Now(),
		Request:    arw.request,
		User:       user,
		ReasonCode: code,
		Action:     cadf.GetAction(arw.request.Method),
		Target: AuditResource{
			project:     token.ProjectScopeUUID(),
			domain:      token.DomainScopeUUID(),
			resource:    resource,
			routeParams: mr.Params,
			id:          arw.Header().Get("X-Target-Id"),
			requestBody: arw.requestBody,
		},
	})
}

// Write implements http.ResponseWriter. The stdlib will invoke WriteHeader(200)
// on the first Write if the handler hasn't called it explicitly — we need to
// funnel that through our wrapper so the implicit 200 also records an audit
// event.
func (arw *AuditResponseWriter) Write(b []byte) (int, error) {
	if !arw.written {
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
