package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Configuration
type Config struct {
	BitbucketURL      string `json:"bitbucket_url"`       // e.g., https://bitbucket.company.com
	BitbucketToken    string `json:"bitbucket_token"`     // Personal access token
	JiraURL           string `json:"jira_url"`            // e.g., https://jira.company.com or https://yoursite.atlassian.net
	JiraUsername      string `json:"jira_username"`       // Email for cloud, username for DC
	JiraToken         string `json:"jira_token"`          // API token for cloud, password for DC
	JiraProject       string `json:"jira_project"`        // Project key
	BitbucketProject  string `json:"bitbucket_project"`   // Project key
	BitbucketRepo     string `json:"bitbucket_repo"`      // Repository slug
	DaysToAnalyze     int    `json:"days_to_analyze"`     // Number of days to look back
	IsJiraCloud       bool   `json:"is_jira_cloud"`       // true for Cloud, false for DC
}

// Data structures
type Commit struct {
	Hash         string    `json:"hash"`
	Author       string    `json:"author"`
	Date         time.Time `json:"date"`
	Message      string    `json:"message"`
	LinesAdded   int       `json:"lines_added"`
	LinesDeleted int       `json:"lines_deleted"`
}

type PullRequest struct {
	ID            string     `json:"id"`
	Author        string     `json:"author"`
	CreatedAt     time.Time  `json:"created_at"`
	MergedAt      *time.Time `json:"merged_at,omitempty"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
	FirstReviewAt *time.Time `json:"first_review_at,omitempty"`
	LinesChanged  int        `json:"lines_changed"`
	Reviewers     []string   `json:"reviewers"`
	Status        string     `json:"status"`
}

type JiraStory struct {
	Key          string     `json:"key"`
	Assignee     string     `json:"assignee"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Estimate     float64    `json:"estimate"`
	ActualEffort float64    `json:"actual_effort"`
	Status       string     `json:"status"`
}

// Bitbucket API responses
type BitbucketCommitsResponse struct {
	Size       int `json:"size"`
	Limit      int `json:"limit"`
	IsLastPage bool `json:"isLastPage"`
	Start      int `json:"start"`
	Values     []struct {
		ID              string `json:"id"`
		DisplayID       string `json:"displayId"`
		Author          struct {
			Name         string `json:"name"`
			EmailAddress string `json:"emailAddress"`
		} `json:"author"`
		AuthorTimestamp int64  `json:"authorTimestamp"`
		Message         string `json:"message"`
	} `json:"values"`
	NextPageStart int `json:"nextPageStart"`
}

type BitbucketPRsResponse struct {
	Size       int `json:"size"`
	Limit      int `json:"limit"`
	IsLastPage bool `json:"isLastPage"`
	Start      int `json:"start"`
	Values     []struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		State       string `json:"state"` // OPEN, MERGED, DECLINED
		CreatedDate int64  `json:"createdDate"`
		UpdatedDate int64  `json:"updatedDate"`
		ClosedDate  int64  `json:"closedDate"`
		Author      struct {
			User struct {
				Name string `json:"name"`
			} `json:"user"`
		} `json:"author"`
		Reviewers []struct {
			User struct {
				Name string `json:"name"`
			} `json:"user"`
			Approved bool `json:"approved"`
		} `json:"reviewers"`
	} `json:"values"`
	NextPageStart int `json:"nextPageStart"`
}

type BitbucketPRDiffResponse struct {
	Diffs []struct {
		Hunks []struct {
			Segments []struct {
				Type  string `json:"type"` // ADDED, REMOVED, CONTEXT
				Lines []struct {
					Line string `json:"line"`
				} `json:"lines"`
			} `json:"segments"`
		} `json:"hunks"`
	} `json:"diffs"`
}

