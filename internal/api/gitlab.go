package api

import (
	"encoding/json"
	"fmt"
	"mr-metrics/internal/config"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout = 15 * time.Second
)

type GitLabClient struct {
	token   string
	client  *http.Client
	baseURL string
}

type ProjectMRResponse struct {
	Author struct {
		Username string `json:"username"`
	} `json:"author"`
	ProjectID int `json:"project_id"`
}

func NewGitLabClient(cfg *config.Config) *GitLabClient {
	return &GitLabClient{
		token: cfg.GitLabToken,
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		baseURL: strings.TrimSuffix(cfg.GitLabHostURL, "/") + "/api/v4",
	}
}

// GetMergedMRCounts returns merged MR counts per user for a project.
func (g *GitLabClient) GetMergedMRCounts(projectName string) (map[string]int, int, error) {
	counts := make(map[string]int)
	page := 1
	var projectID int

	for {
		endpointURL := fmt.Sprintf("%s/projects/%s/merge_requests?state=merged&page=%d&per_page=100",
			g.baseURL, pathEscape(projectName), page)

		req, err := http.NewRequest(http.MethodGet, endpointURL, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("create request failed: %w", err)
		}
		req.Header.Add("Private-Token", g.token)

		resp, err := g.client.Do(req)
		if err != nil {
			return nil, 0, fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, 0, fmt.Errorf("API returned %d", resp.StatusCode)
		}

		var mrs []ProjectMRResponse
		if err := json.NewDecoder(resp.Body).Decode(&mrs); err != nil {
			resp.Body.Close()
			return nil, 0, fmt.Errorf("decode failed: %w", err)
		}
		resp.Body.Close()

		projectID = mrs[0].ProjectID

		for _, mr := range mrs {
			if mr.Author.Username != "" {
				counts[mr.Author.Username]++
			}
		}

		if resp.Header.Get("X-Next-Page") == "" {
			break
		}
		page++
	}

	return counts, projectID, nil
}

func pathEscape(s string) string {
	return strings.ReplaceAll(url.PathEscape(s), ".", "%2F")
}
