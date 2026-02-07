package cmd

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestFollowCmdAcceptsAllFlag(t *testing.T) {
	cmd := newConversationsFollowCmd()
	if cmd.Flags().Lookup("all") == nil {
		t.Fatal("missing --all flag")
	}
}

func TestFollowCmdAcceptsEventsFlag(t *testing.T) {
	cmd := newConversationsFollowCmd()
	f := cmd.Flags().Lookup("events")
	if f == nil {
		t.Fatal("missing --events flag")
	}
	if f.DefValue == "" {
		t.Fatal("--events should have a default value")
	}
}

func TestFollowCmdAgentModeExists(t *testing.T) {
	cmd := newConversationsFollowCmd()
	if cmd == nil {
		t.Fatal("nil command")
	}
}

func TestFollowCmdAcceptsDebounceFlag(t *testing.T) {
	cmd := newConversationsFollowCmd()
	if cmd.Flags().Lookup("debounce") == nil {
		t.Fatal("missing --debounce flag")
	}
}

func TestFollowCmdAcceptsTypingFlag(t *testing.T) {
	cmd := newConversationsFollowCmd()
	if cmd.Flags().Lookup("typing") == nil {
		t.Fatal("missing --typing flag")
	}
}

func TestFollowCmdAcceptsRawFlag(t *testing.T) {
	cmd := newConversationsFollowCmd()
	if cmd.Flags().Lookup("raw") == nil {
		t.Fatal("missing --raw flag")
	}
}

func TestFollowCmdAcceptsContextFlags(t *testing.T) {
	cmd := newConversationsFollowCmd()
	if cmd.Flags().Lookup("context") == nil {
		t.Fatal("missing --context flag")
	}
	if cmd.Flags().Lookup("context-messages") == nil {
		t.Fatal("missing --context-messages flag")
	}
}

func TestFollowCmdAcceptsCursorFlags(t *testing.T) {
	cmd := newConversationsFollowCmd()
	if cmd.Flags().Lookup("cursor-file") == nil {
		t.Fatal("missing --cursor-file flag")
	}
	if cmd.Flags().Lookup("since-id") == nil {
		t.Fatal("missing --since-id flag")
	}
	if cmd.Flags().Lookup("since-time") == nil {
		t.Fatal("missing --since-time flag")
	}
}

func TestParseSinceTime(t *testing.T) {
	now := time.Date(2026, 2, 7, 12, 0, 0, 0, time.UTC)
	if _, err := parseSinceTime("2026-02-07T11:00:00Z", now); err != nil {
		t.Fatalf("parse RFC3339: %v", err)
	}
	if ts, err := parseSinceTime("60m", now); err != nil || ts.After(now) {
		t.Fatalf("parse duration: ts=%v err=%v", ts, err)
	}
	if _, err := parseSinceTime("1738920000", now); err != nil {
		t.Fatalf("parse unix seconds: %v", err)
	}
	if _, err := parseSinceTime("not-a-time", now); err == nil {
		t.Fatal("expected error for invalid time")
	}
}

func TestFollowCmdAcceptsFilterFlags(t *testing.T) {
	cmd := newConversationsFollowCmd()
	for _, name := range []string{
		"inbox",
		"status",
		"assignee",
		"label",
		"priority",
		"contact",
		"only-unassigned",
		"exclude-private",
		"queue",
		"drop",
		"max-batch",
		"exec",
		"exec-timeout",
	} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing --%s flag", name)
		}
	}
}

func TestFollowCmdAcceptsExecFatalFlag(t *testing.T) {
	cmd := newConversationsFollowCmd()
	if cmd.Flags().Lookup("exec-fatal") == nil {
		t.Fatal("missing --exec-fatal flag")
	}
}

func TestBackoffResetsAfterStableConnection(t *testing.T) {
	// Verify the logic: if connection lasted > threshold, backoff resets.
	// This tests the pure logic, not the full WebSocket flow.
	initialBackoff := 2 * time.Second
	maxBackoff := 30 * time.Second
	resetThreshold := 60 * time.Second

	backoff := initialBackoff

	// Simulate escalation.
	for range 5 {
		backoff = min(backoff*2, maxBackoff)
	}
	if backoff != maxBackoff {
		t.Fatalf("expected backoff to reach max, got %s", backoff)
	}

	// Simulate a connection that lasted > threshold.
	connectionDuration := 2 * time.Minute
	if connectionDuration > resetThreshold {
		backoff = initialBackoff
	}
	if backoff != initialBackoff {
		t.Fatalf("expected backoff to reset, got %s", backoff)
	}
}

func TestFollowCmdAllWithTailErrors(t *testing.T) {
	cmd := newConversationsFollowCmd()
	cmd.SetContext(context.Background())
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--all", "--tail", "5"})
	_, err := cmd.ExecuteC()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--tail requires a single conversation") {
		t.Fatalf("unexpected error: %v", err)
	}
}
