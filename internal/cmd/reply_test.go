package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestReplyCommand_RequiresContent(t *testing.T) {
	// Use t.Setenv which automatically restores values after the test
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"reply", "test-contact"})
	if err == nil {
		t.Error("Expected error when content is missing")
	}
}

func TestReplyCommand_RequiresSearchOrID(t *testing.T) {
	// Use t.Setenv which automatically restores values after the test
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"reply", "--content", "test message"})
	if err == nil {
		t.Error("Expected error when no search query or ID is provided")
	}
}

func TestOutputDisambiguation_MultipleContacts(t *testing.T) {
	contacts := []api.Contact{
		{ID: 1, Name: "John Doe", Email: "john@example.com"},
		{ID: 2, Name: "John Smith", Email: "john.smith@example.com"},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newReplyCmd()
	cmd.SetContext(context.Background())
	err := outputDisambiguation(cmd, "multiple_contacts", contacts)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check text output contains expected information
	if !strings.Contains(output, "Multiple contacts found") {
		t.Error("Expected disambiguation message")
	}
	if !strings.Contains(output, "John Doe") {
		t.Error("Expected first contact name in output")
	}
	if !strings.Contains(output, "john@example.com") {
		t.Error("Expected first contact email in output")
	}
	if !strings.Contains(output, "--contact-id") {
		t.Error("Expected hint about contact-id flag")
	}
}

func TestOutputConversationDisambiguation_MultipleConversations(t *testing.T) {
	displayID := 100
	conversations := []api.Conversation{
		{ID: 1, DisplayID: &displayID, InboxID: 1, Status: "open", LastActivityAt: 1700000000},
		{ID: 2, InboxID: 2, Status: "open", LastActivityAt: 1700001000},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newReplyCmd()
	cmd.SetContext(context.Background())
	err := outputConversationDisambiguation(cmd, conversations, 123)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check text output contains expected information
	if !strings.Contains(output, "Multiple open conversations") {
		t.Error("Expected disambiguation message")
	}
	if !strings.Contains(output, "100") {
		t.Error("Expected display ID in output")
	}
	if !strings.Contains(output, "--conversation-id") {
		t.Error("Expected hint about conversation-id flag")
	}
}

func TestReplyResult_JSONSerialization(t *testing.T) {
	contact := &api.TriageContact{
		ID:    123,
		Name:  "Test User",
		Email: "test@example.com",
	}

	result := api.ReplyResult{
		Action:         "replied",
		ConversationID: 456,
		Contact:        contact,
		MessageID:      789,
		Resolved:       true,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal ReplyResult: %v", err)
	}

	var decoded api.ReplyResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ReplyResult: %v", err)
	}

	if decoded.Action != "replied" {
		t.Errorf("Expected action 'replied', got %s", decoded.Action)
	}
	if decoded.ConversationID != 456 {
		t.Errorf("Expected conversation_id 456, got %d", decoded.ConversationID)
	}
	if decoded.MessageID != 789 {
		t.Errorf("Expected message_id 789, got %d", decoded.MessageID)
	}
	if !decoded.Resolved {
		t.Error("Expected resolved to be true")
	}
}

func TestReplyResult_DisambiguationJSON(t *testing.T) {
	matches := []map[string]any{
		{"id": 1, "name": "Contact 1", "email": "c1@test.com"},
		{"id": 2, "name": "Contact 2", "email": "c2@test.com"},
	}

	result := api.ReplyResult{
		Action:  "disambiguation_needed",
		Type:    "multiple_contacts",
		Matches: matches,
		Hint:    "Use contact ID: cw reply --contact-id <id> --content '...'",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal ReplyResult: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"action":"disambiguation_needed"`) {
		t.Error("Expected action field in JSON")
	}
	if !strings.Contains(jsonStr, `"type":"multiple_contacts"`) {
		t.Error("Expected type field in JSON")
	}
	if !strings.Contains(jsonStr, `"hint"`) {
		t.Error("Expected hint field in JSON")
	}
}

// Integration-style tests with mock server

func setupTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, func()) {
	server := httptest.NewServer(handler)

	// Use t.Setenv which automatically restores values after the test
	t.Setenv("CHATWOOT_BASE_URL", server.URL)
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1") // Skip URL validation for localhost

	cleanup := func() {
		server.Close()
	}

	return server, cleanup
}

func TestReplyByConversationID_Success(t *testing.T) {
	requestCount := 0
	_, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/messages") && r.Method == "POST":
			// Create message
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 100, "conversation_id": 123, "content": "Test message", "message_type": 1, "created_at": 1700000000}`))

		case strings.Contains(r.URL.Path, "/conversations/123") && r.Method == "GET":
			// Get conversation
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 123, "contact_id": 456, "status": "open"}`))

		case strings.Contains(r.URL.Path, "/contacts/456") && r.Method == "GET":
			// Get contact
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"id": 456, "name": "Test Contact", "email": "test@example.com", "created_at": 1700000000}}`))

		default:
			t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer cleanup()

	err := Execute(context.Background(), []string{"reply", "--conversation-id", "123", "--content", "Test message"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestReplyByConversationID_WithResolve(t *testing.T) {
	resolveRequested := false
	_, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/messages") && r.Method == "POST":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 100, "conversation_id": 123, "content": "Done!", "message_type": 1, "created_at": 1700000000}`))

		case strings.Contains(r.URL.Path, "/toggle_status") && r.Method == "POST":
			resolveRequested = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"meta": {}, "payload": {"success": true, "conversation_id": 123, "current_status": "resolved"}}`))

		case strings.Contains(r.URL.Path, "/conversations/123") && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 123, "contact_id": 456, "status": "open"}`))

		case strings.Contains(r.URL.Path, "/contacts/456") && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"id": 456, "name": "Test Contact", "email": "test@example.com", "created_at": 1700000000}}`))

		default:
			t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer cleanup()

	err := Execute(context.Background(), []string{"reply", "--conversation-id", "123", "--content", "Done!", "--resolve"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !resolveRequested {
		t.Error("Expected resolve to be called")
	}
}

func TestReplyByContactSearch_SingleMatch(t *testing.T) {
	_, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/contacts/search"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": [{"id": 789, "name": "John Doe", "email": "john@example.com", "created_at": 1700000000}], "meta": {}}`))

		case strings.Contains(r.URL.Path, "/contacts/789/conversations"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": [{"id": 456, "status": "open", "inbox_id": 1, "last_activity_at": 1700000000}]}`))

		case strings.Contains(r.URL.Path, "/contacts/789") && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"id": 789, "name": "John Doe", "email": "john@example.com", "created_at": 1700000000}}`))

		case strings.Contains(r.URL.Path, "/conversations/456/messages") && r.Method == "POST":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 100, "conversation_id": 456, "content": "Hello!", "message_type": 1, "created_at": 1700000000}`))

		default:
			t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer cleanup()

	err := Execute(context.Background(), []string{"reply", "john", "--content", "Hello!", "--no-input"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestReplyByContactSearch_NoMatches(t *testing.T) {
	_, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/contacts/search") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": [], "meta": {}}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	err := Execute(context.Background(), []string{"reply", "nonexistent", "--content", "Hello!"})
	if err == nil {
		t.Error("Expected error for no matching contacts")
	}
}

func TestReplyByContactSearch_NoMatches_DryRunGuidance(t *testing.T) {
	_, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/contacts/search") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": [], "meta": {}}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	err := Execute(context.Background(), []string{"reply", "nonexistent", "--content", "Hello!", "--dry-run"})
	if err == nil {
		t.Fatal("Expected error for no matching contacts in dry-run mode")
	}
	if !strings.Contains(err.Error(), "dry-run still requires a real contact/conversation") {
		t.Fatalf("expected dry-run guidance in error, got: %v", err)
	}
}

