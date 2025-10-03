package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"devops-metrics/metrics"
)

// ExportToJSON saves metrics to a JSON file
func ExportToJSON(metrics metrics.TeamMetrics, filename string) error {
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// ExportToCSV saves metrics to a CSV file
func ExportToCSV(metrics metrics.TeamMetrics, filename string) error {
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

// PrintMetricsSummary displays a formatted summary to the console
func PrintMetricsSummary(metrics metrics.TeamMetrics) {
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