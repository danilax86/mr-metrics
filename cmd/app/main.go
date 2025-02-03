package main

import (
	"log"
	"mr-metrics/internal/api"
	"mr-metrics/internal/config"
	"mr-metrics/internal/db"
	"mr-metrics/internal/handler"
)

func main() {
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

	log.Printf("Server starting on :%s", cfg.Port)
	log.Fatal(h.Start(cfg.Port))
}
