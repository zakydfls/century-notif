package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Level  string
	Format string
}

func New(config Config) *slog.Logger {
	level := new(slog.LevelVar)
	switch strings.ToLower(config.Level) {
	case "debug":
		level.Set(slog.LevelDebug)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}

	opts := &slog.HandlerOptions{Level: level}
	if strings.EqualFold(config.Format, "text") {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func WithRequestID(ctx context.Context, log *slog.Logger, requestID string) *slog.Logger {
	if requestID == "" {
		return log
	}
	return log.With("request_id", requestID)
}

func Redact(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}
