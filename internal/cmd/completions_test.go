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
	"time"
)

func decodeCompletionItems(t *testing.T, output string) []CompletionItem {
	t.Helper()
	var resp struct {
		Items []CompletionItem `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	return resp.Items
}

func TestCompletionsInboxesCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Email", "channel_type": "Channel::Email"},
				{"id": 2, "name": "Website", "channel_type": "Channel::WebWidget"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"completions", "inboxes"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("completions inboxes failed: %v", err)
	}

	// Verify output contains expected data
	if !strings.Contains(output, "1") || !strings.Contains(output, "Email") {
		t.Errorf("output missing inbox 1 data: %s", output)
	}
	if !strings.Contains(output, "2") || !strings.Contains(output, "Website") {
		t.Errorf("output missing inbox 2 data: %s", output)
	}
}

func TestCompletionsInboxesCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Email", "channel_type": "Channel::Email"},
				{"id": 2, "name": "Website", "channel_type": "Channel::WebWidget"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"completions", "inboxes", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("completions inboxes failed: %v", err)
	}

	items := decodeCompletionItems(t, output)

	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	// Check first item
	if items[0].Value != "1" {
		t.Errorf("expected value '1', got %s", items[0].Value)
	}
	if items[0].Label != "Email" {
		t.Errorf("expected label 'Email', got %s", items[0].Label)
	}
	if items[0].Description != "Channel::Email" {
		t.Errorf("expected description 'Channel::Email', got %s", items[0].Description)
	}
}

func TestCompletionsAgentsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 1, "name": "John Doe", "email": "john@example.com", "role": "agent"},
			{"id": 2, "name": "Jane Smith", "email": "jane@example.com", "role": "admin"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"completions", "agents"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("completions agents failed: %v", err)
	}

	if !strings.Contains(output, "1") || !strings.Contains(output, "John Doe") {
		t.Errorf("output missing agent 1 data: %s", output)
	}
	if !strings.Contains(output, "2") || !strings.Contains(output, "Jane Smith") {
		t.Errorf("output missing agent 2 data: %s", output)
	}
}

func TestCompletionsAgentsCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 1, "name": "John Doe", "email": "john@example.com", "role": "agent"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"completions", "agents", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("completions agents failed: %v", err)
	}

	items := decodeCompletionItems(t, output)

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	if items[0].Value != "1" {
		t.Errorf("expected value '1', got %s", items[0].Value)
	}
	if items[0].Label != "John Doe" {
		t.Errorf("expected label 'John Doe', got %s", items[0].Label)
	}
	if items[0].Description != "john@example.com" {
		t.Errorf("expected description 'john@example.com', got %s", items[0].Description)
	}
}

func TestCompletionsLabelsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{
			"payload": [
				{"id": 1, "title": "bug", "description": "Bug reports", "color": "#FF0000"},
				{"id": 2, "title": "feature", "description": "Feature requests", "color": "#00FF00"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"completions", "labels"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("completions labels failed: %v", err)
	}

	if !strings.Contains(output, "bug") {
		t.Errorf("output missing label 'bug': %s", output)
	}
	if !strings.Contains(output, "feature") {
		t.Errorf("output missing label 'feature': %s", output)
	}
}

func TestCompletionsLabelsCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{
			"payload": [
				{"id": 1, "title": "bug", "description": "Bug reports", "color": "#FF0000"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"completions", "labels", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("completions labels failed: %v", err)
	}

	items := decodeCompletionItems(t, output)

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	// Labels use title as both value and label
	if items[0].Value != "bug" {
		t.Errorf("expected value 'bug', got %s", items[0].Value)
	}
	if items[0].Label != "bug" {
		t.Errorf("expected label 'bug', got %s", items[0].Label)
	}
	if items[0].Description != "Bug reports" {
		t.Errorf("expected description 'Bug reports', got %s", items[0].Description)
	}
}

func TestCompletionsTeamsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams", jsonResponse(200, `[
			{"id": 1, "name": "Support", "description": "Customer support team"},
			{"id": 2, "name": "Sales", "description": "Sales team"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"completions", "teams"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("completions teams failed: %v", err)
	}

	if !strings.Contains(output, "1") || !strings.Contains(output, "Support") {
		t.Errorf("output missing team 1 data: %s", output)
	}
	if !strings.Contains(output, "2") || !strings.Contains(output, "Sales") {
		t.Errorf("output missing team 2 data: %s", output)
	}
}

func TestCompletionsTeamsCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams", jsonResponse(200, `[
			{"id": 1, "name": "Support", "description": "Customer support team"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"completions", "teams", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("completions teams failed: %v", err)
	}

	items := decodeCompletionItems(t, output)

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	if items[0].Value != "1" {
		t.Errorf("expected value '1', got %s", items[0].Value)
	}
	if items[0].Label != "Support" {
		t.Errorf("expected label 'Support', got %s", items[0].Label)
	}
	if items[0].Description != "Customer support team" {
		t.Errorf("expected description 'Customer support team', got %s", items[0].Description)
	}
}

func TestCompletionsStatusesCommand(t *testing.T) {
	// No need to set up a handler since statuses is a static command
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"completions", "statuses"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("completions statuses failed: %v", err)
	}

	// Check all status values are present
	if !strings.Contains(output, "open") {
		t.Errorf("output missing status 'open': %s", output)
	}
	if !strings.Contains(output, "resolved") {
		t.Errorf("output missing status 'resolved': %s", output)
	}
	if !strings.Contains(output, "pending") {
		t.Errorf("output missing status 'pending': %s", output)
	}
	if !strings.Contains(output, "snoozed") {
		t.Errorf("output missing status 'snoozed': %s", output)
	}
}

