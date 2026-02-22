package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestStatusCommand(t *testing.T) {
	// Clear any existing env vars for clean test state
	// Use a non-existent profile to ensure keychain lookup returns nothing
	clearEnv := func(t *testing.T) {
		t.Helper()
		t.Setenv("CHATWOOT_BASE_URL", "")
		t.Setenv("CHATWOOT_API_TOKEN", "")
		t.Setenv("CHATWOOT_ACCOUNT_ID", "")
		t.Setenv("CHATWOOT_PROFILE", "__test_nonexistent_profile__")
	}

	t.Run("shows unauthenticated status when no credentials", func(t *testing.T) {
		clearEnv(t)

		output := captureStdout(t, func() {
			_ = Execute(context.Background(), []string{"status"})
		})

		if !strings.Contains(output, "Authenticated:") {
			t.Errorf("expected 'Authenticated:' in output, got: %s", output)
		}
		if !strings.Contains(output, "CLI Version:") {
			t.Errorf("expected 'CLI Version:' in output, got: %s", output)
		}
		if !strings.Contains(output, "Platform:") {
			t.Errorf("expected 'Platform:' in output, got: %s", output)
		}
	})

	t.Run("shows authenticated status with env credentials", func(t *testing.T) {
		t.Setenv("CHATWOOT_BASE_URL", "https://chatwoot.example.com")
		t.Setenv("CHATWOOT_API_TOKEN", "abcd1234efgh5678")
		t.Setenv("CHATWOOT_ACCOUNT_ID", "42")

		output := captureStdout(t, func() {
			_ = Execute(context.Background(), []string{"status"})
		})

		if !strings.Contains(output, "https://chatwoot.example.com") {
			t.Errorf("expected base URL in output, got: %s", output)
		}
		if !strings.Contains(output, "42") {
			t.Errorf("expected account ID in output, got: %s", output)
		}
		if !strings.Contains(output, "abcd********5678") {
			t.Errorf("expected masked token in output, got: %s", output)
		}
		if !strings.Contains(output, "environment") {
			t.Errorf("expected 'environment' config source in output, got: %s", output)
		}
	})

	t.Run("JSON output with env credentials", func(t *testing.T) {
		t.Setenv("CHATWOOT_BASE_URL", "https://chatwoot.example.com")
		t.Setenv("CHATWOOT_API_TOKEN", "abcd1234efgh5678")
		t.Setenv("CHATWOOT_ACCOUNT_ID", "42")

		output := captureStdout(t, func() {
			_ = Execute(context.Background(), []string{"status", "--output", "json"})
		})

		var info StatusInfo
		if err := json.Unmarshal([]byte(output), &info); err != nil {
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
		}

		if !info.Authenticated {
			t.Error("expected authenticated to be true")
		}
		if info.BaseURL != "https://chatwoot.example.com" {
			t.Errorf("expected base_url 'https://chatwoot.example.com', got: %s", info.BaseURL)
		}
		if info.AccountID != 42 {
			t.Errorf("expected account_id 42, got: %d", info.AccountID)
		}
		if info.TokenPreview != "abcd********5678" {
			t.Errorf("expected token_preview 'abcd********5678', got: %s", info.TokenPreview)
		}
		if info.ConfigSource != "environment" {
			t.Errorf("expected config_source 'environment', got: %s", info.ConfigSource)
		}
		if info.CLIVersion == "" {
			t.Error("expected cli_version to be set")
		}
		if info.GoVersion == "" {
			t.Error("expected go_version to be set")
		}
		if info.Platform == "" {
			t.Error("expected platform to be set")
		}
	})

	t.Run("JSON output when unauthenticated", func(t *testing.T) {
		clearEnv(t)

		output := captureStdout(t, func() {
			_ = Execute(context.Background(), []string{"status", "--output", "json"})
		})

		var info StatusInfo
		if err := json.Unmarshal([]byte(output), &info); err != nil {
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
		}

		if info.Authenticated {
			t.Error("expected authenticated to be false")
		}
		if info.BaseURL != "" {
			t.Errorf("expected empty base_url, got: %s", info.BaseURL)
		}
		if info.CLIVersion == "" {
			t.Error("expected cli_version to be set even when unauthenticated")
		}
	})

	t.Run("--check flag exits with error when unauthenticated", func(t *testing.T) {
		clearEnv(t)

		// Capture stderr too
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		err := Execute(context.Background(), []string{"status", "--check"})

		_ = w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)

		if err == nil {
			t.Error("expected error when not authenticated with --check flag")
		}
		if !strings.Contains(buf.String(), "not authenticated") {
			t.Errorf("expected 'not authenticated' error message, got: %s", buf.String())
		}
	})

	t.Run("--check flag succeeds when authenticated", func(t *testing.T) {
		t.Setenv("CHATWOOT_BASE_URL", "https://chatwoot.example.com")
		t.Setenv("CHATWOOT_API_TOKEN", "abcd1234efgh5678")
		t.Setenv("CHATWOOT_ACCOUNT_ID", "42")

		output := captureStdout(t, func() {
			err := Execute(context.Background(), []string{"status", "--check"})
			if err != nil {
				t.Errorf("unexpected error with --check flag when authenticated: %v", err)
			}
		})

		if !strings.Contains(output, "authenticated") {
			t.Errorf("expected 'authenticated' in output, got: %s", output)
		}
	})

	t.Run("--check flag with JSON output when authenticated", func(t *testing.T) {
		t.Setenv("CHATWOOT_BASE_URL", "https://chatwoot.example.com")
		t.Setenv("CHATWOOT_API_TOKEN", "abcd1234efgh5678")
		t.Setenv("CHATWOOT_ACCOUNT_ID", "42")

		output := captureStdout(t, func() {
			err := Execute(context.Background(), []string{"status", "--check", "--output", "json"})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})

		var info StatusInfo
		if err := json.Unmarshal([]byte(output), &info); err != nil {
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
		}

		if !info.Authenticated {
			t.Error("expected authenticated to be true")
		}
	})
}

