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

func TestCustomFiltersListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters", jsonResponse(200, `[
			{"id": 1, "name": "Open Conversations", "filter_type": "conversation", "query": {"status": "open"}},
			{"id": 2, "name": "Active Contacts", "filter_type": "contact", "query": {"active": true}}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-filters", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-filters list failed: %v", err)
	}

	if !strings.Contains(output, "Open Conversations") {
		t.Errorf("output missing filter name: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "TYPE") || !strings.Contains(output, "QUERY") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestCustomFiltersListCommand_WithType(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters", func(w http.ResponseWriter, r *http.Request) {
			filterType := r.URL.Query().Get("filter_type")
			if filterType != "conversation" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id": 1, "name": "Open Conversations", "filter_type": "conversation", "query": {"status": "open"}}]`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-filters", "list", "--type", "conversation"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-filters list failed: %v", err)
	}

	if !strings.Contains(output, "Open Conversations") {
		t.Errorf("output missing filter name: %s", output)
	}
}

func TestCustomFiltersListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters", jsonResponse(200, `[
			{"id": 1, "name": "Open Conversations", "filter_type": "conversation", "query": {"status": "open"}}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-filters", "list", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-filters list failed: %v", err)
	}

	filters := decodeItems(t, output)
	if len(filters) != 1 {
		t.Errorf("expected 1 filter, got %d", len(filters))
	}
}

func TestCustomFiltersListCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-filters", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-filters list failed: %v", err)
	}

	if !strings.Contains(output, "No custom filters found") {
		t.Errorf("expected empty message, got: %s", output)
	}
}

func TestCustomFiltersGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters/1", jsonResponse(200, `{
			"id": 1,
			"name": "Open Conversations",
			"filter_type": "conversation",
			"query": {"status": "open", "priority": "high"}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-filters", "get", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-filters get failed: %v", err)
	}

	if !strings.Contains(output, "Open Conversations") {
		t.Errorf("output missing filter name: %s", output)
	}
	if !strings.Contains(output, "conversation") {
		t.Errorf("output missing filter type: %s", output)
	}
	if !strings.Contains(output, "status") || !strings.Contains(output, "open") {
		t.Errorf("output missing query details: %s", output)
	}
}

func TestCustomFiltersGetCommand_AcceptsHashAndPrefixedIDs(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters/1", jsonResponse(200, `{
			"id": 1,
			"name": "Open Conversations",
			"filter_type": "conversation",
			"query": {"status": "open"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-filters", "get", "#1"}); err != nil {
			t.Fatalf("custom-filters get hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output, "Open Conversations") {
		t.Errorf("output missing filter name: %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-filters", "get", "custom-filter:1"}); err != nil {
			t.Fatalf("custom-filters get prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "Open Conversations") {
		t.Errorf("output missing filter name: %s", output2)
	}
}

func TestCustomFiltersGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters/1", jsonResponse(200, `{
			"id": 1,
			"name": "Open Conversations",
			"filter_type": "conversation",
			"query": {"status": "open"}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-filters", "get", "1", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-filters get failed: %v", err)
	}

	var filter map[string]any
	if err := json.Unmarshal([]byte(output), &filter); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if filter["name"] != "Open Conversations" {
		t.Errorf("expected name 'Open Conversations', got %v", filter["name"])
	}
}

func TestCustomFiltersGetCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"custom-filters", "get", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "custom filter ID") {
		t.Errorf("expected 'custom filter ID' error, got: %v", err)
	}
}

func TestCustomFiltersCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/custom_filters", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "New Filter", "filter_type": "conversation", "query": {"status": "open"}}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"custom-filters", "create",
		"--name", "New Filter",
		"--type", "conversation",
		"--query", `{"status":"open"}`,
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-filters create failed: %v", err)
	}

	if !strings.Contains(output, "Created custom filter 1: New Filter") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["name"] != "New Filter" {
		t.Errorf("expected name 'New Filter', got %v", receivedBody["name"])
	}
	if receivedBody["filter_type"] != "conversation" {
		t.Errorf("expected filter_type 'conversation', got %v", receivedBody["filter_type"])
	}
}

func TestCustomFiltersCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/custom_filters", jsonResponse(200, `{
			"id": 1,
			"name": "New Filter",
			"filter_type": "conversation",
			"query": {"status": "open"}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"custom-filters", "create",
		"--name", "New Filter",
		"--type", "conversation",
		"--query", `{"status":"open"}`,
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-filters create failed: %v", err)
	}

	var filter map[string]any
	if err := json.Unmarshal([]byte(output), &filter); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestCustomFiltersCreateCommand_MissingName(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"custom-filters", "create",
		"--type", "conversation",
		"--query", `{"status":"open"}`,
	})
	if err == nil {
		t.Error("expected error when name is missing")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected '--name is required' error, got: %v", err)
	}
}

