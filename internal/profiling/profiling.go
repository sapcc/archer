// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

// Package profiling exposes net/http/pprof profiling endpoints on a dedicated
// listener. It uses go-bits/pprofapi, which wraps net/http/pprof behind a
// gorilla/mux router so that the pprof handlers are never registered on
// http.DefaultServeMux (which Archer's Prometheus listeners rely on).
package profiling

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/sapcc/go-bits/httpapi"
	"github.com/sapcc/go-bits/httpapi/pprofapi"

	"github.com/sapcc/archer/v2/internal/config"
)

// Start launches the pprof listener in a goroutine if profiling is enabled.
// Access is restricted to requests originating from localhost.
func Start() {
	if !config.Global.Default.Pprof {
		return
	}

	handler := httpapi.Compose(
		pprofapi.API{IsAuthorized: pprofapi.IsRequestFromLocalhost},
		httpapi.WithoutLogging(),
	)
	srv := &http.Server{
		Addr:    config.Global.Default.PprofListen,
		Handler: handler,
	}

	go func() {
		log.Infof("Serving pprof at http://%s/debug/pprof/", config.Global.Default.PprofListen)
		if err := srv.ListenAndServe(); err != nil {
			log.Warnf("pprof listener exited: %v", err)
		}
	}()
}
