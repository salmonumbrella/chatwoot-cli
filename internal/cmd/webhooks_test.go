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

func TestWebhooksListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{
			"payload": {
				"webhooks": [
					{"id": 1, "url": "https://example.com/webhook1", "subscriptions": ["message_created", "conversation_created"]},
					{"id": 2, "url": "https://example.com/webhook2", "subscriptions": ["message_updated"]}
				]
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"webhooks", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("webhooks list failed: %v", err)
	}

	if !strings.Contains(output, "https://example.com/webhook1") {
		t.Errorf("output missing webhook URL: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "URL") || !strings.Contains(output, "SUBSCRIPTIONS") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestWebhooksListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{
			"payload": {
				"webhooks": [
					{"id": 1, "url": "https://example.com/webhook1", "subscriptions": ["message_created"]}
				]
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"webhooks", "list", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("webhooks list failed: %v", err)
	}

	webhooks := decodeItems(t, output)
	if len(webhooks) != 1 {
		t.Errorf("expected 1 webhook, got %d", len(webhooks))
	}
}

func TestWebhooksGetCommand(t *testing.T) {
	// GetWebhook actually calls ListWebhooks and filters
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{
			"payload": {
				"webhooks": [
					{"id": 123, "url": "https://example.com/webhook", "subscriptions": ["message_created", "conversation_created"]}
				]
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"webhooks", "get", "123"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("webhooks get failed: %v", err)
	}

	if !strings.Contains(output, "https://example.com/webhook") {
		t.Errorf("output missing webhook URL: %s", output)
	}
	if !strings.Contains(output, "message_created") {
		t.Errorf("output missing subscription: %s", output)
	}
}

func TestWebhooksGetCommand_AcceptsHashAndPrefixedIDs(t *testing.T) {
	// GetWebhook actually calls ListWebhooks and filters
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{
			"payload": {
				"webhooks": [
					{"id": 123, "url": "https://example.com/webhook", "subscriptions": ["message_created"]}
				]
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"webhooks", "get", "#123"}); err != nil {
			t.Fatalf("webhooks get hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output, "https://example.com/webhook") {
		t.Errorf("output missing webhook URL: %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"webhooks", "get", "webhook:123"}); err != nil {
			t.Fatalf("webhooks get prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "https://example.com/webhook") {
		t.Errorf("output missing webhook URL: %s", output2)
	}
}

func TestWebhooksGetCommand_JSON(t *testing.T) {
	// GetWebhook actually calls ListWebhooks and filters
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{
			"payload": {
				"webhooks": [
					{"id": 123, "url": "https://example.com/webhook", "subscriptions": ["message_created"]}
				]
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"webhooks", "get", "123", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("webhooks get failed: %v", err)
	}

	var webhook map[string]any
	if err := json.Unmarshal([]byte(output), &webhook); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if webhook["url"] != "https://example.com/webhook" {
		t.Errorf("expected url 'https://example.com/webhook', got %v", webhook["url"])
	}
}

func TestWebhooksGetCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"webhooks", "get", "abc"})
	if err == nil {
		t.Error("expected error for invalid webhook ID")
	}

	err = Execute(context.Background(), []string{"webhooks", "get", "-1"})
	if err == nil {
		t.Error("expected error for negative webhook ID")
	}
}

func TestWebhooksGetCommand_MissingID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"webhooks", "get"})
	if err == nil {
		t.Error("expected error when webhook ID is missing")
	}
}

func TestWebhooksCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/webhooks", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"webhook": {"id": 456, "url": "https://example.com/new-webhook", "subscriptions": ["message_created"]}}}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"webhooks", "create",
		"--url", "https://example.com/new-webhook",
		"--subscriptions", "message_created",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("webhooks create failed: %v", err)
	}

	if !strings.Contains(output, "Created webhook 456") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["url"] != "https://example.com/new-webhook" {
		t.Errorf("expected url 'https://example.com/new-webhook', got %v", receivedBody["url"])
	}
}

func TestWebhooksCreateCommand_MultipleSubscriptions(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/webhooks", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"webhook": {"id": 789, "url": "https://example.com/webhook", "subscriptions": ["message_created", "conversation_created"]}}}`))
		})

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{
		"webhooks", "create",
		"--url", "https://example.com/webhook",
		"--subscriptions", "message_created,conversation_created",
	})
	if err != nil {
		t.Errorf("webhooks create failed: %v", err)
	}

	subs, ok := receivedBody["subscriptions"].([]any)
	if !ok {
		t.Fatalf("expected subscriptions to be array, got %T", receivedBody["subscriptions"])
	}
	if len(subs) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(subs))
	}
}

func TestWebhooksCreateCommand_SubscriptionsFromStdin(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/webhooks", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"webhook": {"id": 789, "url": "https://example.com/webhook", "subscriptions": ["message_created", "conversation_created"]}}}`))
		})

	setupTestEnvWithHandler(t, handler)

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	_, _ = w.WriteString("message_created\nconversation_created\n")
	_ = w.Close()

	err := Execute(context.Background(), []string{
		"webhooks", "create",
		"--url", "https://example.com/webhook",
		"--subscriptions", "@-",
	})
	if err != nil {
		t.Errorf("webhooks create failed: %v", err)
	}

	subs, ok := receivedBody["subscriptions"].([]any)
	if !ok {
		t.Fatalf("expected subscriptions to be array, got %T", receivedBody["subscriptions"])
	}
	if len(subs) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(subs))
	}
}

