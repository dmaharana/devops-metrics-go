package web

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"devops-metrics/bitbucket"
	"devops-metrics/config"
	"devops-metrics/github"
	"devops-metrics/jira"
	"devops-metrics/metrics"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server handles HTTP requests
type Server struct {
	Router *chi.Mux
	config config.Config
}

// NewServer creates a new web server
func NewServer() *Server {
	s := &Server{}

	// Load configuration
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Printf("Warning: Could not load config.json, trying environment variables: %v", err)
	}
	s.config = cfg

	// Validate configuration
	if cfg.BitbucketURL == "" || cfg.JiraURL == "" {
		log.Fatal("❌ Configuration Error! Please set BITBUCKET_* and JIRA_* environment variables or create config.json")
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	r := chi.NewRouter()

	// Request logging middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(2 * time.Minute)) // 2 minute timeout for API requests

	// Health check endpoint
	r.Get("/health", s.healthCheck)

	// API endpoints
	r.Route("/api", func(r chi.Router) {
		r.Get("/bitbucket/metrics", s.getBitbucketMetrics)
		r.Get("/github/metrics", s.getGitHubMetrics)
		r.Get("/jira/metrics", s.getJiraMetrics)
		r.Get("/metrics", s.getAllMetrics)
	})

	s.Router = r
}

// healthCheck returns server health status
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "devops-metrics-api",
	})
}

// getBitbucketMetrics calculates and returns Bitbucket metrics
func (s *Server) getBitbucketMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	bbClient := bitbucket.NewClient(s.config)

	// Fetch Bitbucket data
	commits, err := bbClient.FetchCommits()
	if err != nil {
		log.Printf("❌ Error fetching commits: %v", err)
		http.Error(w, "Error fetching commits", http.StatusInternalServerError)
		return
	}

	prs, err := bbClient.FetchPRs()
	if err != nil {
		log.Printf("❌ Error fetching PRs: %v", err)
		http.Error(w, "Error fetching PRs", http.StatusInternalServerError)
		return
	}

	// Calculate Bitbucket metrics
	commitMetrics := metrics.CalculateCommitMetrics(commits)
	prMetrics := metrics.CalculatePRMetrics(prs)

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"commit_metrics": commitMetrics,
			"pr_metrics":     prMetrics,
		},
		"stats": map[string]int{
			"commits": len(commits),
			"prs":     len(prs),
		},
		"timestamp": time.Now().UTC(),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getGitHubMetrics calculates and returns GitHub metrics
func (s *Server) getGitHubMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ghClient := github.NewClient(s.config)

	// Fetch GitHub data
	commits, err := ghClient.FetchCommits()
	if err != nil {
		log.Printf("❌ Error fetching GitHub commits: %v", err)
		http.Error(w, "Error fetching GitHub commits", http.StatusInternalServerError)
		return
	}

	prs, err := ghClient.FetchPRs()
	if err != nil {
		log.Printf("❌ Error fetching GitHub PRs: %v", err)
		http.Error(w, "Error fetching GitHub PRs", http.StatusInternalServerError)
		return
	}

	// Convert to Bitbucket format for metrics calculation
	bbCommits := make([]bitbucket.Commit, len(commits))
	for i, c := range commits {
		bbCommits[i] = bitbucket.Commit{
			Hash:         c.Hash,
			Author:       c.Author,
			Date:         c.Date,
			Message:      c.Message,
			LinesAdded:   c.LinesAdded,
			LinesDeleted: c.LinesDeleted,
		}
	}

	bbPRs := make([]bitbucket.PullRequest, len(prs))
	for i, p := range prs {
		bbPRs[i] = bitbucket.PullRequest{
			ID:            p.ID,
			Author:        p.Author,
			CreatedAt:     p.CreatedAt,
			MergedAt:      p.MergedAt,
			ClosedAt:      p.ClosedAt,
			FirstReviewAt: p.FirstReviewAt,
			LinesChanged:  p.LinesChanged,
			Reviewers:     p.Reviewers,
			Status:        p.Status,
		}
	}

	// Calculate GitHub metrics
	commitMetrics := metrics.CalculateCommitMetrics(bbCommits)
	prMetrics := metrics.CalculatePRMetrics(bbPRs)

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"commit_metrics": commitMetrics,
			"pr_metrics":     prMetrics,
		},
		"stats": map[string]int{
			"commits": len(commits),
			"prs":     len(prs),
		},
		"timestamp": time.Now().UTC(),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getJiraMetrics calculates and returns Jira metrics
