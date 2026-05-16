package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alakkaya/openscout/internal/domain"
)

type TelegramSender struct {
    botToken string
    client   *http.Client
}

func NewTelegramSender(botToken string) *TelegramSender {
    return &TelegramSender{
        botToken: botToken,
        client: &http.Client{Timeout: 10 * time.Second},
    }
}

func (t *TelegramSender) Send(ctx context.Context, user *domain.User, issues []domain.Issue, analyses map[string]domain.IssueAnalysis) error {
    if user == nil || user.TelegramChatID == nil {
        return fmt.Errorf("no telegram chat id")
    }
    msg := buildTelegramMessage(issues, analyses)
    payload := map[string]interface{}{
        "chat_id": user.TelegramChatID,
        "text": msg,
        "parse_mode": "HTML",
        "disable_web_page_preview": true,
    }
    body, _ := json.Marshal(payload)
    url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
    req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    res, err := t.client.Do(req)
    if err != nil {
        return err
    }
    defer res.Body.Close()
    if res.StatusCode != http.StatusOK {
        return fmt.Errorf("telegram returned %d", res.StatusCode)
    }
    return nil
}

func buildTelegramMessage(issues []domain.Issue, analyses map[string]domain.IssueAnalysis) string {
    var sb strings.Builder
    sb.WriteString("🔭 OpenScout — Today's opportunities\n\n")
    for i, iss := range issues {
        if i >= 5 { break }
        a, _ := analyses[iss.ID]
        sb.WriteString(fmt.Sprintf("%d. <b>%s</b>\n", i+1, iss.Title))
        sb.WriteString(fmt.Sprintf("   📦 %s\n", iss.Repository.Name))
        sb.WriteString(fmt.Sprintf("   🟢 Difficulty: %d/5  ~%dh\n", a.Complexity, a.EstimatedHours))
        sb.WriteString(fmt.Sprintf("   🔗 %s\n\n", iss.URL))
    }
    return sb.String()
}