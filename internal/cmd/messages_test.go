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

func TestMessagesListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Hello", "message_type": 0, "private": false, "created_at": 1704067200},
				{"id": 2, "content": "World", "message_type": 1, "private": true, "created_at": 1704153600}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"messages", "list", "123"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("messages list failed: %v", err)
	}

	if !strings.Contains(output, "Hello") {
		t.Errorf("output missing 'Hello': %s", output)
	}
	if !strings.Contains(output, "World") {
		t.Errorf("output missing 'World': %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "TYPE") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestMessagesListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Hello", "message_type": 0, "private": false, "created_at": 1704067200}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"messages", "list", "123", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("messages list failed: %v", err)
	}

	messages := decodeItems(t, output)
	if len(messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(messages))
	}
}

func TestMessagesListCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "  \nCustomer hello  ", "message_type": 0, "private": false, "created_at": 1704067200, "sender": {"id": 42, "name": "Alice"}},
				{"id": 2, "content": "Agent reply", "message_type": 1, "private": false, "created_at": 1704067300, "attachments": [{"file_type":"image"}]},
				{"id": 3, "content": "Internal note", "message_type": 1, "private": true, "created_at": 1704067400},
				{"id": 4, "content": "Assigned conversation", "message_type": 2, "private": false, "created_at": 1704067500}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"messages", "list", "123", "--li"})
		if err != nil {
			t.Fatalf("messages list --li failed: %v", err)
		}
	})

	var payload []struct {
		ID          int    `json:"id"`
		MessageType int    `json:"mt"`
		Private     bool   `json:"prv"`
		Content     string `json:"ct"`
		Sender      *struct {
			Name string `json:"nm"`
		} `json:"sn,omitempty"`
		Attachments []string `json:"att"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light messages output: %v\noutput: %s", err, output)
	}
	if len(payload) != 3 {
		t.Fatalf("expected 3 non-activity messages, got %d", len(payload))
	}
	if payload[0].MessageType != 0 || payload[1].MessageType != 1 {
		t.Fatalf("unexpected message types: %#v", payload)
	}
	if payload[1].Attachments[0] != "image" {
		t.Fatalf("expected attachment type image, got %#v", payload[1].Attachments)
	}
	if payload[0].Content != "Customer hello" {
		t.Fatalf("expected trimmed message content, got %q", payload[0].Content)
	}
	if payload[0].Sender == nil || payload[0].Sender.Name != "Alice" {
		t.Fatalf("expected sender name Alice, got %#v", payload[0].Sender)
	}
	if strings.Contains(output, `"sn":{"id"`) {
		t.Fatal("light output should not include sender id")
	}
	if !payload[2].Private {
		t.Fatal("expected private note to keep private=true")
	}
	if strings.Contains(output, `"items"`) {
		t.Fatal("light output should not include an items wrapper")
	}
	if strings.Contains(output, `"content_type"`) {
		t.Fatal("light output should not include content_type")
	}
	if strings.Contains(output, `"conversation_id"`) {
		t.Fatal("light output should not include conversation_id")
	}
}

func TestMessagesListCommand_Light_QueryKeepsItemsCompatibility(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Customer hello", "message_type": 0, "private": false, "created_at": 1704067200}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"messages", "list", "123", "--li", "--jq", ".items[0].id"})
		if err != nil {
			t.Fatalf("messages list --li --jq .items[0].id failed: %v", err)
		}
	})

	if strings.TrimSpace(output) != "1" {
		t.Fatalf("expected .items[0].id query to return 1, got %q", strings.TrimSpace(output))
	}
}

func TestMessagesListCommand_LightTranscriptConflict(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"messages", "list", "123", "--light", "--transcript"})
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "--light cannot be combined with --transcript") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMessagesListCommand_AgentResolveNames(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Hello", "message_type": 0, "private": false, "created_at": 1704067200}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 7,
			"contact_id": 42,
			"status": "open",
			"unread_count": 0,
			"created_at": 1700000000
		}`)).
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 7, "name": "Support"}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/42", jsonResponse(200, `{
			"payload": {"id": 42, "name": "Jane Doe"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"messages", "list", "123", "--output", "agent", "--resolve-names"})
		if err != nil {
			t.Errorf("messages list --output agent --resolve-names failed: %v", err)
		}
	})

	var payload struct {
		Meta map[string]any `json:"meta"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	conversation, ok := payload.Meta["conversation"].(map[string]any)
	if !ok {
		t.Fatalf("expected conversation detail in meta, got %#v", payload.Meta["conversation"])
	}
	path, ok := conversation["path"].([]any)
	if !ok || len(path) == 0 {
		t.Fatalf("expected conversation path in meta, got %#v", conversation["path"])
	}
}

func TestMessagesListCommand_Limit(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/accounts/1/conversations/123/messages" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch r.URL.Query().Get("before") {
		case "":
			_, _ = w.Write([]byte(`{
				"payload": [
					{"id": 5, "content": "A", "message_type": 0, "private": false, "created_at": 1704067200},
					{"id": 4, "content": "B", "message_type": 0, "private": false, "created_at": 1704067201},
					{"id": 3, "content": "C", "message_type": 0, "private": false, "created_at": 1704067202}
				]
			}`))
		case "3":
			_, _ = w.Write([]byte(`{
				"payload": [
					{"id": 2, "content": "D", "message_type": 0, "private": false, "created_at": 1704067203},
					{"id": 1, "content": "E", "message_type": 0, "private": false, "created_at": 1704067204}
				]
			}`))
		default:
			_, _ = w.Write([]byte(`{"payload": []}`))
		}
	})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"messages", "list", "123", "--limit", "4", "-o", "json"}); err != nil {
			t.Fatalf("messages list --limit failed: %v", err)
		}
	})

	items := decodeItems(t, output)
	if len(items) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(items))
	}
	if items[3]["id"] != float64(2) {
		t.Errorf("expected last message id 2, got %v", items[3]["id"])
	}
}

func TestMessagesListCommand_AllShorthand(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/accounts/1/conversations/123/messages" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch r.URL.Query().Get("before") {
		case "":
			_, _ = w.Write([]byte(`{
				"payload": [
					{"id": 2, "content": "Newest", "message_type": 0, "private": false, "created_at": 1704067200},
					{"id": 1, "content": "Older", "message_type": 0, "private": false, "created_at": 1704067100}
				]
			}`))
		default:
			_, _ = w.Write([]byte(`{"payload": []}`))
		}
	})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"messages", "list", "123", "-a", "-o", "json"}); err != nil {
			t.Fatalf("messages list -a failed: %v", err)
		}
	})

	items := decodeItems(t, output)
	if len(items) != 2 {
		t.Fatalf("expected 2 messages with -a, got %d", len(items))
	}
}

func TestMessagesListCommand_InvalidLimit(t *testing.T) {
	err := Execute(context.Background(), []string{"messages", "list", "123", "--limit", "0"})
	if err == nil {
		t.Fatal("expected error for zero limit")
	}
	if !strings.Contains(err.Error(), "--limit must be at least 1") {
		t.Fatalf("expected limit validation error, got: %v", err)
	}
}

func TestMessagesListCommand_KeywordAlias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Shipping update", "message_type": 0, "private": false, "created_at": 1704067200},
				{"id": 2, "content": "Refund approved", "message_type": 1, "private": false, "created_at": 1704067300},
				{"id": 3, "content": "Order received", "message_type": 0, "private": false, "created_at": 1704067400}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"messages", "list", "123", "--kw", "refund", "-o", "json"}); err != nil {
			t.Fatalf("messages list --kw failed: %v", err)
		}
	})

	items := decodeItems(t, output)
	if len(items) != 1 {
		t.Fatalf("expected 1 filtered message, got %d", len(items))
	}
	if items[0]["id"] != float64(2) {
		t.Fatalf("expected message id 2, got %v", items[0]["id"])
	}
}

func TestMessagesListCommand_InvalidKeyword(t *testing.T) {
	err := Execute(context.Background(), []string{"messages", "list", "123", "--keyword", "   "})
	if err == nil {
		t.Fatal("expected error for empty keyword")
	}
	if !strings.Contains(err.Error(), "--keyword must be non-empty") {
		t.Fatalf("expected keyword validation error, got: %v", err)
	}
}

func TestMessagesListCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"messages", "list", "abc"})
	if err == nil {
		t.Error("expected error for invalid conversation ID")
	}

	err = Execute(context.Background(), []string{"messages", "list", "-1"})
	if err == nil {
		t.Error("expected error for negative conversation ID")
	}
}

func TestMessagesListCommand_MissingID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"messages", "list"})
	if err == nil {
		t.Error("expected error when conversation ID is missing")
	}
}

func TestMessagesCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 456, "content": "Test message", "message_type": 1, "private": false, "created_at": 1704067200}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"messages", "create", "123",
		"--content", "Test message",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("messages create failed: %v", err)
	}

	if !strings.Contains(output, "Created message 456") {
		t.Errorf("expected success message, got: %s", output)
	}
	if !strings.Contains(output, "456") {
		t.Errorf("expected message ID in output, got: %s", output)
	}

	if receivedBody["content"] != "Test message" {
		t.Errorf("expected content 'Test message', got %v", receivedBody["content"])
	}
}

func TestMessagesCreateCommand_Private(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 456, "content": "Private note", "message_type": 1, "private": true, "created_at": 1704067200}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"messages", "create", "123",
		"--content", "Private note",
		"--private",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("messages create failed: %v", err)
	}

	if receivedBody["private"] != true {
		t.Errorf("expected private true, got %v", receivedBody["private"])
	}
}

func TestMessagesCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"id": 456,
			"content": "Test message",
			"message_type": 1,
			"private": false,
			"created_at": 1704067200
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"messages", "create", "123",
		"--content", "Test message",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("messages create failed: %v", err)
	}

	var message map[string]any
	if err := json.Unmarshal([]byte(output), &message); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if message["content"] != "Test message" {
		t.Errorf("expected content 'Test message', got %v", message["content"])
	}
}

func TestMessagesCreateCommand_MissingContent(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"messages", "create", "123"})
	if err == nil {
		t.Error("expected error when content is missing")
	}
	if !strings.Contains(err.Error(), "either --content or --attachment is required") {
		t.Errorf("expected '--content or --attachment is required' error, got: %v", err)
	}
}

func TestMessagesCreateCommand_InvalidConversationID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"messages", "create", "abc", "--content", "test"})
	if err == nil {
		t.Error("expected error for invalid conversation ID")
	}
}

func TestMessagesDeleteCommand(t *testing.T) {
	deleteRequested := false
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/conversations/123/messages/456", func(w http.ResponseWriter, r *http.Request) {
			deleteRequested = true
			w.WriteHeader(http.StatusOK)
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"messages", "delete", "123", "456"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("messages delete failed: %v", err)
	}

	if !deleteRequested {
		t.Error("expected DELETE request to be made")
	}

	if !strings.Contains(output, "Deleted message 456") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestMessagesDeleteCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/conversations/123/messages/456", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"messages", "delete", "123", "456", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("messages delete failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if result["deleted"] != true {
		t.Errorf("expected deleted true, got %v", result["deleted"])
	}
}

func TestMessagesDeleteCommand_InvalidIDs(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"messages", "delete", "abc", "456"})
	if err == nil {
		t.Error("expected error for invalid conversation ID")
	}

	err = Execute(context.Background(), []string{"messages", "delete", "123", "abc"})
	if err == nil {
		t.Error("expected error for invalid message ID")
	}
}

func TestMessagesDeleteCommand_MissingArgs(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"messages", "delete", "123"})
	if err == nil {
		t.Error("expected error when message ID is missing")
	}

	err = Execute(context.Background(), []string{"messages", "delete"})
	if err == nil {
		t.Error("expected error when both IDs are missing")
	}
}

func TestMessagesUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/conversations/123/messages/456", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 456, "content": "Updated content", "message_type": 1, "private": false, "created_at": 1704067200}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"messages", "update", "123", "456",
		"--content", "Updated content",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("messages update failed: %v", err)
	}

	if !strings.Contains(output, "Updated message 456") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["content"] != "Updated content" {
		t.Errorf("expected content 'Updated content', got %v", receivedBody["content"])
	}
}

func TestMessagesUpdateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/conversations/123/messages/456", jsonResponse(200, `{
			"id": 456,
			"content": "Updated content",
			"message_type": 1,
			"private": false,
			"created_at": 1704067200
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"messages", "update", "123", "456",
		"--content", "Updated content",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("messages update failed: %v", err)
	}

	var message map[string]any
	if err := json.Unmarshal([]byte(output), &message); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestMessagesUpdateCommand_MissingContent(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"messages", "update", "123", "456"})
	if err == nil {
		t.Error("expected error when content is missing")
	}
}

