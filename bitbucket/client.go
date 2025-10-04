package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"devops-metrics/config"
)

// Client handles Bitbucket API operations
type Client struct {
	config config.Config
}

// Bitbucket API responses
type bitbucketBranchesResponse struct {
	Size        int `json:"size"`
	Limit       int `json:"limit"`
	IsLastPage  bool `json:"isLastPage"`
	Start       int `json:"start"`
	Values      []struct {
		ID   string `json:"id"`
		DisplayID string `json:"displayId"`
		LatestCommit struct {
			ID string `json:"id"`
		} `json:"latestCommit"`
	} `json:"values"`
	NextPageStart int `json:"nextPageStart"`
}

type bitbucketCommitsResponse struct {
	Size        int `json:"size"`
	Limit       int `json:"limit"`
	IsLastPage  bool `json:"isLastPage"`
	Start       int `json:"start"`
	Values      []struct {
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

type bitbucketPRsResponse struct {
	Size        int `json:"size"`
	Limit       int `json:"limit"`
	IsLastPage  bool `json:"isLastPage"`
	Start       int `json:"start"`
	Values      []struct {
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

type bitbucketPRDiffResponse struct {
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

// NewClient creates a new Bitbucket client
func NewClient(config config.Config) Client {
	return Client{
		config: config,
	}
}

// makeRequest makes an HTTP request with proper authentication and exponential backoff for 429 errors
func (c Client) makeRequest(url, method, username, token string) ([]byte, error) {
	const maxRetries = 5
	const baseDelay = 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
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

		if resp.StatusCode == http.StatusOK {
			return io.ReadAll(resp.Body)
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRetries {
			// Exponential backoff with jitter
			delay := time.Duration(baseDelay.Nanoseconds() * (1 << attempt))
			// Add jitter (up to 50%)
			jitter := time.Duration(time.Now().UnixNano()%int64(time.Second/2)) % (delay / 2)
			time.Sleep(delay + jitter)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("API request failed after %d attempts", maxRetries+1)
}

// FetchCommits retrieves commits from all branches in Bitbucket
func (c Client) FetchCommits() ([]Commit, error) {
	// Get all branches first
	branches, err := c.getBranches()
	if err != nil {
		return nil, fmt.Errorf("error fetching branches: %w", err)
	}

	var allCommits []Commit
	since := time.Now().AddDate(0, 0, -c.config.DaysToAnalyze)

	// Process branches starting with those that have the most recent commits
	for _, branch := range branches {
		branchCommits, shouldContinue, err := c.fetchCommitsFromBranch(branch, since)
		if err != nil {
			// Log error but continue with other branches
			fmt.Printf("Error fetching commits from branch %s: %v\n", branch.DisplayID, err)
			continue
		}

		allCommits = append(allCommits, branchCommits...)

		// If no commits in time range were found in this branch and we already have commits,
		// we can skip remaining branches (assuming branches are sorted by latest activity)
		if !shouldContinue && len(allCommits) > 0 {
			break
		}
	}

	return allCommits, nil
}

// BranchWithActivity represents a branch with its latest commit info for sorting
type BranchWithActivity struct {
	ID        string
	DisplayID string
}

// getBranches retrieves all branches from the repository and sorts them by activity
func (c Client) getBranches() ([]BranchWithActivity, error) {
	var branches []BranchWithActivity
	start := 0
	limit := 100

	for {
		url := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/branches?limit=%d&start=%d",
			c.config.BitbucketURL,
			c.config.BitbucketProject,
			c.config.BitbucketRepo,
			limit,
			start,
		)

		body, err := c.makeRequest(url, "GET", "", c.config.BitbucketToken)
		if err != nil {
			return nil, fmt.Errorf("error fetching branches: %w", err)
		}

		var response bitbucketBranchesResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("error parsing branches response: %w", err)
		}

		for _, branch := range response.Values {
			branches = append(branches, BranchWithActivity{
				ID:        branch.ID,
				DisplayID: branch.DisplayID,
			})
		}

		if response.IsLastPage {
			break
		}
		start = response.NextPageStart
	}

	// Sort branches - we'll keep the default order for now, but could be enhanced to sort by latest commit date
	return branches, nil
}

// fetchCommitsFromBranch retrieves commits from a specific branch and returns whether to continue checking other branches
func (c Client) fetchCommitsFromBranch(branch BranchWithActivity, since time.Time) ([]Commit, bool, error) {
	var commits []Commit
	start := 0
	limit := 100
	hasRecentCommits := false

	for {
		url := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/commits?limit=%d&start=%d&until=%s",
			c.config.BitbucketURL,
			c.config.BitbucketProject,
			c.config.BitbucketRepo,
			limit,
			start,
			branch.ID,
		)

		body, err := c.makeRequest(url, "GET", "", c.config.BitbucketToken)
		if err != nil {
			return nil, true, fmt.Errorf("error fetching commits for branch %s: %w", branch.DisplayID, err)
		}

		var response bitbucketCommitsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, true, fmt.Errorf("error parsing commits response for branch %s: %w", branch.DisplayID, err)
		}

		for _, commit := range response.Values {
			commitDate := time.Unix(commit.AuthorTimestamp/1000, 0)
			if commitDate.Before(since) {
				// No more recent commits in this branch
				return commits, hasRecentCommits, nil
			}

			hasRecentCommits = true
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

	return commits, hasRecentCommits, nil
}

// FetchPRs retrieves pull requests from Bitbucket
func (c Client) FetchPRs() ([]PullRequest, error) {
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

			var response bitbucketPRsResponse
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
				status := pr.State

				if pr.ClosedDate > 0 {
					t := time.Unix(pr.ClosedDate/1000, 0)
					if status == "MERGED" {
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
					var diffResp bitbucketPRDiffResponse
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