package domain

import "time"

type NotificationChannel string

const (
    ChannelEmail    NotificationChannel = "email"
    ChannelTelegram NotificationChannel = "telegram"
)

type User struct {
    ID             string     `gorm:"primaryKey" json:"id"`
    Name           string     `json:"name"`
    Email          string     `gorm:"uniqueIndex" json:"email"`
    TelegramChatID *string    `json:"telegram_chat_id,omitempty"`
    Active         bool       `gorm:"default:true" json:"active"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
}

type UserPreference struct {
    UserID        string `gorm:"primaryKey" json:"user_id"`
    Languages     string `json:"languages"` // JSON array stored as string
    Labels        string `json:"labels"`    // JSON array stored as string
    MaxComplexity int    `gorm:"default:5" json:"max_complexity"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}

type Issue struct {
    ID           string     `json:"id"`
    Title        string     `json:"title"`
    URL          string     `json:"url"`
    Body         string     `json:"body"`
    CreatedAt    time.Time  `json:"created_at"`
    CommentCount int        `json:"comment_count"`
    Labels       string     `json:"labels"` // JSON array as string
    Repository   Repository `gorm:"embedded" json:"repository"`
}

type Repository struct {
    Name             string     `json:"name"`
    Description      string     `json:"description"`
    Stars            int        `json:"stars"`
    LicenseName      string     `json:"license_name"`
    ContributorCount int        `json:"contributor_count"`
    HasReadme        bool       `json:"has_readme"`
    LastCommitAt     *time.Time `json:"last_commit_at,omitempty"`
}

type IssueAnalysis struct {
    IssueID        string    `gorm:"primaryKey" json:"issue_id"`
    Complexity     int       `json:"complexity"`
    EstimatedHours int       `json:"estimated_hours"`
    SkillsNeeded   string    `json:"skills_needed"` // JSON array as string
    WhySolvable    string    `json:"why_solvable"`
    Warning        *string   `json:"warning,omitempty"`
    AnalyzedAt     time.Time `json:"analyzed_at"`
}

type UserIssueFeedback struct {
    ID        string    `gorm:"primaryKey" json:"id"`
    UserID    string    `json:"user_id"`
    IssueID   string    `json:"issue_id"`
    Status    string    `json:"status"` // "solved" | "not_interested"
    Note      *string   `json:"note,omitempty"`
    CreatedAt time.Time `json:"created_at"`
}

type SentNotification struct {
    ID      string              `gorm:"primaryKey" json:"id"`
    UserID  string              `json:"user_id"`
    IssueID string              `json:"issue_id"`
    Channel NotificationChannel `json:"channel"`
    SentAt  time.Time           `json:"sent_at"`
}