// SPDX-FileCopyrightText: 2025 Danila Gorelko <hello@danilax86.space>
//
// SPDX-License-Identifier: MIT

package model

import "time"

type AggregatedStats struct {
	Developers map[string]map[string]int
	Projects   []string
	DateString string
	// Total amount of merged requests per developer
	DevTotals map[string]int
	// Total amount of merged requests per repo
	RepoTotals map[string]int
}

type ProjectMRCounts struct {
	ProjectID   int
	ProjectName string
	Counts      map[string]int
}

type MergeRequest struct {
	Username string
	MergedAt time.Time
}
