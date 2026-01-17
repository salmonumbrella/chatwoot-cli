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
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
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

	// Check contacts section
	if !strings.Contains(output, "Contacts (2)") {
		t.Errorf("output missing contacts count: %s", output)
	}
	if !strings.Contains(output, "John Doe") {
		t.Errorf("output missing 'John Doe': %s", output)
	}
	if !strings.Contains(output, "John Smith") {
		t.Errorf("output missing 'John Smith': %s", output)
	}

	// Check conversations section
	if !strings.Contains(output, "Conversations (1)") {
		t.Errorf("output missing conversations count: %s", output)
	}
	if !strings.Contains(output, "#100") {
		t.Errorf("output missing conversation ID: %s", output)
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
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
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

	if !strings.Contains(output, "Contacts (1)") {
		t.Errorf("output missing contacts count: %s", output)
	}
	if strings.Contains(output, "Conversations") {
		t.Errorf("output should not contain Conversations when filtering by contacts: %s", output)
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
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
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
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
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
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
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
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
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
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
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

	if !strings.Contains(output, "Conversations (1)") {
		t.Errorf("output missing conversations count: %s", output)
	}
	if strings.Contains(output, "Contacts") {
		t.Errorf("output should not contain Contacts when filtering by conversations: %s", output)
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
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
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
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
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
		On("GET", "/api/v1/accounts/1/conversations/search", func(w http.ResponseWriter, r *http.Request) {
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
