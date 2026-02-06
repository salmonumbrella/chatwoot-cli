package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

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
