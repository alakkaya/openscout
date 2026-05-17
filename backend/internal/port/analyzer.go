package port

import (
	"context"

	"github.com/alakkaya/openscout/internal/domain"
)

type AnalyzerClient interface {
	// Output: complexity, estimated hours, skills needed, why solvable, warning (if any) for each issue
    Analyze(ctx context.Context, issues []domain.Issue) ([]domain.IssueAnalysis, error)
}