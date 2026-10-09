package logging

import (
	"context"
	"log/slog"
	"testing"
)

func TestEnvironmentLogLevels(t *testing.T) {
	for _, mode := range []string{"development", "production", "staging"} {
		t.Setenv("APP_ENV", mode)
		t.Setenv("LOG_LEVEL", "")
		logger, err := New()
		if err != nil {
			t.Fatal(err)
		}
		if logger.Enabled(context.Background(), slog.LevelInfo) != (mode == "development") {
			t.Fatal("incorrect environment default", mode)
		}
	}
	t.Setenv("LOG_LEVEL", "debug")
	logger, err := New()
	if err != nil || !logger.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("explicit log level ignored")
	}
	t.Setenv("LOG_LEVEL", "typo")
	if _, err := New(); err == nil {
		t.Fatal("invalid log level accepted")
	}
}
