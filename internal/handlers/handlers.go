// SPDX-FileCopyrightText: 2025 Danila Gorelko <hello@danilax86.space>
//
// SPDX-License-Identifier: MIT

package handlers

import (
	"mr-metrics/internal/config"
	"mr-metrics/internal/db"
	"net/http"
	"time"
)

const defaultServerTimeout = 3 * time.Second

func Start(db *db.PostgresStore, cfg *config.Config) error {
	mux := http.NewServeMux()

	stats := NewStatsHandler(db, cfg)

	mux.HandleFunc("GET /", stats.handleStatsByDate)
	mux.HandleFunc("GET /static/style.css", handleStyle)

	server := http.Server{
		Addr:              ":" + cfg.Port,
		ReadHeaderTimeout: defaultServerTimeout,
		Handler:           mux,
	}

	return server.ListenAndServe()
}
