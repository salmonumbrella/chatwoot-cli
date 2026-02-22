package cmd

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMentionsListCommand(t *testing.T) {
	// Set up mock routes for:
	// 1. GetProfile to get user ID
	// 2. ListConversations to get conversations to search
	// 3. ListAllMessages to search for mentions in each conversation
	handler := newRouteHandler().
		On("GET", "/api/v1/profile", jsonResponse(200, `{
			"id": 42,
			"name": "Test Agent",
			"email": "agent@example.com"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"meta": {"total_pages": 1, "current_page": 1},
				"payload": [
					{"id": 100, "status": "open", "last_activity_at": 1704153600}
				]
			}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/100/messages", jsonResponse(200, `{
			"payload": [
				{
					"id": 1,
					"content": "Hey mention://user/42/TestAgent check this out",
					"message_type": 2,
					"private": true,
					"created_at": 1704067200,
					"sender": {"id": 10, "name": "Other Agent"}
				},
				{
					"id": 2,
					"content": "Regular message",
					"message_type": 1,
					"private": false,
					"created_at": 1704067100
				}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"mentions", "list"})
		if err != nil {
			t.Errorf("mentions list failed: %v", err)
		}
	})

	// Should show the mention content
	if !strings.Contains(output, "mention://user/42") {
		t.Errorf("output missing mention content: %s", output)
	}
	// Should show the sender name
	if !strings.Contains(output, "Other Agent") {
		t.Errorf("output missing sender name: %s", output)
	}
	// Should have table headers
	if !strings.Contains(output, "CONV") || !strings.Contains(output, "MSG") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestMentionsListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/profile", jsonResponse(200, `{
			"id": 42,
			"name": "Test Agent",
			"email": "agent@example.com"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"meta": {"total_pages": 1, "current_page": 1},
				"payload": [
					{"id": 100, "status": "open", "last_activity_at": 1704153600}
				]
			}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/100/messages", jsonResponse(200, `{
			"payload": [
				{
					"id": 1,
					"content": "Hey mention://user/42/TestAgent check this out",
					"message_type": 2,
					"private": true,
					"created_at": 1704067200,
					"sender": {"id": 10, "name": "Other Agent"}
				}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"mentions", "list", "-o", "json"})
		if err != nil {
			t.Errorf("mentions list --json failed: %v", err)
		}
	})

	mentions := decodeItems(t, output)
	if len(mentions) != 1 {
		t.Fatalf("expected 1 mention, got %d", len(mentions))
	}

	m := mentions[0]
	if m["conversation_id"] != float64(100) {
		t.Errorf("expected conversation_id 100, got %v", m["conversation_id"])
	}
	if m["message_id"] != float64(1) {
		t.Errorf("expected message_id 1, got %v", m["message_id"])
	}
	if m["content"] != "Hey mention://user/42/TestAgent check this out" {
		t.Errorf("expected content with mention, got %v", m["content"])
	}
	if m["sender_name"] != "Other Agent" {
		t.Errorf("expected sender_name 'Other Agent', got %v", m["sender_name"])
	}
	if m["created_at"] == nil {
		t.Error("expected created_at to be present")
	}
}

func TestMentionsListCommand_InvalidConversationID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"mentions", "list", "--conversation-id", "-1"})
	if err == nil {
		t.Error("expected error for negative conversation ID")
	}
	if !strings.Contains(err.Error(), "conversation-id must be a positive integer") {
		t.Errorf("expected 'conversation-id must be a positive integer' error, got: %v", err)
	}
}

func TestMentionsListCommand_InvalidLimit(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"mentions", "list", "--limit", "0"})
	if err == nil {
		t.Error("expected error for zero limit")
	}
	if !strings.Contains(err.Error(), "limit must be at least 1") {
		t.Errorf("expected 'limit must be at least 1' error, got: %v", err)
	}

	err = Execute(context.Background(), []string{"mentions", "list", "--limit", "-5"})
	if err == nil {
		t.Error("expected error for negative limit")
	}
}

func TestMentionsListCommand_InvalidSince(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	tests := []struct {
		name  string
		since string
	}{
		{"invalid format", "abc"},
		{"negative days", "-1d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Execute(context.Background(), []string{"mentions", "list", "--since", tt.since})
			if err == nil {
				t.Errorf("expected error for invalid --since value: %s", tt.since)
			}
		})
	}
}

func TestMentionsListCommand_WithFilters(t *testing.T) {
	var profileCalled, convCalled bool

	handler := newRouteHandler().
		On("GET", "/api/v1/profile", func(w http.ResponseWriter, r *http.Request) {
			profileCalled = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id": 42, "name": "Test Agent"}`))
		}).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", func(w http.ResponseWriter, r *http.Request) {
			convCalled = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"payload": []}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"mentions", "list",
			"--conversation-id", "123",
			"--limit", "10",
		})
		if err != nil {
			t.Errorf("mentions list with filters failed: %v", err)
		}
	})

	if !profileCalled {
		t.Error("expected profile API to be called")
	}

	if !convCalled {
		t.Error("expected conversation messages API to be called")
	}

	// Should show "No mentions found" for empty result
	if !strings.Contains(output, "No mentions found") {
		t.Errorf("expected 'No mentions found' message, got: %s", output)
	}
}

func TestMentionsListCommand_NoMentions(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/profile", jsonResponse(200, `{
			"id": 42,
			"name": "Test Agent"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"meta": {"total_pages": 0},
				"payload": []
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"mentions", "list"})
		if err != nil {
			t.Errorf("mentions list failed: %v", err)
		}
	})

	if !strings.Contains(output, "No mentions found") {
		t.Errorf("expected 'No mentions found' message, got: %s", output)
	}
}

func TestMentionsListCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/profile", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"mentions", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

// Tests for parseDuration function
func TestParseDuration_ValidInputs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{"24 hours", "24h", 24 * time.Hour},
		{"1 hour", "1h", 1 * time.Hour},
		{"7 days", "7d", 7 * 24 * time.Hour},
		{"1 day", "1d", 24 * time.Hour},
		{"1 week", "1w", 7 * 24 * time.Hour},
		{"2 weeks", "2w", 14 * 24 * time.Hour},
		{"30 minutes", "30m", 30 * time.Minute},
		{"90 seconds", "90s", 90 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDuration(tt.input)
			if err != nil {
				t.Errorf("parseDuration(%q) unexpected error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDuration_InvalidInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"invalid format", "abc"},
		{"negative days", "-1d"},
		{"zero days", "0d"},
		{"negative weeks", "-1w"},
		{"zero weeks", "0w"},
		{"empty string", ""},
		{"only whitespace", "   "},
		{"mixed invalid", "1x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDuration(tt.input)
			if err == nil {
				t.Errorf("parseDuration(%q) expected error, got nil", tt.input)
			}
		})
	}
}

func TestParseDuration_Whitespace(t *testing.T) {
	// Test that whitespace is trimmed
	result, err := parseDuration("  24h  ")
	if err != nil {
		t.Errorf("parseDuration with whitespace failed: %v", err)
	}
	if result != 24*time.Hour {
		t.Errorf("expected 24h, got %v", result)
	}
}
