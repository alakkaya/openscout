package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/alakkaya/openscout/internal/domain"
	"github.com/alakkaya/openscout/internal/port"
)

type AnalyzeIssuesUseCase struct {
    Analyzer port.AnalyzerClient
}

func NewAnalyzeIssuesUseCase(a port.AnalyzerClient) *AnalyzeIssuesUseCase {
    return &AnalyzeIssuesUseCase{Analyzer: a}
}

// Execute calls the analyzer and returns a map(issueID -> analysis).
func (uc *AnalyzeIssuesUseCase) Execute(ctx context.Context, issues []domain.Issue) (map[string]domain.IssueAnalysis, error) {
    if uc.Analyzer == nil {
        return nil, fmt.Errorf("analyzer client is nil")
    }
    analyses, err := uc.Analyzer.Analyze(ctx, issues)
    if err != nil {
        return nil, err
    }

    now := time.Now().UTC()
    out := make(map[string]domain.IssueAnalysis, len(analyses))
    for _, a := range analyses {
        // basic validation
        if a.IssueID == "" || a.Complexity < 1 || a.Complexity > 5 {
            return nil, fmt.Errorf("invalid analysis for issue %q", a.IssueID)
        }
        // set analyzed time if not set
        if a.AnalyzedAt.IsZero() {
            a.AnalyzedAt = now
        }
        out[a.IssueID] = a
    }
    return out, nil
}