package sqlite

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/alakkaya/openscout/internal/domain"
	"github.com/alakkaya/openscout/internal/repository"
)

type SqliteUserRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) repository.UserRepository {
    return &SqliteUserRepository{db: db}
}

func (r *SqliteUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
    return r.db.WithContext(ctx).Create(user).Error
}

func (r *SqliteUserRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
    var user domain.User
    if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, domain.ErrUserNotFound
        }
        return nil, err
    }
    return &user, nil
}

func (r *SqliteUserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
    var user domain.User
    if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, domain.ErrUserNotFound
        }
        return nil, err
    }
    return &user, nil
}

func (r *SqliteUserRepository) GetAllActiveUsers(ctx context.Context) ([]*domain.User, error) {
    var users []*domain.User
    if err := r.db.WithContext(ctx).Where("active = ?", true).Find(&users).Error; err != nil {
        return nil, err
    }
    return users, nil
}

func (r *SqliteUserRepository) UpdateUser(ctx context.Context, user *domain.User) error {
    user.UpdatedAt = time.Now().UTC()
    return r.db.WithContext(ctx).Model(user).Updates(user).Error
}

func (r *SqliteUserRepository) DeactivateUser(ctx context.Context, userID string) error {
    return r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Update("active", false).Error
}

/* Preferences repo */

type SqliteUserPreferenceRepository struct {
    db *gorm.DB
}

func NewUserPreferenceRepository(db *gorm.DB) repository.UserPreferenceRepository {
    return &SqliteUserPreferenceRepository{db: db}
}

func (r *SqliteUserPreferenceRepository) CreateOrUpdatePreference(ctx context.Context, pref *domain.UserPreference) error {
    pref.UpdatedAt = time.Now().UTC()
    return r.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(pref).Error
}

func (r *SqliteUserPreferenceRepository) GetPreferenceByUserID(ctx context.Context, userID string) (*domain.UserPreference, error) {
    var pref domain.UserPreference
    if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&pref).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, domain.ErrUserPreferenceNotFound
        }
        return nil, err
    }
    return &pref, nil
}

/* Feedback repo */

type SqliteFeedbackRepository struct {
    db *gorm.DB
}

func NewFeedbackRepository(db *gorm.DB) repository.FeedbackRepository {
    return &SqliteFeedbackRepository{db: db}
}

func (r *SqliteFeedbackRepository) SaveFeedback(ctx context.Context, userID, issueID, feedback string) error {
    fb := &domain.UserIssueFeedback{
        ID:        uuid.NewString(),
        UserID:    userID,
        IssueID:   issueID,
        Status:    feedback,
        CreatedAt: time.Now().UTC(),
    }
    return r.db.WithContext(ctx).Create(fb).Error
}

func (r *SqliteFeedbackRepository) HasUserRespondedToIssue(ctx context.Context, userID, issueID string) (bool, error) {
    var count int64
    if err := r.db.WithContext(ctx).Model(&domain.UserIssueFeedback{}).
        Where("user_id = ? AND issue_id = ?", userID, issueID).
        Count(&count).Error; err != nil {
        return false, err
    }
    return count > 0, nil
}

func (r *SqliteFeedbackRepository) GetUserFeedback(ctx context.Context, userID, issueID string) (*domain.UserIssueFeedback, error) {
    var fb domain.UserIssueFeedback
    if err := r.db.WithContext(ctx).Where("user_id = ? AND issue_id = ?", userID, issueID).First(&fb).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, nil
        }
        return nil, err
    }
    return &fb, nil
}

/* Notification repo */

type SqliteNotificationRepository struct {
    db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) repository.NotificationRepository {
    return &SqliteNotificationRepository{db: db}
}

func (r *SqliteNotificationRepository) SaveSentNotification(ctx context.Context, notif *domain.SentNotification) error {
    return r.db.WithContext(ctx).Create(notif).Error
}

func (r *SqliteNotificationRepository) GetSentNotificationHistory(ctx context.Context, userID string, limit int) ([]*domain.SentNotification, error) {
    var notifs []*domain.SentNotification
    if err := r.db.WithContext(ctx).Where("user_id = ?", userID).
        Order("sent_at DESC").
        Limit(limit).
        Find(&notifs).Error; err != nil {
        return nil, err
    }
    return notifs, nil
}

func (r *SqliteNotificationRepository) HasBeenSentToday(ctx context.Context, userID, issueID, channel string) (bool, error) {
    now := time.Now().UTC()
    startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
    var count int64
    if err := r.db.WithContext(ctx).Model(&domain.SentNotification{}).
        Where("user_id = ? AND issue_id = ? AND channel = ? AND sent_at >= ?", userID, issueID, channel, startOfDay).
        Count(&count).Error; err != nil {
        return false, err
    }
    return count > 0, nil
}