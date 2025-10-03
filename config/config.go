package config

import (
	"encoding/json"
	"os"
	"strconv"
)

// Config represents the application configuration
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

// LoadConfig loads configuration from file or environment variables
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

// CreateSampleConfig creates a sample configuration file
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