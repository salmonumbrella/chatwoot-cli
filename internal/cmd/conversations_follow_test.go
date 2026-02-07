package cmd

import (
	"context"
	"io"
	"strings"
	"testing"
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
