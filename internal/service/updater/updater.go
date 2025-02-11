package updater

import (
	"context"
	"log"
	"time"

	"mr-metrics/internal/config"
)

type StatsUpdater interface {
	UpdateProjectCache(projectID int, projectName string, counts map[string]int) error
}

type StatsClient interface {
	GetMergedMRCounts(projectName string) (map[string]int, int, error)
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
		counts, projectID, err := u.gitlab.GetMergedMRCounts(projectName)
		if err != nil {
			log.Printf("Failed to fetch data for project %s: %v", projectName, err)
			continue
		}

		if err := u.updater.UpdateProjectCache(projectID, projectName, counts); err != nil {
			log.Printf("Failed to update cache for project %s: %v", projectName, err)
		}
	}
}
