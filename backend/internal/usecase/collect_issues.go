package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/alakkaya/openscout/internal/domain"
	"github.com/alakkaya/openscout/internal/port"
)

type CollectIssuesUseCase struct {
    Github port.GitHubClient
    // concurrency/worker limits can be handled inside the GitHub adapter
    SeenWindow time.Duration // dedupe window if you want
}

func NewCollectIssuesUseCase(g port.GitHubClient) *CollectIssuesUseCase {
    return &CollectIssuesUseCase{Github: g, SeenWindow: 24 * time.Hour}
}

// Execute fetches issues for the configured languages/labels via the GitHub client.
// It applies repo/issue quality filters inside the GitHub client or here if needed.
// Returns deduped issues ready for analysis.
func (uc *CollectIssuesUseCase) Execute(ctx context.Context, languages, labels []string) ([]domain.Issue, error) {
    if uc.Github == nil {
        return nil, errors.New("github client is nil")
    }
    issues, err := uc.Github.FetchIssues(ctx, languages, labels)
    if err != nil {
        return nil, err
    }

    // simple dedupe by id in-memory
    seen := make(map[string]struct{}, len(issues))
    out := make([]domain.Issue, 0, len(issues))
    for _, iss := range issues {
        if iss.ID == "" {
            continue
        }
        if _, ok := seen[iss.ID]; ok {
            continue
        }
        seen[iss.ID] = struct{}{}
        out = append(out, iss)
    }

    return out, nil
}