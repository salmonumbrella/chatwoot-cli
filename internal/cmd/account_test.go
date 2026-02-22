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

func TestAccountGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1", jsonResponse(200, `{
			"id": 1,
			"name": "Test Account",
			"locale": "en",
			"domain": "test.chatwoot.com"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"account", "get"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("account get failed: %v", err)
	}

	if !strings.Contains(output, "Test Account") {
		t.Errorf("output missing account name: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "LOCALE") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestAccountGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1", jsonResponse(200, `{
			"id": 1,
			"name": "Test Account",
			"locale": "en",
			"domain": "test.chatwoot.com"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"account", "get", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("account get failed: %v", err)
	}

	var account map[string]any
	if err := json.Unmarshal([]byte(output), &account); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if account["name"] != "Test Account" {
		t.Errorf("expected name 'Test Account', got %v", account["name"])
	}
}

func TestAccountUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "Updated Account", "locale": "en"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"account", "update",
		"--name", "Updated Account",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("account update failed: %v", err)
	}

	if !strings.Contains(output, "Updated account 1") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["name"] != "Updated Account" {
		t.Errorf("expected name 'Updated Account', got %v", receivedBody["name"])
	}
}

func TestAccountUpdateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1", jsonResponse(200, `{
			"id": 1,
			"name": "Updated Account",
			"locale": "en"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"account", "update",
		"--name", "Updated Account",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("account update failed: %v", err)
	}

	var account map[string]any
	if err := json.Unmarshal([]byte(output), &account); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestAccountUpdateCommand_MissingName(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"account", "update"})
	if err == nil {
		t.Error("expected error when name is missing")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected '--name is required' error, got: %v", err)
	}
}

func TestAccountCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"account", "get"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}
