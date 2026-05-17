package infrastructure

import (
	"log/slog"
	"os"
	"strings"
)

func NewLogger(level string) *slog.Logger {
    var logLevel slog.Level
    switch strings.ToLower(level) {
    case "debug":
        logLevel = slog.LevelDebug
    case "warn":
        logLevel = slog.LevelWarn
    case "error":
        logLevel = slog.LevelError
    default:
        logLevel = slog.LevelInfo
    }

    opts := &slog.HandlerOptions{
        Level: logLevel,
    }
    handler := slog.NewTextHandler(os.Stdout, opts)
    return slog.New(handler)
}