func TestCompletionsStatusesCommand_JSON(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"completions", "statuses", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("completions statuses failed: %v", err)
	}

	items := decodeCompletionItems(t, output)

	if len(items) != 4 {
		t.Errorf("expected 4 items, got %d", len(items))
	}

	// Verify the static status values
	expectedStatuses := map[string]bool{
		"open":     false,
		"resolved": false,
		"pending":  false,
		"snoozed":  false,
	}

	for _, item := range items {
		if _, ok := expectedStatuses[item.Value]; ok {
			expectedStatuses[item.Value] = true
		}
	}

	for status, found := range expectedStatuses {
		if !found {
			t.Errorf("expected status '%s' not found in output", status)
		}
	}
}

func TestCompletionsInboxesCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{"payload": []}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"completions", "inboxes", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("completions inboxes failed: %v", err)
	}

	items := decodeCompletionItems(t, output)

	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestCompletionsCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"completions", "inboxes"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestCompletionsCommand_Unauthorized(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(401, `{"error": "Unauthorized"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"completions", "agents"})
	if err == nil {
		t.Error("expected error for unauthorized request")
	}
}

func TestCompletionsCache_ReusesItems(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("CHATWOOT_COMPLETIONS_CACHE_DIR", cacheDir)

	var calls int
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"payload": [{"id": 1, "name": "Inbox", "channel_type": "web"}]}`))
		})

	setupTestEnvWithHandler(t, handler)

	if err := Execute(context.Background(), []string{"completions", "inboxes"}); err != nil {
		t.Fatalf("completions inboxes failed: %v", err)
	}
	if err := Execute(context.Background(), []string{"completions", "inboxes"}); err != nil {
		t.Fatalf("completions inboxes failed: %v", err)
	}

	if calls != 1 {
		t.Fatalf("expected 1 API call due to cache, got %d", calls)
	}
}

func TestCompletionsCache_DisabledEnv(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("CHATWOOT_COMPLETIONS_CACHE_DIR", cacheDir)
	t.Setenv("CHATWOOT_COMPLETIONS_NO_CACHE", "1")

	var calls int
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"payload": [{"id": 1, "name": "Inbox", "channel_type": "web"}]}`))
		})

	setupTestEnvWithHandler(t, handler)

	if err := Execute(context.Background(), []string{"completions", "inboxes"}); err != nil {
		t.Fatalf("completions inboxes failed: %v", err)
	}
	if err := Execute(context.Background(), []string{"completions", "inboxes"}); err != nil {
		t.Fatalf("completions inboxes failed: %v", err)
	}

	if calls != 2 {
		t.Fatalf("expected 2 API calls with cache disabled, got %d", calls)
	}
}

func TestCompletionsCache_Expired(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("CHATWOOT_COMPLETIONS_CACHE_DIR", cacheDir)

	var calls int
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"payload": [{"id": 1, "name": "Inbox", "channel_type": "web"}]}`))
		})

	setupTestEnvWithHandler(t, handler)

	if err := Execute(context.Background(), []string{"completions", "inboxes"}); err != nil {
		t.Fatalf("completions inboxes failed: %v", err)
	}

	client, err := getClient()
	if err != nil {
		t.Fatalf("failed to get client: %v", err)
	}
	path, err := completionsCachePath(client, "inboxes")
	if err != nil {
		t.Fatalf("failed to get cache path: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read cache file: %v", err)
	}
	var cached completionsCache
	if err := json.Unmarshal(data, &cached); err != nil {
		t.Fatalf("failed to decode cache: %v", err)
	}
	cached.CachedAt = time.Now().Add(-2 * completionsCacheTTL)
	updated, err := json.Marshal(cached)
	if err != nil {
		t.Fatalf("failed to marshal cache: %v", err)
	}
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}

	if err := Execute(context.Background(), []string{"completions", "inboxes"}); err != nil {
		t.Fatalf("completions inboxes failed: %v", err)
	}

	if calls != 2 {
		t.Fatalf("expected 2 API calls after cache expiry, got %d", calls)
	}
}