// Jira API responses
type JiraIssuesResponse struct {
	Issues []struct {
		Key    string `json:"key"`
		Fields struct {
			Summary   string `json:"summary"`
			Status    struct {
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
			Changelog      *struct {
				Histories []struct {
					Created string `json:"created"`
					Items   []struct {
						Field      string `json:"field"`
						FromString string `json:"fromString"`
						ToString   string `json:"toString"`
					} `json:"items"`
				} `json:"histories"`
			} `json:"changelog"`
		} `json:"fields"`
	} `json:"issues"`
	Total int `json:"total"`
}

// API Client
type APIClient struct {
	config     Config
	httpClient *http.Client
}

func NewAPIClient(config Config) *APIClient {
	return &APIClient{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *APIClient) makeRequest(url, method, username, token string) ([]byte, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	if username != "" {
		req.SetBasicAuth(username, token)
	} else {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
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

// Fetch Bitbucket Commits
func (c *APIClient) FetchBitbucketCommits() ([]Commit, error) {
	var commits []Commit
	start := 0
	limit := 100
	since := time.Now().AddDate(0, 0, -c.config.DaysToAnalyze)

	for {
		url := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/commits?limit=%d&start=%d",
			c.config.BitbucketURL,
			c.config.BitbucketProject,
			c.config.BitbucketRepo,
			limit,
			start,
		)

		body, err := c.makeRequest(url, "GET", "", c.config.BitbucketToken)
		if err != nil {
			return nil, fmt.Errorf("error fetching commits: %w", err)
		}

		var response BitbucketCommitsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("error parsing commits response: %w", err)
		}

		for _, commit := range response.Values {
			commitDate := time.Unix(commit.AuthorTimestamp/1000, 0)
			if commitDate.Before(since) {
				return commits, nil
			}

			commits = append(commits, Commit{
				Hash:    commit.ID,
				Author:  commit.Author.Name,
				Date:    commitDate,
				Message: commit.Message,
				// Note: Bitbucket API doesn't provide line counts directly
				// You'd need to fetch diff for each commit for accurate counts
				LinesAdded:   0,
				LinesDeleted: 0,
			})
		}

		if response.IsLastPage {
			break
		}
		start = response.NextPageStart
	}

	return commits, nil
}

// Fetch Bitbucket Pull Requests
func (c *APIClient) FetchBitbucketPRs() ([]PullRequest, error) {
	var prs []PullRequest
	start := 0
	limit := 100
	states := []string{"ALL"}

	for _, state := range states {
		start = 0
		for {
			url := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/pull-requests?state=%s&limit=%d&start=%d",
				c.config.BitbucketURL,
				c.config.BitbucketProject,
				c.config.BitbucketRepo,
				state,
				limit,
				start,
			)

			body, err := c.makeRequest(url, "GET", "", c.config.BitbucketToken)
			if err != nil {
				return nil, fmt.Errorf("error fetching PRs: %w", err)
			}

			var response BitbucketPRsResponse
			if err := json.Unmarshal(body, &response); err != nil {
				return nil, fmt.Errorf("error parsing PRs response: %w", err)
			}

			for _, pr := range response.Values {
				createdAt := time.Unix(pr.CreatedDate/1000, 0)
				since := time.Now().AddDate(0, 0, -c.config.DaysToAnalyze)
				
				if createdAt.Before(since) {
					continue
				}

				var mergedAt, closedAt, firstReviewAt *time.Time
				status := strings.ToLower(pr.State)

				if pr.ClosedDate > 0 {
					t := time.Unix(pr.ClosedDate/1000, 0)
					if status == "merged" {
						mergedAt = &t
					} else {
						closedAt = &t
					}
				}

				// Find first review time
				for _, reviewer := range pr.Reviewers {
					if reviewer.Approved && firstReviewAt == nil {
						// Approximate with updated date
						t := time.Unix(pr.UpdatedDate/1000, 0)
						firstReviewAt = &t
						break
					}
				}

				var reviewers []string
				for _, reviewer := range pr.Reviewers {
					reviewers = append(reviewers, reviewer.User.Name)
				}

				// Fetch diff to get line counts
				linesChanged := 0
				diffURL := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/diff",
					c.config.BitbucketURL,
					c.config.BitbucketProject,
					c.config.BitbucketRepo,
					pr.ID,
				)
				
				diffBody, err := c.makeRequest(diffURL, "GET", "", c.config.BitbucketToken)
				if err == nil {
					var diffResp BitbucketPRDiffResponse
					if err := json.Unmarshal(diffBody, &diffResp); err == nil {
						for _, diff := range diffResp.Diffs {
							for _, hunk := range diff.Hunks {
								for _, segment := range hunk.Segments {
									if segment.Type == "ADDED" || segment.Type == "REMOVED" {
										linesChanged += len(segment.Lines)
									}
								}
							}
						}
					}
				}

				prs = append(prs, PullRequest{
					ID:            fmt.Sprintf("PR-%d", pr.ID),
					Author:        pr.Author.User.Name,
					CreatedAt:     createdAt,
					MergedAt:      mergedAt,
					ClosedAt:      closedAt,
					FirstReviewAt: firstReviewAt,
					LinesChanged:  linesChanged,
					Status:        status,
					Reviewers:     reviewers,
				})
			}

			if response.IsLastPage {
				break
			}
			start = response.NextPageStart
		}
	}

	return prs, nil
}