func TestWebhooksCreateCommand_SubscriptionsJSON(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/webhooks", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"webhook": {"id": 789, "url": "https://example.com/webhook", "subscriptions": ["message_created", "conversation_created"]}}}`))
		})

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{
		"webhooks", "create",
		"--url", "https://example.com/webhook",
		"--subscriptions", `["message_created","conversation_created"]`,
	})
	if err != nil {
		t.Errorf("webhooks create failed: %v", err)
	}

	subs, ok := receivedBody["subscriptions"].([]any)
	if !ok {
		t.Fatalf("expected subscriptions to be array, got %T", receivedBody["subscriptions"])
	}
	if len(subs) != 2 {
		t.Errorf("expected 2 subscriptions, got %d", len(subs))
	}
}

func TestWebhooksCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{
			"payload": {
				"webhook": {
					"id": 456,
					"url": "https://example.com/webhook",
					"subscriptions": ["message_created"]
				}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"webhooks", "create",
		"--url", "https://example.com/webhook",
		"--subscriptions", "message_created",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("webhooks create failed: %v", err)
	}

	var webhook map[string]any
	if err := json.Unmarshal([]byte(output), &webhook); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestWebhooksCreateCommand_MissingURL(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"webhooks", "create",
		"--subscriptions", "message_created",
	})
	if err == nil {
		t.Error("expected error when URL is missing")
	}
	if !strings.Contains(err.Error(), "--url is required") {
		t.Errorf("expected '--url is required' error, got: %v", err)
	}
}

func TestWebhooksCreateCommand_MissingSubscriptions(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"webhooks", "create",
		"--url", "https://example.com/webhook",
	})
	if err == nil {
		t.Error("expected error when subscriptions is missing")
	}
	if !strings.Contains(err.Error(), "--subscriptions is required") {
		t.Errorf("expected '--subscriptions is required' error, got: %v", err)
	}
}

func TestWebhooksCreateCommand_InvalidURL(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"webhooks", "create",
		"--url", "not-a-valid-url",
		"--subscriptions", "message_created",
	})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestWebhooksUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/webhooks/123", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"webhook": {"id": 123, "url": "https://example.com/updated", "subscriptions": ["message_created"]}}}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"webhooks", "update", "123",
		"--url", "https://example.com/updated",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("webhooks update failed: %v", err)
	}

	if !strings.Contains(output, "Updated webhook 123") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["url"] != "https://example.com/updated" {
		t.Errorf("expected url 'https://example.com/updated', got %v", receivedBody["url"])
	}
}

func TestWebhooksUpdateCommand_Subscriptions(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/webhooks/123", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"webhook": {"id": 123, "url": "https://example.com/webhook", "subscriptions": ["conversation_created"]}}}`))
		})

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{
		"webhooks", "update", "123",
		"--subscriptions", "conversation_created",
	})
	if err != nil {
		t.Errorf("webhooks update failed: %v", err)
	}

	subs, ok := receivedBody["subscriptions"].([]any)
	if !ok {
		t.Fatalf("expected subscriptions to be array, got %T", receivedBody["subscriptions"])
	}
	if len(subs) != 1 || subs[0] != "conversation_created" {
		t.Errorf("expected subscriptions ['conversation_created'], got %v", subs)
	}
}

func TestWebhooksUpdateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/webhooks/123", jsonResponse(200, `{
			"payload": {
				"webhook": {
					"id": 123,
					"url": "https://example.com/updated",
					"subscriptions": ["message_created"]
				}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"webhooks", "update", "123",
		"--url", "https://example.com/updated",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("webhooks update failed: %v", err)
	}

	var webhook map[string]any
	if err := json.Unmarshal([]byte(output), &webhook); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestWebhooksUpdateCommand_NoFlags(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"webhooks", "update", "123"})
	if err == nil {
		t.Error("expected error when no update flags provided")
	}
	if !strings.Contains(err.Error(), "at least one of --url or --subscriptions must be provided") {
		t.Errorf("expected 'at least one' error, got: %v", err)
	}
}

func TestWebhooksUpdateCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"webhooks", "update", "abc", "--url", "https://example.com"})
	if err == nil {
		t.Error("expected error for invalid webhook ID")
	}
}

func TestWebhooksDeleteCommand(t *testing.T) {
	deleteRequested := false
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/webhooks/456", func(w http.ResponseWriter, r *http.Request) {
			deleteRequested = true
			w.WriteHeader(http.StatusOK)
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"webhooks", "delete", "456"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("webhooks delete failed: %v", err)
	}

	if !deleteRequested {
		t.Error("expected DELETE request to be made")
	}

	if !strings.Contains(output, "Deleted webhook 456") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestWebhooksDeleteCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"webhooks", "delete", "abc"})
	if err == nil {
		t.Error("expected error for invalid webhook ID")
	}
}

func TestWebhooksDeleteCommand_MissingID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"webhooks", "delete"})
	if err == nil {
		t.Error("expected error when webhook ID is missing")
	}
}

func TestWebhooksCommand_Alias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{"payload": {"webhooks": []}}`))

	setupTestEnvWithHandler(t, handler)

	// Test using 'webhook' alias instead of 'webhooks'
	err := Execute(context.Background(), []string{"webhook", "list"})
	if err != nil {
		t.Errorf("webhook alias should work: %v", err)
	}

	// Test using 'wh' alias
	err = Execute(context.Background(), []string{"wh", "list"})
	if err != nil {
		t.Errorf("wh alias should work: %v", err)
	}
}

func TestWebhooksCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/webhooks", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"webhooks", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestWebhooksGetCommand_NotFound(t *testing.T) {
	// GetWebhook calls ListWebhooks and filters, so not found is returned client-side
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{"payload": {"webhooks": []}}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"webhooks", "get", "999"})
	if err == nil {
		t.Error("expected error for not found webhook")
	}
}