func TestReplyByContactSearch_MultipleMatches(t *testing.T) {
	_, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/contacts/search") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"payload": [
					{"id": 1, "name": "John Doe", "email": "john@example.com", "created_at": 1700000000},
					{"id": 2, "name": "John Smith", "email": "johns@example.com", "created_at": 1700000000}
				],
				"meta": {}
			}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	// Capture stdout to verify disambiguation output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reply", "john", "--content", "Hello!", "--no-input"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// The command should not return an error for disambiguation
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should show disambiguation message
	if !strings.Contains(output, "Multiple contacts found") {
		t.Error("Expected disambiguation message in output")
	}
}

func TestReplyByContactSearch_MultipleMatches_Interactive(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com", "created_at": 1700000000},
				{"id": 2, "name": "John Smith", "email": "johns@example.com", "created_at": 1700000000}
			],
			"meta": {}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/1", jsonResponse(200, `{
			"payload": {"id": 1, "name": "John Doe", "email": "john@example.com", "created_at": 1700000000}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/1/conversations", jsonResponse(200, `{
			"payload": [{"id": 456, "status": "open", "inbox_id": 1, "last_activity_at": 1700000000}]
		}`)).
		On("POST", "/api/v1/accounts/1/conversations/456/messages", jsonResponse(200, `{
			"id": 100,
			"conversation_id": 456,
			"content": "Hello!",
			"message_type": 1,
			"created_at": 1700000000
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
		err := Execute(context.Background(), []string{"reply", "john", "--content", "Hello!"})
		if err != nil {
			t.Errorf("reply --content failed: %v", err)
		}
	})

	if !strings.Contains(output, "Sent message 100") {
		t.Errorf("expected message sent output, got: %s", output)
	}
	if !strings.Contains(output, "Conversation: 456") {
		t.Errorf("expected conversation output, got: %s", output)
	}
}

