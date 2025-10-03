package jira

import "time"

// types.go - Data structures for Jira integration

// JiraStory represents a Jira story/issue
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