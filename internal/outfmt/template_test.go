package outfmt

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestWithTemplate(t *testing.T) {
	ctx := WithTemplate(context.Background(), "{{.name}}")
	if GetTemplate(ctx) != "{{.name}}" {
		t.Error("GetTemplate should return the template set with WithTemplate")
	}
}

func TestGetTemplate_EmptyByDefault(t *testing.T) {
	ctx := context.Background()
	if GetTemplate(ctx) != "" {
		t.Error("GetTemplate should return empty string by default")
	}
}

func TestWriteTemplate_SimpleField(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"name": "test", "id": "123"}
	err := WriteTemplate(&buf, data, "Name: {{.name}}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "Name: test" {
		t.Errorf("expected 'Name: test', got: %s", buf.String())
	}
}

func TestWriteTemplate_MultipleFields(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"name": "test", "id": "123"}
	err := WriteTemplate(&buf, data, "{{.id}}: {{.name}}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "123: test" {
		t.Errorf("expected '123: test', got: %s", buf.String())
	}
}

func TestWriteTemplate_JSONFunc(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"name": "test"}
	err := WriteTemplate(&buf, data, "{{json .}}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, `"name"`) || !strings.Contains(output, `"test"`) {
		t.Errorf("expected JSON output with name:test, got: %s", output)
	}
}

func TestWriteTemplate_InvalidTemplate(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"name": "test"}
	err := WriteTemplate(&buf, data, "{{.name")
	if err == nil {
		t.Error("expected error for invalid template")
	}
	if !strings.Contains(err.Error(), "invalid template") {
		t.Errorf("error should mention invalid template, got: %v", err)
	}
}

func TestWriteTemplate_MissingKey(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"name": "test"}
	// With missingkey=zero option, missing keys should render as zero value
	err := WriteTemplate(&buf, data, "{{.missing}}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Missing key renders as empty string (zero value for string)
	if buf.String() != "" {
		t.Errorf("expected empty output for missing key, got: %s", buf.String())
	}
}

func TestWriteTemplate_Array(t *testing.T) {
	var buf bytes.Buffer
	data := []map[string]string{
		{"name": "alice"},
		{"name": "bob"},
	}
	err := WriteTemplate(&buf, data, "{{range .}}{{.name}} {{end}}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "alice bob " {
		t.Errorf("expected 'alice bob ', got: %s", buf.String())
	}
}
