package cmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
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

func TestHelpJSON_IncludesCommandContractMetadata(t *testing.T) {
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"assign", "--help-json"})
		if err != nil {
			t.Fatalf("Execute assign --help-json failed: %v", err)
		}
	})

	var payload CommandHelp
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	if payload.Name != "assign" {
		t.Fatalf("expected name assign, got %q", payload.Name)
	}
	if !payload.Mutates {
		t.Fatal("expected mutates=true")
	}
	if !payload.SupportsDryRun {
		t.Fatal("expected supports_dry_run=true")
	}
	if len(payload.Args) != 1 {
		t.Fatalf("expected 1 positional arg, got %d", len(payload.Args))
	}
	if payload.Args[0].Name != "conversation-id" || !payload.Args[0].Required {
		t.Fatalf("unexpected arg metadata: %#v", payload.Args[0])
	}
}

func TestHelpJSON_IncludesCommandContractMetadataForCanonicalSubcommands(t *testing.T) {
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
	if !payload.Mutates {
		t.Fatal("expected mutates=true")
	}
	if !payload.SupportsDryRun {
		t.Fatal("expected supports_dry_run=true")
	}
	if len(payload.Args) != 1 {
		t.Fatalf("expected 1 positional arg, got %d", len(payload.Args))
	}
	if payload.Args[0].Name != "id" || !payload.Args[0].Required || !payload.Args[0].Variadic {
		t.Fatalf("unexpected arg metadata: %#v", payload.Args[0])
	}
}

func TestHelpJSON_IncludesMutationMetadataForMessagesCreate(t *testing.T) {
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"messages", "create", "--help-json"})
		if err != nil {
			t.Fatalf("Execute messages create --help-json failed: %v", err)
		}
	})

	var payload CommandHelp
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	if !payload.Mutates {
		t.Fatal("expected mutates=true")
	}
	if !payload.SupportsDryRun {
		t.Fatal("expected supports_dry_run=true")
	}
}

func TestHelpJSON_IncludesMutationMetadataForHandoff(t *testing.T) {
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"handoff", "--help-json"})
		if err != nil {
			t.Fatalf("Execute handoff --help-json failed: %v", err)
		}
	})

	var payload CommandHelp
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	if !payload.Mutates {
		t.Fatal("expected mutates=true")
	}
	if !payload.SupportsDryRun {
		t.Fatal("expected supports_dry_run=true")
	}
}

func TestHelpJSON_ExposesFieldPresetsAndSchema(t *testing.T) {
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "list", "--help-json"})
		if err != nil {
			t.Fatalf("Execute conversations list --help-json failed: %v", err)
		}
	})

	var payload CommandHelp
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	if payload.FieldSchema != "conversation" {
		t.Fatalf("expected field_schema=conversation, got %q", payload.FieldSchema)
	}
	if len(payload.FieldPresets) == 0 {
		t.Fatal("expected field presets")
	}
	if _, ok := payload.FieldPresets["minimal"]; !ok {
		t.Fatalf("expected minimal field preset, got %#v", payload.FieldPresets)
	}
}

func TestHelpJSON_ParsesPositionalArgsFromAlternation(t *testing.T) {
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"ctx", "--help-json"})
		if err != nil {
			t.Fatalf("Execute ctx --help-json failed: %v", err)
		}
	})

	var payload CommandHelp
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	if len(payload.Args) != 1 {
		t.Fatalf("expected 1 positional arg, got %d", len(payload.Args))
	}
	if payload.Args[0].Name != "conversation-id|url" || !payload.Args[0].Required {
		t.Fatalf("unexpected arg metadata: %#v", payload.Args[0])
	}
}

func TestHelpJSON_MergesRequiredVariadicUsage(t *testing.T) {
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
	if len(payload.Args) != 1 {
		t.Fatalf("expected 1 positional arg, got %d", len(payload.Args))
	}
	if payload.Args[0].Name != "id" || !payload.Args[0].Required || !payload.Args[0].Variadic {
		t.Fatalf("unexpected arg metadata: %#v", payload.Args[0])
	}
}

func TestRegisterCommandContract_DoesNotImplicitlyMarkMutating(t *testing.T) {
	cmd := &cobra.Command{}

	registerCommandContract(cmd, false, true)

	if commandMutates(cmd) {
		t.Fatal("expected mutates=false")
	}
	if !commandSupportsDryRun(cmd) {
		t.Fatal("expected supports_dry_run=true")
	}
}
