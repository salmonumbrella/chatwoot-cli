package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestSearchCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com"},
				{"id": 5, "name": "John Smith", "email": "jsmith@test.com"}
			],
			"meta": {"count": 2}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 1}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "john"})
		if err != nil {
			t.Errorf("search failed: %v", err)
		}
	})

	// Check unified output contains contacts
	if !strings.Contains(output, "John Doe") {
		t.Errorf("output missing 'John Doe': %s", output)
	}
	if !strings.Contains(output, "John Smith") {
		t.Errorf("output missing 'John Smith': %s", output)
	}
	if !strings.Contains(output, "[contact]") {
		t.Errorf("output missing contact type indicator: %s", output)
	}

	// Check unified output contains conversation
	if !strings.Contains(output, "#100") {
		t.Errorf("output missing conversation ID: %s", output)
	}
	if !strings.Contains(output, "[conv]") {
		t.Errorf("output missing conversation type indicator: %s", output)
	}

	// Check totals at bottom
	if !strings.Contains(output, "2 contacts") {
		t.Errorf("output missing contacts total: %s", output)
	}
	if !strings.Contains(output, "1 conversations") {
		t.Errorf("output missing conversations total: %s", output)
	}
}

func TestSearchCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com"}
			],
			"meta": {"count": 1}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 1}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "john", "--output", "json"})
		if err != nil {
			t.Errorf("search --json failed: %v", err)
		}
	})

	// Parse JSON output
	var results SearchResults
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if results.Query != "john" {
		t.Errorf("expected query 'john', got %q", results.Query)
	}

	if len(results.Contacts) != 1 {
		t.Errorf("expected 1 contact, got %d", len(results.Contacts))
	}

	if len(results.Conversations) != 1 {
		t.Errorf("expected 1 conversation, got %d", len(results.Conversations))
	}

	if results.Summary["contacts"] != 1 {
		t.Errorf("expected contacts summary 1, got %d", results.Summary["contacts"])
	}

	if results.Summary["conversations"] != 1 {
		t.Errorf("expected conversations summary 1, got %d", results.Summary["conversations"])
	}
}

func TestSearchCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com", "last_activity_at": 1700000100}
			],
			"meta": {"count": 1}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 48, "last_activity_at": 1700000200, "meta": {"sender": {"name": "John Doe"}}}
				],
				"meta": {"count": 1, "total_pages": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "john", "--light"})
		if err != nil {
			t.Fatalf("search --light failed: %v", err)
		}
	})

	var payload struct {
		Query   string `json:"q"`
		Results []struct {
			Type   string `json:"type"`
			ID     int    `json:"id"`
			Status string `json:"st"`
			Inbox  int    `json:"ib"`
		} `json:"rs"`
		Summary map[string]int `json:"sm"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light search output: %v\noutput: %s", err, output)
	}
	if payload.Query != "john" {
		t.Fatalf("expected query john, got %q", payload.Query)
	}
	if len(payload.Results) != 2 {
		t.Fatalf("expected 2 light results, got %d", len(payload.Results))
	}
	// Most recent result should be the conversation (last_activity_at 1700000200).
	if payload.Results[0].Type != "conversation" || payload.Results[0].ID != 100 {
		t.Fatalf("expected first result conversation #100, got %#v", payload.Results[0])
	}
	if payload.Summary["contacts"] != 1 || payload.Summary["conversations"] != 1 {
		t.Fatalf("unexpected summary: %#v", payload.Summary)
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		t.Fatalf("failed to parse raw payload: %v", err)
	}
	if _, ok := raw["contacts"]; ok {
		t.Fatal("light payload should not include full contacts array")
	}
	if _, ok := raw["conversations"]; ok {
		t.Fatal("light payload should not include full conversations array")
	}
}

func TestSearchCommand_LightJSONLiteralQuery(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com", "last_activity_at": 1700000100}
			],
			"meta": {"count": 1}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 48, "last_activity_at": 1700000200, "meta": {"sender": {"name": "John Doe"}}}
				],
				"meta": {"count": 1, "total_pages": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"search", "john", "--light", "-o", "json", "--jq", ".rs[0].st",
		})
		if err != nil {
			t.Fatalf("search john --light -o json --jq .rs[0].st failed: %v", err)
		}
	})

	got := strings.TrimSpace(output)
	if got != `"o"` {
		t.Fatalf("expected jq output %q for short-key light status, got %q", `"o"`, got)
	}
}

func TestSearchCommand_Best_EmitID(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com"}
			],
			"meta": {"count": 1}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 1, "last_activity_at": 2000}
				],
				"meta": {"count": 1, "total_pages": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "john", "--best", "--emit", "id"})
		if err != nil {
			t.Errorf("search --best --emit id failed: %v", err)
		}
	})
	if strings.TrimSpace(output) != "100" {
		t.Fatalf("expected best ID 100, got %q", output)
	}
}

func TestSearchCommand_Best_AgentEnvelope(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com"}
			],
			"meta": {"count": 1}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 1, "last_activity_at": 2000}
				],
				"meta": {"count": 1, "total_pages": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "john", "--best", "-o", "agent"})
		if err != nil {
			t.Errorf("search --best -o agent failed: %v", err)
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if payload["kind"] != "search.best" {
		t.Fatalf("expected kind search.best, got %#v", payload["kind"])
	}
	item, ok := payload["item"].(map[string]any)
	if !ok {
		t.Fatalf("expected item object, got %#v", payload["item"])
	}
	if item["type"] != "conversation" {
		t.Fatalf("expected type conversation, got %#v", item["type"])
	}
	if int(item["id"].(float64)) != 100 {
		t.Fatalf("expected id 100, got %#v", item["id"])
	}
}

func TestSearchCommand_EmitRequiresBest(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [{"id": 1, "name": "John Doe", "email": "john@example.com"}],
			"meta": {"count": 1}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {"payload": [], "meta": {"count": 0}}
		}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"search", "john", "--emit", "id"})
	if err == nil {
		t.Fatal("expected error when --emit is used without --best")
	}
	if !strings.Contains(err.Error(), "--emit requires --best") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSearchCommand_TypeFilter_ContactsOnly(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com"}
			],
			"meta": {"count": 1}
		}`))
	// Note: No conversations endpoint registered - should not be called

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "john", "--type", "contacts"})
		if err != nil {
			t.Errorf("search --type contacts failed: %v", err)
		}
	})

	// Unified output shows results sorted by activity, with totals at the bottom
	if !strings.Contains(output, "[contact]") {
		t.Errorf("output missing contact result: %s", output)
	}
	if !strings.Contains(output, "1 contacts") {
		t.Errorf("output missing contacts total: %s", output)
	}
	if strings.Contains(output, "[conv]") {
		t.Errorf("output should not contain conversations when filtering by contacts: %s", output)
	}
}

func TestSearchCommand_SelectRequiresInteractive(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com"}
			],
			"meta": {"count": 1}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 1}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"search", "john", "--select", "--no-input"})
	if err == nil {
		t.Error("expected error when --select is used without interactive input")
	}
}

func TestSearchCommand_SelectInteractive(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com"}
			],
			"meta": {"count": 1}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 1}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)
	t.Setenv("CHATWOOT_FORCE_INTERACTIVE", "true")

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.Write([]byte("1\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "john", "--select"})
		if err != nil {
			t.Errorf("search --select failed: %v", err)
		}
	})

	if !strings.Contains(output, "Contact #1") {
		t.Errorf("expected selected contact details, got: %s", output)
	}
}

