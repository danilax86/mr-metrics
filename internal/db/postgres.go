package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"mr-metrics/internal/model"
	"sort"
	"strings"
	"time"
)

const (
	maxConns        = 25
	maxConnLifetime = 5 * time.Minute
)

const oneDay = 24 * time.Hour

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxConns)
	db.SetConnMaxLifetime(maxConnLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func (p PostgresStore) UpdateProjectCache(projectID int, projectName string, mrs []model.MergeRequest) error {
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO projects(project_id, project_name, last_updated) 
		VALUES($1, $2, NOW())
		ON CONFLICT(project_id) DO UPDATE SET 
			project_name = EXCLUDED.project_name,
			last_updated = NOW()
	`, projectID, projectName)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	userDates := groupMRsByUserAndDate(mrs)

	if err := updateDailyCumulativeCounts(tx, userDates, projectID); err != nil {
		return fmt.Errorf("failed to update daily cumulative counts: %w", err)
	}

	return tx.Commit()
}

func (p PostgresStore) GetAggregatedDataForDate(projectNames []string, targetDate time.Time) (*model.AggregatedStats, error) {
	rows, err := p.db.Query(`
        WITH latest_data AS (
            SELECT DISTINCT ON (m.username, p.project_id)
                m.username,
                p.project_name,
                m.merge_count
            FROM merged_mrs m
            JOIN projects p ON m.project_id = p.project_id
            WHERE p.project_name = ANY($1)
            AND m.merged_at <= $2
            ORDER BY m.username, p.project_id, m.merged_at DESC
        )
        SELECT 
            username,
            project_name,
            merge_count
        FROM latest_data
    `, pq.Array(projectNames), targetDate)

	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	developerStats := make(map[string]map[string]int)
	projectsSet := make(map[string]struct{})

	for rows.Next() {
		var username, fullProjectName string
		var count int

		if err := rows.Scan(&username, &fullProjectName, &count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		projectName := extractProjectName(fullProjectName)
		if _, exists := developerStats[username]; !exists {
			developerStats[username] = make(map[string]int)
		}
		developerStats[username][projectName] = count
		projectsSet[projectName] = struct{}{}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	projects := sortedKeys(projectsSet)

	return &model.AggregatedStats{
		Developers: developerStats,
		Projects:   projects,
	}, nil
}

func extractProjectName(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return fullName
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func groupMRsByUserAndDate(mrs []model.MergeRequest) map[string]map[time.Time]int {
	userDates := make(map[string]map[time.Time]int)
	for _, mr := range mrs {
		date := mr.MergedAt.UTC().Truncate(oneDay)
		if _, exists := userDates[mr.Username]; !exists {
			userDates[mr.Username] = make(map[time.Time]int)
		}
		userDates[mr.Username][date]++
	}
	return userDates
}

func updateDailyCumulativeCounts(tx *sql.Tx, userDates map[string]map[time.Time]int, projectID int) error {
	for username, dates := range userDates {
		var sortedDates []time.Time
		for date := range dates {
			sortedDates = append(sortedDates, date)
		}
		sort.Slice(sortedDates, func(i, j int) bool {
			return sortedDates[i].Before(sortedDates[j])
		})

		cumulative := 0
		for _, date := range sortedDates {
			cumulative += dates[date]

			// Check if a row already exists for the given date
			var existingCount int
			err := tx.QueryRow(`
				SELECT merge_count
				FROM merged_mrs
				WHERE username = $1 AND project_id = $2 AND merged_at = $3
			`, username, projectID, date).Scan(&existingCount)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to check existing row: %w", err)
			}

			// If a row exists, update the merge_count
			if existingCount > 0 {
				_, err = tx.Exec(`
					UPDATE merged_mrs
					SET merge_count = $1
					WHERE username = $2 AND project_id = $3 AND merged_at = $4
				`, cumulative, username, projectID, date)
				if err != nil {
					return fmt.Errorf("failed to update existing row: %w", err)
				}
			} else {
				// If no row exists, insert a new one
				_, err = tx.Exec(`
					INSERT INTO merged_mrs (username, project_id, merge_count, merged_at)
					VALUES ($1, $2, $3, $4)
				`, username, projectID, cumulative, date)
				if err != nil {
					return fmt.Errorf("failed to add row: %w", err)
				}
			}
		}
	}
	return nil
}
