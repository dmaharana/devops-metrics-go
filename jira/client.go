package jira

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"devops-metrics/config"
)

// Client handles Jira API operations
type Client struct {
	config config.Config
}

// Jira API response structures
type jiraIssuesResponse struct {
	Issues []struct {
		Key       string `json:"key"`
		Expand    string `json:"expand"`
		Fields    struct {
			Summary        string `json:"summary"`
			Status         struct {
				Name string `json:"name"`
			} `json:"status"`
			Assignee *struct {
				DisplayName string `json:"displayName"`
				Name        string `json:"name"`
			} `json:"assignee"`
			Created        string  `json:"created"`
			Updated        string  `json:"updated"`
			Resolutiondate *string `json:"resolutiondate"`
			StoryPoints    float64 `json:"customfield_10016"` // Common story points field
			TimeEstimate   int     `json:"timeestimate"`
			TimeSpent      int     `json:"timespent"`
		} `json:"fields"`
		Changelog *struct {
			Histories []struct {
				Created string `json:"created"`
				Items   []struct {
					Field      string `json:"field"`
					FromString string `json:"fromString"`
					ToString   string `json:"toString"`
				} `json:"items"`
			} `json:"histories"`
		} `json:"changelog"`
	} `json:"issues"`
	Total int `json:"total"`
}

// NewClient creates a new Jira client
func NewClient(config config.Config) Client {
	return Client{
		config: config,
	}
}

// makeRequest makes an HTTP request with proper authentication
func (c Client) makeRequest(url, method, username, token string) ([]byte, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	if username != "" {
		req.SetBasicAuth(username, token)
	} else {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// FetchIssues retrieves issues from Jira
func (c Client) FetchIssues() ([]JiraStory, error) {
	var stories []JiraStory
	startAt := 0
	maxResults := 100
	since := time.Now().AddDate(0, 0, -c.config.DaysToAnalyze).Format("2006-01-02")

	for {
		jql := fmt.Sprintf("project = %s AND created >= %s ORDER BY created DESC",
			c.config.JiraProject, since)

		var url string
		if c.config.IsJiraCloud {
			url = fmt.Sprintf("%s/rest/api/3/search?jql=%s&maxResults=%d&startAt=%d&expand=changelog",
				c.config.JiraURL, jql, maxResults, startAt)
		} else {
			url = fmt.Sprintf("%s/rest/api/2/search?jql=%s&maxResults=%d&startAt=%d&expand=changelog",
				c.config.JiraURL, jql, maxResults, startAt)
		}

		body, err := c.makeRequest(url, "GET", c.config.JiraUsername, c.config.JiraToken)
		if err != nil {
			return nil, fmt.Errorf("error fetching Jira issues: %w", err)
		}

		var response jiraIssuesResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("error parsing Jira response: %w", err)
		}

		for _, issue := range response.Issues {
			createdAt, _ := time.Parse(time.RFC3339, issue.Fields.Created)

			var completedAt, startedAt *time.Time
			if issue.Fields.Resolutiondate != nil && *issue.Fields.Resolutiondate != "" {
				t, _ := time.Parse(time.RFC3339, *issue.Fields.Resolutiondate)
				completedAt = &t
			}

			// Find when issue moved to "In Progress"
			if issue.Changelog != nil {
				for _, history := range issue.Changelog.Histories {
					for _, item := range history.Items {
						if item.Field == "status" &&
							(strings.Contains(strings.ToLower(item.ToString), "progress") ||
								strings.Contains(strings.ToLower(item.ToString), "development")) {
							t, _ := time.Parse(time.RFC3339, history.Created)
							if startedAt == nil || t.Before(*startedAt) {
								startedAt = &t
							}
						}
					}
				}
			}

			assignee := "Unassigned"
			if issue.Fields.Assignee != nil {
				if c.config.IsJiraCloud {
					assignee = issue.Fields.Assignee.DisplayName
				} else {
					assignee = issue.Fields.Assignee.Name
				}
			}

			estimate := issue.Fields.StoryPoints
			if estimate == 0 && issue.Fields.TimeEstimate > 0 {
				estimate = float64(issue.Fields.TimeEstimate) / 3600 // Convert seconds to hours
			}

			actualEffort := float64(0)
			if issue.Fields.TimeSpent > 0 {
				actualEffort = float64(issue.Fields.TimeSpent) / 3600
			}

			stories = append(stories, JiraStory{
				Key:          issue.Key,
				Assignee:     assignee,
				CreatedAt:    createdAt,
				StartedAt:    startedAt,
				CompletedAt:  completedAt,
				Estimate:     estimate,
				ActualEffort: actualEffort,
				Status:       issue.Fields.Status.Name,
			})
		}

		if len(response.Issues) < maxResults {
			break
		}
		startAt += maxResults
	}

	return stories, nil
}