func (s *Server) getJiraMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	jClient := jira.NewClient(s.config)

	// Fetch Jira data
	stories, err := jClient.FetchIssues()
	if err != nil {
		log.Printf("❌ Error fetching Jira issues: %v", err)
		http.Error(w, "Error fetching Jira issues", http.StatusInternalServerError)
		return
	}

	// Calculate Jira metrics
	jiraMetrics := metrics.CalculateJiraMetrics(stories)

	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"jira_metrics": jiraMetrics,
		},
		"stats": map[string]int{
			"stories": len(stories),
		},
		"timestamp": time.Now().UTC(),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getAllMetrics calculates and returns all metrics
func (s *Server) getAllMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var commits []bitbucket.Commit
	var prs []bitbucket.PullRequest
	var stories []jira.JiraStory

	// Fetch Bitbucket data
	if s.config.BitbucketURL != "" {
		bbClient := bitbucket.NewClient(s.config)
		bbCommits, err := bbClient.FetchCommits()
		if err != nil {
			log.Printf("❌ Error fetching Bitbucket commits: %v", err)
		} else {
			commits = append(commits, bbCommits...)
		}

		bbPRs, err := bbClient.FetchPRs()
		if err != nil {
			log.Printf("❌ Error fetching Bitbucket PRs: %v", err)
		} else {
			prs = append(prs, bbPRs...)
		}
	}

	// Fetch GitHub data
	if s.config.GitHubURL != "" {
		ghClient := github.NewClient(s.config)
		ghCommits, err := ghClient.FetchCommits()
		if err != nil {
			log.Printf("❌ Error fetching GitHub commits: %v", err)
		} else {
			// Convert GitHub commits to Bitbucket format
			for _, c := range ghCommits {
				commits = append(commits, bitbucket.Commit{
					Hash:         c.Hash,
					Author:       c.Author,
					Date:         c.Date,
					Message:      c.Message,
					LinesAdded:   c.LinesAdded,
					LinesDeleted: c.LinesDeleted,
				})
			}
		}

		ghPRs, err := ghClient.FetchPRs()
		if err != nil {
			log.Printf("❌ Error fetching GitHub PRs: %v", err)
		} else {
			// Convert GitHub PRs to Bitbucket format
			for _, p := range ghPRs {
				prs = append(prs, bitbucket.PullRequest{
					ID:            p.ID,
					Author:        p.Author,
					CreatedAt:     p.CreatedAt,
					MergedAt:      p.MergedAt,
					ClosedAt:      p.ClosedAt,
					FirstReviewAt: p.FirstReviewAt,
					LinesChanged:  p.LinesChanged,
					Reviewers:     p.Reviewers,
					Status:        p.Status,
				})
			}
		}
	}

	// Fetch Jira data
	if s.config.JiraURL != "" {
		jClient := jira.NewClient(s.config)
		var err error
		stories, err = jClient.FetchIssues()
		if err != nil {
			log.Printf("❌ Error fetching Jira issues: %v", err)
			stories = []jira.JiraStory{}
		}
	}

	// Calculate all metrics
	teamMetrics := metrics.CalculateTeamMetrics(commits, prs, stories)

	// Generate reports
	jsonData, err := json.Marshal(teamMetrics)
	if err != nil {
		http.Error(w, "Error generating JSON", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status": "success",
		"data":   teamMetrics,
		"stats": map[string]int{
			"commits": len(commits),
			"prs":     len(prs),
			"stories": len(stories),
		},
		"timestamp": time.Now().UTC(),
		"export": map[string]string{
			"json": string(jsonData),
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Start starts the web server
func (s *Server) Start(port string) {
	log.Printf("🚀 Starting DevOps Metrics API Server on port %s", port)
	log.Printf("📊 Available endpoints:")
	log.Printf("   GET /health - Health check")
	log.Printf("   GET /api/bitbucket/metrics - Bitbucket metrics")
	log.Printf("   GET /api/jira/metrics - Jira metrics")
	log.Printf("   GET /api/metrics - All metrics")
	log.Printf("   GET /api/metrics/csv - Download CSV report")

	if err := http.ListenAndServe(":"+port, s.Router); err != nil {
		log.Fatal("❌ Failed to start server:", err)
	}
}