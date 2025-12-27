// internal/outfmt/query_test.go
package outfmt

import (
	"bytes"
	"context"
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
	err := WriteJSONFiltered(&buf, data, "")
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
	err := WriteJSONFiltered(&buf, data, ".name")
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
	err := WriteJSONFiltered(&buf, data, "invalid[[[")
	if err == nil {
		t.Error("expected error for invalid query")
	}
}
