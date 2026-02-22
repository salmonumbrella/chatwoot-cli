// Package debug provides context-based debug mode with structured logging.
package debug

import (
	"context"
	"log/slog"
	"os"
)

type contextKey string

const debugKey contextKey = "debug_enabled"

// WithDebug returns a context with debug mode enabled/disabled.
func WithDebug(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, debugKey, enabled)
}

// IsEnabled returns true if debug mode is enabled in the context.
func IsEnabled(ctx context.Context) bool {
	if v, ok := ctx.Value(debugKey).(bool); ok {
		return v
	}
	return false
}

// SetupLogger configures slog based on debug mode.
func SetupLogger(debugEnabled bool) {
	var level slog.Level
	if debugEnabled {
		level = slog.LevelDebug
	} else {
		level = slog.LevelWarn
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}
