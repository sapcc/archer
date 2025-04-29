// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"net/http"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/internal/config"
)

type PrometheusMonitor struct {
	processJobCount  *prometheus.CounterVec
	jobTiming        *prometheus.HistogramVec
	pendingJobTiming *prometheus.HistogramVec
}

func NewPrometheusMonitor() *PrometheusMonitor {
	processJobCount := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "archer_job_processed",
		Help: "The total number of processed jobs",
	}, []string{"name", "outcome", "id"})
	jobTiming := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "archer_job_timing",
		Help:    "The time taken to process a job",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 14),
	}, []string{"name", "id"})
	pendingJobTiming := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "archer_sync_job_timing",
		Help:    "The time taken to process a pending sync job",
		Buckets: prometheus.DefBuckets,
	}, nil)

	prometheus.DefaultRegisterer.MustRegister(processJobCount, jobTiming, pendingJobTiming)

	return &PrometheusMonitor{
		processJobCount:  processJobCount,
		jobTiming:        jobTiming,
		pendingJobTiming: pendingJobTiming,
	}
}

func (p PrometheusMonitor) IncrementJob(_ uuid.UUID, name string, tags []string, status gocron.JobStatus) {
	labels := prometheus.Labels{"name": name, "outcome": string(status), "id": ""}
	if len(tags) == 1 {
		labels["id"] = tags[0]
	}
	p.processJobCount.With(labels).Inc()
}

func (p PrometheusMonitor) RecordJobTiming(startTime, endTime time.Time, _ uuid.UUID, name string, tags []string) {
	if name == "PendingSyncLoop" {
		p.pendingJobTiming.WithLabelValues().Observe(endTime.Sub(startTime).Seconds())
	} else {
		labels := prometheus.Labels{"name": name, "id": ""}
		if len(tags) == 1 {
			labels["id"] = tags[0]
		}
		p.jobTiming.With(labels).Observe(endTime.Sub(startTime).Seconds())
	}
}

func PrometheusListenerThread() {
	if config.Global.Default.Prometheus {
		http.Handle("/metrics", promhttp.Handler())
		log.Infof("Serving prometheus metrics to %s/metrics", config.Global.Default.PrometheusListen)
		if err := http.ListenAndServe(config.Global.Default.PrometheusListen, nil); err != nil {
			log.Fatal(err.Error())
		}
	}
}
