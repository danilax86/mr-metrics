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
	GetAggregatedDataForDate(projectNames []string, targetDate time.Time) (*model.AggregatedStats, error)
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

	mux.HandleFunc("GET /", h.handleStatsByDate)
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

func (h *StatsHandler) handleStatsByDate(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		h.handleStats(w, r) // Default for current date
		return
	}

	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	endOfDay := time.Date(
		targetDate.Year(),
		targetDate.Month(),
		targetDate.Day(),
		23,
		59,
		59,
		999999999,
		targetDate.Location(),
	).UTC()

	data, err := h.store.GetAggregatedDataForDate(h.cfg.ProjectNames, endOfDay)
	if err != nil {
		http.Error(w, "Failed to retrieve historical data", http.StatusInternalServerError)
		return
	}

	data.DateString = targetDate.Format("2006-01-02")

	if err := h.tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
