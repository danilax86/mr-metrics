package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func getProjectID(projectName string) (int, error) {
	url := fmt.Sprintf(
		"https://%s/api/v4/projects/%s",
		config.GitLabHost,
		url.PathEscape(projectName),
	)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("PRIVATE-TOKEN", config.GitLabToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch project ID: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API returned %d for project %s", resp.StatusCode, projectName)
	}

	var project struct {
		ID   int    `json:"id"`
		Name string `json:"name_with_namespace"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return 0, fmt.Errorf("failed to decode project response: %v", err)
	}

	return project.ID, nil
}

func fetchProjectMRs(projectID int) (map[string]int, error) {
	counts := make(map[string]int)
	page := 1

	for {
		url := fmt.Sprintf(
			"https://%s/api/v4/projects/%d/merge_requests?state=merged&page=%d",
			config.GitLabHost,
			projectID,
			page,
		)

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Add("PRIVATE-TOKEN", config.GitLabToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API returned %d", resp.StatusCode)
		}

		var mrs []struct {
			Author struct {
				Username string `json:"username"`
			} `json:"author"`
		}

		json.NewDecoder(resp.Body).Decode(&mrs)

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

	return counts, nil
}
