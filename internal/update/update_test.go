// internal/update/update_test.go
package update

import (
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{"0.1.0", "v0.1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeVersion(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
