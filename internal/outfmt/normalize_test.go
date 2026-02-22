package outfmt

import (
	"encoding/json"
	"testing"
)

func TestNormalizeJSONOutput_NilSlice(t *testing.T) {
	// A nil []string slice should produce {"items": []} not {"items": null}.
	var nilSlice []string
	result := normalizeJSONOutput(nilSlice)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	items, ok := parsed["items"]
	if !ok {
		t.Fatal("expected items key in output")
	}
	if items == nil {
		t.Fatal("items should be [] not null for nil slice input")
	}
	arr, ok := items.([]any)
	if !ok {
		t.Fatalf("items should be an array, got %T", items)
	}
	if len(arr) != 0 {
		t.Fatalf("items should be empty array, got %v", arr)
	}
}

func TestNormalizeJSONOutput_EmptySlice(t *testing.T) {
	emptySlice := []string{}
	result := normalizeJSONOutput(emptySlice)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	items := parsed["items"]
	arr, ok := items.([]any)
	if !ok {
		t.Fatalf("items should be an array, got %T", items)
	}
	if len(arr) != 0 {
		t.Fatalf("items should be empty array, got %v", arr)
	}
}

func TestNormalizeJSONOutput_PopulatedSlice(t *testing.T) {
	slice := []string{"a", "b"}
	result := normalizeJSONOutput(slice)

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	items := parsed["items"]
	arr, ok := items.([]any)
	if !ok {
		t.Fatalf("items should be an array, got %T", items)
	}
	if len(arr) != 2 {
		t.Fatalf("items should have 2 elements, got %d", len(arr))
	}
}

func TestNormalizeJSONOutput_NilInput(t *testing.T) {
	result := normalizeJSONOutput(nil)
	if result != nil {
		t.Fatalf("expected nil for nil input, got %v", result)
	}
}

func TestNormalizeJSONOutput_Map(t *testing.T) {
	m := map[string]any{"key": "value"}
	result := normalizeJSONOutput(m)
	// Maps should pass through unchanged
	if _, ok := result.(map[string]any); !ok {
		t.Fatalf("map input should pass through, got %T", result)
	}
}
