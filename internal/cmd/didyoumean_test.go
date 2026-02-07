package cmd

import "testing"

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "b", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
	}
	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSuggestCommand(t *testing.T) {
	commands := []string{"conversations", "contacts", "messages", "search", "assign", "reply", "snooze", "handoff", "dashboard", "version"}
	tests := []struct {
		input string
		want  string
	}{
		{"conversatins", "conversations"},
		{"contacs", "contacts"},
		{"mesages", "messages"},
		{"serach", "search"},
		{"asign", "assign"},
		{"replly", "reply"},
		{"snoze", "snooze"},
		{"handof", "handoff"},
		{"dashbord", "dashboard"},
		{"zzzzzzzzz", ""}, // too far, no suggestion
	}
	for _, tt := range tests {
		got := suggestCommand(tt.input, commands)
		if got != tt.want {
			t.Errorf("suggestCommand(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSuggestFlag(t *testing.T) {
	flags := []string{"--status", "--priority", "--agent", "--team", "--page", "--output"}
	tests := []struct {
		input string
		want  string
	}{
		{"--staus", "--status"},
		{"--pririty", "--priority"},
		{"--agnt", "--agent"},
		{"--tem", "--team"},
		{"--pge", "--page"},
		{"--outpt", "--output"},
		{"--zzzzzzz", ""}, // too far
	}
	for _, tt := range tests {
		got := suggestFlag(tt.input, flags)
		if got != tt.want {
			t.Errorf("suggestFlag(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSuggestFlag_StripsDashes(t *testing.T) {
	flags := []string{"--status", "-s"}
	got := suggestFlag("--staus", flags)
	if got != "--status" {
		t.Errorf("suggestFlag(--staus) = %q, want --status", got)
	}
}
