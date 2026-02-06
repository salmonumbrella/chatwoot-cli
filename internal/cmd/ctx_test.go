package cmd

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCtxCommand_Agent_WithURL(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"contact_id": 0,
			"status": "open",
			"inbox_id": 1,
			"created_at": 1700000000
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Hello", "message_type": 0, "private": false, "created_at": 1700000001}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"ctx", "https://app.chatwoot.com/app/accounts/1/conversations/123", "-o", "agent"})
		if err != nil {
			t.Fatalf("ctx failed: %v", err)
		}
	})

	var payload struct {
		Kind string         `json:"kind"`
		Item map[string]any `json:"item"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	if payload.Kind != "ctx" {
		t.Fatalf("expected kind ctx, got %q", payload.Kind)
	}
	if _, ok := payload.Item["messages"]; !ok {
		t.Fatalf("expected messages in payload item, got %#v", payload.Item)
	}
}
