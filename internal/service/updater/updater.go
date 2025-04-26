// SPDX-FileCopyrightText: 2025 Danila Gorelko <hello@danilax86.space>
//
// SPDX-License-Identifier: MIT

package updater

import (
	"context"
	"log"
	"mr-metrics/internal/consts"
	"mr-metrics/internal/model"
	"time"

	"mr-metrics/internal/config"
)

type StatsUpdater interface {
	UpdateProjectCache(projectID int, projectName string, counts []model.MergeRequest) error
	GetLastUpdatedDate(projectName string) (time.Time, error)
}

type StatsClient interface {
	GetMergedMRCounts(projectName string, since time.Time) ([]model.MergeRequest, int, error)
}

type BackgroundUpdater struct {
	cfg     *config.Config
	updater StatsUpdater
	ticker  *time.Ticker
	gitlab  StatsClient
}

func New(store StatsUpdater, gitlab StatsClient, cfg *config.Config) *BackgroundUpdater {
	return &BackgroundUpdater{
		cfg:     cfg,
		updater: store,
		gitlab:  gitlab,
		ticker:  time.NewTicker(cfg.CacheTTL),
	}
}

func (u *BackgroundUpdater) Start(ctx context.Context) {
	go u.updateAllProjects()

	go func() {
		for {
			select {
			case <-u.ticker.C:
				u.updateAllProjects()
			case <-ctx.Done():
				u.ticker.Stop()
				return
			}
		}
	}()
}

func (u *BackgroundUpdater) updateAllProjects() {
	for _, projectName := range u.cfg.ProjectNames {
		lastUpdated, err := u.updater.GetLastUpdatedDate(projectName)
		if err != nil {
			log.Printf("Failed to fetch last updated date for project %s. Fetch all merged requests", projectName)

			// If the last updated date is not found, fetch all data
			counts, projectID, err := u.gitlab.GetMergedMRCounts(projectName, time.Time{}.UTC())
			if err != nil {
				log.Printf("Failed to fetch data for project %s: %v", projectName, err)
				continue
			}

			if err := u.updater.UpdateProjectCache(projectID, projectName, counts); err != nil {
				log.Printf("Failed to update cache for project %s: %v", projectName, err)
			}
			continue
		}

		// Add a delta (yesterday) to the last updated date to avoid losing requests
		since := lastUpdated.Add(-1 * consts.OneDay)

		counts, projectID, err := u.gitlab.GetMergedMRCounts(projectName, since)
		if err != nil {
			log.Printf("Failed to fetch data for project %s: %v", projectName, err)
			continue
		}

		if err := u.updater.UpdateProjectCache(projectID, projectName, counts); err != nil {
			log.Printf("Failed to update cache for project %s: %v", projectName, err)
		}
	}
}
