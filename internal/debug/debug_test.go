// internal/debug/debug_test.go
package debug

import (
	"context"
	"log/slog"
	"testing"
)

func TestWithDebug(t *testing.T) {
	ctx := WithDebug(context.Background(), true)
	if !IsEnabled(ctx) {
		t.Error("IsEnabled should return true when debug is enabled")
	}
}

func TestIsEnabled_DefaultFalse(t *testing.T) {
	ctx := context.Background()
	if IsEnabled(ctx) {
		t.Error("IsEnabled should return false by default")
	}
}

func TestWithDebug_Disabled(t *testing.T) {
	ctx := WithDebug(context.Background(), false)
	if IsEnabled(ctx) {
		t.Error("IsEnabled should return false when debug is disabled")
	}
}

func TestSetupLogger_DebugEnabled(t *testing.T) {
	// Call SetupLogger with debug enabled
	SetupLogger(true)

	// Verify that debug-level logging is enabled
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("SetupLogger(true) should enable debug level logging")
	}
}

func TestSetupLogger_DebugDisabled(t *testing.T) {
	// Call SetupLogger with debug disabled
	SetupLogger(false)

	// Verify that debug-level logging is disabled
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("SetupLogger(false) should disable debug level logging")
	}

	// Verify that warn-level logging is still enabled
	if !slog.Default().Enabled(context.Background(), slog.LevelWarn) {
		t.Error("SetupLogger(false) should enable warn level logging")
	}
}
