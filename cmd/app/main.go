package main

import (
	"context"
	"log"
	"mr-metrics/internal/api"
	"mr-metrics/internal/config"
	"mr-metrics/internal/db"
	"mr-metrics/internal/handler"
	"mr-metrics/internal/service/updater"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	gitlabClient := api.NewGitLabClient(cfg)

	store, err := db.NewPostgresStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	h := handler.New(store, cfg, gitlabClient)

	u := updater.New(store, gitlabClient, cfg)
	go u.Start(ctx)

	log.Fatal(h.Start(cfg.Port))
}
