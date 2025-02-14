package handlers

import (
	"fmt"
	"html/template"
	"mr-metrics/internal/config"
	"mr-metrics/internal/model"
	"mr-metrics/internal/web"
	"net/http"
	"time"
)

type StatsStore interface {
	GetAggregatedDataForDate(projectNames []string, targetDate time.Time) (*model.AggregatedStats, error)
}

type StatsClient interface {
	GetMergedMRCounts(projectName string) ([]model.MergeRequest, int, error)
}
type StatsHandler struct {
	store  StatsStore
	client StatsClient
	cfg    *config.Config
	tmpl   *template.Template
}

func NewStatsHandler(store StatsStore, cfg *config.Config, client StatsClient) *StatsHandler {
	return &StatsHandler{
		store:  store,
		cfg:    cfg,
		client: client,
		tmpl:   web.TemplateStats(),
	}
}

func (h *StatsHandler) handleStats(w http.ResponseWriter, _ *http.Request) {
	today := time.Now().UTC()
	endOfDay := time.Date(
		time.Now().Year(),
		today.Month(),
		today.Day(),
		23,
		59,
		59,
		999999999,
		today.Location(),
	).UTC()

	data, err := h.store.GetAggregatedDataForDate(h.cfg.ProjectNames, endOfDay)
	if err != nil {
		http.Error(w, "Failed to get data", http.StatusInternalServerError)
		return
	}

	if err := web.TemplateExec(w, h.tmpl, data); err != nil {
		http.Error(w, fmt.Errorf("template error: %w", err).Error(), http.StatusInternalServerError)
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
