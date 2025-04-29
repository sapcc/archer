// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package f5

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/IBM/pgxpoolprometheus"
	sq "github.com/Masterminds/squirrel"
	"github.com/go-co-op/gocron/v2"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
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
	"github.com/sapcc/archer/models"
)

type Agent struct {
	scheduler gocron.Scheduler
	pool      db.PgxIface // thread safe
	neutron   *neutron.NeutronClient
	bigips    []*as3.BigIP
	vcmps     []*as3.BigIP
	bigip     *as3.BigIP // active target
}

func (a *Agent) GetScheduler() gocron.Scheduler {
	return a.scheduler
}

func (a *Agent) GetPool() db.PgxIface {
	return a.pool
}

func NewAgent() *Agent {
	config.ResolveHost()

	agent := new(Agent)
	agent.scheduler = common.NewScheduler()

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
	providerClient, err := clientconfig.AuthenticatedClient(context.Background(), &clientconfig.ClientOpts{
		AuthInfo: &authInfo})
	if err != nil {
		log.WithError(err).Fatal("Error while connecting to Keystone")
	}

	if agent.neutron, err = neutron.ConnectToNeutron(providerClient); err != nil {
		log.WithError(err).Fatalf("Error while connecting to Neutron")
	}
	log.Infof("Connected to Neutron %s", agent.neutron.Endpoint)
	agent.neutron.InitCache()

	common.RegisterAgent(agent.pool, "tenant")
	return agent
}

func (a *Agent) Run() {
	go common.DBNotificationThread(context.Background(), a)
	go common.PrometheusListenerThread()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		sig := <-sigs
		log.Infof("Received signal %s", sig)
		done <- true
	}()

	// sync pending services
	if _, err := a.scheduler.NewJob(
		gocron.DurationJob(config.Global.Agent.PendingSyncInterval),
		gocron.NewTask(a.PendingSyncLoop),
		gocron.WithName("PendingSyncLoop"),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		log.Fatal(err)
	}

	if _, err := a.scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(
			gocron.NewAtTime(0, 0, 0)),
		),
		gocron.NewTask(a.cleanupL2),
		gocron.WithName("cleanupL2"),
	); err != nil {
		log.Fatal(err)
	}

	// start the scheduler
	a.scheduler.Start()

	// block until done
	log.Infof("Agent running...")
	<-done
	if err := a.scheduler.Shutdown(); err != nil {
		log.Fatal(err)
	}
}

func (a *Agent) PendingSyncLoop() error {
	var id strfmt.UUID
	var rows pgx.Rows
	var ret pgconn.CommandTag
	var err error

	sql, args := db.Select("1").
		From("service").
		Where("provider = ?", models.ServiceProviderTenant).
		Where(sq.Eq{"status": []string{
			models.ServiceStatusPENDINGDELETE,
			models.ServiceStatusPENDINGCREATE,
			models.ServiceStatusPENDINGUPDATE}}).
		Where("host = ?", config.Global.Default.Host).
		MustSql()
	ret, err = a.pool.Exec(context.Background(), sql, args...)
	if err != nil {
		return err
	}

	if ret.RowsAffected() > 0 {
		if _, err := a.scheduler.NewJob(
			gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
			gocron.NewTask(a.ProcessServices(context.Background())),
			gocron.WithName("ProcessServices"),
		); err != nil {
			return err
		}
	}

	sql, args = db.Select("id").
		From("endpoint").
		Where(sq.And{
			sq.Eq{"status": []models.EndpointStatus{
				models.EndpointStatusPENDINGDELETE,
				models.EndpointStatusPENDINGCREATE,
				models.EndpointStatusPENDINGUPDATE,
				models.EndpointStatusPENDINGREJECTED,
			}},
			db.Select("1").
				Prefix("EXISTS(").
				From("service").
				Where("service.id = endpoint.service_id").
				Where("service.provider = ?", models.ServiceProviderTenant).
				Where("service.host = ?", config.Global.Default.Host).
				Suffix(")"),
		}).
		MustSql()
	rows, err = a.pool.Query(context.Background(), sql, args...)
	if err != nil {
		return err
	}
	if _, err = pgx.ForEachRow(rows, []any{&id}, func() error {
		if _, err := a.scheduler.NewJob(
			gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
			gocron.NewTask(a.ProcessEndpoint(context.Background(), id)),
			gocron.WithName("ProcessEndpoint"),
			gocron.WithTags(id.String()),
		); err != nil {
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
