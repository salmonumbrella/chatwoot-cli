package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestCommentCommand_SendsMessage(t *testing.T) {
	var received map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&received)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 55, "conversation_id": 123, "content": "Hello", "message_type": 1, "private": false}`))
		})

	setupTestEnvWithHandler(t, handler)

	_ = captureStdout(t, func() {
		err := Execute(context.Background(), []string{"comment", "123", "Hello", "-o", "json"})
		if err != nil {
			t.Fatalf("comment failed: %v", err)
		}
	})

	if received["content"] != "Hello" {
		t.Fatalf("expected content Hello, got %#v", received["content"])
	}
	if received["private"] != false {
		t.Fatalf("expected private false, got %#v", received["private"])
	}
	if received["message_type"] != "outgoing" {
		t.Fatalf("expected message_type outgoing, got %#v", received["message_type"])
	}
}

func TestCommentCommand_LightOutput(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{"id": 55, "conversation_id": 123, "content": "Hello", "message_type": 1, "private": false}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"payload": {"success": true, "current_status": "pending", "conversation_id": 123}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"comment", "123", "Hello", "--pending", "--light", "-o", "agent"})
		if err != nil {
			t.Fatalf("comment --pending --light failed: %v", err)
		}
	})

	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"item"`) {
		t.Fatalf("light output should bypass agent envelope, got: %s", output)
	}
	if strings.Contains(output, "\n  ") {
		t.Fatalf("light output should be compact by default, got: %s", output)
	}

	var result struct {
		ID        int    `json:"id"`
		MessageID int    `json:"mid"`
		Status    string `json:"st"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse light output: %v\noutput: %s", err, output)
	}
	if result.ID != 123 {
		t.Fatalf("expected id 123, got %d", result.ID)
	}
	if result.MessageID != 55 {
		t.Fatalf("expected mid 55, got %d", result.MessageID)
	}
	if result.Status != "p" {
		t.Fatalf("expected short status p, got %q", result.Status)
	}
}

func TestCommentCommand_AgentOutput_CompactAliases(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{"id": 55, "conversation_id": 123, "content": "Hello", "message_type": 1, "private": false}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"payload": {"success": true, "current_status": "pending", "conversation_id": 123}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"comment", "123", "Hello", "--pending", "-o", "agent"})
		if err != nil {
			t.Fatalf("comment --pending -o agent failed: %v", err)
		}
	})

	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"item"`) || strings.Contains(output, `"data"`) {
		t.Fatalf("agent output should be flat summary, got: %s", output)
	}
	var result struct {
		ID     int    `json:"id"`
		Mid    int    `json:"mid"`
		Status string `json:"st"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse compact output: %v\noutput: %s", err, output)
	}
	if result.ID != 123 || result.Mid != 55 || result.Status != "p" {
		t.Fatalf("unexpected compact comment payload: %#v", result)
	}
}

func TestCommentAcceptsSideEffectFlags(t *testing.T) {
	for _, name := range []string{"label", "priority", "snooze-for"} {
		if rootCmd := newCommentCmd(); rootCmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing --%s flag on comment command", name)
		}
	}
}

func TestNoteAcceptsSideEffectFlags(t *testing.T) {
	for _, name := range []string{"label", "priority", "snooze-for"} {
		if rootCmd := newNoteCmd(); rootCmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing --%s flag on note command", name)
		}
	}
}

func TestReplyAcceptsSideEffectFlags(t *testing.T) {
	for _, name := range []string{"label", "priority", "snooze-for"} {
		if rootCmd := newReplyCmd(); rootCmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing --%s flag on reply command", name)
		}
	}
}

func TestCommentCommand_Resolve(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{"id": 55, "conversation_id": 123, "content": "Done", "message_type": 1, "private": false}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"payload": {"success": true, "current_status": "resolved", "conversation_id": 123}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"comment", "123", "Done", "--resolve", "-o", "json"})
		if err != nil {
			t.Fatalf("comment --resolve failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["resolved"] != true {
		t.Fatalf("expected resolved true, got %#v", result["resolved"])
	}
}

func TestCommentCommand_Pending(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{"id": 56, "conversation_id": 123, "content": "Waiting on customer", "message_type": 1, "private": false}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"payload": {"success": true, "current_status": "pending", "conversation_id": 123}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"comment", "123", "Waiting on customer", "--pending", "-o", "json"})
		if err != nil {
			t.Fatalf("comment --pending failed: %v", err)
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

func TestCommentCommand_SnoozeFailureReturnsError(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{"id": 55, "conversation_id": 123, "content": "text", "message_type": 1, "private": false}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(500, `{"error": "server error"}`))

	setupTestEnvWithHandler(t, handler)

	_ = captureStdout(t, func() {
		err := Execute(context.Background(), []string{"comment", "123", "text", "--snooze-for", "2h", "-o", "json"})
		if err == nil {
			t.Fatal("expected error when snooze fails, got nil")
		}
		if !strings.Contains(err.Error(), "failed to snooze") {
			t.Fatalf("expected snooze failure error, got: %v", err)
		}
	})
}

func TestCommentCommand_ResolveAndPendingExclusive(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{"id": 57, "conversation_id": 123, "content": "conflict", "message_type": 1, "private": false}`))

	setupTestEnvWithHandler(t, handler)

	assertStatusFlagsMutuallyExclusive(t, []string{"comment", "123", "conflict", "--resolve", "--pending"})
}
