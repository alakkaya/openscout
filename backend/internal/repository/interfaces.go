package repository

import (
	"context"

	"github.com/alakkaya/openscout/internal/domain"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
    GetAllActiveUsers(ctx context.Context) ([]*domain.User, error)
	UpdateUser(ctx context.Context, user *domain.User) error
    DeactivateUser(ctx context.Context, userID string) error
}

type UserPreferenceRepository interface {
	CreateOrUpdatePreference(ctx context.Context, pref *domain.UserPreference) error
	GetPreferenceByUserID(ctx context.Context, userID string) (*domain.UserPreference, error)
}

type FeedbackRepository interface {
	SaveFeedback(ctx context.Context, userID, issueID, feedback string) error
    HasUserRespondedToIssue(ctx context.Context, userID, issueID string) (bool, error) // checks if user has already given feedback for the issue
    GetUserFeedback(ctx context.Context, userID, issueID string) (*domain.UserIssueFeedback, error)
}

type NotificationRepository interface {
    SaveSentNotification(ctx context.Context, notif *domain.SentNotification) error
    GetSentNotificationHistory(ctx context.Context, userID string, limit int) ([]*domain.SentNotification, error)
	HasBeenSentToday(ctx context.Context, userID, issueID, channel string) (bool, error)
}
