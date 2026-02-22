// internal/outfmt/query_test.go
package outfmt

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestWithQuery(t *testing.T) {
	ctx := WithQuery(context.Background(), ".name")
	if GetQuery(ctx) != ".name" {
		t.Error("GetQuery should return the query set with WithQuery")
	}
}

func TestGetQuery_EmptyByDefault(t *testing.T) {
	ctx := context.Background()
	if GetQuery(ctx) != "" {
		t.Error("GetQuery should return empty string by default")
	}
}

func TestWriteJSONFiltered_EmptyQuery(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"name": "test"}
	err := WriteJSONFiltered(&buf, data, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify it outputs valid JSON
	if !strings.Contains(buf.String(), "name") {
		t.Error("expected name in output")
	}
}

func TestWriteJSONFiltered_WithQuery(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"name": "test", "id": "123"}
	err := WriteJSONFiltered(&buf, data, ".name", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != `"test"` {
		t.Errorf("expected filtered output, got: %s", buf.String())
	}
}

func TestWriteJSONFiltered_InvalidQuery(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"name": "test"}
	err := WriteJSONFiltered(&buf, data, "invalid[[[", false)
	if err == nil {
		t.Error("expected error for invalid query")
	}
}

func TestWriteJSONFiltered_WrapsSlice(t *testing.T) {
	var buf bytes.Buffer
	data := []string{"a", "b"}
	err := WriteJSONFiltered(&buf, data, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "\"items\"") {
		t.Errorf("expected items wrapper, got: %s", buf.String())
	}
}

func TestApplyQuery_EmptyQuery(t *testing.T) {
	data := map[string]string{"name": "test"}
	result, err := ApplyQuery(data, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With empty query, should return original data structure
	m, ok := result.(map[string]string)
	if !ok {
		t.Fatalf("expected map[string]string, got %T", result)
	}
	if m["name"] != "test" {
		t.Errorf("expected name=test, got %v", m["name"])
	}
}

func TestApplyQuery_WithQuery(t *testing.T) {
	data := map[string]string{"name": "test", "id": "123"}
	result, err := ApplyQuery(data, ".name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("expected 'test', got %v", result)
	}
}

func TestApplyQuery_InvalidQuery(t *testing.T) {
	data := map[string]string{"name": "test"}
	_, err := ApplyQuery(data, "invalid[[[")
	if err == nil {
		t.Error("expected error for invalid query")
	}
}

func TestApplyQuery_ArrayFilter(t *testing.T) {
	data := []map[string]string{
		{"name": "alice"},
		{"name": "bob"},
	}
	result, err := ApplyQuery(data, ".items[0].name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "alice" {
		t.Errorf("expected 'alice', got %v", result)
	}
}

func TestWriteJSONFiltered_Compact(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{"name": "test", "nested": map[string]int{"a": 1}}
	err := WriteJSONFiltered(&buf, data, "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if strings.Contains(out, "\n") {
		t.Errorf("compact output should be a single line, got: %s", out)
	}
	if !strings.Contains(out, `"name":"test"`) {
		t.Errorf("expected compact JSON, got: %s", out)
	}
}

func TestWriteJSONFiltered_CompactWithQuery(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{"name": "test", "items": []int{1, 2, 3}}
	err := WriteJSONFiltered(&buf, data, "{n: .name, it: .items}", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	if strings.Contains(out, "\n") {
		t.Errorf("compact output should be a single line, got: %s", out)
	}
}

func TestWriteJSONFiltered_NotCompactByDefault(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{"name": "test", "nested": map[string]int{"a": 1}}
	err := WriteJSONFiltered(&buf, data, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "\n  ") {
		t.Errorf("default output should be indented, got: %s", out)
	}
}

func TestWithCompact(t *testing.T) {
	ctx := WithCompact(context.Background(), true)
	if !IsCompact(ctx) {
		t.Error("IsCompact should return true after WithCompact(true)")
	}
}

func TestIsCompact_FalseByDefault(t *testing.T) {
	ctx := context.Background()
	if IsCompact(ctx) {
		t.Error("IsCompact should return false by default")
	}
}

func TestWriteJSONFiltered_RawMessageUnchanged(t *testing.T) {
	raw := json.RawMessage(`{"it":"literal","items":"canonical"}`)
	original := append([]byte(nil), raw...)

	var buf bytes.Buffer
	if err := WriteJSONFiltered(&buf, raw, `.["it"]`, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(buf.String()) != `"literal"` {
		t.Fatalf("expected literal lookup result, got %q", buf.String())
	}
	if !bytes.Equal(raw, original) {
		t.Fatalf("raw JSON payload was mutated: got %s want %s", raw, original)
	}
}
