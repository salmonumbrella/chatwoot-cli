// internal/filter/filter_test.go
package filter

import (
	"bytes"
	"testing"
)

func TestApply_EmptyExpression(t *testing.T) {
	data := map[string]interface{}{"name": "test"}
	result, err := Apply(data, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(map[string]interface{})["name"] != "test" {
		t.Error("empty expression should return data unchanged")
	}
}

func TestApply_SelectField(t *testing.T) {
	data := map[string]interface{}{"name": "test", "id": 123}
	result, err := Apply(data, ".name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("expected 'test', got %v", result)
	}
}

func TestApply_FilterArray(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"status": "open"},
		map[string]interface{}{"status": "closed"},
	}
	result, err := Apply(data, `.[] | select(.status == "open")`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["status"] != "open" {
		t.Errorf("expected status 'open', got %v", m["status"])
	}
}

func TestApply_InvalidExpression(t *testing.T) {
	data := map[string]interface{}{"name": "test"}
	_, err := Apply(data, "invalid[[[")
	if err == nil {
		t.Error("expected error for invalid expression")
	}
}

func TestApplyToJSON_ValidJSON(t *testing.T) {
	jsonData := []byte(`{"name": "test", "id": 123}`)
	result, err := ApplyToJSON(jsonData, ".name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(result, []byte(`"test"`)) {
		t.Error("expected JSON output to contain filtered result")
	}
}

func TestApplyToJSON_InvalidJSON(t *testing.T) {
	_, err := ApplyToJSON([]byte(`{invalid}`), ".name")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestApplyToJSON_EmptyExpression(t *testing.T) {
	jsonData := []byte(`{"name": "test"}`)
	result, err := ApplyToJSON(jsonData, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(jsonData, result) {
		t.Errorf("empty expression should return original JSON unchanged")
	}
}
