# DevOps Metrics API

A web server that provides REST API endpoints for Bitbucket and Jira metrics using the chi router.

## Endpoints

### Health Check
- `GET /health` - Server health status

### Bitbucket Metrics
- `GET /api/bitbucket/metrics` - Returns commit and PR metrics
  - **Response**: 
    ```json
    {
      "status": "success",
      "data": {
        "commit_metrics": { ... },
        "pr_metrics": { ... }
      },
      "stats": {
        "commits": 150,
        "prs": 25
      },
      "timestamp": "2024-01-15T10:30:00Z"
    }
    ```

### GitHub Metrics (new!)
- `GET /api/github/metrics` - Returns commit and PR metrics
  - **Response**: Same format as Bitbucket metrics

### Jira Metrics  
- `GET /api/jira/metrics` - Returns Jira story metrics
  - **Response**:
    ```json
    {
      "status": "success", 
      "data": {
        "jira_metrics": { ... }
      },
      "stats": {
        "stories": 45
      },
      "timestamp": "2024-01-15T10:30:00Z"
    }
    ```

### All Metrics
- `GET /api/metrics` - Returns all metrics combined from all sources
  - **Response**: Complete team metrics including all data

## Usage

### Start the Web Server

```bash
# Using the built-in server flag
./devops-metrics -server -port 8080

# Or using make
make web

# Or using go run
go run cmd/web/main.go
```

### Environment Variables

Set these environment variables or create a `config.json` file:

```bash
# Bitbucket
BITBUCKET_URL=https://bitbucket.example.com
BITBUCKET_TOKEN=your-token
BITBUCKET_PROJECT=PROJ
BITBUCKET_REPO=repo-name

# Jira  
JIRA_URL=https://your-domain.atlassian.net
JIRA_USERNAME=your-email@example.com
JIRA_TOKEN=api-token
JIRA_PROJECT=PROJ
JIRA_IS_CLOUD=true

# Optional
DAYS_TO_ANALYZE=30
```

### Example API Calls

```bash
# Health check
curl http://localhost:8080/health

# Bitbucket metrics
curl http://localhost:8080/api/bitbucket/metrics

# Jira metrics  
curl http://localhost:8080/api/jira/metrics

# All metrics
curl http://localhost:8080/api/metrics
```

## Features

- ✅ Multi-branch commit support (fetches from all branches)
- ✅ Rate limiting handling with exponential backoff
- ✅ CORS enabled for web applications
- ✅ Request logging and timeout handling
- ✅ Graceful error handling
- ✅ JSON responses with metadata
- ✅ Configurable time ranges
- ✅ Health check endpoint

## Configuration

Create a `config.json` file or use environment variables. See `config.sample.json` for the structure.

## Dependencies

- [github.com/go-chi/chi/v5](https://github.com/go-chi/chi) - HTTP router
- [github.com/go-chi/cors](https://github.com/go-chi/cors) - CORS middleware