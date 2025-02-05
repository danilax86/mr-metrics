package updater

import (
	"context"
	"log"
	"mr-metrics/internal/handler"
	"time"

	"mr-metrics/internal/api"
	"mr-metrics/internal/config"
)

type BackgroundUpdater struct {
	cfg    *config.Config
	store  handler.Store
	ticker *time.Ticker
	gitlab *api.GitLabClient
}

func New(store handler.Store, gitlab *api.GitLabClient, cfg *config.Config) *BackgroundUpdater {
	return &BackgroundUpdater{
		cfg:    cfg,
		store:  store,
		gitlab: gitlab,
		ticker: time.NewTicker(cfg.CacheTTL),
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

		if err := u.store.UpdateProjectCache(projectID, projectName, counts); err != nil {
			log.Printf("Failed to update cache for project %s: %v", projectName, err)
		}
	}
}
