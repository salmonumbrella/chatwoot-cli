package outfmt

import (
	"bytes"
	"context"
	"testing"
)

func TestFormatter_Output_JSON(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithMode(context.Background(), JSON)
	f := NewFormatter(ctx, &buf, &buf)

	data := map[string]string{"name": "test"}
	if err := f.Output(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte(`"name"`)) {
		t.Error("output should contain JSON")
	}
}

func TestFormatter_Table(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithMode(context.Background(), Text)
	f := NewFormatter(ctx, &buf, &buf)

	f.StartTable([]string{"ID", "NAME"})
	f.Row("1", "test")
	_ = f.EndTable()

	if !bytes.Contains(buf.Bytes(), []byte("ID")) {
		t.Error("output should contain table header")
	}
}

func TestFormatter_Empty(t *testing.T) {
	var out, errOut bytes.Buffer
	ctx := WithMode(context.Background(), Text)
	f := NewFormatter(ctx, &out, &errOut)

	f.Empty("No results found")

	if !bytes.Contains(errOut.Bytes(), []byte("No results found")) {
		t.Error("empty message should be written to stderr")
	}
}

func TestFormatter_Output_Text(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithMode(context.Background(), Text)
	f := NewFormatter(ctx, &buf, &buf)

	data := map[string]string{"name": "test"}
	if err := f.Output(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// In text mode, Output does nothing (returns nil without writing)
	if buf.Len() != 0 {
		t.Errorf("expected no output in text mode, got: %s", buf.String())
	}
}

func TestFormatter_Output_JSONWithQuery(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithMode(context.Background(), JSON)
	ctx = WithQuery(ctx, ".name")
	f := NewFormatter(ctx, &buf, &buf)

	data := map[string]string{"name": "test", "id": "123"}
	if err := f.Output(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte(`"test"`)) {
		t.Errorf("output should contain filtered result, got: %s", buf.String())
	}
}

func TestFormatter_Output_JSONWithTemplate(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithMode(context.Background(), JSON)
	ctx = WithTemplate(ctx, "Name: {{.name}}")
	f := NewFormatter(ctx, &buf, &buf)

	data := map[string]string{"name": "test", "id": "123"}
	if err := f.Output(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "Name: test" {
		t.Errorf("expected 'Name: test', got: %s", buf.String())
	}
}

func TestFormatter_Output_JSONWithQueryAndTemplate(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithMode(context.Background(), JSON)
	ctx = WithQuery(ctx, ".items[0]")
	ctx = WithTemplate(ctx, "First: {{.name}}")
	f := NewFormatter(ctx, &buf, &buf)

	data := []map[string]string{
		{"name": "alice"},
		{"name": "bob"},
	}
	if err := f.Output(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.String() != "First: alice" {
		t.Errorf("expected 'First: alice', got: %s", buf.String())
	}
}

func TestFormatter_StartTable_JSON(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithMode(context.Background(), JSON)
	f := NewFormatter(ctx, &buf, &buf)

	// In JSON mode, StartTable returns false and writes nothing
	result := f.StartTable([]string{"ID", "NAME"})

	if result {
		t.Error("StartTable should return false in JSON mode")
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output in JSON mode, got: %s", buf.String())
	}
}

func TestFormatter_Row_MultipleColumns(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithMode(context.Background(), Text)
	f := NewFormatter(ctx, &buf, &buf)

	f.StartTable([]string{"ID", "NAME", "EMAIL"})
	f.Row("1", "alice", "alice@example.com")
	f.Row("2", "bob", "bob@example.com")
	_ = f.EndTable()

	output := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("alice")) {
		t.Error("output should contain 'alice'")
	}
	if !bytes.Contains(buf.Bytes(), []byte("bob@example.com")) {
		t.Error("output should contain 'bob@example.com'")
	}
	// Check for tabular formatting
	if !bytes.Contains(buf.Bytes(), []byte("ID")) {
		t.Errorf("output should contain header 'ID', got: %s", output)
	}
}