func TestMessagesUpdateCommand_InvalidIDs(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"messages", "update", "abc", "456", "--content", "test"})
	if err == nil {
		t.Error("expected error for invalid conversation ID")
	}

	err = Execute(context.Background(), []string{"messages", "update", "123", "abc", "--content", "test"})
	if err == nil {
		t.Error("expected error for invalid message ID")
	}
}

func TestMessagesCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"messages", "list", "123"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestMessagesListCommand_LongContent(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "This is a very long message content that should be truncated in the table output to keep things readable and manageable", "message_type": 0, "private": false, "created_at": 1704067200}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"messages", "list", "123"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("messages list failed: %v", err)
	}

	if !strings.Contains(output, "...") {
		t.Errorf("expected truncated content with '...', got: %s", output)
	}
}

func TestMessagesCreateCommand_AttachmentNotFound(t *testing.T) {
	handler := newRouteHandler()
	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"messages", "create", "123", "--attachment", "/nonexistent/file.pdf"})
	if err == nil {
		t.Error("expected error for nonexistent attachment")
	}
	if !strings.Contains(err.Error(), "failed to access attachment") {
		t.Errorf("expected 'failed to access attachment' error, got: %v", err)
	}
}

func TestMessagesCreateCommand_TooManyAttachments(t *testing.T) {
	handler := newRouteHandler()
	setupTestEnvWithHandler(t, handler)

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test-attachment-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_, _ = tmpFile.WriteString("test content")
	_ = tmpFile.Close()

	// Build args with too many attachments (>10)
	args := []string{"messages", "create", "123", "--content", "test"}
	for i := 0; i < 15; i++ {
		args = append(args, "--attachment", tmpFile.Name())
	}

	execErr := Execute(context.Background(), args)
	if execErr == nil {
		t.Error("expected error for too many attachments")
	}
	if !strings.Contains(execErr.Error(), "too many attachments") {
		t.Errorf("expected 'too many attachments' error, got: %v", execErr)
	}
}

