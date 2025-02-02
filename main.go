package main

import (
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Config struct {
	GitLabHost  string
	GitLabToken string
	Projects    []string
	CacheTTL    time.Duration
	DatabaseURL string
}

var (
	db     *sql.DB
	tmpl   *template.Template
	config Config
)

func getEnv(key string, required bool) string {
	value := os.Getenv(key)
	if required && value == "" {
		log.Fatalf("Missing required environment variable: %s", key)
	}
	return value
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config = Config{
		GitLabHost:  getEnv("GITLAB_HOST", true),
		GitLabToken: getEnv("GITLAB_TOKEN", true),
		DatabaseURL: getEnv("DATABASE_URL", true),
		CacheTTL:    10 * time.Minute,
	}

	// Parse project IDs
	for _, id := range strings.Split(getEnv("GITLAB_PROJECT_IDS", true), ",") {
		config.Projects = append(config.Projects, id)
	}

	var err error
	db, err = sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	tmpl = template.Must(template.ParseFiles("templates/index.html"))

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	wg := sync.WaitGroup{}

	for _, projectName := range config.Projects {
		wg.Add(1)
		go func(projectName string) {
			defer wg.Done()

			// Resolve project name to ID
			projectID, err := getProjectID(projectName)
			if err != nil {
				log.Printf("Failed to resolve project ID for %s: %v", projectName, err)
				http.Error(w, fmt.Sprintf("Failed to resolve project ID for %s", projectName), http.StatusInternalServerError)
				return
			}

			lastUpdated, err := getLastUpdated(projectID)
			if err != nil {
				log.Printf("Failed to get last updated time for project %s: %v", projectName, err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}

			if time.Since(lastUpdated) > config.CacheTTL {
				counts, err := fetchProjectMRs(projectID)
				if err != nil {
					log.Printf("Failed to fetch data for project %s: %v", projectName, err)
					http.Error(w, fmt.Sprintf("Failed to fetch data for project %s", projectName), http.StatusInternalServerError)
					return
				}

				if err := updateProjectCache(projectID, projectName, counts); err != nil {
					log.Printf("Failed to update cache for project %s: %v", projectName, err)
					http.Error(w, "Cache update failed", http.StatusInternalServerError)
					return
				}
			}
		}(projectName)

	}

	data, err := getAggregatedData()
	if err != nil {
		log.Printf("Failed to aggregate data: %v", err)
		http.Error(w, "Data aggregation failed", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}
