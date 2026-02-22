package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSchemaListCommand(t *testing.T) {
	// No mock server needed - schema commands don't call the API
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"schema", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("schema list failed: %v", err)
	}

	// Verify output contains expected schemas
	expectedSchemas := []string{"conversation", "contact", "message", "inbox", "agent", "team", "label"}
	for _, name := range expectedSchemas {
		if !strings.Contains(output, name) {
			t.Errorf("output missing schema %q: %s", name, output)
		}
	}

	// Check headers
	if !strings.Contains(output, "RESOURCE") || !strings.Contains(output, "DESCRIPTION") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestSchemaListCommand_JSON(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"schema", "list", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("schema list failed: %v", err)
	}

	schemas := decodeItems(t, output)

	if len(schemas) < 7 {
		t.Errorf("expected at least 7 schemas, got %d", len(schemas))
	}

	// Verify each schema has name and description
	for i, s := range schemas {
		if s["name"] == nil {
			t.Errorf("schema %d missing 'name'", i)
		}
		if s["description"] == nil {
			t.Errorf("schema %d missing 'description'", i)
		}
	}
}

func TestSchemaShowCommand(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"schema", "show", "conversation"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("schema show failed: %v", err)
	}

	// Verify output contains schema details
	if !strings.Contains(output, "Schema: conversation") {
		t.Errorf("output missing schema name: %s", output)
	}
	if !strings.Contains(output, "Type: object") {
		t.Errorf("output missing type: %s", output)
	}
	if !strings.Contains(output, "Fields:") {
		t.Errorf("output missing fields section: %s", output)
	}

	// Check for expected fields
	expectedFields := []string{"id", "inbox_id", "status", "priority"}
	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("output missing field %q: %s", field, output)
		}
	}

	// Check for enum values
	if !strings.Contains(output, "open") || !strings.Contains(output, "resolved") {
		t.Errorf("output missing status enum values: %s", output)
	}
}

func TestSchemaShowCommand_Contact(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"schema", "show", "contact"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("schema show failed: %v", err)
	}

	// Verify output contains contact schema details
	if !strings.Contains(output, "Schema: contact") {
		t.Errorf("output missing schema name: %s", output)
	}

	expectedFields := []string{"name", "email", "phone_number", "identifier"}
	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("output missing field %q: %s", field, output)
		}
	}
}

func TestSchemaShowCommand_JSON(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"schema", "show", "conversation", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("schema show failed: %v", err)
	}

	// Verify it's valid JSON
	var s map[string]any
	if err := json.Unmarshal([]byte(output), &s); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if s["type"] != "object" {
		t.Errorf("expected type 'object', got %v", s["type"])
	}

	properties, ok := s["properties"].(map[string]any)
	if !ok {
		t.Errorf("expected properties to be an object, got %T", s["properties"])
	}

	if properties["id"] == nil {
		t.Error("expected properties to contain 'id'")
	}
	if properties["status"] == nil {
		t.Error("expected properties to contain 'status'")
	}
}

func TestSchemaShowCommand_NotFound(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"schema", "show", "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent schema")
	}
}

func TestSchemaShowCommand_MissingArg(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"schema", "show"})
	if err == nil {
		t.Error("expected error when resource name is missing")
	}
}

func TestSchemaShowCommand_AllSchemas(t *testing.T) {
	// Test that all registered schemas can be shown
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	schemas := []string{"conversation", "contact", "message", "inbox", "agent", "team", "label"}

	for _, name := range schemas {
		t.Run(name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := Execute(context.Background(), []string{"schema", "show", name})

			_ = w.Close()
			os.Stdout = old

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			output := buf.String()

			if err != nil {
				t.Errorf("schema show %s failed: %v", name, err)
			}

			if !strings.Contains(output, "Schema: "+name) {
				t.Errorf("output missing schema name %s: %s", name, output)
			}
		})
	}
}

func TestSchemaShowCommand_RequiredFields(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"schema", "show", "conversation"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("schema show failed: %v", err)
	}

	// Check that required fields are marked
	if !strings.Contains(output, "(required)") {
		t.Errorf("output should indicate required fields: %s", output)
	}

	// Check the Required line at the bottom
	if !strings.Contains(output, "Required:") {
		t.Errorf("output should have Required section: %s", output)
	}
}

func TestSchemaShowCommand_EnumValues(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"schema", "show", "agent"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("schema show failed: %v", err)
	}

	// Check that enum values are shown
	if !strings.Contains(output, "Allowed values:") {
		t.Errorf("output should show allowed values for enums: %s", output)
	}

	// Check specific enum values
	if !strings.Contains(output, "agent") || !strings.Contains(output, "administrator") {
		t.Errorf("output should show role enum values: %s", output)
	}
}
