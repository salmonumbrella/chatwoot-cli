package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderCustomerInfo(t *testing.T) {
	var out bytes.Buffer
	renderCustomerInfo(&out, map[string]any{
		"customer_name":        "Jane Doe",
		"membership_tier_name": "Gold",
		"total_spend":          1234.0,
	})
	text := out.String()
	checks := []string{"Customer: Jane Doe", "Member: Gold", "Total Spend: $1234"}
	for _, want := range checks {
		if !strings.Contains(text, want) {
			t.Fatalf("renderCustomerInfo missing %q in %q", want, text)
		}
	}
}

func TestRenderCustomerInfo_Empty(t *testing.T) {
	var out bytes.Buffer
	renderCustomerInfo(&out, map[string]any{})
	if out.Len() != 0 {
		t.Fatalf("expected no output for empty customer info, got %q", out.String())
	}
}
