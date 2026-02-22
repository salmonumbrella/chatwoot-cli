package filter

import "testing"

func TestApplyLiteral_SkipsAliasNormalization(t *testing.T) {
	// "st" is a query alias for "status" in normal mode.
	// In literal mode, it should NOT be expanded.
	data := map[string]any{
		"st":     "o",
		"status": "should-not-match",
	}
	result, err := ApplyLiteral(data, ".st")
	if err != nil {
		t.Fatalf("ApplyLiteral failed: %v", err)
	}
	if result != "o" {
		t.Fatalf("expected 'o', got %v (alias was expanded)", result)
	}
}

func TestApplyLiteral_FixesShellEscapes(t *testing.T) {
	data := map[string]any{"a": 1, "b": nil}
	result, err := ApplyLiteral(data, `[.a, .b] | map(select(. \!= null))`)
	if err != nil {
		t.Fatalf("ApplyLiteral failed: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	// gojq preserves int when input is int (not JSON-unmarshaled float64)
	if len(arr) != 1 || arr[0] != 1 {
		t.Fatalf("expected [1], got %v", arr)
	}
}

func TestApplyFromJSONLiteral(t *testing.T) {
	jsonData := []byte(`{"st": "o", "ib": 48}`)
	result, err := ApplyFromJSONLiteral(jsonData, "{st: .st, ib: .ib}")
	if err != nil {
		t.Fatalf("ApplyFromJSONLiteral failed: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["st"] != "o" {
		t.Fatalf("expected st=o, got %v", m["st"])
	}
	if m["ib"] != float64(48) {
		t.Fatalf("expected ib=48, got %v", m["ib"])
	}
}
