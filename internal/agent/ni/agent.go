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

package ni

import (
	"context"
	"net/http"
	"time"

	"github.com/IBM/pgxpoolprometheus"
	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/go-bits/jobloop"
	"github.com/sapcc/go-bits/logg"

	common "github.com/sapcc/archer/internal/agent"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
)

type Agent struct {
	pool      *pgxpool.Pool // thread safe
	neutron   *gophercloud.ServiceClient
	haproxy   *HAProxyController
	serviceID strfmt.UUID
}

func NewAgent() *Agent {
	config.ResolveHost()
	agent := new(Agent)

	var err error

	// Connect to database
	connConfig, err := pgxpool.ParseConfig(config.Global.Database.Connection)
	if err != nil {
		logg.Fatal(err.Error())
	}
	if config.Global.Database.Trace {
		logger := tracelog.TraceLog{
			Logger:   db.NewLogger(),
			LogLevel: tracelog.LogLevelDebug,
		}
		connConfig.ConnConfig.Tracer = &logger
	}
	if agent.pool, err = pgxpool.NewWithConfig(context.Background(), connConfig); err != nil {
		logg.Fatal(err.Error())
	}

	// install postgres status exporter
	dbConfig := agent.pool.Config()
	collector := pgxpoolprometheus.NewCollector(agent.pool, map[string]string{"db_name": dbConfig.ConnConfig.Database})
	prometheus.MustRegister(collector)
	logg.Info("Connected to PostgreSQL host=%s, max_conns=%d, health_check_period=%s",
		dbConfig.ConnConfig.Host, dbConfig.MaxConns, dbConfig.HealthCheckPeriod)

	if err := agent.SetupOpenStack(); err != nil {
		logg.Fatal(err.Error())
	}
	logg.Info("Connected to Neutron host=%s", agent.neutron.Endpoint)

	if err := agent.discoverService(); err != nil {
		logg.Fatal(err.Error())
	}

	common.RegisterAgent(agent.pool, "cp")
	return agent
}

func (a *Agent) Run() error {
	if config.Global.Default.Prometheus {
		http.Handle("/metrics", promhttp.Handler())
		go a.PrometheusListenerThread()
	}

	// initial run
	if err := a.periodicInjectionLoop(context.Background(), nil); err != nil {
		logg.Error(err.Error())
	}

	// periodic sync
	jl := a.EventTranslationJob(prometheus.DefaultRegisterer)
	jl.Run(context.Background())
	return nil
}

func (e *Agent) EventTranslationJob(registerer prometheus.Registerer) jobloop.Job {
	return (&jobloop.CronJob{
		Metadata: jobloop.JobMetadata{
			ReadableName:    "periodic_injection_loop",
			ConcurrencySafe: false,
			CounterOpts:     prometheus.CounterOpts{Name: "periodic_injection_loop"},
			CounterLabels:   nil,
		},
		Interval: 240 * time.Second,
		Task:     e.periodicInjectionLoop,
	}).Setup(registerer)
}

func (e *Agent) periodicInjectionLoop(ctx context.Context, _ prometheus.Labels) error {
	sql, args, err := db.Select("e.id", "e.status", "ep.port_id", "ep.network", "ep.ip_address", "s.port").
		From("endpoint e").
		Join("endpoint_port ep ON ep.endpoint_id = e.id").
		Join("service s ON s.id = service_id").
		Where("s.id = ?", e.serviceID).
		ToSql()
	if err != nil {
		return err
	}

	var items []*ServiceInjection
	if err = pgxscan.Select(ctx, e.pool, &items, sql, args...); err != nil {
		return err
	}

	for _, si := range items {
		switch si.Status {
		case "PENDING_REJECTED":
			fallthrough
		case "PENDING_DELETE":
			if err = e.DisableInjection(si); err != nil {
				return err
			}
			sql, args, err = db.Delete("endpoint").
				Where("id = ?", si.Id).
				ToSql()
			if err != nil {
				return err
			}
			if _, err = e.pool.Exec(ctx, sql, args...); err != nil {
				return err
			}
		case "AVAILABLE":
			if err = e.EnableInjection(si); err != nil {
				return err
			}
		case "PENDING_UPDATE":
			fallthrough
		case "FAILED":
			fallthrough
		case "PENDING_CREATE":
			if err = e.EnableInjection(si); err != nil {
				return err
			}
			sql, args, err = db.Update("endpoint").
				Set("status", "AVAILABLE").
				Set("updated_at", sq.Expr("NOW()")).
				Where("id = ?", si.Id).
				ToSql()
			if err != nil {
				return err
			}
			if _, err = e.pool.Exec(ctx, sql, args...); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Agent) PrometheusListenerThread() {
	logg.Info("Serving prometheus metrics to %s/metrics", config.Global.Default.PrometheusListen)
	if err := http.ListenAndServe(config.Global.Default.PrometheusListen, nil); err != nil {
		logg.Fatal(err.Error())
	}
}
