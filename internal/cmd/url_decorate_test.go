package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestAgentOutputIncludesConversationURL(t *testing.T) {
	var baseURL string
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":123,"account_id":1,"inbox_id":1,"status":"open","created_at":1700000000}`))
		})

	env := setupTestEnvWithHandler(t, handler)
	baseURL = env.server.URL

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "-o", "agent"})
		if err != nil {
			t.Fatalf("conversations get -o agent failed: %v", err)
		}
	})

	var payload struct {
		Kind string         `json:"kind"`
		Item map[string]any `json:"item"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if payload.Kind != "conversations.get" {
		t.Fatalf("expected kind conversations.get, got %q", payload.Kind)
	}
	if payload.Item["url"] != baseURL+"/app/accounts/1/conversations/123" {
		t.Fatalf("expected url %q, got %#v", baseURL+"/app/accounts/1/conversations/123", payload.Item["url"])
	}
}
