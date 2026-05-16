package notification

import (
	"context"
	"fmt"

	"github.com/alakkaya/openscout/internal/domain"
	"github.com/alakkaya/openscout/internal/port"
)

type CompositeNotifier struct {
    EmailSender    *EmailSender
    TelegramSender *TelegramSender
}

func NewCompositeNotifier(email *EmailSender, tg *TelegramSender) *CompositeNotifier {
    return &CompositeNotifier{EmailSender: email, TelegramSender: tg}
}

func (c *CompositeNotifier) Notify(ctx context.Context, user *domain.User, channel domain.NotificationChannel, issues []domain.Issue, analyses map[string]domain.IssueAnalysis) error {
    switch channel {
    case domain.ChannelEmail:
        if c.EmailSender == nil {
            return fmt.Errorf("email sender not configured")
        }
        return c.EmailSender.Send(ctx, user, issues, analyses)
    case domain.ChannelTelegram:
        if c.TelegramSender == nil {
            return fmt.Errorf("telegram sender not configured")
        }
        return c.TelegramSender.Send(ctx, user, issues, analyses)
    default:
        return fmt.Errorf("unsupported channel: %s", channel)
    }
}

// Ensure CompositeNotifier implements port.Notifier
var _ port.Notifier = (*CompositeNotifier)(nil)