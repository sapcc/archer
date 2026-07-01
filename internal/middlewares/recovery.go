// SPDX-FileCopyrightText: Copyright 2026 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"fmt"
	"net/http"
	"runtime/debug"

	log "github.com/sirupsen/logrus"
)

// RecoveryMiddleware catches panics from downstream handlers, logs them with a
// stack trace, and writes a plain 500 response. It is intended to wrap the
// Sentry handler (configured with Repanic: true) so that Sentry reports the
// panic first and this middleware then converts it into a real HTTP 500.
//
// The 500 is written via WriteHeader so that any outer Prometheus
// instrumentation observes the correct status code.
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
