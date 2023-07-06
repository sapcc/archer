// Copyright 2023 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agent

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/config"
)

var (
	processJobCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "job_processed",
		Help: "The total number of processed jobs",
	}, []string{"model", "outcome"})
)

func InitalizePrometheus() {
	prometheus.DefaultRegisterer.MustRegister(processJobCount)
	processJobCount.WithLabelValues("service", "unknown").Add(0)
}

func RunPrometheus() {
	if config.Global.Default.Prometheus {
		http.Handle("/metrics", promhttp.Handler())
		logg.Info("Serving prometheus metrics to %s/metrics", config.Global.Default.PrometheusListen)
		go prometheusListenerThread()
	}
}

func prometheusListenerThread() {
	if err := http.ListenAndServe(config.Global.Default.PrometheusListen, nil); err != nil {
		logg.Fatal(err.Error())
	}
}
