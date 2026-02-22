package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestNoteCommand_Pending(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{"id": 99, "conversation_id": 123, "content": "Internal note", "message_type": 1, "private": true}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"payload": {"success": true, "current_status": "pending", "conversation_id": 123}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"note", "123", "Internal note", "--pending", "-o", "json"})
		if err != nil {
			t.Fatalf("note --pending failed: %v", err)
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

func TestNoteCommand_WithMention(t *testing.T) {
	var received map[string]any
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 7, "name": "Lily Chen", "email": "lily@example.com", "role": "agent"}
		]`)).
		On("POST", "/api/v1/accounts/1/conversations/123/messages", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&received)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 99, "conversation_id": 123, "content": "ok", "message_type": 1, "private": true}`))
		})

	setupTestEnvWithHandler(t, handler)

	_ = captureStdout(t, func() {
		err := Execute(context.Background(), []string{"note", "123", "--mention", "lily", "Please review", "-o", "json"})
		if err != nil {
			t.Fatalf("note failed: %v", err)
		}
	})

	content, _ := received["content"].(string)
	if !strings.Contains(content, "mention://user/7/") {
		t.Fatalf("expected content to include mention URL, got %q", content)
	}
	if received["private"] != true {
		t.Fatalf("expected private true, got %#v", received["private"])
	}
}

func TestNoteCommand_WithMentionAliasMTAndLight(t *testing.T) {
	var received map[string]any
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 7, "name": "Lily Chen", "email": "lily@example.com", "role": "agent"}
		]`)).
		On("POST", "/api/v1/accounts/1/conversations/123/messages", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&received)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 99, "conversation_id": 123, "content": "ok", "message_type": 1, "private": true}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"note", "123", "--mt", "lily", "Please review", "--light", "-o", "agent"})
		if err != nil {
			t.Fatalf("note --mt --light failed: %v", err)
		}
	})

	content, _ := received["content"].(string)
	if !strings.Contains(content, "mention://user/7/") {
		t.Fatalf("expected content to include mention URL, got %q", content)
	}
	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"item"`) {
		t.Fatalf("light output should bypass agent envelope, got: %s", output)
	}

	var result struct {
		ID        int `json:"id"`
		MessageID int `json:"mid"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse light output: %v\noutput: %s", err, output)
	}
	if result.ID != 123 {
		t.Fatalf("expected id 123, got %d", result.ID)
	}
	if result.MessageID != 99 {
		t.Fatalf("expected mid 99, got %d", result.MessageID)
	}
}

func TestNoteCommand_AgentOutput_CompactAliases(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{"id": 99, "conversation_id": 123, "content": "Internal note", "message_type": 1, "private": true}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"payload": {"success": true, "current_status": "pending", "conversation_id": 123}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"note", "123", "Internal note", "--pending", "-o", "agent"})
		if err != nil {
			t.Fatalf("note --pending -o agent failed: %v", err)
		}
	})

	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"item"`) || strings.Contains(output, `"data"`) {
		t.Fatalf("agent output should be flat summary, got: %s", output)
	}
	var result struct {
		ID  int    `json:"id"`
		Mid int    `json:"mid"`
		Prv bool   `json:"prv"`
		St  string `json:"st"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse compact output: %v\noutput: %s", err, output)
	}
	if result.ID != 123 || result.Mid != 99 || !result.Prv || result.St != "p" {
		t.Fatalf("unexpected compact note payload: %#v", result)
	}
}

func TestNoteCommand_ResolveAndPendingExclusive(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{"id": 99, "conversation_id": 123, "content": "conflict", "message_type": 1, "private": true}`))

	setupTestEnvWithHandler(t, handler)

	assertStatusFlagsMutuallyExclusive(t, []string{"note", "123", "conflict", "--resolve", "--pending"})
}