func TestGetConfigSource(t *testing.T) {
	t.Run("returns environment when all env vars set", func(t *testing.T) {
		t.Setenv("CHATWOOT_BASE_URL", "https://example.com")
		t.Setenv("CHATWOOT_API_TOKEN", "token")
		t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
		t.Setenv("CHATWOOT_PROFILE", "")

		source := getConfigSource()
		if source != "environment" {
			t.Errorf("expected 'environment', got: %s", source)
		}
	})

	t.Run("returns environment (profile) when CHATWOOT_PROFILE set", func(t *testing.T) {
		t.Setenv("CHATWOOT_BASE_URL", "")
		t.Setenv("CHATWOOT_API_TOKEN", "")
		t.Setenv("CHATWOOT_ACCOUNT_ID", "")
		t.Setenv("CHATWOOT_PROFILE", "work")

		source := getConfigSource()
		if source != "environment (profile)" {
			t.Errorf("expected 'environment (profile)', got: %s", source)
		}
	})

	t.Run("returns keychain when no env vars set", func(t *testing.T) {
		t.Setenv("CHATWOOT_BASE_URL", "")
		t.Setenv("CHATWOOT_API_TOKEN", "")
		t.Setenv("CHATWOOT_ACCOUNT_ID", "")
		t.Setenv("CHATWOOT_PROFILE", "")

		source := getConfigSource()
		if source != "keychain" {
			t.Errorf("expected 'keychain', got: %s", source)
		}
	})

	t.Run("returns keychain when only partial env vars set", func(t *testing.T) {
		t.Setenv("CHATWOOT_BASE_URL", "https://example.com")
		t.Setenv("CHATWOOT_API_TOKEN", "")
		t.Setenv("CHATWOOT_ACCOUNT_ID", "")
		t.Setenv("CHATWOOT_PROFILE", "")

		source := getConfigSource()
		if source != "keychain" {
			t.Errorf("expected 'keychain', got: %s", source)
		}
	})
}

func TestStatusWithPing(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/health", jsonResponse(http.StatusOK, `{"status":"woot"}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"status", "--ping"})
		if err != nil {
			t.Errorf("status --ping failed: %v", err)
		}
	})

	if !strings.Contains(output, "reachable") {
		t.Errorf("expected 'reachable' in output, got: %s", output)
	}
}

func TestStatusWithPing_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/health", jsonResponse(http.StatusOK, `{"status":"woot"}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"status", "--ping", "-o", "json"})
		if err != nil {
			t.Errorf("status --ping JSON failed: %v", err)
		}
	})

	var info map[string]any
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
	if info["server_reachable"] != true {
		t.Errorf("expected server_reachable true, got %v", info["server_reachable"])
	}
}