// Fetch Jira Issues
func (c *APIClient) FetchJiraIssues() ([]JiraStory, error) {
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

		var response JiraIssuesResponse
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
			if issue.Fields.Changelog != nil {
				for _, history := range issue.Fields.Changelog.Histories {
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

// Metrics structures (same as before)
type CommitMetrics struct {
	TotalCommits      int            `json:"total_commits"`
	CommitsPerDay     float64        `json:"commits_per_day"`
	CommitsByAuthor   map[string]int `json:"commits_by_author"`
	CommitsByWeekday  map[string]int `json:"commits_by_weekday"`
	TotalLinesAdded   int            `json:"total_lines_added"`
	TotalLinesDeleted int            `json:"total_lines_deleted"`
	ActiveDays        int            `json:"active_days"`
	DateRange         string         `json:"date_range"`
}

type PRMetrics struct {
	TotalPRs           int            `json:"total_prs"`
	MergedPRs          int            `json:"merged_prs"`
	ClosedPRs          int            `json:"closed_prs"`
	OpenPRs            int            `json:"open_prs"`
	AvgCycleTimeHours  float64        `json:"avg_cycle_time_hours"`
	AvgReviewTimeHours float64        `json:"avg_review_time_hours"`
	AvgPRSize          float64        `json:"avg_pr_size"`
	PRsByAuthor        map[string]int `json:"prs_by_author"`
	MergeSuccessRate   float64        `json:"merge_success_rate"`
}

type JiraMetrics struct {
	TotalStories      int            `json:"total_stories"`
	CompletedStories  int            `json:"completed_stories"`
	AvgLeadTimeDays   float64        `json:"avg_lead_time_days"`
	AvgCycleTimeDays  float64        `json:"avg_cycle_time_days"`
	Throughput        float64        `json:"throughput_per_week"`
	AvgEstimate       float64        `json:"avg_estimate"`
	AvgActualEffort   float64        `json:"avg_actual_effort"`
	EstimateAccuracy  float64        `json:"estimate_accuracy_percent"`
	StoriesByAssignee map[string]int `json:"stories_by_assignee"`
}

type TeamMetrics struct {
	CommitMetrics CommitMetrics `json:"commit_metrics"`
	PRMetrics     PRMetrics     `json:"pr_metrics"`
	JiraMetrics   JiraMetrics   `json:"jira_metrics"`
	GeneratedAt   time.Time     `json:"generated_at"`
}

// Metric calculation functions (same as before)
func CalculateCommitMetrics(commits []Commit) CommitMetrics {
	metrics := CommitMetrics{
		CommitsByAuthor:  make(map[string]int),
		CommitsByWeekday: make(map[string]int),
	}

	if len(commits) == 0 {
		return metrics
	}

	metrics.TotalCommits = len(commits)
	activeDaysMap := make(map[string]bool)

	var minDate, maxDate time.Time
	for i, c := range commits {
		if i == 0 || c.Date.Before(minDate) {
			minDate = c.Date
		}
		if i == 0 || c.Date.After(maxDate) {
			maxDate = c.Date
		}

		metrics.CommitsByAuthor[c.Author]++
		weekday := c.Date.Weekday().String()
		metrics.CommitsByWeekday[weekday]++
		metrics.TotalLinesAdded += c.LinesAdded
		metrics.TotalLinesDeleted += c.LinesDeleted

		dateKey := c.Date.Format("2006-01-02")
		activeDaysMap[dateKey] = true
	}

	metrics.ActiveDays = len(activeDaysMap)
	daysDiff := maxDate.Sub(minDate).Hours() / 24
	if daysDiff > 0 {
		metrics.CommitsPerDay = float64(metrics.TotalCommits) / daysDiff
	}
	metrics.DateRange = fmt.Sprintf("%s to %s", minDate.Format("2006-01-02"), maxDate.Format("2006-01-02"))

	return metrics
}

func CalculatePRMetrics(prs []PullRequest) PRMetrics {
	metrics := PRMetrics{
		PRsByAuthor: make(map[string]int),
	}

	if len(prs) == 0 {
		return metrics
	}

	metrics.TotalPRs = len(prs)
	var totalCycleTime, totalReviewTime, totalSize float64
	var cycleTimeCount, reviewTimeCount int

	for _, pr := range prs {
		metrics.PRsByAuthor[pr.Author]++

		switch pr.Status {
		case "merged":
			metrics.MergedPRs++
		case "declined", "closed":
			metrics.ClosedPRs++
		case "open":
			metrics.OpenPRs++
		}

		if pr.MergedAt != nil {
			cycleTime := pr.MergedAt.Sub(pr.CreatedAt).Hours()
			totalCycleTime += cycleTime
			cycleTimeCount++
		}

		if pr.FirstReviewAt != nil {
			reviewTime := pr.FirstReviewAt.Sub(pr.CreatedAt).Hours()
			totalReviewTime += reviewTime
			reviewTimeCount++
		}

		totalSize += float64(pr.LinesChanged)
	}

	if cycleTimeCount > 0 {
		metrics.AvgCycleTimeHours = totalCycleTime / float64(cycleTimeCount)
	}
	if reviewTimeCount > 0 {
		metrics.AvgReviewTimeHours = totalReviewTime / float64(reviewTimeCount)
	}
	if metrics.TotalPRs > 0 {
		metrics.AvgPRSize = totalSize / float64(metrics.TotalPRs)
		metrics.MergeSuccessRate = float64(metrics.MergedPRs) / float64(metrics.TotalPRs) * 100
	}

	return metrics
}

func CalculateJiraMetrics(stories []JiraStory) JiraMetrics {
	metrics := JiraMetrics{
		StoriesByAssignee: make(map[string]int),
	}

	if len(stories) == 0 {
		return metrics
	}

	metrics.TotalStories = len(stories)
	var totalLeadTime, totalCycleTime, totalEstimate, totalActual float64
	var leadTimeCount, cycleTimeCount int

	var minDate, maxDate time.Time
	for i, s := range stories {
		if i == 0 || s.CreatedAt.Before(minDate) {
			minDate = s.CreatedAt
		}
		if s.CompletedAt != nil && (i == 0 || s.CompletedAt.After(maxDate)) {
			maxDate = *s.CompletedAt
		}

		metrics.StoriesByAssignee[s.Assignee]++

		if strings.Contains(strings.ToLower(s.Status), "done") ||
			strings.Contains(strings.ToLower(s.Status), "completed") ||
			strings.Contains(strings.ToLower(s.Status), "resolved") {
			metrics.CompletedStories++
		}

		if s.CompletedAt != nil {
			leadTime := s.CompletedAt.Sub(s.CreatedAt).Hours() / 24
			totalLeadTime += leadTime
			leadTimeCount++

			if s.StartedAt != nil {
				cycleTime := s.CompletedAt.Sub(*s.StartedAt).Hours() / 24
				totalCycleTime += cycleTime
				cycleTimeCount++
			}
		}

		totalEstimate += s.Estimate
		totalActual += s.ActualEffort
	}

	if leadTimeCount > 0 {
		metrics.AvgLeadTimeDays = totalLeadTime / float64(leadTimeCount)
	}
	if cycleTimeCount > 0 {
		metrics.AvgCycleTimeDays = totalCycleTime / float64(cycleTimeCount)
	}
	if metrics.TotalStories > 0 {
		metrics.AvgEstimate = totalEstimate / float64(metrics.TotalStories)
		metrics.AvgActualEffort = totalActual / float64(metrics.TotalStories)
	}
	if totalEstimate > 0 {
		metrics.EstimateAccuracy = (1 - abs(totalActual-totalEstimate)/totalEstimate) * 100
	}

	weeksDiff := maxDate.Sub(minDate).Hours() / 24 / 7
	if weeksDiff > 0 {
		metrics.Throughput = float64(metrics.CompletedStories) / weeksDiff
	}

	return metrics
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Export functions
func ExportToJSON(metrics TeamMetrics, filename string) error {
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func ExportToCSV(metrics TeamMetrics, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"Metric Category", "Metric Name", "Value"})

	writer.Write([]string{"Commits", "Total Commits", strconv.Itoa(metrics.CommitMetrics.TotalCommits)})
	writer.Write([]string{"Commits", "Commits Per Day", fmt.Sprintf("%.2f", metrics.CommitMetrics.CommitsPerDay)})
	writer.Write([]string{"Commits", "Active Days", strconv.Itoa(metrics.CommitMetrics.ActiveDays)})
	writer.Write([]string{"Commits", "Lines Added", strconv.Itoa(metrics.CommitMetrics.TotalLinesAdded)})
	writer.Write([]string{"Commits", "Lines Deleted", strconv.Itoa(metrics.CommitMetrics.TotalLinesDeleted)})

	writer.Write([]string{"Pull Requests", "Total PRs", strconv.Itoa(metrics.PRMetrics.TotalPRs)})
	writer.Write([]string{"Pull Requests", "Merged PRs", strconv.Itoa(metrics.PRMetrics.MergedPRs)})
	writer.Write([]string{"Pull Requests", "Avg Cycle Time (hours)", fmt.Sprintf("%.2f", metrics.PRMetrics.AvgCycleTimeHours)})
	writer.Write([]string{"Pull Requests", "Avg Review Time (hours)", fmt.Sprintf("%.2f", metrics.PRMetrics.AvgReviewTimeHours)})
	writer.Write([]string{"Pull Requests", "Merge Success Rate (%)", fmt.Sprintf("%.2f", metrics.PRMetrics.MergeSuccessRate)})

	writer.Write([]string{"Jira Stories", "Total Stories", strconv.Itoa(metrics.JiraMetrics.TotalStories)})
	writer.Write([]string{"Jira Stories", "Completed Stories", strconv.Itoa(metrics.JiraMetrics.CompletedStories)})
	writer.Write([]string{"Jira Stories", "Avg Lead Time (days)", fmt.Sprintf("%.2f", metrics.JiraMetrics.AvgLeadTimeDays)})
	writer.Write([]string{"Jira Stories", "Avg Cycle Time (days)", fmt.Sprintf("%.2f", metrics.JiraMetrics.AvgCycleTimeDays)})
	writer.Write([]string{"Jira Stories", "Throughput (per week)", fmt.Sprintf("%.2f", metrics.JiraMetrics.Throughput)})
	writer.Write([]string{"Jira Stories", "Estimate Accuracy (%)", fmt.Sprintf("%.2f", metrics.JiraMetrics.EstimateAccuracy)})

	return nil
}

func PrintMetricsSummary(metrics TeamMetrics) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("DEVOPS & PRODUCTIVITY METRICS REPORT")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("\n📊 COMMIT METRICS")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Total Commits: %d\n", metrics.CommitMetrics.TotalCommits)
	fmt.Printf("Commits Per Day: %.2f\n", metrics.CommitMetrics.CommitsPerDay)
	fmt.Printf("Active Days: %d\n", metrics.CommitMetrics.ActiveDays)
	fmt.Printf("Lines Added: %d | Lines Deleted: %d\n",
		metrics.CommitMetrics.TotalLinesAdded, metrics.CommitMetrics.TotalLinesDeleted)
	fmt.Printf("Date Range: %s\n", metrics.CommitMetrics.DateRange)

	fmt.Println("\nCommits by Author:")
	authors := make([]string, 0, len(metrics.CommitMetrics.CommitsByAuthor))
	for author := range metrics.CommitMetrics.CommitsByAuthor {
		authors = append(authors, author)
	}
	sort.Strings(authors)
	for _, author := range authors {
		fmt.Printf("  - %s: %d commits\n", author, metrics.CommitMetrics.CommitsByAuthor[author])
	}

	fmt.Println("\n🔀 PULL REQUEST METRICS")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Total PRs: %d (Merged: %d, Closed: %d, Open: %d)\n",
		metrics.PRMetrics.TotalPRs, metrics.PRMetrics.MergedPRs,
		metrics.PRMetrics.ClosedPRs, metrics.PRMetrics.OpenPRs)
	fmt.Printf("Avg Cycle Time: %.2f hours\n", metrics.PRMetrics.AvgCycleTimeHours)
	fmt.Printf("Avg Review Time: %.2f hours\n", metrics.PRMetrics.AvgReviewTimeHours)
	fmt.Printf("Avg PR Size: %.0f lines\n", metrics.PRMetrics.AvgPRSize)
	fmt.Printf("Merge Success Rate: %.2f%%\n", metrics.PRMetrics.MergeSuccessRate)

	fmt.Println("\n📋 JIRA STORY METRICS")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Total Stories: %d (Completed: %d)\n",
		metrics.JiraMetrics.TotalStories, metrics.JiraMetrics.CompletedStories)
	fmt.Printf("Avg Lead Time: %.2f days\n", metrics.JiraMetrics.AvgLeadTimeDays)
	fmt.Printf("Avg Cycle Time: %.2f days\n", metrics.JiraMetrics.AvgCycleTimeDays)
	fmt.Printf("Throughput: %.2f stories/week\n", metrics.JiraMetrics.Throughput)
	fmt.Printf("Avg Estimate: %.2f | Avg Actual: %.2f\n",
		metrics.JiraMetrics.AvgEstimate, metrics.JiraMetrics.AvgActualEffort)
	fmt.Printf("Estimate Accuracy: %.2f%%\n", metrics.JiraMetrics.EstimateAccuracy)

	fmt.Println("\n" + strings.Repeat("=", 60))
}

