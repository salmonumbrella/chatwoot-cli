package cmd

import "testing"

func TestFormatPosition(t *testing.T) {
	tests := []struct {
		name     string
		position int
		total    int
		expected string
	}{
		{"first of one", 1, 1, "[1/1]"},
		{"first of ten", 1, 10, "[1/10]"},
		{"fifth of ten", 5, 10, "[5/10]"},
		{"last of ten", 10, 10, "[10/10]"},
		{"large numbers", 100, 500, "[100/500]"},
		{"zero position", 0, 10, "[0/10]"},
		{"zero total", 1, 0, "[1/0]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPosition(tt.position, tt.total)
			if result != tt.expected {
				t.Errorf("formatPosition(%d, %d) = %q, want %q", tt.position, tt.total, result, tt.expected)
			}
		})
	}
}

func TestFormatMessageCount(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		expected string
	}{
		{"zero count", 0, "-"},
		{"single message", 1, "[1 msgs]"},
		{"multiple messages", 5, "[5 msgs]"},
		{"large count", 100, "[100 msgs]"},
		{"very large count", 9999, "[9999 msgs]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMessageCount(tt.count)
			if result != tt.expected {
				t.Errorf("formatMessageCount(%d) = %q, want %q", tt.count, result, tt.expected)
			}
		})
	}
}
