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

func TestApply_ShellEscapedNotEqual(t *testing.T) {
	// Zsh escapes != to \!= even in single quotes
	data := []interface{}{
		map[string]interface{}{"value": nil},
		map[string]interface{}{"value": "test"},
	}
	// Expression as it arrives from zsh: select(.value \!= null)
	result, err := Apply(data, `.[] | select(.value \!= null)`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["value"] != "test" {
		t.Errorf("expected value 'test', got %v", m["value"])
	}
}

func TestNormalizeExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`select(.x \!= null)`, `select(.x != null)`},
		{`select(.x != null)`, `select(.x != null)`},
		{`.[] | select(.a \!= .b)`, `.[] | select(.a != .b)`},
		{`select(.x == "test")`, `select(.x == "test")`},
		{`.it[] | select(.st == "open")`, `.items[] | select(.status == "open")`},
		{`.["st"] | .st`, `.["st"] | .status`},
		{`.St | .st`, `.St | .status`},
		{`.st # .st in comment`, `.status # .st in comment`},
		{`sl(.mty == 1)`, `select(.message_type == 1)`},
		{`sl(.ct | ts("x"; "i"))`, `select(.content | test("x"; "i"))`},
		{`.cu.blk and .cu.mtr`, `.custom_attributes.blacklist and .custom_attributes.membership_tier`},
	}
	for _, tt := range tests {
		got := NormalizeExpression(tt.input)
		if got != tt.expected {
			t.Errorf("NormalizeExpression(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestApply_QueryAliases(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"status": "open"},
			map[string]any{"status": "resolved"},
		},
	}

	result, err := Apply(data, `.it[] | select(.st == "open") | .st`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "open" {
		t.Fatalf("expected open, got %v", result)
	}
}

func TestApply_QueryAliases_DoNotRewriteQuotedBracketLiteral(t *testing.T) {
	data := map[string]any{
		"it":    "literal",
		"items": "canonical",
	}

	result, err := Apply(data, `.["it"]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "literal" {
		t.Fatalf("expected literal key lookup to remain unchanged, got %v", result)
	}
}

func TestApply_QueryAliases_MessageFields(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{
				"message_type": 1,
				"sender":       map[string]any{"name": "Ada"},
			},
		},
	}

	result, err := Apply(data, `.it[] | select(.mty == 1) | .sd.n`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Ada" {
		t.Fatalf("expected Ada, got %v", result)
	}
}

func TestApply_QueryFunctionAliases(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"message_type": 1, "content": "refund pending"},
			map[string]any{"message_type": 0, "content": "hello"},
		},
	}

	result, err := Apply(data, `[.it[] | sl(.mty == 1) | sl(.ct | ts("refund"; "i"))] | length`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != 1 {
		t.Fatalf("expected 1, got %v", result)
	}
}

func TestApplyFromJSON_EmptyExpression(t *testing.T) {
	jsonData := []byte(`{"name": "test", "id": 42}`)
	result, err := ApplyFromJSON(jsonData, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["name"] != "test" {
		t.Errorf("expected name=test, got %v", m["name"])
	}
}

func TestApplyFromJSON_WithExpression(t *testing.T) {
	jsonData := []byte(`{"name": "test", "id": 42}`)
	result, err := ApplyFromJSON(jsonData, ".name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("expected 'test', got %v", result)
	}
}

func TestApplyFromJSON_InvalidJSON(t *testing.T) {
	_, err := ApplyFromJSON([]byte(`{invalid}`), ".name")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestApplyWith_CustomNormalizer(t *testing.T) {
	data := map[string]any{"st": "o", "status": "open"}
	// Identity normalizer — no changes, acts like ApplyLiteral
	result, err := applyWith(data, ".st", func(s string) string { return s })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "o" {
		t.Fatalf("expected 'o', got %v", result)
	}
}

func TestApply_QueryAliases_CustomAttributes(t *testing.T) {
	data := map[string]any{
		"custom_attributes": map[string]any{
			"blacklist":       true,
			"membership_tier": "gold",
		},
	}

	result, err := Apply(data, `.cu | {blk: .blk, mtr: .mtr}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if out["blk"] != true {
		t.Fatalf("expected blk=true, got %v", out["blk"])
	}
	if out["mtr"] != "gold" {
		t.Fatalf("expected mtr=gold, got %v", out["mtr"])
	}
}

func TestApply_RootArrayQueryFallsBackToItems(t *testing.T) {
	data := map[string]any{
		"items": []any{
			map[string]any{"inbox": map[string]any{"id": 11}},
			map[string]any{"inbox": map[string]any{"id": 22}},
		},
		"meta": map[string]any{"total": 2},
	}

	result, err := Apply(data, `.[].inbox.id`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	values, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any result, got %T (%v)", result, result)
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 results, got %d (%v)", len(values), values)
	}
	if values[0] != 11 || values[1] != 22 {
		t.Fatalf("unexpected values: %v", values)
	}
}

func TestApply_RootArrayQueryWithoutItemsStillErrors(t *testing.T) {
	data := map[string]any{
		"payload": []any{map[string]any{"id": 1}},
	}

	_, err := Apply(data, `.[].id`)
	if err == nil {
		t.Fatal("expected error for root-array query on non-items object")
	}
}
