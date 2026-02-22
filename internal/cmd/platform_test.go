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

func setupPlatformTestEnv(t *testing.T, handler *routeHandler) {
	t.Helper()
	env := setupTestEnvWithHandler(t, handler)

	t.Setenv("CHATWOOT_BASE_URL", env.server.URL)
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-platform-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")
}

// Platform Accounts Tests

func TestPlatformAccountsCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/platform/api/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "Test Account"}`))
		})

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "accounts", "create", "--name", "Test Account"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform accounts create failed: %v", err)
	}

	if !strings.Contains(output, "Created account 1") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["name"] != "Test Account" {
		t.Errorf("expected name 'Test Account', got %v", receivedBody["name"])
	}
}

func TestPlatformAccountsCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/platform/api/v1/accounts", jsonResponse(200, `{"id": 1, "name": "Test Account"}`))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "accounts", "create", "--name", "Test Account", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform accounts create failed: %v", err)
	}

	var account map[string]any
	if err := json.Unmarshal([]byte(output), &account); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestPlatformAccountsCreateCommand_MissingName(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{"platform", "accounts", "create"})
	if err == nil {
		t.Error("expected error when name is missing")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected '--name is required' error, got: %v", err)
	}
}

func TestPlatformAccountsCreateCommand_CustomAttributes(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/platform/api/v1/accounts", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "Test Account"}`))
		})

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"platform", "accounts", "create",
		"--name", "Test Account",
		"--custom-attributes", `{"key": "value"}`,
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("platform accounts create failed: %v", err)
	}

	attrs, ok := receivedBody["custom_attributes"].(map[string]any)
	if !ok || attrs["key"] != "value" {
		t.Errorf("expected custom_attributes with key=value, got %v", receivedBody["custom_attributes"])
	}
}

func TestPlatformAccountsCreateCommand_InvalidCustomAttributes(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{
		"platform", "accounts", "create",
		"--name", "Test Account",
		"--custom-attributes", "invalid-json",
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid custom-attributes JSON") {
		t.Errorf("expected 'invalid custom-attributes JSON' error, got: %v", err)
	}
}

func TestPlatformAccountsCreateCommand_InvalidLimits(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{
		"platform", "accounts", "create",
		"--name", "Test Account",
		"--limits", "invalid-json",
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid limits JSON") {
		t.Errorf("expected 'invalid limits JSON' error, got: %v", err)
	}
}

func TestPlatformAccountsGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/platform/api/v1/accounts/1", jsonResponse(200, `{"id": 1, "name": "Test Account"}`))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "accounts", "get", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform accounts get failed: %v", err)
	}

	if !strings.Contains(output, "Account 1: Test Account") {
		t.Errorf("expected account info, got: %s", output)
	}
}

func TestPlatformAccountsGetCommand_AcceptsHashAndPrefixedIDs(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/platform/api/v1/accounts/1", jsonResponse(200, `{"id": 1, "name": "Test Account"}`))

	setupPlatformTestEnv(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"platform", "accounts", "get", "#1"}); err != nil {
			t.Fatalf("platform accounts get hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output, "Account 1: Test Account") {
		t.Errorf("expected account info, got: %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"platform", "accounts", "get", "account:1"}); err != nil {
			t.Fatalf("platform accounts get prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "Account 1: Test Account") {
		t.Errorf("expected account info, got: %s", output2)
	}
}

func TestPlatformAccountsGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/platform/api/v1/accounts/1", jsonResponse(200, `{"id": 1, "name": "Test Account"}`))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "accounts", "get", "1", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform accounts get failed: %v", err)
	}

	var account map[string]any
	if err := json.Unmarshal([]byte(output), &account); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestPlatformAccountsGetCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{"platform", "accounts", "get", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "account ID") {
		t.Errorf("expected 'account ID' error, got: %v", err)
	}
}

func TestPlatformAccountsUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/platform/api/v1/accounts/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "Updated Account"}`))
		})

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "accounts", "update", "1", "--name", "Updated Account"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform accounts update failed: %v", err)
	}

	if !strings.Contains(output, "Updated account 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPlatformAccountsUpdateCommand_NoFields(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{"platform", "accounts", "update", "1"})
	if err == nil {
		t.Error("expected error when no fields provided")
	}
	if !strings.Contains(err.Error(), "at least one field") {
		t.Errorf("expected 'at least one field' error, got: %v", err)
	}
}

func TestPlatformAccountsDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/platform/api/v1/accounts/1", jsonResponse(200, ``))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "accounts", "delete", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform accounts delete failed: %v", err)
	}

	if !strings.Contains(output, "Deleted account 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

// Platform Users Tests

func TestPlatformUsersCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/platform/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "Test User", "email": "test@example.com"}`))
		})

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"platform", "users", "create",
		"--name", "Test User",
		"--email", "test@example.com",
		"--password", "password123",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform users create failed: %v", err)
	}

	if !strings.Contains(output, "Created user 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPlatformUsersCreateCommand_MissingFields(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{"platform", "users", "create"})
	if err == nil {
		t.Error("expected error when required fields are missing")
	}
	if !strings.Contains(err.Error(), "--name, --email, and --password are required") {
		t.Errorf("expected required fields error, got: %v", err)
	}
}

func TestPlatformUsersGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/platform/api/v1/users/1", jsonResponse(200, `{"id": 1, "name": "Test User", "email": "test@example.com"}`))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "users", "get", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform users get failed: %v", err)
	}

	if !strings.Contains(output, "User 1: test@example.com") {
		t.Errorf("expected user info, got: %s", output)
	}
}

func TestPlatformUsersGetCommand_AcceptsHashAndPrefixedIDs(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/platform/api/v1/users/1", jsonResponse(200, `{"id": 1, "name": "Test User", "email": "test@example.com"}`))

	setupPlatformTestEnv(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"platform", "users", "get", "#1"}); err != nil {
			t.Fatalf("platform users get hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output, "User 1: test@example.com") {
		t.Errorf("expected user info, got: %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"platform", "users", "get", "user:1"}); err != nil {
			t.Fatalf("platform users get prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "User 1: test@example.com") {
		t.Errorf("expected user info, got: %s", output2)
	}
}

func TestPlatformUsersUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/platform/api/v1/users/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "Updated User", "email": "test@example.com"}`))
		})

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "users", "update", "1", "--name", "Updated User"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform users update failed: %v", err)
	}

	if !strings.Contains(output, "Updated user 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPlatformUsersUpdateCommand_NoFields(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{"platform", "users", "update", "1"})
	if err == nil {
		t.Error("expected error when no fields provided")
	}
	if !strings.Contains(err.Error(), "at least one field") {
		t.Errorf("expected 'at least one field' error, got: %v", err)
	}
}

func TestPlatformUsersDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/platform/api/v1/users/1", jsonResponse(200, ``))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "users", "delete", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform users delete failed: %v", err)
	}

	if !strings.Contains(output, "Deleted user 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPlatformUsersLoginCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/platform/api/v1/users/1/login", jsonResponse(200, `{"url": "https://example.com/sso/login?token=abc"}`))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "users", "login", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform users login failed: %v", err)
	}

	if !strings.Contains(output, "https://example.com/sso/login") {
		t.Errorf("expected login URL, got: %s", output)
	}
}

// Platform Account Users Tests

func TestPlatformAccountUsersListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/platform/api/v1/accounts/1/account_users", jsonResponse(200, `[
			{"id": 1, "account_id": 1, "user_id": 10, "role": "administrator"},
			{"id": 2, "account_id": 1, "user_id": 20, "role": "agent"}
		]`))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "account-users", "list", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform account-users list failed: %v", err)
	}

	if !strings.Contains(output, "ID") || !strings.Contains(output, "USER") || !strings.Contains(output, "ROLE") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "administrator") {
		t.Errorf("output missing role: %s", output)
	}
}

func TestPlatformAccountUsersCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/platform/api/v1/accounts/1/account_users", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "account_id": 1, "user_id": 10, "role": "agent"}`))
		})

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"platform", "account-users", "create", "1",
		"--user-id", "10",
		"--role", "agent",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform account-users create failed: %v", err)
	}

	if !strings.Contains(output, "Added user 10 to account 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPlatformAccountUsersCreateCommand_MissingUserID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{"platform", "account-users", "create", "1", "--role", "agent"})
	if err == nil {
		t.Error("expected error when user-id is missing")
	}
	if !strings.Contains(err.Error(), "--user-id is required") {
		t.Errorf("expected '--user-id is required' error, got: %v", err)
	}
}

func TestPlatformAccountUsersCreateCommand_MissingRole(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{"platform", "account-users", "create", "1", "--user-id", "10"})
	if err == nil {
		t.Error("expected error when role is missing")
	}
	if !strings.Contains(err.Error(), "--role is required") {
		t.Errorf("expected '--role is required' error, got: %v", err)
	}
}

func TestPlatformAccountUsersDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/platform/api/v1/accounts/1/account_users", jsonResponse(200, ``))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"platform", "account-users", "delete", "1",
		"--user-id", "10",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform account-users delete failed: %v", err)
	}

	if !strings.Contains(output, "Removed user 10 from account 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPlatformAccountUsersDeleteCommand_MissingUserID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{"platform", "account-users", "delete", "1"})
	if err == nil {
		t.Error("expected error when user-id is missing")
	}
	if !strings.Contains(err.Error(), "--user-id is required") {
		t.Errorf("expected '--user-id is required' error, got: %v", err)
	}
}

// Platform Agent Bots Tests

func TestPlatformAgentBotsListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/platform/api/v1/agent_bots", jsonResponse(200, `[
			{"id": 1, "name": "Bot 1", "bot_type": "webhook", "outgoing_url": "https://example.com/webhook"},
			{"id": 2, "name": "Bot 2", "bot_type": "webhook", "outgoing_url": ""}
		]`))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "agent-bots", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform agent-bots list failed: %v", err)
	}

	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "TYPE") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "Bot 1") {
		t.Errorf("output missing bot name: %s", output)
	}
}

func TestPlatformAgentBotsGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/platform/api/v1/agent_bots/1", jsonResponse(200, `{
			"id": 1,
			"name": "Test Bot",
			"description": "A test bot",
			"bot_type": "webhook",
			"outgoing_url": "https://example.com/webhook"
		}`))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "agent-bots", "get", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform agent-bots get failed: %v", err)
	}

	if !strings.Contains(output, "Agent Bot 1: Test Bot") {
		t.Errorf("expected bot info, got: %s", output)
	}
	if !strings.Contains(output, "Description: A test bot") {
		t.Errorf("output missing description: %s", output)
	}
}

func TestPlatformAgentBotsGetCommand_MinimalInfo(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/platform/api/v1/agent_bots/1", jsonResponse(200, `{
			"id": 1,
			"name": "Test Bot"
		}`))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "agent-bots", "get", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform agent-bots get failed: %v", err)
	}

	if !strings.Contains(output, "Agent Bot 1: Test Bot") {
		t.Errorf("expected bot info, got: %s", output)
	}
	// Description and URL should not appear when empty
	if strings.Contains(output, "Description:") {
		t.Errorf("output should not show empty description: %s", output)
	}
}

func TestPlatformAgentBotsCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/platform/api/v1/agent_bots", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "New Bot"}`))
		})

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"platform", "agent-bots", "create",
		"--name", "New Bot",
		"--description", "A new bot",
		"--outgoing-url", "https://example.com/hook",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform agent-bots create failed: %v", err)
	}

	if !strings.Contains(output, "Created agent bot 1") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["name"] != "New Bot" {
		t.Errorf("expected name 'New Bot', got %v", receivedBody["name"])
	}
}

func TestPlatformAgentBotsCreateCommand_MissingName(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{"platform", "agent-bots", "create"})
	if err == nil {
		t.Error("expected error when name is missing")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected '--name is required' error, got: %v", err)
	}
}

func TestPlatformAgentBotsUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/platform/api/v1/agent_bots/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "Updated Bot"}`))
		})

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "agent-bots", "update", "1", "--name", "Updated Bot"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform agent-bots update failed: %v", err)
	}

	if !strings.Contains(output, "Updated agent bot 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPlatformAgentBotsUpdateCommand_NoFields(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{"platform", "agent-bots", "update", "1"})
	if err == nil {
		t.Error("expected error when no fields provided")
	}
	if !strings.Contains(err.Error(), "at least one field") {
		t.Errorf("expected 'at least one field' error, got: %v", err)
	}
}

func TestPlatformAgentBotsDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/platform/api/v1/agent_bots/1", jsonResponse(200, ``))

	setupPlatformTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"platform", "agent-bots", "delete", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("platform agent-bots delete failed: %v", err)
	}

	if !strings.Contains(output, "Deleted agent bot 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPlatformAgentBotsDeleteCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_PLATFORM_TOKEN", "test-token")

	err := Execute(context.Background(), []string{"platform", "agent-bots", "delete", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "bot ID") {
		t.Errorf("expected 'bot ID' error, got: %v", err)
	}
}

// API Error Tests

func TestPlatformAccountsGetCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/platform/api/v1/accounts/1", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupPlatformTestEnv(t, handler)

	err := Execute(context.Background(), []string{"platform", "accounts", "get", "1"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}
