package github

import "time"

// types.go - Data structures for GitHub integration

// Commit represents a git commit
type Commit struct {
	Hash         string    `json:"hash"`
	Author       string    `json:"author"`
	Date         time.Time `json:"date"`
	Message      string    `json:"message"`
	LinesAdded   int       `json:"lines_added"`
	LinesDeleted int       `json:"lines_deleted"`
}

// PullRequest represents a pull request
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