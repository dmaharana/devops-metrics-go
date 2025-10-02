## 🚀 Getting Started

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

## 📊 Features

✅ **Bitbucket Integration:**
- Fetches commits with author and timestamps
- Retrieves all PRs with status, reviewers, and diff stats
- Calculates line changes from PR diffs

✅ **Jira Integration:**
- Fetches issues with full changelog
- Extracts story points or time estimates
- Tracks status transitions (In Progress → Done)
- Works with both Jira Cloud and Data Center

✅ **Environment Variables Support:**
```bash
export BITBUCKET_URL="https://bitbucket.company.com"
export BITBUCKET_TOKEN="your-token"
# ... etc
go run main.go
```

## 📈 Output

The tool generates:
- **Console Report**: Beautiful formatted summary
- **metrics.json**: Full detailed metrics
- **metrics.csv**: Import into Excel/Google Sheets

Would you like me to add features like:
- Scheduled runs with cron
- Slack/email notifications
- Trend analysis over time
- Custom date ranges
- Additional metrics (DORA metrics, deployment frequency)?
