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
	"github.com/IBM/pgxpoolprometheus"
	"github.com/f5devcentral/go-bigip"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/go-bits/jobloop"
	"github.com/sapcc/go-bits/logg"
	"net/http"

	"github.com/sapcc/archer/internal/agent/as3"
	"github.com/sapcc/archer/internal/agent/neutron"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
)

type Agent struct {
	jobLoop  jobloop.Job
	jobQueue *JobChan
	pool     *pgxpool.Pool // thread safe
	neutron  *gophercloud.ServiceClient
	bigip    *bigip.BigIP
	cache    *lru.Cache[string, int]
}

func NewAgent() *Agent {
	config.ResolveHost()
	initalizePrometheus()
	agent := new(Agent)

	jl := jobloop.CronJob{
		Metadata: jobloop.JobMetadata{
			ReadableName:    "Sync Loop",
			ConcurrencySafe: false,
			CounterOpts: prometheus.CounterOpts{
				Name: "archer_sync_counter",
				Help: "The total number of sync events",
			},
			CounterLabels: nil,
		},
		Interval: config.Global.Agent.PendingSyncInterval,
		Task:     agent.PendingSyncLoop,
	}

	agent.jobLoop = jl.Setup(nil)
	jobQueue := make(JobChan, 100)
	agent.jobQueue = &jobQueue

	var err error
	if agent.cache, err = lru.New[string, int](128); err != nil {
		logg.Fatal(err.Error())
	}

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

	agent.bigip, err = as3.GetBigIPSession()
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

	authInfo := clientconfig.AuthInfo(config.Global.ServiceAuth)
	providerClient, err := clientconfig.AuthenticatedClient(&clientconfig.ClientOpts{
		AuthInfo: &authInfo})
	if err != nil {
		logg.Fatal(err.Error())
	}

	if agent.neutron, err = neutron.ConnectToNeutron(providerClient); err != nil {
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
	go a.WorkerThread(context.Background())
	go func() {
		// run pending sync scan immediately
		_ = a.PendingSyncLoop(nil, nil)
	}()
	a.jobLoop.Run(context.Background(), nil)

	return nil
}

func (a *Agent) PendingSyncLoop(context.Context, prometheus.Labels) error {
	var id, networkId strfmt.UUID
	var rows pgx.Rows
	var ret pgconn.CommandTag
	var err error

	logg.Debug("pending sync scan")
	ret, err = a.pool.Exec(context.Background(),
		`SELECT 1 FROM service WHERE status LIKE 'PENDING_%' AND host = $1`,
		config.Global.Default.Host)
	if err != nil {
		return err
	}

	if ret.RowsAffected() > 0 {
		if err := a.jobQueue.Enqueue(job{model: "service"}); err != nil {
			return err
		}
	}

	rows, err = a.pool.Query(context.Background(),
		`SELECT endpoint.id, "target.network" 
              FROM endpoint
                    INNER JOIN service ON service.id = service_id AND service.status = 'AVAILABLE' 
              WHERE endpoint.status LIKE 'PENDING_%' AND host = $1`,
		config.Global.Default.Host)
	if err != nil {
		return err
	}
	if _, err = pgx.ForEachRow(rows, []any{&id, &networkId}, func() error {
		if err := a.jobQueue.Enqueue(job{model: "service", id: id}); err != nil {
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

	if _, err := conn.Exec(ctx, "listen service; listen endpoint;"); err != nil {
		logg.Fatal(err.Error())
	}

	logg.Info("DBNotificationThread: Listening to service and endpoint notifications")

	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if !pgconn.Timeout(err) {
				logg.Fatal(err.Error())
			}
		}

		logg.Debug("received notification, channel=%s, payload=%s", notification.Channel, notification.Payload)
		j := job{model: notification.Channel}
		if notification.Payload != "" {
			j.id = strfmt.UUID(notification.Payload)
		}
		if err := a.jobQueue.Enqueue(j); err != nil {
			logg.Error(err.Error())
		}
	}
}

func (a *Agent) PrometheusListenerThread() {
	logg.Info("Serving prometheus metrics to %s/metrics", config.Global.Default.PrometheusListen)
	if err := http.ListenAndServe(config.Global.Default.PrometheusListen, nil); err != nil {
		logg.Fatal(err.Error())
	}
}

func (a *Agent) WorkerThread(ctx context.Context) {
	for job := range *a.jobQueue {
		var err error
		logg.Debug("received message %v", job)

		switch job.model {
		case "service":
			if err = a.ProcessServices(ctx); err != nil {
				logg.Error(err.Error())
			}
		case "endpoint":
			if err = a.ProcessEndpoint(ctx, job.id); err != nil {
				logg.Error(err.Error())
			}
		}

		outcome := "success"
		if err != nil {
			outcome = "failure"
		}
		processJobCount.WithLabelValues(job.model, outcome).Inc()
	}
}
