package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/alakkaya/openscout/internal/domain"
	"github.com/alakkaya/openscout/internal/port"
	"github.com/alakkaya/openscout/internal/repository"
)

type SendNotificationsUseCase struct {
    UserRepo         repository.UserRepository
    PrefRepo         repository.UserPreferenceRepository
    FeedbackRepo     repository.FeedbackRepository
    NotificationRepo repository.NotificationRepository
    Notifier         port.Notifier
}

func NewSendNotificationsUseCase(
    u repository.UserRepository,
    p repository.UserPreferenceRepository,
    f repository.FeedbackRepository,
    n repository.NotificationRepository,
    not port.Notifier,
) *SendNotificationsUseCase {
    return &SendNotificationsUseCase{
        UserRepo: u, PrefRepo: p, FeedbackRepo: f, NotificationRepo: n, Notifier: not,
    }
}

func (uc *SendNotificationsUseCase) Execute(ctx context.Context, issues []domain.Issue, analyses map[string]domain.IssueAnalysis) error {
    if uc.UserRepo == nil || uc.PrefRepo == nil || uc.FeedbackRepo == nil || uc.NotificationRepo == nil || uc.Notifier == nil {
        return errors.New("dependencies not provided")
    }

    users, err := uc.UserRepo.GetAllActiveUsers(ctx)
    if err != nil {
        return err
    }

    for _, user := range users {
        // Only fall back to default preferences when the preference is not found.
        // Returning other repository errors surfaces real failures (DB/connection
        // issues) instead of silently using defaults which can mask problems.
        pref, err := uc.PrefRepo.GetPreferenceByUserID(ctx, user.ID)
        if err != nil {
            if errors.Is(err, domain.ErrUserPreferenceNotFound) {
                pref = &domain.UserPreference{
                    UserID:        user.ID,
                    Languages:     `["Go","Python","TypeScript"]`,
                    Labels:        `["good first issue","help wanted"]`,
                    MaxComplexity: 5,
                }
            } else {
                return err
            }
        }

        candidates := filterForUser(issues, analyses, pref)
        if len(candidates) == 0 {
            continue
        }

        toSend := make([]domain.Issue, 0, len(candidates))
        for _, iss := range candidates {
            responded, err := uc.FeedbackRepo.HasUserRespondedToIssue(ctx, user.ID, iss.ID)
            if err != nil {
                continue
            }
            if responded {
                continue
            }

            already, err := uc.NotificationRepo.HasBeenSentToday(ctx, user.ID, iss.ID, string(domain.ChannelEmail))
            if err != nil {
                continue
            }
            if already {
                continue
            }
            toSend = append(toSend, iss)
        }

        if len(toSend) == 0 {
            continue
        }

        err = uc.Notifier.Notify(ctx, user, domain.ChannelEmail, toSend, analyses)
        if err != nil {
            if user.TelegramChatID != nil {
                _ = uc.Notifier.Notify(ctx, user, domain.ChannelTelegram, toSend, analyses)
            }
        } else {
            for _, iss := range toSend {
                _ = uc.NotificationRepo.SaveSentNotification(ctx, &domain.SentNotification{
                    ID:      uuid.NewString(),
                    UserID:  user.ID,
                    IssueID: iss.ID,
                    Channel: domain.ChannelEmail,
                    SentAt:  time.Now().UTC(),
                })
            }
        }
    }
    return nil
}

func filterForUser(issues []domain.Issue, analyses map[string]domain.IssueAnalysis, pref *domain.UserPreference) []domain.Issue {
    out := make([]domain.Issue, 0, len(issues))
    languages := decodeStringSlice(pref.Languages)
    wantedLabels := decodeStringSlice(pref.Labels)
nextIssue:
    for _, iss := range issues {
        analysis, ok := analyses[iss.ID]
        if !ok {
            continue
        }
        if analysis.Complexity > pref.MaxComplexity {
            continue
        }

        if len(languages) > 0 {
            matched := false
            issueLabels := decodeStringSlice(iss.Labels)
            for _, lang := range languages {
                for _, l := range issueLabels {
                    if strings.EqualFold(l, lang) {
                        matched = true
                        break
                    }
                }
                if matched {
                    break
                }
                if containsIgnoreCase(iss.Repository.Name, lang) {
                    matched = true
                    break
                }
            }
            if !matched {
                continue
            }
        }

        if len(wantedLabels) > 0 {
            issueLabels := decodeStringSlice(iss.Labels)
            for _, want := range wantedLabels {
                has := false
                for _, l := range issueLabels {
                    if l == want {
                        has = true
                        break
                    }
                }
                if !has {
                    continue nextIssue
                }
            }
        }
        out = append(out, iss)
    }
    return out
}

func decodeStringSlice(raw string) []string {
    if raw == "" {
        return nil
    }
    var values []string
    if err := json.Unmarshal([]byte(raw), &values); err != nil {
        return nil
    }
    return values
}

func containsIgnoreCase(s, sub string) bool {
    return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}