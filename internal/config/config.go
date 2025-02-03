package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port          string
	GitLabToken   string
	GitLabHostURL string
	ProjectNames  []string
	DatabaseURL   string
	CacheTTL      time.Duration
}

func Load() (*Config, error) {
	var errors []string

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	gitlabToken := os.Getenv("GITLAB_TOKEN")
	if gitlabToken == "" {
		errors = append(errors, "GITLAB_TOKEN is required")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		errors = append(errors, "DATABASE_URL is required")
	}

	gitlabHostURL := getEnv("GITLAB_HOST_URL", "https://gitlab.com")
	if _, err := url.Parse(gitlabHostURL); err != nil {
		errors = append(errors, fmt.Sprintf("invalid GITLAB_HOST_URL: %v", err))
	}

	projectNames := strings.Split(os.Getenv("GITLAB_PROJECT_NAMES"), ",")
	if len(projectNames) == 0 {
		errors = append(errors, "GITLAB_PROJECT_NAMES is required")
	}

	cacheTTL := parseDuration(getEnv("CACHE_TTL", "1h"))

	if len(errors) > 0 {
		return nil, fmt.Errorf("configuration errors:\n- %s", strings.Join(errors, "\n- "))
	}

	return &Config{
		Port:          getEnv("PORT", "8080"),
		GitLabToken:   gitlabToken,
		GitLabHostURL: gitlabHostURL,
		ProjectNames:  projectNames,
		DatabaseURL:   databaseURL,
		CacheTTL:      cacheTTL,
	}, nil
}

func getEnv(key string, defaultVal string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal
	}
	return value
}

func parseDuration(value string) time.Duration {
	d, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("Invalid duration format: %s", value)
	}
	return d
}
