// internal/debug/debug_test.go
package debug

import (
	"context"
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
