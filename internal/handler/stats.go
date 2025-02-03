package handler

import (
	"fmt"
	"html/template"
	"log"
	"mr-metrics/internal/api"
	"mr-metrics/internal/config"
	"mr-metrics/internal/model"
	"net/http"
	"sync"
	"time"
)

var tmpl *template.Template

type Store interface {
	GetLastUpdated(projectID int) (time.Time, error)
	GetLastUpdatedByName(projectName string) (time.Time, error)
	UpdateProjectCache(projectID int, projectName string, counts map[string]int) error
	GetAggregatedData(projectNames []string) (*model.AggregatedStats, error)
}
type StatsHandler struct {
	store  Store
	cfg    *config.Config
	client *api.GitLabClient
}

func New(store Store, cfg *config.Config, client *api.GitLabClient) *StatsHandler {
	return &StatsHandler{
		store:  store,
		cfg:    cfg,
		client: client,
	}
}

func (h *StatsHandler) Start(port string) error {
	mux := http.NewServeMux()

	tmpl = template.Must(template.ParseFiles("internal/web/templates/index.html"))

	mux.HandleFunc("/", h.handleStats)
	return http.ListenAndServe(":"+port, mux)
}

func (h *StatsHandler) handleStats(w http.ResponseWriter, _ *http.Request) {
	wg := sync.WaitGroup{}

	for _, projectName := range h.cfg.ProjectNames {
		wg.Add(1)
		go func(projectName string) {
			defer wg.Done()

			lastUpdated, err := h.store.GetLastUpdatedByName(projectName)
			if err != nil {
				log.Printf("Failed to get last updated time for project %s: %v", projectName, err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}

			if time.Since(lastUpdated) > h.cfg.CacheTTL {
				counts, projectID, err := h.client.GetMergedMRCounts(projectName)
				if err != nil {
					log.Printf("Failed to fetch data for project %s: %v", projectName, err)
					http.Error(w, fmt.Sprintf("Failed to fetch data for project %s", projectName), http.StatusInternalServerError)
					return
				}

				if err := h.store.UpdateProjectCache(projectID, projectName, counts); err != nil {
					log.Printf("Failed to update cache for project %s: %v", projectName, err)
					http.Error(w, "Cache update failed", http.StatusInternalServerError)
					return
				}
			}
		}(projectName)

	}

	data, err := h.store.GetAggregatedData(h.cfg.ProjectNames)
	if err != nil {
		log.Printf("Failed to aggregate data: %v", err)
		http.Error(w, "Data aggregation failed", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}
