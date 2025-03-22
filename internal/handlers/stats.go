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

type StatsHandler struct {
	store StatsStore
	cfg   *config.Config
	tmpl  *template.Template
}

func NewStatsHandler(store StatsStore, cfg *config.Config) *StatsHandler {
	return &StatsHandler{
		store: store,
		cfg:   cfg,
		tmpl:  web.TemplateStats(),
	}
}

func (h *StatsHandler) handleStats(w http.ResponseWriter, _ *http.Request) {
	data, err := h.store.GetAggregatedDataForDate(h.cfg.ProjectNames, endOfDay(time.Now()))
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

	data, err := h.store.GetAggregatedDataForDate(h.cfg.ProjectNames, endOfDay(targetDate))
	if err != nil {
		http.Error(w, "Failed to retrieve historical data", http.StatusInternalServerError)
		return
	}

	data.DateString = targetDate.Format("2006-01-02")

	if err := h.tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func endOfDay(date time.Time) time.Time {
	return time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		23,
		59,
		59,
		999999999,
		date.Location(),
	).UTC()
}
