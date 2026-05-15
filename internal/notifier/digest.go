// SPDX-FileCopyrightText: Copyright 2025 SAP SE or an SAP affiliate company
//
// SPDX-License-Identifier: Apache-2.0

package notifier

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	log "github.com/sirupsen/logrus"

	"github.com/sapcc/archer/v2/internal/db"
	"github.com/sapcc/archer/v2/models"
)

type PendingGroup struct {
	ProjectID string
	Services  []ServiceInfo
}

func QueryPendingEndpoints(ctx context.Context, pool db.PgxIface) ([]PendingGroup, error) {
	sql, args := db.Select("*").
		From("endpoint").
		Where("status = ?", "PENDING_APPROVAL").
		OrderBy("service_id", "created_at").
		MustSql()

	var endpoints []models.Endpoint
	if err := pgxscan.Select(ctx, pool, &endpoints, sql, args...); err != nil {
		return nil, err
	}
	if len(endpoints) == 0 {
		return nil, nil
	}

	serviceIDs := make([]interface{}, 0)
	seen := make(map[string]bool)
	for _, ep := range endpoints {
		id := ep.ServiceID.String()
		if !seen[id] {
			seen[id] = true
			serviceIDs = append(serviceIDs, id)
		}
	}

	sql, args = db.Select("*").
		From("service").
		Where(sq.Eq{"id": serviceIDs}).
		MustSql()

	var services []models.Service
	if err := pgxscan.Select(ctx, pool, &services, sql, args...); err != nil {
		return nil, err
	}

	svcMap := make(map[string]*models.Service, len(services))
	for i := range services {
		svcMap[services[i].ID.String()] = &services[i]
	}

	var groups []PendingGroup
	projectIdx := make(map[string]int)

	for i := range endpoints {
		ep := &endpoints[i]
		svc := svcMap[ep.ServiceID.String()]
		if svc == nil {
			continue
		}

		projectID := string(svc.ProjectID)
		idx, exists := projectIdx[projectID]
		if !exists {
			idx = len(groups)
			projectIdx[projectID] = idx
			groups = append(groups, PendingGroup{ProjectID: projectID})
		}

		group := &groups[idx]
		var si *ServiceInfo
		for j := range group.Services {
			if group.Services[j].ID == svc.ID {
				si = &group.Services[j]
				break
			}
		}
		if si == nil {
			group.Services = append(group.Services, ServiceInfo{Service: *svc})
			si = &group.Services[len(group.Services)-1]
		}
		si.Endpoints = append(si.Endpoints, ep)
	}

	return groups, nil
}

func shouldSkipDigest(youngestCreatedAt time.Time) bool {
	return time.Since(youngestCreatedAt) < 24*time.Hour
}

func (n *Notifier) RunDigest(ctx context.Context, pool db.PgxIface) {
	groups, err := QueryPendingEndpoints(ctx, pool)
	if err != nil {
		log.WithError(err).Error("Failed to query pending endpoints for digest")
		return
	}

	if len(groups) == 0 {
		log.Debug("No pending endpoints for digest")
		return
	}

	// Find youngest endpoint across all groups
	var youngest time.Time
	for _, g := range groups {
		for _, s := range g.Services {
			for _, ep := range s.Endpoints {
				if ep.CreatedAt.After(youngest) {
					youngest = ep.CreatedAt
				}
			}
		}
	}

	if shouldSkipDigest(youngest) {
		log.Debug("Skipping digest: youngest pending endpoint is less than 1 day old")
		return
	}

	for _, group := range groups {
		data := NotificationData{
			Type:     "digest",
			Services: group.Services,
		}
		if err := n.SendNotification(ctx, group.ProjectID, data); err != nil {
			log.WithFields(log.Fields{
				"project_id": group.ProjectID,
			}).WithError(err).Error("Failed to send digest notification")
		}
	}
}
