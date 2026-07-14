// SPDX-FileCopyrightText: Copyright 2026 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	log "github.com/sirupsen/logrus"
)

// clientClosedRequest is the (unofficial but widely used) HTTP status code for
// a request whose context was canceled by the client or by hitting the
// request deadline. It signals that the failure is not a server fault.
const clientClosedRequest = 499

// RecoveryMiddleware catches panics from downstream handlers, logs them with a
// stack trace, and writes a plain 500 response. It is intended to wrap the
// Sentry handler (configured with Repanic: true) so that Sentry reports the
// panic first and this middleware then converts it into a real HTTP 500.
//
// The 500 is written via WriteHeader so that any outer Prometheus
// instrumentation observes the correct status code.
//
// Panics caused solely by a canceled or timed-out request context (e.g. a
// handler panicking on the error returned when a FOR UPDATE lock wait exceeds
// the request deadline) are a client-side condition, not a server fault. Those
// are converted to HTTP 499 and logged at info level without a stack trace.
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}

			err, ok := rec.(error)
			if !ok {
				err = fmt.Errorf("%v", rec)
			}

			// A canceled/timed-out request context is not a server bug.
			// Avoid the error-level log + stack trace and report 499.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				log.WithFields(log.Fields{
					"method": r.Method,
					"path":   r.URL.Path,
				}).Infof("request context cancelled: %v", err)

				w.WriteHeader(clientClosedRequest)
				return
			}

			log.WithFields(log.Fields{
				"method": r.Method,
				"path":   r.URL.Path,
				"stack":  string(debug.Stack()),
			}).Errorf("panic recovered: %v", err)

			// Bail out if the response has already been partially sent —
			// once headers are on the wire there is nothing useful we can do.
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error\n"))
		}()

		next.ServeHTTP(w, r)
	})
}
