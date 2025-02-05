package handler

import (
	"html/template"
	"mr-metrics/internal/api"
	"mr-metrics/internal/config"
	"mr-metrics/internal/model"
	"net/http"
	"time"
)

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
	tmpl   *template.Template
}

func New(store Store, cfg *config.Config, client *api.GitLabClient) *StatsHandler {
	return &StatsHandler{
		store:  store,
		cfg:    cfg,
		client: client,
		tmpl:   template.Must(template.ParseFiles("internal/web/templates/index.html")),
	}
}

func (h *StatsHandler) Start(port string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", h.handleStats)
	return http.ListenAndServe(":"+port, mux)
}

func (h *StatsHandler) handleStats(w http.ResponseWriter, _ *http.Request) {
	data, err := h.store.GetAggregatedData(h.cfg.ProjectNames)
	if err != nil {
		http.Error(w, "Failed to get data", http.StatusInternalServerError)
		return
	}

	if err := h.tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
