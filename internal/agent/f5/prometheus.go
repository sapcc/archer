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

package f5

import "github.com/prometheus/client_golang/prometheus"

var (
	processJobCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "archer_job_processed",
		Help: "The total number of processed jobs",
	}, []string{"model", "outcome"})
)

func initalizePrometheus() {
	prometheus.DefaultRegisterer.MustRegister(processJobCount)
	processJobCount.WithLabelValues("service", "unknown").Add(0)
}
