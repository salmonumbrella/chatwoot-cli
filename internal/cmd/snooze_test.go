package cmd

import (
	"context"
	"testing"
	"time"
)

func TestSnoozeCommandExists(t *testing.T) {
	err := Execute(context.Background(), []string{"snooze", "--help"})
	if err != nil {
		t.Fatalf("snooze --help failed: %v", err)
	}
}

func TestParseSnoozeFor(t *testing.T) {
	now := time.Date(2026, 2, 7, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		input    string
		wantErr  bool
		minDelta time.Duration
	}{
		{"2h", false, 2 * time.Hour},
		{"30m", false, 30 * time.Minute},
		{"1h30m", false, 90 * time.Minute},
		{"", true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseSnoozeFor(tt.input, now)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			diff := result.Sub(now)
			if diff < tt.minDelta-time.Second || diff > tt.minDelta+time.Second {
				t.Errorf("expected ~%s from now, got %s", tt.minDelta, diff)
			}
		})
	}
}
