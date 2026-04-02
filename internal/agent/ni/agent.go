// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package ni

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/IBM/pgxpoolprometheus"
	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-co-op/gocron/v2"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	common "github.com/sapcc/archer/internal/agent"
	"github.com/sapcc/archer/internal/agent/ni/haproxy"
	ni "github.com/sapcc/archer/internal/agent/ni/models"
	"github.com/sapcc/archer/internal/agent/ni/proxy"
	"github.com/sapcc/archer/internal/config"
	"github.com/sapcc/archer/internal/db"
	"github.com/sapcc/archer/models"
)

type Agent struct {
	scheduler    gocron.Scheduler
	pool         *pgxpool.Pool // thread safe
	neutron      *gophercloud.ServiceClient
	haproxy      haproxy.HAProxy
	proxyManager *proxy.Manager // manages Unix proxy threads per service
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
	if err != nil || connConfig == nil {
		log.Fatalf("PGX Pool connection failed: %v", err)
	}
	connConfig.ConnConfig.Tracer = db.GetTracer()
	connConfig.ConnConfig.RuntimeParams["application_name"] = "archer-ni-agent"
	if agent.pool, err = pgxpool.NewWithConfig(context.Background(), connConfig); err != nil {
		log.Fatal(err.Error())
	}

	// install postgres status exporter
	dbConfig := agent.pool.Config()
	collector := pgxpoolprometheus.NewCollector(agent.pool, map[string]string{"db_name": dbConfig.ConnConfig.Database})
	prometheus.MustRegister(collector)
	log.Infof("Connected to PostgreSQL host=%s, max_conns=%d, health_check_period=%s",
		dbConfig.ConnConfig.Host, dbConfig.MaxConns, dbConfig.HealthCheckPeriod)

	if err := agent.SetupOpenStack(); err != nil {
		log.Fatal(err.Error())
	}
	log.Infof("Connected to Neutron host=%s", agent.neutron.Endpoint)

	if config.Global.Default.AvailabilityZone != "" {
		log.Infof("Availability zone: %s", config.Global.Default.AvailabilityZone)
	}

	agent.haproxy = haproxy.NewHAProxyController()

	common.RegisterAgent(agent.pool, "cp")

	// Update services without IP addresses using deprecated config (migration path)
	if err := agent.migrateServiceIPAddresses(context.Background()); err != nil {
		log.WithError(err).Warn("Failed to migrate service IP addresses")
	}

	return agent
}

func (a *Agent) Run() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize the proxy manager with the main context
	a.proxyManager = proxy.NewManager(ctx)

	prom := common.NewPrometheusListener()
	go prom.Run()
	go common.DBNotificationThread(ctx, a)
	go a.haproxy.Run(ctx)

	// sync immediately
	if err := a.PendingSyncLoop(context.Background(), true); err != nil {
		log.Fatal(err)
	}

	// background job for pending services
	if _, err := a.scheduler.NewJob(
		gocron.DurationJob(config.Global.Agent.PendingSyncInterval),
		gocron.NewTask(a.PendingSyncLoop, false),
		gocron.WithName("PendingSyncLoop"),
	); err != nil {
		log.Fatal(err)
	}

	// collect metrics
	if _, err := a.scheduler.NewJob(
		gocron.DurationJob(1*time.Minute),
		gocron.NewTask(a.CollectStats),
		gocron.WithName("CollectStats"),
	); err != nil {
		log.Fatal(err)
	}

	// heartbeat job
	if _, err := a.scheduler.NewJob(
		gocron.DurationJob(config.Global.Agent.HeartbeatInterval),
		gocron.NewTask(a.UpdateHeartbeat),
		gocron.WithName("Heartbeat"),
	); err != nil {
		log.Fatal(err)
	}

	a.scheduler.Start()

	log.Infof("Agent running...")

	// block until done
	<-ctx.Done()

	log.Infof("Agent shutting down...")
	_ = prom.Shutdown(context.Background())
	_ = a.scheduler.Shutdown()
	time.Sleep(5 * time.Second)
}

// UpdateHeartbeat updates the agent's heartbeat in the database.
func (a *Agent) UpdateHeartbeat() {
	common.UpdateHeartbeat(a.pool)
}

// migrateServiceIPAddresses updates services assigned to this host that have empty IP addresses
// using the deprecated ServiceUpstreamHost config. This provides a migration path for existing
// services that were created before dynamic IP address support.
func (a *Agent) migrateServiceIPAddresses(ctx context.Context) error {
	upstreamHost := config.Global.Agent.ServiceUpstreamHost
	if upstreamHost == "" {
		return nil
	}

	sql, args := db.Update("service").
		Set("ip_addresses", sq.Expr("ARRAY[?::inet]", upstreamHost)).
		Where("host = ?", config.Global.Default.Host).
		Where("provider = 'cp'").
		Where("ip_addresses = '{}'").
		MustSql()

	result, err := a.pool.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	if result.RowsAffected() > 0 {
		log.Infof("Migrated %d services with IP address %s (from deprecated config)",
			result.RowsAffected(), upstreamHost)
	}

	return nil
}