func TestMessagesCreateCommand_WithAttachment(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"id": 789,
			"content": "See attached",
			"message_type": 1,
			"private": false,
			"attachments": [{"id": 1, "file_type": "file", "data_url": "https://example.com/file.txt"}]
		}`))

	setupTestEnvWithHandler(t, handler)

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test-attachment-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_, _ = tmpFile.WriteString("test content for upload")
	_ = tmpFile.Close()

	output := captureStdout(t, func() {
		execErr := Execute(context.Background(), []string{"messages", "create", "123", "--content", "See attached", "--attachment", tmpFile.Name()})
		if execErr != nil {
			t.Errorf("messages create with attachment failed: %v", execErr)
		}
	})

	if !strings.Contains(output, "Created message 789") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestMessagesCreateCommand_AttachmentOnly(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"id": 790,
			"content": "",
			"message_type": 1,
			"private": false,
			"attachments": [{"id": 1, "file_type": "image", "data_url": "https://example.com/image.png"}]
		}`))

	setupTestEnvWithHandler(t, handler)

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test-image-*.png")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	_, _ = tmpFile.WriteString("fake image content")
	_ = tmpFile.Close()

	output := captureStdout(t, func() {
		execErr := Execute(context.Background(), []string{"messages", "create", "123", "--attachment", tmpFile.Name()})
		if execErr != nil {
			t.Errorf("messages create with attachment only failed: %v", execErr)
		}
	})

	if !strings.Contains(output, "Created message 790") {
		t.Errorf("expected success message, got: %s", output)
	}
	if !strings.Contains(output, "Attachments: 1") {
		t.Errorf("expected attachment info, got: %s", output)
	}
}

