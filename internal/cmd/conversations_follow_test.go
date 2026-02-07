package cmd

import "testing"

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