func (a *Agent) ProcessServices(ctx context.Context) error {
	type serviceInfo struct {
		ID          strfmt.UUID `db:"id"`
		IPAddresses []string    `db:"ip_addresses"`
		Ports       []int32     `db:"ports"`
		Status      string      `db:"status"`
	}

	// Query all services assigned to this host
	sql, args := db.Select("id", "ip_addresses", "ports", "status").
		From("service").
		Where("host = ?", config.Global.Default.Host).
		Where("provider = 'cp'").
		MustSql()

	var services []serviceInfo
	if err := pgxscan.Select(ctx, a.pool, &services, sql, args...); err != nil {
		return err
	}

	var toDelete []strfmt.UUID
	for _, svc := range services {
		if svc.Status == string(models.ServiceStatusPENDINGDELETE) {
			a.proxyManager.StopProxy(svc.ID)
			toDelete = append(toDelete, svc.ID)
		} else if !a.proxyManager.IsRunning(svc.ID) && len(svc.IPAddresses) > 0 && len(svc.Ports) > 0 {
			a.proxyManager.StartProxy(svc.ID, svc.IPAddresses[0], svc.Ports)
		}
	}

	if len(toDelete) == 0 {
		return nil
	}

	// Delete services pending deletion
	sql, args = db.Delete("service").
		Where(sq.Eq{"id": toDelete}).
		MustSql()
	_, err := a.pool.Exec(ctx, sql, args...)
	return err
}

func (a *Agent) ProcessEndpoint(ctx context.Context, id strfmt.UUID) error {
	log.Infof("Processing endpoint: %s", id)
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	return pgx.BeginFunc(ctx, a.pool, func(tx pgx.Tx) error {
		var si ni.ServiceInjection
		var err error

		sql, args := db.Select("e.id", "e.status", "ep.port_id", "ep.network", "ep.ip_address",
			"s.id AS service_id", "s.protocol AS service_protocol", "s.ports AS service_ports",
			"s.ip_addresses[1] AS service_ip_address").
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
		case models.EndpointStatusPENDINGREJECTED:
			log.Infof("ProcessEndpoint: Rejecting endpoint %s", si.ID)
			if err = a.DisableInjection(&si); err != nil {
				return err
			}
			sql, args = db.Update("endpoint").
				Set("status", models.EndpointStatusREJECTED).
				Set("updated_at", sq.Expr("NOW()")).
				Where("id = ?", si.ID).
				MustSql()
			if _, err = tx.Exec(ctx, sql, args...); err != nil {
				return err
			}
		case models.EndpointStatusPENDINGDELETE:
			if err = a.DisableInjection(&si); err != nil {
				return err
			}
			sql, args, err = db.Delete("endpoint").
				Where("id = ?", si.ID).
				ToSql()
			if err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, sql, args...); err != nil {
				return err
			}
		case models.EndpointStatusAVAILABLE:
			if err = a.EnableInjection(&si); err != nil {
				return err
			}
		case models.EndpointStatusPENDINGUPDATE:
			fallthrough
		case models.EndpointStatusFAILED:
			fallthrough
		case models.EndpointStatusPENDINGCREATE:
			if err = a.EnableInjection(&si); err != nil {
				return err
			}
			sql, args, err = db.Update("endpoint").
				Set("status", models.EndpointStatusAVAILABLE).
				Set("updated_at", sq.Expr("NOW()")).
				Where("id = ?", si.ID).
				ToSql()
			if err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, sql, args...); err != nil {
				return err
			}
		}
		log.Debugf("ProcessEndpoint: finished processing endpoint %s (status=%s)", id, si.Status)
		return nil
	})
}

func (a *Agent) PendingSyncLoop(ctx context.Context, syncAll bool) error {
	log.Debugf("PendingSyncLoop(syncAll=%t)", syncAll)

	// Check for pending services first (similar to F5 agent pattern)
	sql, args := db.Select("1").
		From("service").
		Where("provider = 'cp'").
		Where(sq.Eq{"status": []models.ServiceStatus{
			models.ServiceStatusPENDINGDELETE,
			models.ServiceStatusPENDINGCREATE,
			models.ServiceStatusPENDINGUPDATE}}).
		Where("host = ?", config.Global.Default.Host).
		MustSql()

	ret, err := a.pool.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}

	if ret.RowsAffected() > 0 {
		if _, err := a.scheduler.NewJob(
			gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
			gocron.NewTask(a.ProcessServices),
			gocron.WithName("ProcessServices"),
		); err != nil {
			return err
		}
	}

	// Query all endpoints for services assigned to this host
	q := db.Select("e.id").
		From("endpoint e").
		Join("service s ON e.service_id = s.id").
		Where("s.host = ?", config.Global.Default.Host).
		Where("s.provider = 'cp'")

	if syncAll {
		// initial run, sync everything
		q = q.Where("e.status != ?", models.EndpointStatusREJECTED)
	} else {
		// sync only pending
		q = q.Where(sq.Eq{"e.status": []models.EndpointStatus{
			models.EndpointStatusPENDINGCREATE,
			models.EndpointStatusPENDINGUPDATE,
			models.EndpointStatusREJECTED,
			models.EndpointStatusPENDINGREJECTED,
			models.EndpointStatusPENDINGDELETE,
			models.EndpointStatusFAILED}})
	}

	var id strfmt.UUID
	sql, args = q.MustSql()
	rows, err := a.pool.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	_, err = pgx.ForEachRow(rows, []any{&id}, func() error {
		log.Debugf("Scheduling ProcessEndpoint for %s", id)
		if _, err = a.scheduler.NewJob(
			gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
			gocron.NewTask(a.ProcessEndpoint, id),
			gocron.WithName("ProcessEndpoint"),
			gocron.WithTags(id.String()),
		); err != nil {
			return err
		}
		return nil
	})
	return err
}
