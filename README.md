# DevOps & Productivity Metrics Generator

A Go application that collects and analyzes development metrics from Bitbucket and Jira to help teams measure productivity and identify process improvements.

## 🚀 Quick Start

### 1. **Generate Sample Configuration**
```bash
go run main.go --sample-config
```
This creates `config.sample.json` with all the required fields.

### 2. **Configure Your Credentials**

Edit `config.json` with your details:

**For Bitbucket Data Center:**
```json
{
  "bitbucket_url": "https://bitbucket.company.com",
  "bitbucket_token": "your-personal-access-token",
  "bitbucket_project": "PROJ",
  "bitbucket_repo": "my-repo"
}
```

**For GitHub (new!):**
```json
{
  "github_url": "https://github.com",
  "github_token": "your-personal-access-token", 
  "github_owner": "company",
  "github_repo": "my-repo"
}
```

**For Jira Data Center:**
```json
{
  "jira_url": "https://jira.company.com",
  "jira_username": "your-username",
  "jira_token": "your-password",
  "jira_project": "PROJ",
  "is_jira_cloud": false
}
```

**For Jira Cloud:**
```json
{
  "jira_url": "https://yoursite.atlassian.net",
  "jira_username": "your-email@company.com",
  "jira_token": "your-api-token",
  "jira_project": "PROJ",
  "is_jira_cloud": true
}
```

### 3. **Run the Analysis**

**CLI Mode (traditional):**
```bash
go run main.go
```

**Web Server Mode (new!):**
```bash
# Start web API server
go run main.go --server

# Or specify port
go run main.go --server --port 8080
```

**Web API Endpoints:**
- `GET /health` - Health check
- `GET /api/bitbucket/metrics` - Bitbucket metrics
- `GET /api/github/metrics` - GitHub metrics (new!)
- `GET /api/jira/metrics` - Jira metrics
- `GET /api/metrics` - All metrics combined

## 🔑 Getting API Tokens

**Bitbucket Data Center:**
- Go to Profile → Manage account → Personal access tokens
- Create token with `REPO_READ` permission

**GitHub:**
- Go to Settings → Developer settings → Personal access tokens
- Create token with `repo` scope for private repos, `public_repo` for public repos

**Jira Cloud:**
- Go to https://id.atlassian.com/manage-profile/security/api-tokens
- Create API token

**Jira Data Center:**
- Use your regular password or configure personal access tokens

## ✅ Features

**GitHub Integration (new!):**
- Fetches commits from all branches with author info
- Retrieves pull requests with reviews and timeline
- Supports GitHub's GraphQL API for efficient data fetching
- Rate limiting with exponential backoff

**Bitbucket Integration:**
- Fetches commits with author and timestamps
- Retrieves all PRs with status, reviewers, and diff stats
- Calculates line changes from PR diffs

**Jira Integration:**
- Fetches issues with full changelog
- Extracts story points or time estimates
- Tracks status transitions (In Progress → Done)
- Works with both Jira Cloud and Data Center

**Environment Variables Support:**
```bash
# GitHub (new!)
export GITHUB_URL="https://github.com"          # Optional for GitHub.com
export GITHUB_TOKEN="your-token"
export GITHUB_OWNER="company"
export GITHUB_REPO="repo-name"

# Bitbucket  
export BITBUCKET_URL="https://bitbucket.company.com"
export BITBUCKET_TOKEN="your-token"
export BITBUCKET_PROJECT="PROJECT"
export BITBUCKET_REPO="repo-slug"

# Jira
export JIRA_URL="https://yoursite.atlassian.net"
export JIRA_USERNAME="your-email@company.com"
export JIRA_TOKEN="api-token"
export JIRA_PROJECT="PROJ"
export JIRA_IS_CLOUD="true"

# Optional
export DAYS_TO_ANALYZE=30
go run main.go
```

## 📊 Output

The tool generates:
- **Console Report**: Beautiful formatted summary
- **metrics.json**: Full detailed metrics
- **metrics.csv**: Import into Excel/Google Sheets

## 🏗️ Project Structure

```
devops-metrics-go/
├── cmd/
│   ├── main.go
│   └── web/
│       └── main.go
├── config/
│   └── config.go
├── api/
│   └── client.go
├── bitbucket/
│   ├── types.go
│   └── client.go
├── github/          # New!
│   ├── types.go
│   └── client.go
├── jira/
│   ├── types.go
│   └── client.go
├── metrics/
│   └── calculator.go
├── report/
│   └── reporter.go
├── web/            # New!
│   └── server.go
├── go.mod
├── config.sample.json
├── WEB_API.md      # API documentation
└── README.md
```

## 📈 Key Metrics

### Top 3 Development Metrics

1. **Lead Time (Idea to Done)** - Directly measures speed of value delivery
2. **Cycle Time (Development Start to Done)** - Measures actual development efficiency  
3. **Deployment Frequency (PRs Merged/Released)** - Key DORA metric for team maturity

### Additional Metrics

**From Commits:**
- Commit Frequency, Code Churn, Active Days, Bus Factor

**From Pull Requests:**  
- PR Cycle Time, Review Time, Merge Success Rate, Review Load

**From Jira Stories:**
- Throughput, Velocity, Work In Progress, Effort Accuracy

## 💡 Pro Tip: The Golden Ratio

**Lead Time ÷ Cycle Time = Flow Efficiency**

If Lead Time is 20 days but Cycle Time is 5 days, your work is sitting idle 75% of the time! This reveals waste in your process.

**Elite teams:** Flow Efficiency > 40%
**Average teams:** Flow Efficiency < 15%