func TestMessagesListSinceLastAgent(t *testing.T) {
	tests := []struct {
		name           string
		messages       string
		expectedCount  int
		expectedIDs    []float64
		expectedOutput string
	}{
		{
			name: "filters to messages after last agent reply",
			messages: `{
				"payload": [
					{"id": 1, "content": "Customer hello", "message_type": 0, "private": false, "created_at": 1704067200},
					{"id": 2, "content": "Agent reply", "message_type": 1, "private": false, "created_at": 1704067300},
					{"id": 3, "content": "Customer follow-up", "message_type": 0, "private": false, "created_at": 1704067400},
					{"id": 4, "content": "Customer question", "message_type": 0, "private": false, "created_at": 1704067500}
				]
			}`,
			expectedCount: 2,
			expectedIDs:   []float64{3, 4},
		},
		{
			name: "returns empty when agent message is last",
			messages: `{
				"payload": [
					{"id": 1, "content": "Customer hello", "message_type": 0, "private": false, "created_at": 1704067200},
					{"id": 2, "content": "Agent reply", "message_type": 1, "private": false, "created_at": 1704067300}
				]
			}`,
			expectedCount: 0,
			expectedIDs:   nil,
		},
		{
			name: "returns all when no agent messages",
			messages: `{
				"payload": [
					{"id": 1, "content": "Customer hello", "message_type": 0, "private": false, "created_at": 1704067200},
					{"id": 2, "content": "Customer follow-up", "message_type": 0, "private": false, "created_at": 1704067300}
				]
			}`,
			expectedCount: 2,
			expectedIDs:   []float64{1, 2},
		},
		{
			name: "handles single customer message after agent",
			messages: `{
				"payload": [
					{"id": 1, "content": "Agent initial", "message_type": 1, "private": false, "created_at": 1704067200},
					{"id": 2, "content": "Customer reply", "message_type": 0, "private": false, "created_at": 1704067300}
				]
			}`,
			expectedCount: 1,
			expectedIDs:   []float64{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newRouteHandler().
				On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, tt.messages))

			setupTestEnvWithHandler(t, handler)

			output := captureStdout(t, func() {
				err := Execute(context.Background(), []string{"messages", "list", "123", "--since-last-agent", "-o", "json"})
				if err != nil {
					t.Errorf("messages list --since-last-agent failed: %v", err)
				}
			})

			items := decodeItems(t, output)
			if len(items) != tt.expectedCount {
				t.Errorf("expected %d messages, got %d", tt.expectedCount, len(items))
			}

			if tt.expectedIDs != nil {
				for i, expectedID := range tt.expectedIDs {
					if i >= len(items) {
						t.Errorf("missing expected message at index %d", i)
						continue
					}
					if items[i]["id"] != expectedID {
						t.Errorf("expected message id %v at index %d, got %v", expectedID, i, items[i]["id"])
					}
				}
			}
		})
	}
}

func TestMessagesListSinceLastAgent_TextOutput(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Customer hello", "message_type": 0, "private": false, "created_at": 1704067200},
				{"id": 2, "content": "Agent reply", "message_type": 1, "private": false, "created_at": 1704067300},
				{"id": 3, "content": "New question", "message_type": 0, "private": false, "created_at": 1704067400}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"messages", "list", "123", "--since-last-agent"})
		if err != nil {
			t.Errorf("messages list --since-last-agent failed: %v", err)
		}
	})

	// Should only show message 3
	if !strings.Contains(output, "New question") {
		t.Errorf("expected 'New question' in output, got: %s", output)
	}
	// Should NOT show the agent reply
	if strings.Contains(output, "Agent reply") {
		t.Errorf("did not expect 'Agent reply' in output, got: %s", output)
	}
}

func TestMessagesBatchSend_InvalidConcurrency(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.Write([]byte(`[{"conversation_id":123,"content":"hello"}]`))
		_ = w.Close()
	}()

	err = Execute(context.Background(), []string{"messages", "batch-send", "--concurrency", "0"})
	if err == nil {
		t.Fatal("expected error for invalid --concurrency")
	}
	if !strings.Contains(err.Error(), "--concurrency must be greater than 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}
