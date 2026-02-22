package cmd

import (
	"context"
	"encoding/json"
	"testing"
)

func TestHelpJSON_Root(t *testing.T) {
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"--help-json"})
		if err != nil {
			t.Fatalf("Execute --help-json failed: %v", err)
		}
	})

	var payload CommandHelp
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	if payload.Name != "cw" {
		t.Fatalf("expected name cw, got %q", payload.Name)
	}
	if len(payload.Subcommands) == 0 {
		t.Fatalf("expected subcommands, got none")
	}
}

func TestHelpJSON_BypassesArgsValidation(t *testing.T) {
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "assign", "--help-json"})
		if err != nil {
			t.Fatalf("Execute conversations assign --help-json failed: %v", err)
		}
	})

	var payload CommandHelp
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	if payload.Name != "assign" {
		t.Fatalf("expected name assign, got %q", payload.Name)
	}
	if len(payload.Flags) == 0 {
		t.Fatalf("expected flags, got none")
	}
}
