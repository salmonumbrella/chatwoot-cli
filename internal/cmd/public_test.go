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

func setupPublicTestEnv(t *testing.T, handler *routeHandler) {
	t.Helper()
	env := setupTestEnvWithHandler(t, handler)
	// Public API only needs base URL - clear token to verify it's not used
	t.Setenv("CHATWOOT_API_TOKEN", "")
	_ = env // env is used for cleanup via t.Cleanup
}

// Public Inboxes Tests

func TestPublicInboxesGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox123", jsonResponse(200, `{
			"name": "Support Inbox",
			"working_hours_enabled": true,
			"timezone": "America/New_York",
			"csat_survey_enabled": true
		}`))

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"public", "inboxes", "get", "inbox123"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public inboxes get failed: %v", err)
	}

	if !strings.Contains(output, "NAME") || !strings.Contains(output, "WORKING_HOURS") || !strings.Contains(output, "TIMEZONE") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "Support Inbox") {
		t.Errorf("output missing inbox name: %s", output)
	}
}

func TestPublicInboxesGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox123", jsonResponse(200, `{
			"name": "Support Inbox",
			"working_hours_enabled": true,
			"timezone": "America/New_York",
			"csat_survey_enabled": true
		}`))

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"public", "inboxes", "get", "inbox123", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public inboxes get failed: %v", err)
	}

	var inbox map[string]any
	if err := json.Unmarshal([]byte(output), &inbox); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

// Public Contacts Tests

func TestPublicContactsCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/public/api/v1/inboxes/inbox123/contacts", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "source_id": "src123", "name": "John Doe", "email": "john@example.com"}`))
		})

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"public", "contacts", "create", "inbox123",
		"--name", "John Doe",
		"--email", "john@example.com",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public contacts create failed: %v", err)
	}

	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "EMAIL") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "John Doe") {
		t.Errorf("output missing contact name: %s", output)
	}
}

func TestPublicContactsCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/public/api/v1/inboxes/inbox123/contacts", jsonResponse(200, `{
			"id": 1,
			"source_id": "src123",
			"name": "John Doe",
			"email": "john@example.com"
		}`))

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"public", "contacts", "create", "inbox123",
		"--name", "John Doe",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public contacts create failed: %v", err)
	}

	var contact map[string]any
	if err := json.Unmarshal([]byte(output), &contact); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestPublicContactsGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox123/contacts/contact456", jsonResponse(200, `{
			"id": 1,
			"source_id": "contact456",
			"name": "Jane Doe",
			"email": "jane@example.com"
		}`))

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"public", "contacts", "get", "inbox123", "contact456"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public contacts get failed: %v", err)
	}

	if !strings.Contains(output, "Jane Doe") {
		t.Errorf("output missing contact name: %s", output)
	}
}

func TestPublicContactsUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/public/api/v1/inboxes/inbox123/contacts/contact456", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "source_id": "contact456", "name": "Jane Updated", "email": "jane@example.com"}`))
		})

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"public", "contacts", "update", "inbox123", "contact456",
		"--name", "Jane Updated",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public contacts update failed: %v", err)
	}

	if !strings.Contains(output, "Jane Updated") {
		t.Errorf("output missing updated name: %s", output)
	}
}

func TestPublicContactsUpdateCommand_NoFields(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{"public", "contacts", "update", "inbox123", "contact456"})
	if err == nil {
		t.Error("expected error when no fields provided")
	}
	if !strings.Contains(err.Error(), "at least one") {
		t.Errorf("expected 'at least one' error, got: %v", err)
	}
}

// Public Conversations Tests

func TestPublicConversationsListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox123/contacts/contact456/conversations", jsonResponse(200, `[
			{"id": 1, "status": "open"},
			{"id": 2, "status": "resolved"}
		]`))

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"public", "conversations", "list", "inbox123", "contact456"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public conversations list failed: %v", err)
	}

	if !strings.Contains(output, "ID") || !strings.Contains(output, "STATUS") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "open") {
		t.Errorf("output missing status: %s", output)
	}
}

func TestPublicConversationsListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox123/contacts/contact456/conversations", jsonResponse(200, `[
			{"id": 1, "status": "open"}
		]`))

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"public", "conversations", "list", "inbox123", "contact456", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public conversations list failed: %v", err)
	}

	_ = decodeItems(t, output)
}

func TestPublicConversationsGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox123/contacts/contact456/conversations/1", jsonResponse(200, `{
			"id": 1,
			"status": "open"
		}`))

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"public", "conversations", "get", "inbox123", "contact456", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public conversations get failed: %v", err)
	}

	if !strings.Contains(output, "open") {
		t.Errorf("output missing status: %s", output)
	}
}

func TestPublicConversationsGetCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{"public", "conversations", "get", "inbox123", "contact456", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "conversation ID") {
		t.Errorf("expected 'conversation ID' error, got: %v", err)
	}
}

func TestPublicConversationsCreateCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/public/api/v1/inboxes/inbox123/contacts/contact456/conversations", jsonResponse(200, `{
			"id": 1,
			"status": "open"
		}`))

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"public", "conversations", "create", "inbox123", "contact456"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public conversations create failed: %v", err)
	}

	if !strings.Contains(output, "ID") || !strings.Contains(output, "STATUS") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestPublicConversationsResolveCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/public/api/v1/inboxes/inbox123/contacts/contact456/conversations/1/toggle_status", jsonResponse(200, `{
			"id": 1,
			"status": "resolved"
		}`))

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"public", "conversations", "resolve", "inbox123", "contact456", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public conversations resolve failed: %v", err)
	}

	if !strings.Contains(output, "Conversation 1 status: resolved") {
		t.Errorf("expected status message, got: %s", output)
	}
}

// Public Messages Tests

func TestPublicMessagesListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox123/contacts/contact456/conversations/1/messages", jsonResponse(200, `[
			{"id": 1, "message_type": "incoming", "content": "Hello!"},
			{"id": 2, "message_type": "outgoing", "content": "Hi there, how can I help?"}
		]`))

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"public", "messages", "list", "inbox123", "contact456", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public messages list failed: %v", err)
	}

	if !strings.Contains(output, "ID") || !strings.Contains(output, "TYPE") || !strings.Contains(output, "CONTENT") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "Hello!") {
		t.Errorf("output missing message content: %s", output)
	}
}

func TestPublicMessagesListCommand_LongContent(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox123/contacts/contact456/conversations/1/messages", jsonResponse(200, `[
			{"id": 1, "message_type": "incoming", "content": "This is a very long message that should be truncated because it exceeds the fifty character limit for display purposes"}
		]`))

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"public", "messages", "list", "inbox123", "contact456", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public messages list failed: %v", err)
	}

	// Long content should be truncated with "..."
	if !strings.Contains(output, "...") {
		t.Errorf("output should truncate long content: %s", output)
	}
}

func TestPublicMessagesCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/public/api/v1/inboxes/inbox123/contacts/contact456/conversations/1/messages", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 10}`))
		})

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"public", "messages", "create", "inbox123", "contact456", "1",
		"--content", "Test message",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public messages create failed: %v", err)
	}

	if !strings.Contains(output, "Message 10 created") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["content"] != "Test message" {
		t.Errorf("expected content 'Test message', got %v", receivedBody["content"])
	}
}

func TestPublicMessagesCreateCommand_MissingContent(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{"public", "messages", "create", "inbox123", "contact456", "1"})
	if err == nil {
		t.Error("expected error when content is missing")
	}
	if !strings.Contains(err.Error(), "--content is required") {
		t.Errorf("expected '--content is required' error, got: %v", err)
	}
}

func TestPublicMessagesUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/public/api/v1/inboxes/inbox123/contacts/contact456/conversations/1/messages/10", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 10}`))
		})

	setupPublicTestEnv(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"public", "messages", "update", "inbox123", "contact456", "1", "10",
		"--content", "Updated message",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("public messages update failed: %v", err)
	}

	if !strings.Contains(output, "Message 10 updated") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPublicMessagesUpdateCommand_MissingContent(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{"public", "messages", "update", "inbox123", "contact456", "1", "10"})
	if err == nil {
		t.Error("expected error when content is missing")
	}
	if !strings.Contains(err.Error(), "--content is required") {
		t.Errorf("expected '--content is required' error, got: %v", err)
	}
}

func TestPublicMessagesUpdateCommand_InvalidMessageID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"public", "messages", "update", "inbox123", "contact456", "1", "invalid",
		"--content", "Test",
	})
	if err == nil {
		t.Error("expected error for invalid message ID")
	}
	if !strings.Contains(err.Error(), "message ID") {
		t.Errorf("expected 'message ID' error, got: %v", err)
	}
}

// API Error Tests

func TestPublicInboxesGetCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/public/api/v1/inboxes/inbox123", jsonResponse(404, `{"error": "Not found"}`))

	setupPublicTestEnv(t, handler)

	err := Execute(context.Background(), []string{"public", "inboxes", "get", "inbox123"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}