func TestSearchCommand_SelectJSONOutput(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com"}
			],
			"meta": {"count": 1}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 1}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)
	t.Setenv("CHATWOOT_FORCE_INTERACTIVE", "true")

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.Write([]byte("1\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "john", "--select", "--output", "json"})
		if err != nil {
			t.Errorf("search --select --json failed: %v", err)
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if payload["type"] != "contact" {
		t.Errorf("expected type contact, got %v", payload["type"])
	}
	item, ok := payload["item"].(map[string]any)
	if !ok {
		t.Fatalf("expected item object, got %v", payload["item"])
	}
	if item["id"] != float64(1) {
		t.Errorf("expected selected item id 1, got %v", item["id"])
	}
}

func TestSearchCommand_SelectJSONRaw(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com"}
			],
			"meta": {"count": 1}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 1}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)
	t.Setenv("CHATWOOT_FORCE_INTERACTIVE", "true")

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.Write([]byte("1\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "john", "--select", "--select-raw", "--output", "json"})
		if err != nil {
			t.Errorf("search --select --select-raw failed: %v", err)
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if payload["id"] != float64(1) {
		t.Errorf("expected selected item id 1, got %v", payload["id"])
	}
}

func TestSearchCommand_TypeFilter_ConversationsOnly(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 1}
				],
				"meta": {"count": 1}
			}
		}`))
	// Note: No contacts endpoint registered - should not be called

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "support", "--type", "conversations"})
		if err != nil {
			t.Errorf("search --type conversations failed: %v", err)
		}
	})

	// Unified output shows results sorted by activity, with totals at the bottom
	if !strings.Contains(output, "[conv]") {
		t.Errorf("output missing conversation result: %s", output)
	}
	if !strings.Contains(output, "1 conversations") {
		t.Errorf("output missing conversations total: %s", output)
	}
	if strings.Contains(output, "[contact]") {
		t.Errorf("output should not contain contacts when filtering by conversations: %s", output)
	}
}

func TestSearchCommand_InvalidType(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"search", "john", "--type", "invalid"})
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestSearchCommand_MissingQuery(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"search"})
	if err == nil {
		t.Error("expected error for missing query argument")
	}
}

func TestSearchCommand_Limit(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com"},
				{"id": 2, "name": "John Smith", "email": "jsmith@test.com"},
				{"id": 3, "name": "John Jones", "email": "jjones@test.com"}
			],
			"meta": {"count": 3}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "john", "--type", "contacts", "--limit", "2", "--output", "json"})
		if err != nil {
			t.Errorf("search --limit failed: %v", err)
		}
	})

	var results SearchResults
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if len(results.Contacts) != 2 {
		t.Errorf("expected 2 contacts (limited), got %d", len(results.Contacts))
	}
}

func TestSearchCommand_NoResults(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [],
			"meta": {"count": 0}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [],
				"meta": {"count": 0}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "nonexistent"})
		if err != nil {
			t.Errorf("search failed: %v", err)
		}
	})

	if !strings.Contains(output, "No results found") {
		t.Errorf("output should indicate no results: %s", output)
	}
}

func TestSearchCommand_ContactsError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(500, `{"error": "internal error"}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [],
				"meta": {"count": 0}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"search", "john"})
	if err == nil {
		t.Error("expected error when contacts search fails")
	}
}

func TestSearchCommand_ContextCancellation(t *testing.T) {
	// Track how many requests were made to each endpoint
	var contactsRequests, conversationsRequests int
	var mu sync.Mutex

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			contactsRequests++
			mu.Unlock()

			// Add delay to simulate network latency - this gives context cancellation time to be detected
			time.Sleep(5 * time.Millisecond)

			// Simulate API with many pages
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{
				"payload": [{"id": 1, "name": "John Doe", "email": "john@example.com"}],
				"meta": {"count": 1000, "current_page": 1, "total_pages": 100}
			}`))
		}).
		On("GET", "/api/v1/accounts/1/conversations", func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			conversationsRequests++
			mu.Unlock()

			// Add delay to simulate network latency
			time.Sleep(5 * time.Millisecond)

			// Simulate API with many pages
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{
				"data": {
					"payload": [{"id": 100, "status": "open", "inbox_id": 1}],
					"meta": {"count": 1000, "current_page": 1, "total_pages": 100}
				}
			}`))
		})

	setupTestEnvWithHandler(t, handler)

	// Create a context that we'll cancel after a few requests
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after allowing a few requests to go through
	go func() {
		time.Sleep(25 * time.Millisecond)
		cancel()
	}()

	// Run the search with a high limit to ensure pagination would occur
	_ = Execute(ctx, []string{"search", "john", "--limit", "10000"})

	// Wait a bit for any in-flight requests to complete
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	contactsReqs := contactsRequests
	conversationsReqs := conversationsRequests
	mu.Unlock()

	// Without context cancellation check in the loop, we'd see 100 requests per type.
	// With cancellation, after ~25ms (allowing ~5 requests at 5ms each), we should stop.
	// Allow some margin for timing variance, but should be well under 100.
	if contactsReqs >= 20 || conversationsReqs >= 20 {
		t.Errorf("context cancellation did not stop pagination early: contacts=%d, conversations=%d requests (expected <20 each)",
			contactsReqs, conversationsReqs)
	}
}

