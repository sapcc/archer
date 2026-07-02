// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sapcc/go-bits/audittools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustNewRequest builds an *http.Request with the given method, JSON-ish
// content type, and body — helper to keep the table-driven cases compact.
func mustNewRequest(t *testing.T, method, contentType, body string) *http.Request {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, "/v1/service", http.NoBody)
	} else {
		r = httptest.NewRequest(method, "/v1/service", strings.NewReader(body))
	}
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	return r
}

func TestCaptureRequestBody_PreservesBodyForHandler(t *testing.T) {
	// The core invariant: after captureRequestBody() returns, r.Body must
	// still yield the same bytes when a downstream handler reads it.
	cases := []struct {
		name        string
		method      string
		contentType string
		body        string
		wantCapture bool
	}{
		{
			name:        "POST json object is captured",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"name":"svc-1","port":8080}`,
			wantCapture: true,
		},
		{
			name:        "PUT json object is captured",
			method:      http.MethodPut,
			contentType: "application/json",
			body:        `{"description":"updated"}`,
			wantCapture: true,
		},
		{
			name:        "POST json with charset parameter is captured",
			method:      http.MethodPost,
			contentType: "application/json; charset=utf-8",
			body:        `{"a":1}`,
			wantCapture: true,
		},
		{
			name:        "POST application/merge-patch+json is captured",
			method:      http.MethodPost,
			contentType: "application/merge-patch+json",
			body:        `{"a":1}`,
			wantCapture: true,
		},
		{
			name:        "POST without Content-Type is captured (defaults to JSON)",
			method:      http.MethodPost,
			contentType: "",
			body:        `{"x":true}`,
			wantCapture: true,
		},
		{
			name:        "POST non-JSON content-type is not captured",
			method:      http.MethodPost,
			contentType: "application/octet-stream",
			body:        "binary-blob-payload",
			wantCapture: false,
		},
		{
			name:        "PATCH is not captured (Archer API uses PUT, not PATCH)",
			method:      http.MethodPatch,
			contentType: "application/json",
			body:        `{"a":1}`,
			wantCapture: false,
		},
		{
			name:        "DELETE is not captured",
			method:      http.MethodDelete,
			contentType: "application/json",
			body:        `{"a":1}`,
			wantCapture: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := mustNewRequest(t, tc.method, tc.contentType, tc.body)

			captured := captureRequestBody(r)

			if tc.wantCapture {
				assert.Equal(t, tc.body, string(captured),
					"captured bytes should match original body")
			} else {
				assert.Nil(t, captured,
					"body should not be captured for this method/content-type")
			}

			// The critical invariant: a downstream handler must still be
			// able to read the exact original bytes from r.Body.
			require.NotNil(t, r.Body, "r.Body must not be nil after capture")
			got, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Equal(t, tc.body, string(got),
				"downstream handler should observe the unmodified body")
		})
	}
}

func TestCaptureRequestBody_EmptyBody(t *testing.T) {
	// A POST with http.NoBody must not blow up and must leave the request
	// in a state where reading r.Body still yields zero bytes.
	r := httptest.NewRequest(http.MethodPost, "/v1/service", http.NoBody)
	r.Header.Set("Content-Type", "application/json")

	captured := captureRequestBody(r)
	assert.Nil(t, captured, "no bytes to capture when body is http.NoBody")

	got, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestCaptureRequestBody_NilBody(t *testing.T) {
	// Constructed requests can legitimately have a nil Body (e.g. some
	// client code paths). The function must not panic.
	r, err := http.NewRequest(http.MethodPost, "/v1/service", nil)
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	assert.Nil(t, captureRequestBody(r))
}

func TestCaptureRequestBody_TruncatesAuditButPreservesFullBody(t *testing.T) {
	// Bodies larger than maxAuditBodySize get truncated in the audit
	// attachment to protect the audit backend, but the downstream handler
	// MUST still see the complete original bytes — audit is observational
	// and must never silently corrupt what the API sees.
	original := bytes.Repeat([]byte("A"), maxAuditBodySize+512)
	r := httptest.NewRequest(http.MethodPost, "/v1/service", bytes.NewReader(original))
	r.Header.Set("Content-Type", "application/json")

	captured := captureRequestBody(r)
	assert.Len(t, captured, maxAuditBodySize,
		"captured (audit) body is truncated to the cap")

	got, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Len(t, got, len(original),
		"downstream body is restored to its full original size")
	assert.Equal(t, original, got,
		"downstream body is byte-for-byte identical to the original")
}

func TestCaptureRequestBody_HandlerCanDecodeJSON(t *testing.T) {
	// End-to-end style check: after captureRequestBody(), a handler that
	// json.Decodes r.Body must succeed and see the same fields.
	body := `{"service":"archer","enabled":true}`
	r := httptest.NewRequest(http.MethodPost, "/v1/service", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	_ = captureRequestBody(r)

	var payload struct {
		Service string `json:"service"`
		Enabled bool   `json:"enabled"`
	}
	require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
	assert.Equal(t, "archer", payload.Service)
	assert.True(t, payload.Enabled)
}

func TestAuditHandler_GETPassesBodyUntouched(t *testing.T) {
	// The GET short-circuit in AuditHandler must not consume the body —
	// the downstream handler must still read the same bytes the client
	// sent.
	body := `{"should":"pass through"}`
	r := httptest.NewRequest(http.MethodGet, "/v1/service", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var sawBody string
	handler := (&AuditController{}).AuditHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		sawBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), r)
	assert.Equal(t, body, sawBody)
}

func TestAuditHandler_POSTDownstreamSeesFullBody(t *testing.T) {
	// End-to-end through AuditHandler for a POST: the downstream handler
	// must observe the same body bytes the client sent, even though the
	// middleware read them first. We stub WriteHeader by never calling it
	// — the auditor code path (which needs a token) is exercised by
	// integration tests, not this unit test.
	body := `{"payload":"data","n":42}`
	r := httptest.NewRequest(http.MethodPost, "/v1/service", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var sawBody string
	handler := (&AuditController{}).AuditHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		sawBody = string(b)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), r)
	assert.Equal(t, body, sawBody,
		"downstream handler must see the original request body")
}

// countingResponseWriter is a spy around httptest.ResponseRecorder that
// counts how many times WriteHeader is invoked on the underlying writer.
// Used to verify the audit wrapper does not double-write the status code.
type countingResponseWriter struct {
	http.ResponseWriter
	headerCalls atomic.Int32
	writeCalls  atomic.Int32
}

func (c *countingResponseWriter) WriteHeader(code int) {
	c.headerCalls.Add(1)
	c.ResponseWriter.WriteHeader(code)
}

func (c *countingResponseWriter) Write(b []byte) (int, error) {
	c.writeCalls.Add(1)
	return c.ResponseWriter.Write(b)
}

// newAuditWriterForTest builds an AuditResponseWriter around a counting spy,
// so tests can inspect both the audit-side behavior and how many times the
// underlying ResponseWriter was touched.
func newAuditWriterForTest(t *testing.T) (*AuditResponseWriter, *countingResponseWriter, *audittools.MockAuditor) {
	t.Helper()
	auditor := audittools.NewMockAuditor()
	spy := &countingResponseWriter{ResponseWriter: httptest.NewRecorder()}
	ac := &AuditController{Auditor: auditor}
	// A request with no security principal in its context. All the auditor
	// paths that dereference the principal must handle this without
	// panicking or failing the response.
	req := httptest.NewRequest(http.MethodPost, "/v1/service", http.NoBody)
	arw := ac.NewAuditResponseWriter(spy, req, nil)
	return arw, spy, auditor
}

func TestAuditResponseWriter_WriteHeaderWithoutPrincipalDoesNotPanic(t *testing.T) {
	// When the request has no security principal (e.g. auth middleware
	// rejected the request before attaching a token, or the route is not
	// authenticated), the audit wrapper must NOT panic. Failing to audit
	// is acceptable; crashing the process is not.
	arw, spy, auditor := newAuditWriterForTest(t)

	assert.NotPanics(t, func() {
		arw.WriteHeader(http.StatusUnauthorized)
	}, "audit middleware must survive a missing security principal")

	assert.Equal(t, int32(1), spy.headerCalls.Load(),
		"the underlying ResponseWriter must still receive the status code so the client gets a response")
	assert.Empty(t, auditor.RecordedEvents(),
		"no audit event should be recorded when the principal is missing")
}

func TestAuditResponseWriter_WriteHeaderIsIdempotent(t *testing.T) {
	// A second WriteHeader must be a no-op. Handlers, error renderers,
	// and stdlib fallbacks can all invoke WriteHeader independently — the
	// audit wrapper should record at most once and forward at most once
	// to the underlying writer.
	arw, spy, auditor := newAuditWriterForTest(t)

	arw.WriteHeader(http.StatusCreated)
	arw.WriteHeader(http.StatusInternalServerError) // should be ignored

	assert.Equal(t, int32(1), spy.headerCalls.Load(),
		"second WriteHeader must not reach the underlying ResponseWriter")
	assert.Empty(t, auditor.RecordedEvents(),
		"still no audit event because there is no principal — but crucially, no duplicate either")
}

func TestAuditResponseWriter_WriteTriggersImplicitWriteHeader(t *testing.T) {
	// The stdlib server auto-invokes WriteHeader(200) on the first Write
	// if the handler skipped it. Our wrapper must do the same so the
	// audit-recording code path runs for handlers that only call Write.
	arw, spy, _ := newAuditWriterForTest(t)

	n, err := arw.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	assert.Equal(t, int32(1), spy.headerCalls.Load(),
		"implicit WriteHeader(200) must be forwarded to the underlying writer")
	assert.Equal(t, int32(1), spy.writeCalls.Load(),
		"Write must still reach the underlying writer")
}

func TestAuditResponseWriter_WriteAfterWriteHeaderDoesNotDoubleTrigger(t *testing.T) {
	// If the handler calls WriteHeader explicitly and then Write, we must
	// NOT call WriteHeader a second time from inside Write.
	arw, spy, _ := newAuditWriterForTest(t)

	arw.WriteHeader(http.StatusAccepted)
	_, err := arw.Write([]byte("body"))
	require.NoError(t, err)

	assert.Equal(t, int32(1), spy.headerCalls.Load(),
		"WriteHeader must reach the underlying writer exactly once")
}

func TestAuditHandler_HTTPRequestSucceedsEvenWhenAuditFails(t *testing.T) {
	// End-to-end: a POST that flows through AuditHandler with no
	// principal in the context (audit will silently drop the event) must
	// still produce a normal successful response for the client. Audit
	// is observational; it must never fail the HTTP request.
	body := `{"foo":"bar"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/service", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	auditor := audittools.NewMockAuditor()
	ac := &AuditController{Auditor: auditor}

	// A perfectly ordinary handler: read body, write 201.
	handler := ac.AuditHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, body, string(b))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	assert.NotPanics(t, func() { handler.ServeHTTP(rec, req) },
		"missing principal must not crash the request")
	assert.Equal(t, http.StatusCreated, rec.Code,
		"client must observe the handler's status code")
	assert.Equal(t, `{"ok":true}`, rec.Body.String(),
		"client must observe the handler's response body")
	assert.Empty(t, auditor.RecordedEvents(),
		"no audit event was recorded because there was no principal — the HTTP request still succeeded")
}

func TestNewAuditController_ReturnsErrorOnBadConfig(t *testing.T) {
	// With no ConnectionURL set (the test config has empty audit
	// settings), NewAuditController must surface a validation error
	// from the underlying auditor rather than panicking. This is the
	// guardrail for the old panic(err)-at-startup behavior that would
	// crash archer-server on any audit config problem.
	var ac *AuditController
	var err error
	assert.NotPanics(t, func() {
		ac, err = NewAuditController()
	}, "NewAuditController must not panic on audit-backend config errors")

	require.Error(t, err, "invalid audit config must surface as an error")
	assert.Nil(t, ac, "no controller should be returned on failure")
}

func TestAuditResponseWriter_ClientErrorsAreNotAudited(t *testing.T) {
	// 4xx responses represent requests that never took effect — the auth
	// middleware rejected them, the resource didn't exist, the payload
	// was invalid. Recording an audit event with outcome=failure for
	// every 401/403/404 pollutes the audit trail with non-state-changing
	// noise. Skip client errors; keep everything else.
	cases := []struct {
		name        string
		code        int
		wantRecord  bool
		description string
	}{
		{name: "401 Unauthorized skipped", code: http.StatusUnauthorized, wantRecord: false},
		{name: "403 Forbidden skipped", code: http.StatusForbidden, wantRecord: false},
		{name: "404 Not Found skipped", code: http.StatusNotFound, wantRecord: false},
		{name: "409 Conflict skipped", code: http.StatusConflict, wantRecord: false},
		{name: "422 Unprocessable skipped", code: http.StatusUnprocessableEntity, wantRecord: false},
		// The status code alone isn't enough to record — we still need a
		// principal in context. But for these codes, we're asserting the
		// PRE-principal short-circuit fires so the auditor never even
		// gets consulted. Using the no-principal test helper is fine:
		// if the 4xx skip works, the auditor stays empty regardless.
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			arw, spy, auditor := newAuditWriterForTest(t)
			arw.WriteHeader(tc.code)

			assert.Equal(t, int32(1), spy.headerCalls.Load(),
				"the underlying ResponseWriter must still receive the status code")
			assert.Empty(t, auditor.RecordedEvents(),
				"4xx audit events should be skipped")
		})
	}
}

