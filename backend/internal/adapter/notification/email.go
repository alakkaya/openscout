package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/alakkaya/openscout/internal/domain"
)

type EmailConfig struct {
    Host     string
    Port     int
    Username string
    Password string
    From     string
}

type EmailSender struct {
    cfg EmailConfig
    auth smtp.Auth
    addr string
}

func NewEmailSender(cfg EmailConfig) *EmailSender {
    addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
    auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
    return &EmailSender{cfg: cfg, auth: auth, addr: addr}
}

func (s *EmailSender) Send(ctx context.Context, user *domain.User, issues []domain.Issue, analyses map[string]domain.IssueAnalysis) error {
    if user == nil || user.Email == "" {
        return fmt.Errorf("invalid user email")
    }

    subject := "OpenScout — daily opportunities"
    body := buildEmailBody(issues, analyses)

    // simple RFC5322 headers
    msg := strings.Builder{}
    msg.WriteString(fmt.Sprintf("From: %s\r\n", s.cfg.From))
    msg.WriteString(fmt.Sprintf("To: %s\r\n", user.Email))
    msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
    msg.WriteString("MIME-Version: 1.0\r\n")
    msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
    msg.WriteString("\r\n")
    msg.WriteString(body)

    // Use TLS connection to SMTP server
    conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", s.addr, &tls.Config{
        InsecureSkipVerify: false,
        ServerName:         s.cfg.Host,
    })
    if err != nil {
        return err
    }
    c, err := smtp.NewClient(conn, s.cfg.Host)
    if err != nil {
        return err
    }
    defer c.Close()

    if err = c.Auth(s.auth); err != nil {
        return err
    }
    if err = c.Mail(s.cfg.From); err != nil {
        return err
    }
    if err = c.Rcpt(user.Email); err != nil {
        return err
    }
    w, err := c.Data()
    if err != nil {
        return err
    }
    _, err = w.Write([]byte(msg.String()))
    if err != nil {
        _ = w.Close()
        return err
    }
    return w.Close()
}

func buildEmailBody(issues []domain.Issue, analyses map[string]domain.IssueAnalysis) string {
    //TODO minimal plain-text body; can extend to HTML later
    var b strings.Builder
    b.WriteString("OpenScout — Today's opportunities\n\n")
    for i, iss := range issues {
        if i >= 10 { break }
        a, _ := analyses[iss.ID]
        b.WriteString(fmt.Sprintf("%d) %s\n", i+1, iss.Title))
        b.WriteString(fmt.Sprintf("   Repo: %s\n", iss.Repository.Name))
        b.WriteString(fmt.Sprintf("   Difficulty: %d/5  ~%dh\n", a.Complexity, a.EstimatedHours))
        b.WriteString(fmt.Sprintf("   Link: %s\n\n", iss.URL))
    }
    return b.String()
}