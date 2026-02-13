package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestParseFieldsWithSchemaValidation(t *testing.T) {
	cmd := &cobra.Command{}
	registerFieldSchema(cmd, "contact")

	fields, err := parseFieldsWithPresets(cmd, "id,email")
	if err != nil {
		t.Fatalf("expected valid fields, got error: %v", err)
	}
	if len(fields) != 2 || fields[0] != "id" || fields[1] != "email" {
		t.Fatalf("unexpected fields: %v", fields)
	}

	_, err = parseFieldsWithPresets(cmd, "id,unknown_field")
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}

	// Nested fields should validate against the root property.
	fields, err = parseFieldsWithPresets(cmd, "custom_attributes.plan")
	if err != nil {
		t.Fatalf("expected nested field to be valid, got error: %v", err)
	}
	if len(fields) != 1 || fields[0] != "custom_attributes.plan" {
		t.Fatalf("unexpected fields: %v", fields)
	}
}

func TestParseFieldsWithSchemaValidation_Aliases(t *testing.T) {
	cmd := &cobra.Command{}
	registerFieldSchema(cmd, "conversation")

	fields, err := parseFieldsWithPresets(cmd, "i,st,ii,la,cu.plan")
	if err != nil {
		t.Fatalf("expected aliased fields to be valid, got error: %v", err)
	}

	want := []string{"id", "status", "inbox_id", "last_activity_at", "custom_attributes.plan"}
	if len(fields) != len(want) {
		t.Fatalf("unexpected fields length: got %v want %v", fields, want)
	}
	for i := range fields {
		if fields[i] != want[i] {
			t.Fatalf("unexpected fields: got %v want %v", fields, want)
		}
	}
}

func TestParseFieldsWithSchemaValidation_MixedCaseAliasesNotRewritten(t *testing.T) {
	cmd := &cobra.Command{}
	registerFieldSchema(cmd, "conversation")

	_, err := parseFieldsWithPresets(cmd, "St")
	if err == nil {
		t.Fatal("expected mixed-case alias to fail schema validation")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestSchemaPresetsMergeWithManualPresets(t *testing.T) {
	cmd := &cobra.Command{}
	registerFieldSchema(cmd, "contact")
	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id"},
	})

	presets, err := fieldPresetsForCommand(cmd)
	if err != nil {
		t.Fatalf("fieldPresetsForCommand returned error: %v", err)
	}

	minimal := presets["minimal"]
	if len(minimal) != 1 || minimal[0] != "id" {
		t.Fatalf("expected manual minimal preset to be preserved, got %v", minimal)
	}

	if _, ok := presets["debug"]; !ok {
		t.Fatalf("expected schema-derived presets to include debug, got %v", presets)
	}
}
