// SPDX-FileCopyrightText: 2025 Danila Gorelko <hello@danilax86.space>
//
// SPDX-License-Identifier: MIT

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"mr-metrics/internal/config"
	"mr-metrics/internal/model"
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
	ProjectID int        `json:"project_id"`
	MergedAt  *time.Time `json:"merged_at"`
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
func (g *GitLabClient) GetMergedMRCounts(projectName string, since time.Time) ([]model.MergeRequest, int, error) {
	var (
		mrs       []model.MergeRequest
		projectID int
		page      = 1
	)

	for {
		endpointURL := g.getMergeRequestsEndpointURL(projectName, since, page)
		resp, err := g.sendGetRequest(endpointURL)
		if err != nil {
			return nil, 0, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, 0, fmt.Errorf("API returned %d", resp.StatusCode)
		}

		apiMRs, err := g.decodeMergeRequests(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, 0, err
		}

		if len(apiMRs) == 0 {
			resp.Body.Close()
			break
		}

		if projectID == 0 {
			projectID = apiMRs[0].ProjectID
		}

		mrs = append(mrs, g.extractMergeRequests(apiMRs)...)

		if !g.hasNextPage(resp.Header) {
			resp.Body.Close()
			break
		}
		resp.Body.Close()
		page++
	}

	return mrs, projectID, nil
}

func (g *GitLabClient) getMergeRequestsEndpointURL(projectName string, since time.Time, page int) string {
	return fmt.Sprintf(
		"%s/projects/%s/merge_requests?state=merged&page=%d&updated_after=%s&per_page=100",
		g.baseURL, pathEscape(projectName), page, since.Format(time.RFC3339),
	)
}

func (g *GitLabClient) sendGetRequest(endpointURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Add("Private-Token", g.token)

	return g.client.Do(req)
}

func (g *GitLabClient) decodeMergeRequests(body io.Reader) ([]ProjectMRResponse, error) {
	var apiMRs []ProjectMRResponse
	if err := json.NewDecoder(body).Decode(&apiMRs); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	return apiMRs, nil
}

func (g *GitLabClient) extractMergeRequests(apiMRs []ProjectMRResponse) []model.MergeRequest {
	mrs := make([]model.MergeRequest, 0, len(apiMRs))
	for _, mr := range apiMRs {
		if mr.Author.Username == "" || mr.MergedAt == nil {
			continue
		}
		mrs = append(mrs, model.MergeRequest{
			Username: mr.Author.Username,
			MergedAt: *mr.MergedAt,
		})
	}
	return mrs
}

func (g *GitLabClient) hasNextPage(header http.Header) bool {
	return header.Get("X-Next-Page") != ""
}

func pathEscape(s string) string {
	return strings.ReplaceAll(url.PathEscape(s), ".", "%2F")
}
