package logger

import (
	"log/slog"
	"os"
	"strings"
)

const defaultService = "project-service"

var defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

func Init(service, env, level string) *slog.Logger {
	if service == "" {
		service = defaultService
	}
	if env == "" {
		env = "local"
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	})
	defaultLogger = slog.New(handler).With(
		"service", service,
		"env", env,
	)
	slog.SetDefault(defaultLogger)
	return defaultLogger
}

func L() *slog.Logger {
	return defaultLogger
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
