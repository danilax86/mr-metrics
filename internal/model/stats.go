package model

type AggregatedStats struct {
	Developers map[string]map[string]int
	Projects   []string
	DateString string
}

type ProjectMRCounts struct {
	ProjectID   int
	ProjectName string
	Counts      map[string]int
}