func TestSearchIncludeSnippet(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [],
			"meta": {"count": 0}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 1},
					{"id": 200, "status": "pending", "inbox_id": 2}
				],
				"meta": {"count": 2}
			}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/100/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Hello there, I really need a refund for my order #12345 from last week because the item was damaged", "message_type": 0, "created_at": 1700000000},
				{"id": 2, "content": "Thanks for your help", "message_type": 1, "created_at": 1700000100}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/200/messages", jsonResponse(200, `{
			"payload": [
				{"id": 3, "content": "Can I get a refund?", "message_type": 0, "created_at": 1700000200}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "refund", "--type", "conversations", "--include-snippet", "--output", "json"})
		if err != nil {
			t.Fatalf("search --include-snippet failed: %v", err)
		}
	})

	var result struct {
		Query         string             `json:"query"`
		Conversations []struct{ ID int } `json:"conversations"`
		Snippets      map[string]struct {
			MessageID int    `json:"message_id"`
			Content   string `json:"content"`
			CreatedAt int64  `json:"created_at"`
		} `json:"snippets"`
		Summary map[string]int `json:"summary"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	// Check that we got 2 conversations
	if result.Summary["conversations"] != 2 {
		t.Errorf("expected 2 conversations, got %d", result.Summary["conversations"])
	}

	// Check that snippets map exists and has entries
	if result.Snippets == nil {
		t.Fatal("expected snippets map in output, got nil")
	}
	if len(result.Snippets) != 2 {
		t.Errorf("expected 2 snippets, got %d", len(result.Snippets))
	}

	// Check snippet for conversation 100
	snippet100, ok := result.Snippets["100"]
	if !ok {
		t.Fatal("expected snippet for conversation 100")
	}
	if snippet100.MessageID != 1 {
		t.Errorf("expected message_id 1 for conversation 100, got %d", snippet100.MessageID)
	}
	if !strings.Contains(snippet100.Content, "refund") {
		t.Errorf("expected snippet to contain 'refund', got %q", snippet100.Content)
	}
	// Verify truncation with ellipsis (message is long enough to truncate)
	if !strings.HasPrefix(snippet100.Content, "...") {
		t.Errorf("expected snippet to start with '...', got %q", snippet100.Content)
	}

	// Check snippet for conversation 200
	snippet200, ok := result.Snippets["200"]
	if !ok {
		t.Fatal("expected snippet for conversation 200")
	}
	if snippet200.MessageID != 3 {
		t.Errorf("expected message_id 3 for conversation 200, got %d", snippet200.MessageID)
	}
	if !strings.Contains(snippet200.Content, "refund") {
		t.Errorf("expected snippet to contain 'refund', got %q", snippet200.Content)
	}
}

func TestExtractSnippet_UTF8Safety(t *testing.T) {
	tests := []struct {
		name     string
		messages []api.Message
		query    string
		wantOK   bool
		check    func(t *testing.T, snippet SnippetInfo)
	}{
		{
			name: "Chinese characters",
			messages: []api.Message{
				{ID: 1, Content: "这是一条包含关键字的消息，我们需要测试多字节字符的处理", CreatedAt: 1700000000},
			},
			query:  "关键字",
			wantOK: true,
			check: func(t *testing.T, snippet SnippetInfo) {
				// Ensure query is found in snippet
				if !strings.Contains(snippet.Content, "关键字") {
					t.Errorf("expected snippet to contain query '关键字', got %q", snippet.Content)
				}
				// Ensure no corrupted characters (UTF-8 replacement character)
				if strings.Contains(snippet.Content, "\ufffd") {
					t.Errorf("snippet contains replacement character (corrupted UTF-8): %q", snippet.Content)
				}
			},
		},
		{
			name: "emoji characters",
			messages: []api.Message{
				{ID: 2, Content: "Hello! 😀🎉🚀 This message has emojis 🌟 and we search for star", CreatedAt: 1700000100},
			},
			query:  "star",
			wantOK: true,
			check: func(t *testing.T, snippet SnippetInfo) {
				if !strings.Contains(snippet.Content, "star") {
					t.Errorf("expected snippet to contain 'star', got %q", snippet.Content)
				}
				if strings.Contains(snippet.Content, "\ufffd") {
					t.Errorf("snippet contains replacement character (corrupted UTF-8): %q", snippet.Content)
				}
			},
		},
		{
			name: "mixed multibyte with ellipsis truncation",
			messages: []api.Message{
				// Need >20 runes before and >50 runes after the keyword to trigger truncation
				// Using 30 Chinese chars before + keyword + 55 Chinese chars after
				{ID: 3, Content: "这是一段很长的中文前缀文本用于测试截断功能确保有足够字符数keyword这是一段很长的中文后缀文本用于测试截断功能确保有足够的字符来触发省略号添加更多更多更多文字继续继续结束", CreatedAt: 1700000200},
			},
			query:  "keyword",
			wantOK: true,
			check: func(t *testing.T, snippet SnippetInfo) {
				if !strings.Contains(snippet.Content, "keyword") {
					t.Errorf("expected snippet to contain 'keyword', got %q", snippet.Content)
				}
				// Should have prefix ellipsis since there's content before (>20 runes)
				if !strings.HasPrefix(snippet.Content, "...") {
					t.Errorf("expected snippet to start with '...', got %q", snippet.Content)
				}
				// Should have suffix ellipsis since there's content after (>50 runes)
				if !strings.HasSuffix(snippet.Content, "...") {
					t.Errorf("expected snippet to end with '...', got %q", snippet.Content)
				}
				if strings.Contains(snippet.Content, "\ufffd") {
					t.Errorf("snippet contains replacement character (corrupted UTF-8): %q", snippet.Content)
				}
			},
		},
		{
			name: "Japanese hiragana and katakana",
			messages: []api.Message{
				{ID: 4, Content: "こんにちは、これはテストメッセージです。キーワードを探しています。", CreatedAt: 1700000300},
			},
			query:  "キーワード",
			wantOK: true,
			check: func(t *testing.T, snippet SnippetInfo) {
				if !strings.Contains(snippet.Content, "キーワード") {
					t.Errorf("expected snippet to contain 'キーワード', got %q", snippet.Content)
				}
				if strings.Contains(snippet.Content, "\ufffd") {
					t.Errorf("snippet contains replacement character (corrupted UTF-8): %q", snippet.Content)
				}
			},
		},
		{
			name: "query at start of message",
			messages: []api.Message{
				{ID: 5, Content: "查询词在开头然后是更多的中文内容", CreatedAt: 1700000400},
			},
			query:  "查询词",
			wantOK: true,
			check: func(t *testing.T, snippet SnippetInfo) {
				if !strings.Contains(snippet.Content, "查询词") {
					t.Errorf("expected snippet to contain '查询词', got %q", snippet.Content)
				}
				// Should NOT have prefix ellipsis since query is at start
				if strings.HasPrefix(snippet.Content, "...") {
					t.Errorf("snippet should not start with '...' when query is at start, got %q", snippet.Content)
				}
			},
		},
		{
			name: "query at end of message",
			messages: []api.Message{
				{ID: 6, Content: "这是消息内容结尾词", CreatedAt: 1700000500},
			},
			query:  "结尾词",
			wantOK: true,
			check: func(t *testing.T, snippet SnippetInfo) {
				if !strings.Contains(snippet.Content, "结尾词") {
					t.Errorf("expected snippet to contain '结尾词', got %q", snippet.Content)
				}
				// Should NOT have suffix ellipsis since query is at end
				if strings.HasSuffix(snippet.Content, "...") {
					t.Errorf("snippet should not end with '...' when query is at end, got %q", snippet.Content)
				}
			},
		},
		{
			name: "no match",
			messages: []api.Message{
				{ID: 7, Content: "This message has no match", CreatedAt: 1700000600},
			},
			query:  "xyz",
			wantOK: false,
			check:  func(t *testing.T, snippet SnippetInfo) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snippet, ok := extractSnippet(tt.messages, tt.query)
			if ok != tt.wantOK {
				t.Fatalf("extractSnippet() ok = %v, wantOK = %v", ok, tt.wantOK)
			}
			if ok {
				tt.check(t, snippet)
			}
		})
	}
}

func TestSearchConversationContent(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("q") == "shipping" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{
					"data": {
						"meta": {"total_pages": 1, "current_page": 1},
						"payload": [
							{"id": 1, "status": "open", "inbox_id": 1},
							{"id": 2, "status": "resolved", "inbox_id": 1}
						]
					}
				}`))
			} else {
				http.NotFound(w, r)
			}
		}).
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"meta": {"total_pages": 1, "current_page": 1},
			"payload": []
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "shipping", "--type", "conversations", "--output", "json"})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
	})

	var result struct {
		Conversations []struct{ ID int } `json:"conversations"`
		Summary       map[string]int     `json:"summary"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if result.Summary["conversations"] != 2 {
		t.Errorf("expected 2 conversations, got %d", result.Summary["conversations"])
	}
}

func TestSearchCommand_Senders(t *testing.T) {
	// Test that sender search finds people who message through shared channels
	// using LINE-style bracketed names like "[Jack Su]"
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 100, "status": "open", "inbox_id": 1, "meta": {"sender": {"id": 50, "name": "Welgrow Support"}}}
				],
				"meta": {"count": 1, "total_pages": 1}
			}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/100/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "[Jack Su]   \nHello, this is Jack", "message_type": 0, "created_at": 1700000000, "sender": {"id": 50, "name": "Welgrow Support"}},
				{"id": 2, "content": "[Amy]   \nHi Jack!", "message_type": 0, "created_at": 1700000100, "sender": {"id": 50, "name": "Welgrow Support"}},
				{"id": 3, "content": "[Jack Su]   \nThanks Amy", "message_type": 0, "created_at": 1700000200, "sender": {"id": 50, "name": "Welgrow Support"}}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "Jack", "--type", "senders", "--output", "json"})
		if err != nil {
			t.Fatalf("search --type senders failed: %v", err)
		}
	})

	var result struct {
		Query   string `json:"query"`
		Senders []struct {
			Name           string `json:"name"`
			ContactID      int    `json:"contact_id"`
			ContactName    string `json:"contact_name"`
			ConversationID int    `json:"conversation_id"`
			MessageCount   int    `json:"message_count"`
		} `json:"senders"`
		Summary map[string]int `json:"summary"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if result.Summary["senders"] != 1 {
		t.Errorf("expected 1 sender, got %d", result.Summary["senders"])
	}
	if len(result.Senders) != 1 {
		t.Fatalf("expected 1 sender in array, got %d", len(result.Senders))
	}

	sender := result.Senders[0]
	if sender.Name != "Jack Su" {
		t.Errorf("expected sender name 'Jack Su', got %q", sender.Name)
	}
	if sender.ContactName != "Welgrow Support" {
		t.Errorf("expected contact name 'Welgrow Support', got %q", sender.ContactName)
	}
	if sender.ConversationID != 100 {
		t.Errorf("expected conversation_id 100, got %d", sender.ConversationID)
	}
	if sender.MessageCount != 2 {
		t.Errorf("expected message_count 2 (Jack Su sent 2 messages), got %d", sender.MessageCount)
	}
}

func TestSearchCommand_SendersExcludedByDefault(t *testing.T) {
	// Test that senders are NOT included in default search (no --type flag)
	// because sender search is expensive (O(n*m) API calls).
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [],
			"meta": {"count": 0}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 200, "status": "open", "inbox_id": 1, "meta": {"sender": {"id": 60, "name": "Support Group"}}}
				],
				"meta": {"count": 1, "total_pages": 1}
			}
		}`))
	// Note: No messages endpoint registered - sender search should not run

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"search", "Alice", "--output", "json"})
		if err != nil {
			t.Fatalf("search failed: %v", err)
		}
	})

	var result struct {
		Senders []struct {
			Name string `json:"name"`
		} `json:"senders"`
		Summary map[string]int `json:"summary"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if _, hasSenders := result.Summary["senders"]; hasSenders {
		t.Errorf("expected senders NOT to be included by default, but summary has senders=%d", result.Summary["senders"])
	}
	if len(result.Senders) != 0 {
		t.Errorf("expected no senders in default search, got %v", result.Senders)
	}
}
