package port

import (
	"context"

	"github.com/alakkaya/openscout/internal/domain"
)

type GitHubClient interface {
    // FetchIssues fetches issues according to language/label filters and applies quality filters.
    FetchIssues(ctx context.Context, languages, labels []string) ([]domain.Issue, error)
}