// Load configuration from file or environment
func LoadConfig(filename string) (Config, error) {
	// Try loading from file first
	if _, err := os.Stat(filename); err == nil {
		data, err := os.ReadFile(filename)
		if err != nil {
			return Config{}, err
		}
		var config Config
		if err := json.Unmarshal(data, &config); err != nil {
			return Config{}, err
		}
		return config, nil
	}

	// Fall back to environment variables
	config := Config{
		BitbucketURL:     os.Getenv("BITBUCKET_URL"),
		BitbucketToken:   os.Getenv("BITBUCKET_TOKEN"),
		BitbucketProject: os.Getenv("BITBUCKET_PROJECT"),
		BitbucketRepo:    os.Getenv("BITBUCKET_REPO"),
		JiraURL:          os.Getenv("JIRA_URL"),
		JiraUsername:     os.Getenv("JIRA_USERNAME"),
		JiraToken:        os.Getenv("JIRA_TOKEN"),
		JiraProject:      os.Getenv("JIRA_PROJECT"),
		DaysToAnalyze:    30,
		IsJiraCloud:      os.Getenv("JIRA_IS_CLOUD") == "true",
	}

	if days := os.Getenv("DAYS_TO_ANALYZE"); days != "" {
		if d, err := strconv.Atoi(days); err == nil {
			config.DaysToAnalyze = d
		}
	}

	return config, nil
}

