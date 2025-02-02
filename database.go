package main

import (
	"database/sql"
	"errors"
	"github.com/lib/pq"
	"sort"
	"strings"
	"time"
)

func getLastUpdated(projectID int) (time.Time, error) {
	var lastUpdated time.Time
	err := db.QueryRow(
		"SELECT last_updated FROM projects WHERE projects.project_id = $1",
		projectID,
	).Scan(&lastUpdated)

	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	return lastUpdated, err
}

func updateProjectCache(projectID int, projectName string, counts map[string]int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO projects(project_id, project_name, last_updated) 
		VALUES($1, $2, NOW())
		ON CONFLICT(project_id) DO UPDATE SET 
			project_name = EXCLUDED.project_name,
			last_updated = NOW()
	`, projectID, projectName)

	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("DELETE FROM merged_mrs WHERE project_id = $1", projectID)
	if err != nil {
		tx.Rollback()
		return err
	}

	stmt, err := tx.Prepare(pq.CopyIn("merged_mrs", "username", "project_id", "merge_count"))
	if err != nil {
		tx.Rollback()
		return err
	}

	for username, count := range counts {
		_, err = stmt.Exec(username, projectID, count)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		tx.Rollback()
		return err
	}

	if err = stmt.Close(); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func getAggregatedData() (interface{}, error) {
	type ViewData struct {
		Developers map[string]map[string]int `json:"developers"`
		Projects   []string                  `json:"projects"`
	}

	projectNameWithoutGroup := func(projectName string) string {
		return strings.Split(projectName, "/")[1]
	}

	rows, err := db.Query(`
		WITH project_list AS (
			SELECT project_id, project_name 
			FROM projects 
			WHERE project_name = ANY($1)
		)
		SELECT 
			m.username, 
			p.project_name, 
			SUM(m.merge_count) as total_count
		FROM merged_mrs m
		JOIN project_list p USING (project_id)
		GROUP BY 1, 2
	`, pq.Array(config.Projects))

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Map to store results
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
		developerStats[username][projectNameWithoutGroup(projectName)] = count
		projectsSet[projectNameWithoutGroup(projectName)] = struct{}{}
	}

	// Convert projects set to sorted slice
	projects := make([]string, 0, len(projectsSet))
	for project := range projectsSet {
		projects = append(projects, project)
	}
	sort.Strings(projects)

	return ViewData{
		Developers: developerStats,
		Projects:   projects,
	}, nil
}
