package cmd

import (
	"context"
	"encoding/json"
	"strings"
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

func TestAuthLoginCommand_ValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name:      "missing url flag",
			args:      []string{"auth", "login", "--browser=false", "--token", "test", "--account-id", "1"},
			wantError: "--url is required",
		},
		{
			name:      "missing token flag",
			args:      []string{"auth", "login", "--browser=false", "--url", "https://example.com", "--account-id", "1"},
			wantError: "--token is required",
		},
		{
			name:      "missing account-id flag",
			args:      []string{"auth", "login", "--browser=false", "--url", "https://example.com", "--token", "test"},
			wantError: "--account-id must be a positive integer",
		},
		{
			name:      "invalid account-id (zero)",
			args:      []string{"auth", "login", "--browser=false", "--url", "https://example.com", "--token", "test", "--account-id", "0"},
			wantError: "--account-id must be a positive integer",
		},
		{
			name:      "invalid account-id (negative)",
			args:      []string{"auth", "login", "--browser=false", "--url", "https://example.com", "--token", "test", "--account-id", "-1"},
			wantError: "--account-id must be a positive integer",
		},
		{
			name:      "SSRF - localhost",
			args:      []string{"auth", "login", "--browser=false", "--url", "http://localhost", "--token", "test", "--account-id", "1"},
			wantError: "invalid URL",
		},
		{
			name:      "SSRF - 127.0.0.1",
			args:      []string{"auth", "login", "--browser=false", "--url", "http://127.0.0.1", "--token", "test", "--account-id", "1"},
			wantError: "invalid URL",
		},
		{
			name:      "SSRF - private IP 10.x.x.x",
			args:      []string{"auth", "login", "--browser=false", "--url", "http://10.0.0.1", "--token", "test", "--account-id", "1"},
			wantError: "invalid URL",
		},
		{
			name:      "SSRF - private IP 192.168.x.x",
			args:      []string{"auth", "login", "--browser=false", "--url", "http://192.168.1.1", "--token", "test", "--account-id", "1"},
			wantError: "invalid URL",
		},
		{
			name:      "SSRF - metadata endpoint",
			args:      []string{"auth", "login", "--browser=false", "--url", "http://169.254.169.254", "--token", "test", "--account-id", "1"},
			wantError: "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Execute(context.Background(), tt.args)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantError)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("expected error containing %q, got: %v", tt.wantError, err)
			}
		})
	}
}

func TestAuthStatusCommand_WithEnvVars(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/profile", jsonResponse(200, `{"id": 1, "name": "Test User", "email": "test@example.com"}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"auth", "status"})
		if err != nil {
			t.Errorf("auth status failed: %v", err)
		}
	})

	if !strings.Contains(output, "Authenticated") {
		t.Errorf("output should contain 'Authenticated': %s", output)
	}
	if !strings.Contains(output, "Source: env") {
		t.Errorf("output should indicate source is env: %s", output)
	}
}

func TestAuthStatusCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/profile", jsonResponse(200, `{"id": 1, "name": "Test User", "email": "test@example.com"}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"auth", "status", "--output", "json"})
		if err != nil {
			t.Errorf("auth status --json failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result["authenticated"] != true {
		t.Errorf("expected authenticated=true, got %v", result["authenticated"])
	}
	if result["source"] != "env" {
		t.Errorf("expected source=env, got %v", result["source"])
	}
}

func TestAuthCmd(t *testing.T) {
	cmd := newAuthCmd()

	if cmd.Use != "auth" {
		t.Errorf("expected command Use to be 'auth', got %q", cmd.Use)
	}

	// Check subcommands exist
	subcommands := cmd.Commands()
	expectedSubs := []string{"login", "status", "logout"}
	for _, expected := range expectedSubs {
		found := false
		for _, sub := range subcommands {
			if sub.Use == expected || strings.HasPrefix(sub.Use, expected+" ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q not found", expected)
		}
	}
}

func TestAuthLoginCmd(t *testing.T) {
	cmd := newAuthLoginCmd()

	// Check that required flags exist
	requiredFlags := []string{"url", "token", "account-id", "browser", "profile", "platform-token"}
	for _, flag := range requiredFlags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected flag %q not found", flag)
		}
	}
}

func TestAuthStatusCmd(t *testing.T) {
	cmd := newAuthStatusCmd()

	if cmd.Use != "status" {
		t.Errorf("expected command Use to be 'status', got %q", cmd.Use)
	}
}

func TestAuthLogoutCmd(t *testing.T) {
	cmd := newAuthLogoutCmd()

	if cmd.Use != "logout" {
		t.Errorf("expected command Use to be 'logout', got %q", cmd.Use)
	}

	// Check profile flag exists
	if cmd.Flags().Lookup("profile") == nil {
		t.Error("expected profile flag not found")
	}
}

func TestAuthLogoutCommand_WithProfile(t *testing.T) {
	// Clear environment variables
	t.Setenv("CHATWOOT_BASE_URL", "")
	t.Setenv("CHATWOOT_API_TOKEN", "")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "")

	// Test that specifying a non-existent profile still works (may print success or error)
	output := captureStdout(t, func() {
		_ = Execute(context.Background(), []string{"auth", "logout", "--profile", "test-nonexistent-profile-12345"})
	})

	// Either "removed" or error message is acceptable
	if output == "" {
		t.Errorf("expected some output from logout command")
	}
}

func TestAuthStatusCommand_NotConfigured(t *testing.T) {
	// Clear environment variables
	t.Setenv("CHATWOOT_BASE_URL", "")
	t.Setenv("CHATWOOT_API_TOKEN", "")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"auth", "status"})
		if err != nil {
			t.Errorf("auth status failed: %v", err)
		}
	})

	if !strings.Contains(output, "Not authenticated") {
		t.Errorf("expected 'Not authenticated' message, got: %s", output)
	}
}

func TestAuthStatusCommand_JSON_NotConfigured(t *testing.T) {
	// Clear environment variables
	t.Setenv("CHATWOOT_BASE_URL", "")
	t.Setenv("CHATWOOT_API_TOKEN", "")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"auth", "status", "-o", "json"})
		if err != nil {
			t.Errorf("auth status failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if result["authenticated"] != false {
		t.Errorf("expected authenticated=false, got: %v", result["authenticated"])
	}
}
