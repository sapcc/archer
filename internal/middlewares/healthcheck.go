// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package middlewares

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

// HealthCheckMiddleware provides the GET /healthcheck endpoint.
func HealthCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if r.URL.Path == "/healthcheck" && r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("ok")); err != nil {
				log.Error("Error replying health check")
			}
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
