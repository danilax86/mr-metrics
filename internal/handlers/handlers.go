package handlers

import (
	"mr-metrics/internal/api"
	"mr-metrics/internal/config"
	"mr-metrics/internal/db"
	"net/http"
	"time"
)

const defaultServerTimeout = 3 * time.Second

func Start(db *db.PostgresStore, cfg *config.Config, client *api.GitLabClient) error {
	mux := http.NewServeMux()

	stats := NewStatsHandler(db, cfg, client)

	mux.HandleFunc("GET /", stats.handleStatsByDate)
	mux.HandleFunc("GET /static/style.css", handleStyle)

	server := http.Server{
		Addr:              ":" + cfg.Port,
		ReadHeaderTimeout: defaultServerTimeout,
		Handler:           mux,
	}

	return server.ListenAndServe()
}
