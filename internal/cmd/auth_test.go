package cmd

import (
	"testing"
)

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		// Short tokens (< 8 chars) - should match actual length
		{
			name:     "empty token",
			token:    "",
			expected: "",
		},
		{
			name:     "1 character token",
			token:    "a",
			expected: "*",
		},
		{
			name:     "2 character token",
			token:    "ab",
			expected: "**",
		},
		{
			name:     "3 character token",
			token:    "abc",
			expected: "***",
		},
		{
			name:     "4 character token",
			token:    "abcd",
			expected: "****",
		},
		{
			name:     "5 character token",
			token:    "abcde",
			expected: "*****",
		},
		{
			name:     "6 character token",
			token:    "abcdef",
			expected: "******",
		},
		{
			name:     "7 character token",
			token:    "abcdefg",
			expected: "*******",
		},
		// Boundary case - exactly 8 characters
		{
			name:     "8 character token",
			token:    "abcd1234",
			expected: "abcd1234",
		},
		// Normal tokens (>= 8 chars) - show first 4 and last 4
		{
			name:     "9 character token",
			token:    "abcd12345",
			expected: "abcd*2345",
		},
		{
			name:     "10 character token",
			token:    "abcdefghij",
			expected: "abcd**ghij",
		},
		{
			name:     "16 character token",
			token:    "1234567890abcdef",
			expected: "1234********cdef",
		},
		{
			name:     "32 character token (typical API token length)",
			token:    "abcdefghijklmnopqrstuvwxyz123456",
			expected: "abcd************************3456",
		},
		{
			name:     "64 character token",
			token:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expected: "aaaa********************************************************aaaa",
		},
		// Real-world-like tokens
		{
			name:     "typical API token format",
			token:    "sk-1234567890abcdefghij",
			expected: "sk-1***************ghij",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskToken(tt.token)
			if result != tt.expected {
				t.Errorf("maskToken(%q) = %q, want %q", tt.token, result, tt.expected)
			}

			// Verify the masked token has the same length as the original
			if len(result) != len(tt.token) {
				t.Errorf("maskToken(%q) length = %d, want %d (original length)", tt.token, len(result), len(tt.token))
			}
		})
	}
}

func TestMaskToken_LengthPreservation(t *testing.T) {
	// Property-based test: verify length is always preserved
	testTokens := []string{
		"",
		"a",
		"ab",
		"abc",
		"abcd",
		"abcde",
		"abcdef",
		"abcdefg",
		"abcdefgh",
		"abcdefghi",
		"abcdefghij",
		"this-is-a-very-long-token-with-many-characters-1234567890",
	}

	for _, token := range testTokens {
		t.Run("length_"+string(rune(len(token))), func(t *testing.T) {
			masked := maskToken(token)
			if len(masked) != len(token) {
				t.Errorf("Length mismatch for token of length %d: got %d", len(token), len(masked))
			}
		})
	}
}

func TestMaskToken_NoLeakage(t *testing.T) {
	// Verify that short tokens don't leak length information by having fixed output
	// This test documents the fix: tokens < 8 chars now correctly show their actual length
	shortTokens := map[string]int{
		"a":       1,
		"ab":      2,
		"abc":     3,
		"abcd":    4,
		"abcde":   5,
		"abcdef":  6,
		"abcdefg": 7,
	}

	for token, expectedLen := range shortTokens {
		masked := maskToken(token)
		if len(masked) != expectedLen {
			t.Errorf("Token %q (length %d) masked to %q (length %d), should preserve length",
				token, expectedLen, masked, len(masked))
		}
	}
}

func TestMaskToken_LongTokenFormat(t *testing.T) {
	// Verify that long tokens (>= 8 chars) show first 4 and last 4 characters
	tests := []struct {
		token       string
		wantPrefix  string
		wantSuffix  string
		wantMidMask int // number of asterisks in the middle
	}{
		{
			token:       "abcd1234",
			wantPrefix:  "abcd",
			wantSuffix:  "1234",
			wantMidMask: 0,
		},
		{
			token:       "abcdefghij",
			wantPrefix:  "abcd",
			wantSuffix:  "ghij",
			wantMidMask: 2,
		},
		{
			token:       "1234567890abcdef",
			wantPrefix:  "1234",
			wantSuffix:  "cdef",
			wantMidMask: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			masked := maskToken(tt.token)

			if len(masked) < 8 {
				t.Fatalf("Expected long token format, got short format: %q", masked)
			}

			prefix := masked[:4]
			suffix := masked[len(masked)-4:]

			if prefix != tt.wantPrefix {
				t.Errorf("Prefix = %q, want %q", prefix, tt.wantPrefix)
			}

			if suffix != tt.wantSuffix {
				t.Errorf("Suffix = %q, want %q", suffix, tt.wantSuffix)
			}

			// Check middle is all asterisks
			middle := masked[4 : len(masked)-4]
			expectedMiddle := ""
			for i := 0; i < tt.wantMidMask; i++ {
				expectedMiddle += "*"
			}
			if middle != expectedMiddle {
				t.Errorf("Middle = %q, want %q (%d asterisks)", middle, expectedMiddle, tt.wantMidMask)
			}
		})
	}
}
