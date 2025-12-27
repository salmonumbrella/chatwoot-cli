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