func TestCustomFiltersCreateCommand_MissingType(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"custom-filters", "create",
		"--name", "Test Filter",
		"--query", `{"status":"open"}`,
	})
	if err == nil {
		t.Error("expected error when type is missing")
	}
	if !strings.Contains(err.Error(), "--type is required") {
		t.Errorf("expected '--type is required' error, got: %v", err)
	}
}

func TestCustomFiltersCreateCommand_MissingQuery(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"custom-filters", "create",
		"--name", "Test Filter",
		"--type", "conversation",
	})
	if err == nil {
		t.Error("expected error when query is missing")
	}
	if !strings.Contains(err.Error(), "--query is required") {
		t.Errorf("expected '--query is required' error, got: %v", err)
	}
}

func TestCustomFiltersCreateCommand_InvalidQuery(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"custom-filters", "create",
		"--name", "Test Filter",
		"--type", "conversation",
		"--query", "invalid-json",
	})
	if err == nil {
		t.Error("expected error for invalid JSON query")
	}
	if !strings.Contains(err.Error(), "invalid query JSON") {
		t.Errorf("expected 'invalid query JSON' error, got: %v", err)
	}
}

func TestCustomFiltersUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/custom_filters/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "Updated Filter", "filter_type": "conversation", "query": {"status": "pending"}}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"custom-filters", "update", "1",
		"--name", "Updated Filter",
		"--query", `{"status":"pending"}`,
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-filters update failed: %v", err)
	}

	if !strings.Contains(output, "Updated custom filter 1: Updated Filter") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["name"] != "Updated Filter" {
		t.Errorf("expected name 'Updated Filter', got %v", receivedBody["name"])
	}
}

func TestCustomFiltersUpdateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/custom_filters/1", jsonResponse(200, `{
			"id": 1,
			"name": "Updated Filter",
			"filter_type": "conversation",
			"query": {"status": "pending"}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"custom-filters", "update", "1",
		"--name", "Updated Filter",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-filters update failed: %v", err)
	}

	var filter map[string]any
	if err := json.Unmarshal([]byte(output), &filter); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestCustomFiltersUpdateCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"custom-filters", "update", "invalid", "--name", "Test"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "custom filter ID") {
		t.Errorf("expected 'custom filter ID' error, got: %v", err)
	}
}

func TestCustomFiltersUpdateCommand_NoChanges(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"custom-filters", "update", "1"})
	if err == nil {
		t.Error("expected error when no changes provided")
	}
	if !strings.Contains(err.Error(), "at least one of --name or --query is required") {
		t.Errorf("expected 'at least one of --name or --query is required' error, got: %v", err)
	}
}

func TestCustomFiltersUpdateCommand_InvalidQuery(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"custom-filters", "update", "1",
		"--query", "invalid-json",
	})
	if err == nil {
		t.Error("expected error for invalid JSON query")
	}
	if !strings.Contains(err.Error(), "invalid query JSON") {
		t.Errorf("expected 'invalid query JSON' error, got: %v", err)
	}
}

func TestCustomFiltersDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/custom_filters/1", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-filters", "delete", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-filters delete failed: %v", err)
	}

	if !strings.Contains(output, "Deleted custom filter 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestCustomFiltersDeleteCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/custom_filters/1", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-filters", "delete", "1", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("custom-filters delete failed: %v", err)
	}
	// JSON mode should have no output for delete
}

func TestCustomFiltersDeleteCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"custom-filters", "delete", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "custom filter ID") {
		t.Errorf("expected 'custom filter ID' error, got: %v", err)
	}
}

func TestCustomFiltersListCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"custom-filters", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

// Test the "filters" alias
func TestCustomFiltersListCommand_FiltersAlias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters", jsonResponse(200, `[
			{"id": 1, "name": "Open Conversations", "filter_type": "conversation", "query": {"status": "open"}}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"filters", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("filters list failed: %v", err)
	}

	if !strings.Contains(output, "Open Conversations") {
		t.Errorf("output missing filter name: %s", output)
	}
}

// Test the "cf" alias
func TestCustomFiltersListCommand_CfAlias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters", jsonResponse(200, `[
			{"id": 1, "name": "Open Conversations", "filter_type": "conversation", "query": {"status": "open"}}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"cf", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("cf list failed: %v", err)
	}

	if !strings.Contains(output, "Open Conversations") {
		t.Errorf("output missing filter name: %s", output)
	}
}