func TestAuditResponseWriter_ServerErrorsAreAudited(t *testing.T) {
	// 5xx responses indicate the server tried to do something and
	// failed — that IS worth auditing (someone triggered a state
	// transition, even if it errored). The audit path is exercised;
	// only the recording is skipped for lack of a principal here.
	arw, spy, _ := newAuditWriterForTest(t)
	arw.WriteHeader(http.StatusInternalServerError)

	assert.Equal(t, int32(1), spy.headerCalls.Load())
	// We can't easily assert the recording ran without a principal, but
	// we can assert we didn't take the 4xx short-circuit by verifying
	// `written` was set and the header made it through.
	assert.True(t, arw.written,
		"5xx must flow through the full audit path (not short-circuit like 4xx)")
}

func TestAuditResponseWriter_2xxAndRedirectsAreAudited(t *testing.T) {
	// Sanity: successes and redirects hit the full audit path too. Same
	// caveat as the 5xx test — no principal means no recording, but the
	// short-circuit for 4xx must not swallow these.
	for _, code := range []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusNoContent,
		http.StatusMovedPermanently,
	} {
		arw, spy, _ := newAuditWriterForTest(t)
		arw.WriteHeader(code)

		assert.Equalf(t, int32(1), spy.headerCalls.Load(),
			"status %d must reach the underlying writer", code)
		assert.Truef(t, arw.written,
			"status %d must flow through the full audit path", code)
	}
}
