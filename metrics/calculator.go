package metrics

import (
	"fmt"
	"strings"
	"time"
	"devops-metrics/bitbucket"
	"devops-metrics/jira"
)

// Metric structures
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

// CalculateCommitMetrics computes metrics from commits
func CalculateCommitMetrics(commits []bitbucket.Commit) CommitMetrics {
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

// CalculatePRMetrics computes metrics from pull requests
func CalculatePRMetrics(prs []bitbucket.PullRequest) PRMetrics {
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
		case "MERGED":
			metrics.MergedPRs++
		case "DECLINED", "CLOSED":
			metrics.ClosedPRs++
		case "OPEN":
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

// CalculateJiraMetrics computes metrics from Jira stories
func CalculateJiraMetrics(stories []jira.JiraStory) JiraMetrics {
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

// CalculateTeamMetrics combines all metrics
func CalculateTeamMetrics(commits []bitbucket.Commit, prs []bitbucket.PullRequest, stories []jira.JiraStory) TeamMetrics {
	return TeamMetrics{
		CommitMetrics: CalculateCommitMetrics(commits),
		PRMetrics:     CalculatePRMetrics(prs),
		JiraMetrics:   CalculateJiraMetrics(stories),
		GeneratedAt:   time.Now(),
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}