// Create sample config file
func CreateSampleConfig() error {
	config := Config{
		BitbucketURL:     "https://bitbucket.company.com",
		BitbucketToken:   "your-bitbucket-token",
		BitbucketProject: "PROJECT",
		BitbucketRepo:    "repository-slug",
		JiraURL:          "https://jira.company.com",
		JiraUsername:     "your-username",
		JiraToken:        "your-jira-token",
		JiraProject:      "PROJ",
		DaysToAnalyze:    30,
		IsJiraCloud:      false,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("config.sample.json", data, 0644)
}

func main() {
	fmt.Println("DevOps & Productivity Metrics Generator with API Integration")
	fmt.Println("============================================================\n")

	// Check for --sample-config flag
	if len(os.Args) > 1 && os.Args[1] == "--sample-config" {
		if err := CreateSampleConfig(); err != nil {
			log.Fatalf("Error creating sample config: %v", err)
		}
		fmt.Println("✅ Sample configuration file created: config.sample.json")
		fmt.Println("\nEdit this file with your credentials and rename to config.json")
		return
	}

	// Load configuration
	config, err := LoadConfig("config.json")
	if err != nil {
		log.Printf("Warning: Could not load config.json, trying environment variables: %v\n", err)
	}

	// Validate configuration
	if config.BitbucketURL == "" || config.JiraURL == "" {
		fmt.Println("❌ Configuration Error!")
		fmt.Println("\nYou need to provide configuration either by:")
		fmt.Println("1. Creating a config.json file (run with --sample-config to generate template)")
		fmt.Println("2. Setting environment variables:")
		fmt.Println("   - BITBUCKET_URL, BITBUCKET_TOKEN, BITBUCKET_PROJECT, BITBUCKET_REPO")
		fmt.Println("   - JIRA_URL, JIRA_USERNAME, JIRA_TOKEN, JIRA_PROJECT")
		fmt.Println("   - JIRA_IS_CLOUD=true (for Jira Cloud)")
		fmt.Println("   - DAYS_TO_ANALYZE=30 (optional, defaults to 30)")
		return
	}

	client := NewAPIClient(config)

	fmt.Printf("Analyzing data from the last %d days...\n\n", config.DaysToAnalyze)

	// Fetch Bitbucket data
	fmt.Println("🔄 Fetching Bitbucket commits...")
	commits, err := client.FetchBitbucketCommits()
	if err != nil {
		log.Printf("❌ Error fetching commits: %v", err)
		commits = []Commit{}
	} else {
		fmt.Printf("✅ Fetched %d commits\n", len(commits))
	}

	fmt.Println("🔄 Fetching Bitbucket pull requests...")
	prs, err := client.FetchBitbucketPRs()
	if err != nil {
		log.Printf("❌ Error fetching PRs: %v", err)
		prs = []PullRequest{}
	} else {
		fmt.Printf("✅ Fetched %d pull requests\n", len(prs))
	}

	// Fetch Jira data
	fmt.Println("🔄 Fetching Jira issues...")
	stories, err := client.FetchJiraIssues()
	if err != nil {
		log.Printf("❌ Error fetching Jira issues: %v", err)
		stories = []JiraStory{}
	} else {
		fmt.Printf("✅ Fetched %d Jira stories\n", len(stories))
	}

	// Calculate metrics
	fmt.Println("\n📊 Calculating metrics...")
	metrics := TeamMetrics{
		CommitMetrics: CalculateCommitMetrics(commits),
		PRMetrics:     CalculatePRMetrics(prs),
		JiraMetrics:   CalculateJiraMetrics(stories),
		GeneratedAt:   time.Now(),
	}

	// Print summary
	PrintMetricsSummary(metrics)

	// Export to files
	if err := ExportToJSON(metrics, "metrics.json"); err != nil {
		log.Printf("Error exporting to JSON: %v", err)
	} else {
		fmt.Println("\n✅ Metrics exported to: metrics.json")
	}

	if err := ExportToCSV(metrics, "metrics.csv"); err != nil {
		log.Printf("Error exporting to CSV: %v", err)
	} else {
		fmt.Println("✅ Metrics exported to: metrics.csv")
	}

	fmt.Println("\n🎉 Analysis complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Review metrics.json for detailed analysis")
	fmt.Println("- Import metrics.csv into spreadsheet for visualization")
	fmt.Println("- Schedule this script to run periodically for tracking trends")
}