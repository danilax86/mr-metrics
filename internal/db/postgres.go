// SPDX-FileCopyrightText: 2025 Danila Gorelko <hello@danilax86.space>
//
// SPDX-License-Identifier: MIT

package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/lib/pq"
	"mr-metrics/internal/consts"
	"mr-metrics/internal/model"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"

	// Blank import is necessary to enable the migrate library to use the file source driver,
	// which is used to load migration scripts from files.
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	maxConns        = 25
	maxConnLifetime = 5 * time.Minute
)

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

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &PostgresStore{db: db}, nil
}

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+path.Join("migrations"),
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

func (p PostgresStore) GetLastUpdatedDate(projectName string) (time.Time, error) {
	var lastUpdated time.Time
	err := p.db.QueryRow(`
        SELECT last_updated
        FROM projects
        WHERE project_name = $1
    `, projectName).Scan(&lastUpdated)
	if err != nil {
		return time.Time{}, err
	}
	return lastUpdated, nil
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
	rows, err := p.db.Query(getAggregatedDataSQL(), pq.Array(projectNames), targetDate)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	devTotals := make(map[string]int)
	repoTotals := make(map[string]int)
	projectsSet := make(map[string]struct{})
	developerStats := make(map[string]map[string]int)

	for rows.Next() {
		var devTotal int
		var count int
		var username, fullProjectName string

		if err := rows.Scan(&username, &fullProjectName, &count, &devTotal); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		projectName := extractProjectName(fullProjectName)
		projectsSet[projectName] = struct{}{}

		if username == "TOTAL" {
			// Repository total merged mrs row
			repoTotals[projectName] = count
		} else {
			if _, exists := developerStats[username]; !exists {
				developerStats[username] = make(map[string]int)
			}
			devTotals[username] = devTotal
			developerStats[username][projectName] = count
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	projects := sortedKeys(projectsSet)

	return &model.AggregatedStats{
		Developers: developerStats,
		Projects:   projects,
		DevTotals:  devTotals,
		RepoTotals: repoTotals,
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
		date := mr.MergedAt.UTC().Truncate(consts.OneDay)
		if _, exists := userDates[mr.Username]; !exists {
			userDates[mr.Username] = make(map[time.Time]int)
		}
		userDates[mr.Username][date]++
	}
	return userDates
}

// updateDailyCumulativeCounts updates the daily cumulative counts of merge requests for each user in a project.
func updateDailyCumulativeCounts(tx *sql.Tx, userDates map[string]map[time.Time]int, projectID int) error {
	for username, dates := range userDates {
		sortedDates := getSortedDates(dates)

		if err := updateUserCounts(tx, username, projectID, sortedDates, dates); err != nil {
			return err
		}
	}
	return nil
}

// getSortedDates returns a slice of dates sorted in ascending order.
func getSortedDates(dates map[time.Time]int) []time.Time {
	sortedDates := make([]time.Time, 0, len(dates))
	for date := range dates {
		sortedDates = append(sortedDates, date)
	}
	sort.Slice(sortedDates, func(i, j int) bool {
		return sortedDates[i].Before(sortedDates[j])
	})
	return sortedDates
}

// updateUserCounts updates the cumulative counts for a specific user.
func updateUserCounts(tx *sql.Tx, username string, projectID int, sortedDates []time.Time, dates map[time.Time]int) error {
	// NOTE(d.gorelko): Get the latest cumulative count before the earliest date in the current batch.
	cumulative := 0
	if len(sortedDates) > 0 {
		earliestDate := sortedDates[0]
		err := tx.QueryRow(`
			SELECT merge_count
			FROM merged_mrs
			WHERE username = $1 AND project_id = $2 AND merged_at < $3
			ORDER BY merged_at DESC
			LIMIT 1
		`, username, projectID, earliestDate).Scan(&cumulative)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("failed to get previous cumulative count: %w", err)
		}
	}

	for _, date := range sortedDates {
		cumulative += dates[date]

		if err := updateOrInsertCount(tx, username, projectID, date, cumulative); err != nil {
			return err
		}
	}
	return nil
}

// updateOrInsertCount updates an existing row or inserts a new one for the given date.
func updateOrInsertCount(tx *sql.Tx, username string, projectID int, date time.Time, cumulative int) error {
	var existingCount int
	err := tx.QueryRow(`
		SELECT merge_count
		FROM merged_mrs
		WHERE username = $1 AND project_id = $2 AND merged_at = $3
	`, username, projectID, date).Scan(&existingCount)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existing row: %w", err)
	}

	if existingCount > 0 {
		return updateExistingCount(tx, username, projectID, date, cumulative)
	}

	return insertNewCount(tx, username, projectID, date, cumulative)
}

// updateExistingCount updates an existing row in the database.
func updateExistingCount(tx *sql.Tx, username string, projectID int, date time.Time, cumulative int) error {
	_, err := tx.Exec(`
		UPDATE merged_mrs
		SET merge_count = $1
		WHERE username = $2 AND project_id = $3 AND merged_at = $4
	`, cumulative, username, projectID, date)

	if err != nil {
		return fmt.Errorf("failed to update existing row: %w", err)
	}
	return nil
}

// insertNewCount inserts a new row into the database.
func insertNewCount(tx *sql.Tx, username string, projectID int, date time.Time, cumulative int) error {
	_, err := tx.Exec(`
		INSERT INTO merged_mrs (username, project_id, merge_count, merged_at)
		VALUES ($1, $2, $3, $4)
	`, username, projectID, cumulative, date)

	if err != nil {
		return fmt.Errorf("failed to add row: %w", err)
	}
	return nil
}

func getAggregatedDataSQL() string {
	return `
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
        ),
		user_totals AS (
			SELECT
				username,
				SUM(merge_count) as user_total_mrs
			FROM latest_data
			GROUP BY username
		),
		repo_totals AS (
			SELECT
				project_name,
				SUM(merge_count) as repo_total_mrs
			FROM latest_data
			GROUP BY project_name
		)
        SELECT 
            p.username,
            p.project_name,
            p.merge_count,
            t.user_total_mrs
        FROM latest_data p
        JOIN user_totals t ON p.username = t.username

		UNION ALL

		SELECT
			'TOTAL' as username,
			project_name,
			repo_total_mrs as merge_count,
			0 as user_total
		FROM repo_totals
		ORDER BY username, project_name;
    `
}
