package main

import (
	"flag"
	"fmt"
	"log"
	"devops-metrics/bitbucket"
	"devops-metrics/config"
	"devops-metrics/jira"
	"devops-metrics/metrics"
	"devops-metrics/report"
	"devops-metrics/web"
)

func main() {
	fmt.Println("DevOps & Productivity Metrics Generator with API Integration")
	fmt.Println("============================================================\n")

	// Parse command line flags
	var sampleConfig bool
	var runServer bool
	var port string
	flag.BoolVar(&sampleConfig, "sample-config", false, "Generate sample configuration file")
	flag.BoolVar(&runServer, "server", false, "Run as web server")
	flag.StringVar(&port, "port", "8080", "Port to run the server on (when using -server)")
	flag.Parse()

	if sampleConfig {
		if err := config.CreateSampleConfig(); err != nil {
			log.Fatalf("Error creating sample config: %v", err)
		}
		fmt.Println("✅ Sample configuration file created: config.sample.json")
		fmt.Println("\nEdit this file with your credentials and rename to config.json")
		return
	}

	if runServer {
		// Start web server
		server := web.NewServer()
		server.Start(port)
		return
	}

	// Original CLI mode
	// Load configuration
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Printf("Warning: Could not load config.json, trying environment variables: %v\n", err)
	}

	// Validate configuration
	if cfg.BitbucketURL == "" || cfg.JiraURL == "" {
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

	bbClient := bitbucket.NewClient(cfg)
	jClient := jira.NewClient(cfg)

	fmt.Printf("Analyzing data from the last %d days...\n\n", cfg.DaysToAnalyze)

	// Fetch Bitbucket data
	fmt.Println("🔄 Fetching Bitbucket commits...")
	commits, err := bbClient.FetchCommits()
	if err != nil {
		log.Printf("❌ Error fetching commits: %v", err)
		commits = []bitbucket.Commit{}
	} else {
		fmt.Printf("✅ Fetched %d commits\n", len(commits))
	}

	fmt.Println("🔄 Fetching Bitbucket pull requests...")
	prs, err := bbClient.FetchPRs()
	if err != nil {
		log.Printf("❌ Error fetching PRs: %v", err)
		prs = []bitbucket.PullRequest{}
	} else {
		fmt.Printf("✅ Fetched %d pull requests\n", len(prs))
	}

	// Fetch Jira data
	fmt.Println("🔄 Fetching Jira issues...")
	stories, err := jClient.FetchIssues()
	if err != nil {
		log.Printf("❌ Error fetching Jira issues: %v", err)
		stories = []jira.JiraStory{}
	} else {
		fmt.Printf("✅ Fetched %d Jira stories\n", len(stories))
	}

	// Calculate metrics
	fmt.Println("\n📊 Calculating metrics...")
	teamMetrics := metrics.CalculateTeamMetrics(commits, prs, stories)

	// Print summary
	report.PrintMetricsSummary(teamMetrics)

	// Export to files
	if err := report.ExportToJSON(teamMetrics, "metrics.json"); err != nil {
		log.Printf("Error exporting to JSON: %v", err)
	} else {
		fmt.Println("\n✅ Metrics exported to: metrics.json")
	}

	if err := report.ExportToCSV(teamMetrics, "metrics.csv"); err != nil {
		log.Printf("Error exporting to CSV: %v", err)
	} else {
		fmt.Println("✅ Metrics exported to: metrics.csv")
	}

	fmt.Println("\n🎉 Analysis complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Review metrics.json for detailed analysis")
	fmt.Println("- Import metrics.csv into spreadsheet for visualization")
	fmt.Println("- Schedule this script to run periodically for tracking trends")
	fmt.Println("- Run with --server to start the web API")
}