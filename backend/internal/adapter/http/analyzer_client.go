package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alakkaya/openscout/internal/domain"
)

// HTTP client for Python analyzer service
type AnalyzerHTTPClient struct {
    endpoint   string        // Python analyzer URL (e.g., http://localhost:8000/analyze)
    httpClient *http.Client
    timeout    time.Duration
}

// creates a new analyzer client
func NewAnalyzerHTTPClient(endpoint string, timeout time.Duration) *AnalyzerHTTPClient {
    return &AnalyzerHTTPClient{
        endpoint: endpoint,
        httpClient: &http.Client{
            Timeout: timeout,
        },
        timeout: timeout,
    }
}

// HTTP request payload (Go → Python)
type analyzeRequest struct {
    Issues []issuePayload `json:"issues"`
}

// issue payload format for Python
type issuePayload struct {
    ID           string   `json:"id"`
    Title        string   `json:"title"`
    URL          string   `json:"url"`
    Body         string   `json:"body"`
    CreatedAt    string   `json:"created_at"`
    CommentCount int      `json:"comment_count"`
    Labels       []string `json:"labels"`
    Repository   repoPayload `json:"repository"`
}

// repository information
type repoPayload struct {
    Name             string `json:"name"`
    Description      string `json:"description"`
    Stars            int    `json:"stars"`
    LicenseName      string `json:"license_name"`
    ContributorCount int    `json:"contributor_count"`
    HasReadme        bool   `json:"has_readme"`
    LastCommitAt     *string `json:"last_commit_at,omitempty"` // ISO 8601 string
}

// HTTP response payload (Python → Go)
type analyzeResponse struct {
    Analyses []analysisPayload `json:"analyses"`
    Error    *string           `json:"error,omitempty"`
}

// analysis result
type analysisPayload struct {
    IssueID        string   `json:"issue_id"`
    Complexity     int      `json:"complexity"`
    EstimatedHours int      `json:"estimated_hours"`
    SkillsNeeded   []string `json:"skills_needed"`
    WhySolvable    string   `json:"why_solvable"`
    Warning        *string  `json:"warning,omitempty"`
}

// sends issues to Python analyzer and returns analysis results
func (c *AnalyzerHTTPClient) Analyze(ctx context.Context, issues []domain.Issue) ([]domain.IssueAnalysis, error) {
    payload := analyzeRequest{
        Issues: make([]issuePayload, len(issues)),
    }

    for i, issue := range issues {
        lastCommitStr := ""
        if issue.Repository.LastCommitAt != nil {
            lastCommitStr = issue.Repository.LastCommitAt.Format(time.RFC3339)
        }
        var labels []string
        if issue.Labels != "" {
            _ = json.Unmarshal([]byte(issue.Labels), &labels)
        }

        payload.Issues[i] = issuePayload{
            ID:           issue.ID,
            Title:        issue.Title,
            URL:          issue.URL,
            Body:         issue.Body,
            CreatedAt:    issue.CreatedAt.Format(time.RFC3339),
            CommentCount: issue.CommentCount,
            Labels:       labels,
            Repository: repoPayload{
                Name:             issue.Repository.Name,
                Description:      issue.Repository.Description,
                Stars:            issue.Repository.Stars,
                LicenseName:      issue.Repository.LicenseName,
                ContributorCount: issue.Repository.ContributorCount,
                HasReadme:        issue.Repository.HasReadme,
                LastCommitAt:     &lastCommitStr,
            },
        }
    }

    body, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("analyzer request failed: %w", err)
    }
    defer resp.Body.Close()

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("analyzer returned status %d: %s", resp.StatusCode, string(respBody))
    }

    var analyzeResp analyzeResponse
    if err := json.Unmarshal(respBody, &analyzeResp); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    if analyzeResp.Error != nil {
        return nil, fmt.Errorf("analyzer error: %s", *analyzeResp.Error)
    }

    results := make([]domain.IssueAnalysis, len(analyzeResp.Analyses))
    for i, analysis := range analyzeResp.Analyses {
        skills, err := json.Marshal(analysis.SkillsNeeded)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal analysis skills: %w", err)
        }
        results[i] = domain.IssueAnalysis{
            IssueID:        analysis.IssueID,
            Complexity:     analysis.Complexity,
            EstimatedHours: analysis.EstimatedHours,
            SkillsNeeded:   string(skills),
            WhySolvable:    analysis.WhySolvable,
            Warning:        analysis.Warning,
            AnalyzedAt:     time.Now(),
        }
    }

    return results, nil
}