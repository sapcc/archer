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

import (
	"context"
	"net/http"
	"time"

	"github.com/IBM/pgxpoolprometheus"
	sq "github.com/Masterminds/squirrel"
	"github.com/go-co-op/gocron"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/utils/openstack/clientconfig"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	common "github.com/sapcc/archer/internal/agent"
	"github.com/sapcc/archer/internal/agent/f5/as3"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/internal/neutron"
)

type Agent struct {
	jobQueue *common.JobChan
	pool     db.PgxIface // thread safe
	neutron  *neutron.NeutronClient
	bigips   []*as3.BigIP
	vcmps    []*as3.BigIP
	bigip    *as3.BigIP // active target
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
		log.Fatal(err.Error())
	}
	connConfig.ConnConfig.Tracer = db.GetTracer()
	connConfig.ConnConfig.RuntimeParams["application_name"] = "archer-f5-agent"
	if agent.pool, err = pgxpool.NewWithConfig(context.Background(), connConfig); err != nil {
		log.Fatal(err.Error())
	}

	// install postgres status exporter
	dbConfig := agent.pool.Config()
	collector := pgxpoolprometheus.NewCollector(agent.pool, map[string]string{"db_name": dbConfig.ConnConfig.Database})
	prometheus.MustRegister(collector)
	log.Infof("Connected to PostgreSQL host=%s, max_conns=%d, health_check_period=%s",
		dbConfig.ConnConfig.Host, dbConfig.MaxConns, dbConfig.HealthCheckPeriod)

	// physical network/interface
	log.Infof("Physical Interface Mapping: physical_network=%s, interface=%s",
		config.Global.Agent.PhysicalNetwork, config.Global.Agent.PhysicalInterface)

	// bigips
	for _, url := range config.Global.Agent.Devices {
		var big *as3.BigIP
		big, err = as3.GetBigIPSession(url)
		if err != nil {
			log.Fatalf("BigIP session: %v", err)
		}
		agent.bigips = append(agent.bigips, big)
		if big.GetBigIPDevice(url).FailoverState == "active" {
			agent.bigip = big
		}
	}

	// vcmps
	for _, url := range config.Global.Agent.VCMPs {
		var big *as3.BigIP
		big, err = as3.GetBigIPSession(url)
		if err != nil {
			log.Fatalf("BigIP session: %v", err)
		}

		agent.vcmps = append(agent.vcmps, big)
	}

	authInfo := clientconfig.AuthInfo(config.Global.ServiceAuth)
	providerClient, err := clientconfig.AuthenticatedClient(&clientconfig.ClientOpts{
		AuthInfo: &authInfo})
	if err != nil {
		log.Fatal(err.Error())
	}

	if agent.neutron, err = neutron.ConnectToNeutron(providerClient); err != nil {
		log.Fatalf("While connecting to Neutron: %s", err.Error())
	}
	log.Infof("Connected to Neutron %s", agent.neutron.Endpoint)

	common.RegisterAgent(agent.pool, "tenant")
	return agent
}

func (a *Agent) Run() {
	go common.WorkerThread(context.Background(), a)
	go common.DBNotificationThread(context.Background(), a.pool, a.jobQueue)
	go common.PrometheusListenerThread()

	s := gocron.NewScheduler(time.UTC).SingletonMode()
	// sync pending services
	if _, err := s.
		Every(config.Global.Agent.PendingSyncInterval).
		DoWithJobDetails(a.PendingSyncLoop); err != nil {
		log.Fatal(err)
	}
	if _, err := s.Every(24).Hours().Do(a.cleanOrphanSelfIPs); err != nil {
		log.WithField("cron", "cleanupOrphanSelfIPs").Error(err)
	}
	s.StartBlocking()
}

func (a *Agent) PendingSyncLoop(job gocron.Job) error {
	var id strfmt.UUID
	var rows pgx.Rows
	var ret pgconn.CommandTag
	var err error

	log.WithFields(log.Fields{
		"run_count": job.RunCount(),
		"next_run":  time.Until(job.NextRun()),
	}).Debugf("pending sync loop")
	sql, args := db.Select("1").
		From("service").
		Where("provider = 'tenant'").
		Where(sq.Like{"status": "PENDING_%"}).
		Where("host = ?", config.Global.Default.Host).
		MustSql()
	ret, err = a.pool.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}

	if ret.RowsAffected() > 0 {
		if err := a.jobQueue.Enqueue("service", ""); err != nil {
			return err
		}
	}

	sql, args = db.Select("id").
		From("endpoint").
		Where(sq.And{
			sq.Like{"status": "PENDING_%"},
			db.Select("1").
				Prefix("EXISTS(").
				From("service").
				Where("service.id = endpoint.service_id").
				Where("service.provider = 'tenant'").
				Where("service.host = ?", config.Global.Default.Host).
				Suffix(")"),
		}).
		MustSql()
	rows, err = a.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	if _, err = pgx.ForEachRow(rows, []any{&id}, func() error {
		if err := a.jobQueue.Enqueue("endpoint", id); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (a *Agent) PrometheusListenerThread() {
	log.Infof("Serving prometheus metrics to %s/metrics", config.Global.Default.PrometheusListen)
	if err := http.ListenAndServe(config.Global.Default.PrometheusListen, nil); err != nil {
		log.Fatal(err.Error())
	}
}
