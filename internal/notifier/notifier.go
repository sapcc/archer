// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package notifier

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/go-openapi/strfmt"
	"github.com/gophercloud/gophercloud/v2"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/db"
	"github.com/sapcc/archer/v2/internal/scheduler"
	"github.com/sapcc/archer/v2/models"
)

// notificationLockID is a fixed lock ID for digest notification scheduling.
const notificationLockID = 8675310

type Config struct {
	CampfireURL    string
	TemplatePath   string
	DigestCron     string
	ProviderClient *gophercloud.ProviderClient
}

type Notifier struct {
	campfire        *CampfireClient
	templates       *Templates
	pool            db.PgxIface
	cronExpr        string
	gocronScheduler gocron.Scheduler
	elector         *scheduler.PostgresElector
}

func New(cfg Config, pool db.PgxIface) (*Notifier, error) {
	templates, err := LoadTemplates(cfg.TemplatePath)
	if err != nil {
		return nil, fmt.Errorf("loading notification templates: %w", err)
	}

	elector := scheduler.NewPostgresElector(pool, notificationLockID, "notification")

	gocronSched, err := gocron.NewScheduler(
		gocron.WithStopTimeout(time.Second * 30),
	)
	if err != nil {
		return nil, err
	}

	return &Notifier{
		campfire:        NewCampfireClient(cfg.CampfireURL, cfg.ProviderClient),
		templates:       templates,
		pool:            pool,
		cronExpr:        cfg.DigestCron,
		gocronScheduler: gocronSched,
		elector:         elector,
	}, nil
}

// Start starts the digest notification cron job with distributed leader election.
func (n *Notifier) Start(ctx context.Context) error {
	if err := n.elector.Start(ctx); err != nil {
		return err
	}

	_, err := n.gocronScheduler.NewJob(
		gocron.CronJob(n.cronExpr, false),
		gocron.NewTask(func() {
			n.runDigest(ctx)
		}),
		gocron.WithName("DigestNotification"),
	)
	if err != nil {
		n.elector.Close()
		return err
	}

	log.Infof("Notification scheduler started with cron expression: %s", n.cronExpr)
	n.gocronScheduler.Start()
	return nil
}

// Stop stops the notification scheduler gracefully.
func (n *Notifier) Stop() error {
	n.elector.Close()
	return n.gocronScheduler.Shutdown()
}

// ScheduleImmediate creates a one-shot job that sends an immediate notification for a new pending endpoint.
func (n *Notifier) ScheduleImmediate(ctx context.Context, pool db.PgxIface, serviceID strfmt.UUID, ep *models.Endpoint) {
	_, err := n.gocronScheduler.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
		gocron.NewTask(func() {
			sql, args := db.Select("name", "project_id").
				From("service").
				Where("id = ?", serviceID).
				MustSql()

			var serviceName string
			var ownerProjectID string
			if err := pool.QueryRow(ctx, sql, args...).Scan(&serviceName, &ownerProjectID); err != nil {
				log.WithError(err).WithField("service_id", serviceID).Error("Failed to look up service for notification")
				return
			}

			data := NotificationData{
				Type: "immediate",
				Services: []ServiceInfo{
					{
						Service:   models.Service{Name: serviceName, ID: serviceID},
						Endpoints: []*models.Endpoint{ep},
					},
				},
			}

			if err := n.SendNotification(ctx, ownerProjectID, data); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"service_id":  serviceID,
					"endpoint_id": ep.ID,
				}).Error("Failed to send immediate notification")
			}
		}),
		gocron.WithName("ImmediateNotification"),
	)
	if err != nil {
		log.WithError(err).WithField("service_id", serviceID).Error("Failed to schedule immediate notification")
	}
}

func (n *Notifier) SendNotification(ctx context.Context, projectID string, data NotificationData) error {
	body, err := n.templates.Render(data)
	if err != nil {
		return fmt.Errorf("rendering notification: %w", err)
	}

	subject := buildSubject(data)

	req := &CampfireRequest{
		ProjectID: projectID,
		Subject:   subject,
		MimeType:  "text/plain",
		MailText:  body,
	}

	if err = n.campfire.SendEmail(ctx, req); err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"project_id": projectID,
		"type":       data.Type,
	}).Debug("Notification sent successfully")
	return nil
}

// runDigest executes the digest notification with distributed leadership.
func (n *Notifier) runDigest(ctx context.Context) {
	if err := n.elector.IsLeader(ctx); err != nil {
		return
	}

	n.RunDigest(ctx, n.pool)
}

func buildSubject(data NotificationData) string {
	if data.Type == "immediate" {
		return "Archer Endpoint Services: New endpoint(s) pending approval"
	}
	return fmt.Sprintf("Archer Endpoint Services: %d endpoint(s) awaiting approval", data.TotalEndpoints())
}
