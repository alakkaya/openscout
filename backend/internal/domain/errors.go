package domain

import "errors"

var (
    ErrUserNotFound              = errors.New("user not found")
    ErrIssueNotFound             = errors.New("issue not found")
    ErrUserPreferenceNotFound    = errors.New("user preference not found")
    ErrInvalidEmail              = errors.New("invalid email")
    ErrInvalidChannel            = errors.New("invalid notification channel")
    ErrNotificationAlreadySent   = errors.New("notification already sent")
    ErrInvalidAnalysisResult     = errors.New("invalid analysis result")
)