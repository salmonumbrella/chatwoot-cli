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

func TestIntegrationsAppsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/integrations/apps", jsonResponse(200, `{
			"payload": [
				{"id": "slack", "name": "Slack", "description": "Slack integration", "enabled": true, "hooks": []},
				{"id": "dialogflow", "name": "Dialogflow", "description": "AI chatbot", "enabled": false, "hooks": []}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"integrations", "apps"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("integrations apps failed: %v", err)
	}

	if !strings.Contains(output, "Slack") {
		t.Errorf("output missing app name: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "ENABLED") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "yes") || !strings.Contains(output, "no") {
		t.Errorf("output missing enabled status: %s", output)
	}
}

func TestIntegrationsAppsCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/integrations/apps", jsonResponse(200, `{
			"payload": [{"id": "slack", "name": "Slack", "description": "Slack integration", "enabled": true, "hooks": []}]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"integrations", "apps", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("integrations apps failed: %v", err)
	}

	_ = decodeItems(t, output)
}

func TestIntegrationsHooksCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/integrations/apps", jsonResponse(200, `{
			"payload": [
				{"id": "slack", "name": "Slack", "hooks": [
					{"id": 1, "app_id": "slack", "inbox_id": 1, "account_id": 1},
					{"id": 2, "app_id": "slack", "inbox_id": 0, "account_id": 1}
				]},
				{"id": "dialogflow", "name": "Dialogflow", "hooks": [
					{"id": 3, "app_id": "dialogflow", "inbox_id": 2, "account_id": 1}
				]}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"integrations", "hooks"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("integrations hooks failed: %v", err)
	}

	if !strings.Contains(output, "slack") {
		t.Errorf("output missing app_id: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "APP_ID") || !strings.Contains(output, "INBOX_ID") {
		t.Errorf("output missing expected headers: %s", output)
	}
	// Hook with inbox_id 0 should show "-"
	if !strings.Contains(output, "-") {
		t.Errorf("output should show '-' for inbox_id 0: %s", output)
	}
}

func TestIntegrationsHooksCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/integrations/apps", jsonResponse(200, `{
			"payload": [
				{"id": "slack", "name": "Slack", "hooks": [{"id": 1, "app_id": "slack", "inbox_id": 1, "account_id": 1}]}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"integrations", "hooks", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("integrations hooks failed: %v", err)
	}

	_ = decodeItems(t, output)
}

func TestIntegrationsHookCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/integrations/hooks", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "app_id": "slack", "inbox_id": 1, "account_id": 1}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"integrations", "hook-create",
		"--app-id", "slack",
		"--inbox-id", "1",
		"--settings", `{"webhook_url":"https://hooks.slack.com/test"}`,
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("integrations hook-create failed: %v", err)
	}

	if !strings.Contains(output, "Created integration hook 1") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["app_id"] != "slack" {
		t.Errorf("expected app_id 'slack', got %v", receivedBody["app_id"])
	}
	if receivedBody["inbox_id"] != float64(1) {
		t.Errorf("expected inbox_id 1, got %v", receivedBody["inbox_id"])
	}
}

func TestIntegrationsHookCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/integrations/hooks", jsonResponse(200, `{
			"id": 1,
			"app_id": "slack",
			"inbox_id": 1,
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"integrations", "hook-create",
		"--app-id", "slack",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("integrations hook-create failed: %v", err)
	}

	var hook map[string]any
	if err := json.Unmarshal([]byte(output), &hook); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestIntegrationsHookCreateCommand_MissingAppID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"integrations", "hook-create",
		"--inbox-id", "1",
	})
	if err == nil {
		t.Error("expected error when app-id is missing")
	}
	if !strings.Contains(err.Error(), "--app-id is required") {
		t.Errorf("expected '--app-id is required' error, got: %v", err)
	}
}

func TestIntegrationsHookCreateCommand_InvalidSettings(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"integrations", "hook-create",
		"--app-id", "slack",
		"--settings", "invalid-json",
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid settings JSON") {
		t.Errorf("expected 'invalid settings JSON' error, got: %v", err)
	}
}

func TestIntegrationsHookUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/integrations/hooks/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "app_id": "slack", "inbox_id": 1, "account_id": 1}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"integrations", "hook-update", "1",
		"--settings", `{"webhook_url":"https://hooks.slack.com/updated"}`,
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("integrations hook-update failed: %v", err)
	}

	if !strings.Contains(output, "Updated integration hook 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestIntegrationsHookUpdateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/integrations/hooks/1", jsonResponse(200, `{
			"id": 1,
			"app_id": "slack",
			"inbox_id": 1,
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"integrations", "hook-update", "1",
		"--settings", `{"key":"value"}`,
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("integrations hook-update failed: %v", err)
	}

	var hook map[string]any
	if err := json.Unmarshal([]byte(output), &hook); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestIntegrationsHookUpdateCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"integrations", "hook-update", "invalid",
		"--settings", `{"key":"value"}`,
	})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "hook ID") {
		t.Errorf("expected 'hook ID' error, got: %v", err)
	}
}

func TestIntegrationsHookUpdateCommand_AcceptsHashAndPrefixedIDs(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/integrations/hooks/1", jsonResponse(200, `{
			"id": 1,
			"app_id": "slack",
			"inbox_id": 1,
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{
			"integrations", "hook-update", "#1",
			"--settings", `{"key":"value"}`,
		}); err != nil {
			t.Fatalf("integrations hook-update hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output, "Updated integration hook 1") {
		t.Errorf("expected success message, got: %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{
			"integrations", "hook-update", "hook:1",
			"--settings", `{"key":"value"}`,
		}); err != nil {
			t.Fatalf("integrations hook-update prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "Updated integration hook 1") {
		t.Errorf("expected success message, got: %s", output2)
	}
}

func TestIntegrationsHookUpdateCommand_InvalidSettings(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"integrations", "hook-update", "1",
		"--settings", "invalid-json",
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid settings JSON") {
		t.Errorf("expected 'invalid settings JSON' error, got: %v", err)
	}
}

func TestIntegrationsHookDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/integrations/hooks/1", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"integrations", "hook-delete", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("integrations hook-delete failed: %v", err)
	}

	if !strings.Contains(output, "Deleted integration hook 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestIntegrationsHookDeleteCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/integrations/hooks/1", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"integrations", "hook-delete", "1", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("integrations hook-delete failed: %v", err)
	}
	// JSON mode should have no output for delete
}

func TestIntegrationsHookDeleteCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"integrations", "hook-delete", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "hook ID") {
		t.Errorf("expected 'hook ID' error, got: %v", err)
	}
}

func TestIntegrationsAppsCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/integrations/apps", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"integrations", "apps"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

// Test the "integration" alias
func TestIntegrationsAppsCommand_IntegrationAlias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/integrations/apps", jsonResponse(200, `{
			"payload": [{"id": "slack", "name": "Slack", "description": "Slack integration", "enabled": true, "hooks": []}]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"integration", "apps"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("integration apps failed: %v", err)
	}

	if !strings.Contains(output, "Slack") {
		t.Errorf("output missing app name: %s", output)
	}
}
