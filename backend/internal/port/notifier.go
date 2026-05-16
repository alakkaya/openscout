package port

import (
	"context"

	"github.com/alakkaya/openscout/internal/domain"
)

// Notifier sends notifications via a specific channel.
// Implementations should be safe to call concurrently.
type Notifier interface {
    // Notify sends a list of issues (with analyses) to a single user via channel.
    Notify(ctx context.Context, user *domain.User, channel domain.NotificationChannel, issues []domain.Issue, analyses map[string]domain.IssueAnalysis) error
}