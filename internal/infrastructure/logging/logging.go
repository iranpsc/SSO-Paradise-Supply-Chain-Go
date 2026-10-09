package logging

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

func New() (*slog.Logger, error) {
	level := slog.LevelDebug
	if mode := os.Getenv("APP_ENV"); mode == "production" || mode == "staging" {
		level = slog.LevelWarn
	}
	if raw := strings.TrimSpace(os.Getenv("LOG_LEVEL")); raw != "" {
		if err := level.UnmarshalText([]byte(raw)); err != nil {
			return nil, fmt.Errorf("invalid LOG_LEVEL: %w", err)
		}
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})), nil
}