func TestReplyByContactID_NoOpenConversations(t *testing.T) {
	_, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/contacts/123/conversations"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": [{"id": 1, "status": "resolved"}, {"id": 2, "status": "pending"}]}`))

		case strings.Contains(r.URL.Path, "/contacts/123") && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"id": 123, "name": "Test User", "created_at": 1700000000}}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer cleanup()

	err := Execute(context.Background(), []string{"reply", "--contact-id", "123", "--content", "Hello!", "--no-input"})
	if err == nil {
		t.Error("Expected error for no open conversations")
	}
}

func TestReplyByContactID_MultipleOpenConversations(t *testing.T) {
	_, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/contacts/123/conversations"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": [
				{"id": 1, "status": "open", "inbox_id": 1, "last_activity_at": 1700000000},
				{"id": 2, "status": "open", "inbox_id": 2, "last_activity_at": 1700001000}
			]}`))

		case strings.Contains(r.URL.Path, "/contacts/123") && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"id": 123, "name": "Test User", "created_at": 1700000000}}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer cleanup()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reply", "--contact-id", "123", "--content", "Hello!", "--no-input"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !strings.Contains(output, "Multiple open conversations") {
		t.Error("Expected conversation disambiguation message")
	}
}

func TestReplyPrivateNote(t *testing.T) {
	privateNoteReceived := false
	_, cleanup := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, "/messages") && r.Method == "POST" {
			// Parse request body to check private flag
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				if private, ok := body["private"].(bool); ok && private {
					privateNoteReceived = true
				}
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 100, "conversation_id": 123, "content": "Internal note", "message_type": 1, "private": true, "created_at": 1700000000}`))
			return
		}

		if strings.Contains(r.URL.Path, "/conversations/123") && r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 123, "contact_id": 456, "status": "open"}`))
			return
		}

		if strings.Contains(r.URL.Path, "/contacts/456") && r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"id": 456, "name": "Test", "created_at": 1700000000}}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	err := Execute(context.Background(), []string{"reply", "--conversation-id", "123", "--content", "Internal note", "--private"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !privateNoteReceived {
		t.Error("Expected private note flag to be set in request")
	}
}

func TestReplyDryRun(t *testing.T) {
	messagesSent := false

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"contact_id": 456,
			"inbox_id": 10,
			"status": "open"
		}`)).
		On("GET", "/api/v1/accounts/1/inboxes/10", jsonResponse(200, `{
			"id": 10,
			"name": "LINE Official",
			"channel_type": "Channel::Line"
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {"id": 456, "name": "Test Contact", "email": "test@example.com", "created_at": 1700000000}
		}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/messages", func(w http.ResponseWriter, r *http.Request) {
			messagesSent = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 100, "conversation_id": 123, "content": "Test message", "message_type": 1, "created_at": 1700000000}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"reply", "--conversation-id", "123", "--content", "Hello from dry-run!", "--dry-run"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Should NOT have sent a message
	if messagesSent {
		t.Error("Expected dry-run to NOT send a message")
	}

	// Should contain dry-run indicator
	if !strings.Contains(output, "DRY RUN") && !strings.Contains(output, "DRY-RUN") {
		t.Errorf("Expected output to contain 'DRY RUN', got: %s", output)
	}

	// Should contain channel type info
	if !strings.Contains(output, "Channel::Line") && !strings.Contains(output, "LINE Official") {
		t.Errorf("Expected output to contain channel info, got: %s", output)
	}

	// Should contain the message content preview
	if !strings.Contains(output, "Hello from dry-run!") {
		t.Errorf("Expected output to contain message content, got: %s", output)
	}
}

func TestReplyDryRunPrivateNote(t *testing.T) {
	messagesSent := false

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"contact_id": 456,
			"inbox_id": 10,
			"status": "open"
		}`)).
		On("GET", "/api/v1/accounts/1/inboxes/10", jsonResponse(200, `{
			"id": 10,
			"name": "Support Inbox",
			"channel_type": "Channel::WebWidget"
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {"id": 456, "name": "Test Contact", "email": "test@example.com", "created_at": 1700000000}
		}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/messages", func(w http.ResponseWriter, r *http.Request) {
			messagesSent = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 100, "conversation_id": 123, "content": "Private note", "message_type": 1, "private": true, "created_at": 1700000000}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"reply", "--conversation-id", "123", "--content", "Private note content", "--private", "--dry-run"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Should NOT have sent a message
	if messagesSent {
		t.Error("Expected dry-run to NOT send a message")
	}

	// Should indicate private note
	if !strings.Contains(output, "Private note") || !strings.Contains(output, "internal") {
		t.Errorf("Expected output to indicate private note, got: %s", output)
	}
}

func TestReplyDryRunJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"contact_id": 456,
			"inbox_id": 10,
			"status": "open"
		}`)).
		On("GET", "/api/v1/accounts/1/inboxes/10", jsonResponse(200, `{
			"id": 10,
			"name": "LINE Official",
			"channel_type": "Channel::Line"
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {"id": 456, "name": "Test Contact", "email": "test@example.com", "created_at": 1700000000}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"reply", "--conversation-id", "123", "--content", "Test message", "--dry-run", "-o", "json"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Should be valid JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Expected valid JSON output, got error: %v, output: %s", err, output)
	}

	// Should have dry_run flag
	if dryRun, ok := result["dry_run"].(bool); !ok || !dryRun {
		t.Errorf("Expected dry_run=true in JSON output, got: %v", result)
	}
}

func TestReplyDryRunChannelWarning_LINE(t *testing.T) {
	// Create a message longer than LINE's 2000 character limit
	longMessage := strings.Repeat("x", 2001)

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"contact_id": 456,
			"inbox_id": 10,
			"status": "open"
		}`)).
		On("GET", "/api/v1/accounts/1/inboxes/10", jsonResponse(200, `{
			"id": 10,
			"name": "LINE Official",
			"channel_type": "Channel::Line"
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {"id": 456, "name": "Test Contact", "email": "test@example.com", "created_at": 1700000000}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"reply", "--conversation-id", "123", "--content", longMessage, "--dry-run"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Should contain warning about LINE character limit
	if !strings.Contains(output, "LINE has a 2000 character limit") {
		t.Errorf("Expected LINE character limit warning in output, got: %s", output)
	}

	// Should contain actual character count
	if !strings.Contains(output, "2001 characters") {
		t.Errorf("Expected '2001 characters' in warning, got: %s", output)
	}

	// Should show in Warnings section
	if !strings.Contains(output, "Warnings:") {
		t.Errorf("Expected 'Warnings:' section header, got: %s", output)
	}
}

func TestReplyDryRunChannelWarning_LINE_JSON(t *testing.T) {
	// Create a message longer than LINE's 2000 character limit
	longMessage := strings.Repeat("x", 2500)

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"contact_id": 456,
			"inbox_id": 10,
			"status": "open"
		}`)).
		On("GET", "/api/v1/accounts/1/inboxes/10", jsonResponse(200, `{
			"id": 10,
			"name": "LINE Official",
			"channel_type": "Channel::Line"
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {"id": 456, "name": "Test Contact", "email": "test@example.com", "created_at": 1700000000}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"reply", "--conversation-id", "123", "--content", longMessage, "--dry-run", "-o", "json"})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Should be valid JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Expected valid JSON output, got error: %v, output: %s", err, output)
	}

	// Should have warnings array
	warnings, ok := result["warnings"].([]any)
	if !ok {
		t.Fatalf("Expected warnings array in JSON output, got: %v", result)
	}

	if len(warnings) == 0 {
		t.Error("Expected at least one warning in JSON output")
	}

	// First warning should mention LINE limit
	warning := warnings[0].(string)
	if !strings.Contains(warning, "LINE has a 2000 character limit") {
		t.Errorf("Expected LINE warning in JSON, got: %s", warning)
	}
}

func TestReplyCommand_Pending(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"id": 100, "conversation_id": 123, "content": "Heads up", "message_type": 1, "created_at": 1700000000
		}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"payload": {"success": true, "current_status": "pending", "conversation_id": 123}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123, "contact_id": 456, "status": "open"
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {"id": 456, "name": "Test Contact", "email": "test@example.com", "created_at": 1700000000}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"reply", "--conversation-id", "123", "--content", "Heads up", "--pending", "-o", "json"})
		if err != nil {
			t.Fatalf("reply --pending failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["pending"] != true {
		t.Fatalf("expected pending true, got %#v", result["pending"])
	}
}

func TestReplyCommand_ResolveAndPendingExclusive(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"id": 100, "conversation_id": 123, "content": "conflict", "message_type": 1, "created_at": 1700000000
		}`))

	setupTestEnvWithHandler(t, handler)

	assertStatusFlagsMutuallyExclusive(t, []string{"reply", "--conversation-id", "123", "--content", "conflict", "--resolve", "--pending"})
}
