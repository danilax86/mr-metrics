// @todo #1 Add license for each file

package main

import (
	"context"
	"log"
	"mr-metrics/internal/api"
	"mr-metrics/internal/config"
	"mr-metrics/internal/db"
	"mr-metrics/internal/handlers"
	"mr-metrics/internal/service/updater"

	_ "github.com/lib/pq"
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

	u := updater.New(store, gitlabClient, cfg)
	go u.Start(ctx)

	log.Fatal(handlers.Start(store, cfg, gitlabClient))
}
