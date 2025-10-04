package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"devops-metrics/config"
)

// Client handles GitHub API operations using direct HTTP calls
type Client struct {
	config config.Config
}

// NewClient creates a new GitHub client
func NewClient(config config.Config) Client {
	return Client{
		config: config,
	}
}

// GitHub API response structures
type githubCommitsResponse struct {
	Hash    string `json:"sha"`
	Author  struct {
		Login string `json:"login"`
	} `json:"author"`
	Commit struct {
		Author struct {
			Date  time.Time `json:"date"`
			Name  string  `json:"name"`
			Email string  `json:"email"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
}

type githubBranchesResponse struct {
	Name string `json:"name"`
}

type githubPRsResponse struct {
	Number       int    `json:"number"`
	State        string `json:"state"`
	Title        string `json:"title"`
	User         struct {
		Login string `json:"login"`
	} `json:"user"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MergedAt     *time.Time `json:"merged_at"`
	ClosedAt     *time.Time `json:"closed_at"`
	Additions    int       `json:"additions"`
	Deletions    int       `json:"deletions"`
	ChangedFiles int       `json:"changed_files"`
}

type githubReviewsResponse struct {
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	State       string    `json:"state"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// makeRequest makes an HTTP request with proper authentication
func (c Client) makeRequest(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+c.config.GitHubToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "devops-metrics")

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

// FetchCommits retrieves commits from GitHub
func (c Client) FetchCommits() ([]Commit, error) {
	var commits []Commit
	since := time.Now().AddDate(0, 0, -c.config.DaysToAnalyze)
	
	// Get all branches first
	branchesURL := fmt.Sprintf("%s/repos/%s/%s/branches", c.getBaseURL(), c.config.GitHubOwner, c.config.GitHubRepo)
	branchBody, err := c.makeRequest(branchesURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching branches: %w", err)
	}
	
	var branches []githubBranchesResponse
	if err := json.Unmarshal(branchBody, &branches); err != nil {
		return nil, fmt.Errorf("error parsing branches: %w", err)
	}
	
	for _, branch := range branches {
		page := 1
		for {
			commitsURL := fmt.Sprintf("%s/repos/%s/%s/commits?sha=%s&since=%s&page=%d&per_page=100",
				c.getBaseURL(), c.config.GitHubOwner, c.config.GitHubRepo, branch.Name,
				since.Format(time.RFC3339), page)
			
			commitBody, err := c.makeRequest(commitsURL)
			if err != nil {
				fmt.Printf("Error fetching commits from branch %s: %v\n", branch.Name, err)
				break
			}
			
			var commitList []githubCommitsResponse
			if err := json.Unmarshal(commitBody, &commitList); err != nil {
				break
			}
			
			for _, commit := range commitList {
				commitDate := commit.Commit.Author.Date
				if commitDate.Before(since) {
					break
				}
				
				author := commit.Author.Login
				if author == "" && commit.Commit.Author.Name != "" {
					author = commit.Commit.Author.Name
				}
				
				commits = append(commits, Commit{
					Hash:    commit.Hash,
					Author:  author,
					Date:    commitDate,
					Message: commit.Commit.Message,
					// Line counts require additional API calls
					LinesAdded:   0,
					LinesDeleted: 0,
				})
			}
			
			if len(commitList) < 100 {
				break
			}
			page++
		}
	}
	
	return commits, nil
}

// FetchPRs retrieves pull requests from GitHub
func (c Client) FetchPRs() ([]PullRequest, error) {
	var prs []PullRequest
	since := time.Now().AddDate(0, 0, -c.config.DaysToAnalyze)
	
	page := 1
	for {
		prsURL := fmt.Sprintf("%s/repos/%s/%s/pulls?state=all&sort=updated&direction=desc&page=%d&per_page=100",
			c.getBaseURL(), c.config.GitHubOwner, c.config.GitHubRepo, page)
		
		prBody, err := c.makeRequest(prsURL)
		if err != nil {
			return nil, fmt.Errorf("error fetching PRs: %w", err)
		}
		
		var prList []githubPRsResponse
		if err := json.Unmarshal(prBody, &prList); err != nil {
			return nil, fmt.Errorf("error parsing PRs: %w", err)
		}
		
		for _, pr := range prList {
			if pr.CreatedAt.Before(since) {
				break
			}
			
			// Get reviews for this PR
			reviewsURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/reviews",
				c.getBaseURL(), c.config.GitHubOwner, c.config.GitHubRepo, pr.Number)
			
			reviewBody, _ := c.makeRequest(reviewsURL)
			var reviews []githubReviewsResponse
			json.Unmarshal(reviewBody, &reviews)
			
			var firstReviewAt *time.Time
			for _, review := range reviews {
				if (review.State == "APPROVED" || review.State == "CHANGES_REQUESTED") && firstReviewAt == nil {
					firstReviewAt = &review.SubmittedAt
					break
				}
			}
			
			// Calculate status
			status := "OPEN"
			if pr.MergedAt != nil {
				status = "MERGED"
			} else if pr.State == "closed" {
				status = "CLOSED"
			}
			
			if pr.ChangedFiles > 0 {
				prs = append(prs, PullRequest{
					ID:           fmt.Sprintf("PR-%d", pr.Number),
					Author:       pr.User.Login,
					CreatedAt:    pr.CreatedAt,
					MergedAt:     pr.MergedAt,
					ClosedAt:     pr.ClosedAt,
					FirstReviewAt: firstReviewAt,
					LinesChanged:  pr.Additions + pr.Deletions,
					Status:       status,
					Reviewers:    c.extractReviewers(reviews),
				})
			}
		}
		
		if len(prList) < 100 {
			break
		}
		page++
	}
	
	return prs, nil
}

// getBaseURL returns the GitHub API base URL
func (c Client) getBaseURL() string {
	if c.config.GitHubURL == "" || c.config.GitHubURL == "https://github.com" {
		return "https://api.github.com"
	}
	return c.config.GitHubURL + "/api/v3"
}

// extractReviewers extracts unique reviewer logins
func (c Client) extractReviewers(reviews []githubReviewsResponse) []string {
	seen := make(map[string]bool)
	var reviewers []string
	
	for _, review := range reviews {
		if review.User.Login != "" && !seen[review.User.Login] {
			seen[review.User.Login] = true
			reviewers = append(reviewers, review.User.Login)
		}
	}
	
	return reviewers
}