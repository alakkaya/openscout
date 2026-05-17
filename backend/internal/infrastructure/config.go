package infrastructure

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
    Database DatabaseConfig
    GitHub   GitHubConfig
    Analyzer AnalyzerConfig
    Email    EmailConfig
    Telegram TelegramConfig
    Logger   LoggerConfig
}

type DatabaseConfig struct {
    DSN string
}

type GitHubConfig struct {
    Token string
}

type AnalyzerConfig struct {
    URL string
}

type EmailConfig struct {
    Host     string
    Port     int
    Username string
    Password string
    From     string
}

type TelegramConfig struct {
    BotToken string
}

type LoggerConfig struct {
    Level string // debug, info, warn, error
}

func LoadConfig() (*Config, error) {
    cfg := &Config{
        Database: DatabaseConfig{
            DSN: getEnvRequired("DATABASE_URL"),
        },
        GitHub: GitHubConfig{
            Token: getEnvRequired("GITHUB_TOKEN"),
        },
        Analyzer: AnalyzerConfig{
            URL: getEnvRequired("ANALYZER_URL"),
        },
        Email: EmailConfig{
            Host:     getEnv("SMTP_HOST", ""),
            Port:     getEnvInt("SMTP_PORT", 587),
            Username: getEnv("SMTP_USER", ""),
            Password: getEnv("SMTP_PASS", getEnv("SMTP_PASSWORD", "")),
            From:     getEnv("SMTP_FROM", ""),
        },
        Telegram: TelegramConfig{
            BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
        },
        Logger: LoggerConfig{
            Level: getEnv("LOG_LEVEL", "info"),
        },
    }
    return cfg, nil
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}

func getEnvRequired(key string) string {
    v := os.Getenv(key)
    if v == "" {
        panic(fmt.Sprintf("required env var missing: %s", key))
    }
    return v
}

func getEnvInt(key string, fallback int) int {
    if v := os.Getenv(key); v != "" {
        if i, err := strconv.Atoi(v); err == nil {
            return i
        }
    }
    return fallback
}