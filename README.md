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
```bash
go run main.go
```

## 🔑 Getting API Tokens

**Bitbucket Data Center:**
- Go to Profile → Manage account → Personal access tokens
- Create token with `REPO_READ` permission

**Jira Cloud:**
- Go to https://id.atlassian.com/manage-profile/security/api-tokens
- Create API token

**Jira Data Center:**
- Use your regular password or configure personal access tokens

## ✅ Features

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
export BITBUCKET_URL="https://bitbucket.company.com"
export BITBUCKET_TOKEN="your-token"
# ... etc
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
│   └── main.go
├── config/
│   └── config.go
├── api/
│   └── client.go
├── bitbucket/
│   ├── types.go
│   └── client.go
├── jira/
│   ├── types.go
│   └── client.go
├── metrics/
│   └── calculator.go
├── report/
│   └── reporter.go
├── go.mod
├── config.sample.json
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