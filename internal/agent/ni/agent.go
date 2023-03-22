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

	"github.com/IBM/pgxpoolprometheus"
	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/go-bits/logg"

	common "github.com/sapcc/archer/internal/agent"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
)

type Agent struct {
	jobQueue  *common.JobChan
	pool      *pgxpool.Pool // thread safe
	neutron   *gophercloud.ServiceClient
	haproxy   *HAProxyController
	serviceID strfmt.UUID
}

func (a *Agent) GetJobQueue() *common.JobChan {
	return a.jobQueue
}

func NewAgent() *Agent {
	config.ResolveHost()
	common.InitalizePrometheus()

	agent := new(Agent)
	jobQueue := make(common.JobChan, 100)
	agent.jobQueue = &jobQueue

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

	if config.Global.Default.AvailabilityZone != "" {
		logg.Info("Availability zone: %s", config.Global.Default.AvailabilityZone)
	}

	if err := agent.discoverService(); err != nil {
		logg.Fatal(err.Error())
	}

	common.RegisterAgent(agent.pool, "cp")
	return agent
}

func (a *Agent) Run() {
	go common.WorkerThread(context.Background(), a)
	go common.DBNotificationThread(context.Background(), a.pool, a.jobQueue)
	go common.PrometheusListenerThread()

	// initial run
	go func() {
		if err := a.InitialSync(context.Background()); err != nil {
			logg.Error(err.Error())
		}
	}()

	common.CronJob(a).Run(context.Background())
}

func (e *Agent) ProcessServices(ctx context.Context) error {
	// Cleanup pending delete services
	sql, args := db.Delete("service").
		Where("status = 'PENDING_DELETE'").
		Where("provider = 'cp'").
		MustSql()
	_, err := e.pool.Exec(ctx, sql, args...)
	return err
}

func (e *Agent) ProcessEndpoint(ctx context.Context, id strfmt.UUID) error {
	return pgx.BeginFunc(context.Background(), e.pool, func(tx pgx.Tx) error {
		var si ServiceInjection
		var err error

		sql, args := db.Select("e.id", "e.status", "ep.port_id", "ep.network", "ep.ip_address", "s.port").
			From("endpoint e").
			Join("endpoint_port ep ON ep.endpoint_id = e.id").
			Join("service s ON s.id = service_id").
			Where("e.id = ?", id).
			Suffix("FOR UPDATE OF e").
			MustSql()
		if err = pgxscan.Get(ctx, tx, &si, sql, args...); err != nil {
			return err
		}

		switch si.Status {
		case "PENDING_REJECTED":
			if err = e.DisableInjection(&si); err != nil {
				return err
			}
			sql, args = db.Update("endpoint").
				Set("status", "REJECTED").
				Set("updated_at", sq.Expr("NOW()")).
				Where("id = ?", si.Id).
				MustSql()
			if _, err = tx.Exec(ctx, sql, args...); err != nil {
				return err
			}
		case "PENDING_DELETE":
			if err = e.DisableInjection(&si); err != nil {
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
			if err = e.EnableInjection(&si); err != nil {
				return err
			}
		case "PENDING_UPDATE":
			fallthrough
		case "FAILED":
			fallthrough
		case "PENDING_CREATE":
			if err = e.EnableInjection(&si); err != nil {
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
			if _, err = tx.Exec(ctx, sql, args...); err != nil {
				return err
			}
		}
		return nil
	})
}

func (e *Agent) InitialSync(ctx context.Context) error {
	var id strfmt.UUID

	logg.Debug("initial sync")
	sql, args := db.Select("id").
		From("endpoint").
		Where("status != 'REJECTED'").
		Where("service_id = ?", e.serviceID).
		MustSql()
	rows, err := e.pool.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	_, err = pgx.ForEachRow(rows, []any{&id}, func() error {
		if err = e.jobQueue.Enqueue("endpoint", id); err != nil {
			return err
		}
		return nil
	})
	return err
}

func (e *Agent) PendingSyncLoop(ctx context.Context, _ prometheus.Labels) error {
	var id strfmt.UUID

	logg.Debug("pending sync scan")
	sql, args := db.Select("id").
		From("endpoint").
		Where("status IN ('PENDING_CREATE', 'PENDING_REJECTED', 'PENDING_DELETE', 'FAILED')").
		Where("service_id = ?", e.serviceID).
		MustSql()
	rows, err := e.pool.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	_, err = pgx.ForEachRow(rows, []any{&id}, func() error {
		if err = e.jobQueue.Enqueue("endpoint", id); err != nil {
			return err
		}
		return nil
	})
	return err
}
