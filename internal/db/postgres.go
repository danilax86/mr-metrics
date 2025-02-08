package db

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"mr-metrics/internal/model"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(connStr string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func (p PostgresStore) UpdateProjectCache(projectID int, projectName string, counts map[string]int) error {
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update project metadata
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

	// Batch insert new metrics
	stmt, err := tx.Prepare(pq.CopyIn("merged_mrs", "username", "project_id", "merge_count"))
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert: %w", err)
	}

	for username, count := range counts {
		_, err = stmt.Exec(username, projectID, count)
		if err != nil {
			return fmt.Errorf("failed to add row to batch: %w", err)
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return fmt.Errorf("failed to execute batch insert: %w", err)
	}

	if err = stmt.Close(); err != nil {
		return fmt.Errorf("failed to close statement: %w", err)
	}

	return tx.Commit()
}

func (p PostgresStore) GetAggregatedData(projectNames []string) (*model.AggregatedStats, error) {
	rows, err := p.db.Query(`
		SELECT DISTINCT ON (m.username, p.project_id)
			m.username,
			p.project_name,
			m.merge_count
		FROM merged_mrs m
		JOIN projects p ON m.project_id = p.project_id
		WHERE p.project_name = ANY($1)
		ORDER BY m.username, p.project_id, m.created_at DESC
	`, pq.Array(projectNames))

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	developerStats := make(map[string]map[string]int)
	projectsSet := make(map[string]struct{})

	for rows.Next() {
		var username, projectName string
		var count int
		if err := rows.Scan(&username, &projectName, &count); err != nil {
			return nil, err
		}

		if _, exists := developerStats[username]; !exists {
			developerStats[username] = make(map[string]int)
		}

		projectName = extractProjectName(projectName)

		developerStats[username][projectName] = count
		projectsSet[projectName] = struct{}{}
	}

	projects := sortedKeys(projectsSet)

	return &model.AggregatedStats{
		Developers: developerStats,
		Projects:   projects,
	}, nil
}

func (p PostgresStore) GetAggregatedDataForDate(projectNames []string, targetDate time.Time) (*model.AggregatedStats, error) {
	query := `
			WITH latest_data AS (
				SELECT DISTINCT ON (m.username, p.project_id)
					m.username,
					p.project_name,
					m.merge_count,
					m.created_at
				FROM merged_mrs m
				JOIN projects p ON m.project_id = p.project_id
				WHERE p.project_name = ANY($1)
				AND m.created_at <= $2
				ORDER BY m.username, p.project_id, m.created_at DESC
			)
			SELECT 
				username,
				project_name,
				merge_count as merge_count
			FROM latest_data
		`

	rows, err := p.db.Query(
		query,
		pq.Array(projectNames),
		targetDate.UTC(),
	)

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
