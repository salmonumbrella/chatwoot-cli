package outfmt

import (
	"bytes"
	"context"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input       string
		expected    Mode
		expectError bool
	}{
		{"text", Text, false},
		{"", Text, false},
		{"json", JSON, false},
		{"jsonl", JSONL, false},
		{"invalid", Text, true},
		{"JSON", Text, true}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			mode, err := Parse(tt.input)
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && mode != tt.expected {
				t.Errorf("Expected mode %v, got %v", tt.expected, mode)
			}
		})
	}
}

func TestModeContext(t *testing.T) {
	ctx := context.Background()

	// Default should be Text
	if ModeFromContext(ctx) != Text {
		t.Error("Expected default mode to be Text")
	}
	if IsJSON(ctx) {
		t.Error("Expected IsJSON to be false for default context")
	}

	// With JSON mode
	jsonCtx := WithMode(ctx, JSON)
	if ModeFromContext(jsonCtx) != JSON {
		t.Error("Expected mode to be JSON")
	}
	if !IsJSON(jsonCtx) {
		t.Error("Expected IsJSON to be true")
	}

	// With JSONL mode
	jsonlCtx := WithMode(ctx, JSONL)
	if ModeFromContext(jsonlCtx) != JSONL {
		t.Error("Expected mode to be JSONL")
	}
	if !IsJSON(jsonlCtx) {
		t.Error("Expected IsJSON to be true for JSONL")
	}
	if !IsJSONL(jsonlCtx) {
		t.Error("Expected IsJSONL to be true for JSONL")
	}

	// With Text mode
	textCtx := WithMode(ctx, Text)
	if ModeFromContext(textCtx) != Text {
		t.Error("Expected mode to be Text")
	}
}

func TestModeString(t *testing.T) {
	if Text.String() != "text" {
		t.Errorf("Expected 'text', got %q", Text.String())
	}
	if JSON.String() != "json" {
		t.Errorf("Expected 'json', got %q", JSON.String())
	}
	if JSONL.String() != "jsonl" {
		t.Errorf("Expected 'jsonl', got %q", JSONL.String())
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"key": "value"}

	err := WriteJSON(&buf, data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "{\n  \"key\": \"value\"\n}\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}
