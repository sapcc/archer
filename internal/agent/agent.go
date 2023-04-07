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
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/IBM/pgxpoolprometheus"
	"github.com/f5devcentral/go-bigip"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/go-bits/jobloop"
	"github.com/sapcc/go-bits/logg"

	"github.com/sapcc/archer/internal/config"
)

type job struct {
	model string
	id    strfmt.UUID
}

type Agent struct {
	jobLoop  jobloop.Job
	jobQueue chan job
	pool     *pgxpool.Pool // thread safe
	neutron  *gophercloud.ServiceClient
	bigip    *bigip.BigIP
	cache    *lru.Cache[string, int]
}

func (a *Agent) Enqueue(job job) error {
	select {
	case a.jobQueue <- job:
		logg.Debug("Enqueued job %v", job)
		return nil
	default:
		return fmt.Errorf("Failed enque jobLoop %v", job)
	}
}

func NewAgent() *Agent {
	agent := new(Agent)

	jl := jobloop.CronJob{
		Metadata: jobloop.JobMetadata{
			ReadableName:    "Sync Loop",
			ConcurrencySafe: false,
			CounterOpts: prometheus.CounterOpts{
				Name: "archer_processed_events",
				Help: "The total number of processed events",
			},
			CounterLabels: nil,
		},
		Interval: 30 * time.Second,
		Task:     agent.PendingSyncLoop,
	}

	agent.jobLoop = jl.Setup(prometheus.DefaultRegisterer)
	agent.jobQueue = make(chan job, 100)

	var err error
	if agent.cache, err = lru.New[string, int](128); err != nil {
		logg.Fatal(err.Error())
	}

	if agent.pool, err = pgxpool.New(context.Background(), config.Global.Database.Connection); err != nil {
		logg.Fatal(err.Error())
	}
	dbConfig := agent.pool.Config()
	collector := pgxpoolprometheus.NewCollector(agent.pool, map[string]string{"db_name": dbConfig.ConnConfig.Database})
	prometheus.MustRegister(collector)

	logg.Info("Connected to PostgreSQL host=%s, max_conns=%d, health_check_period=%s",
		dbConfig.ConnConfig.Host, dbConfig.MaxConns, dbConfig.HealthCheckPeriod)

	agent.bigip, err = GetBigIPSession()
	if err != nil {
		logg.Fatal("BigIP session: %v", err)
	}

	devices, err := agent.bigip.GetDevices()
	if err != nil {
		logg.Fatal(err.Error())
	}
	for _, device := range devices {
		logg.Info("Connected to %s, %s (%s)", device.MarketingName, device.Name, device.Version)
		logg.Info("%v", device.ActiveModules)
	}

	if err := agent.ConnectToNeutron(); err != nil {
		logg.Fatal("While connecting to Neutron: %s", err.Error())
	}
	logg.Info("Connected to Neutron %s", agent.neutron.Endpoint)

	return agent
}

func (a *Agent) Run() error {
	if config.Global.Default.Prometheus {
		http.Handle("/metrics", promhttp.Handler())
		go a.PrometheusListenerThread()
	}
	go a.DBNotificationThread(context.Background())
	go a.WorkerThread()
	a.jobLoop.Run(context.Background(), nil)

	return nil
}

func (a *Agent) PendingSyncLoop(prometheus.Labels) error {
	var id, networkId strfmt.UUID
	var rows pgx.Rows
	var err error

	logg.Debug("Pending Sync")
	rows, err = a.pool.Query(context.Background(),
		`SELECT id FROM service WHERE status LIKE 'PENDING_%' AND host = $1`,
		config.HostName())
	if err != nil {
		return err
	}
	if _, err = pgx.ForEachRow(rows, []any{&id}, func() error {
		if err := a.Enqueue(job{model: "service"}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	rows, err = a.pool.Query(context.Background(),
		`SELECT endpoint.id, "target.network" 
              FROM endpoint
                    INNER JOIN service ON service.id = service_id AND service.status = 'AVAILABLE' 
              WHERE endpoint.status LIKE 'PENDING_%' AND host = $1`,
		config.HostName())
	if err != nil {
		return err
	}
	if _, err = pgx.ForEachRow(rows, []any{&id, &networkId}, func() error {
		if err := a.Enqueue(job{model: "endpoint", id: networkId}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (a *Agent) DBNotificationThread(ctx context.Context) {
	// Acquire one Connection for listen events
	conn, err := a.pool.Acquire(ctx)
	if err != nil {
		logg.Fatal(err.Error())
	}

	if _, err := conn.Exec(ctx, "listen sync"); err != nil {
		logg.Fatal(err.Error())
	}

	logg.Info("DBNotificationThread: Listening to sync notifications")

	for {
		_, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if !pgconn.Timeout(err) {
				logg.Fatal(err.Error())
			}
		}

		logg.Debug("Received Notification")
	}
}

func (a *Agent) PrometheusListenerThread() {
	logg.Info("Prometheus listening to %s", config.Global.Default.PrometheusListen)
	if err := http.ListenAndServe(config.Global.Default.PrometheusListen, nil); err != nil {
		logg.Fatal(err.Error())
	}
}
