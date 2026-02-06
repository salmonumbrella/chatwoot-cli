package cmd

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCloseCommand_JSONSummary(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"payload": {"success": true, "current_status": "resolved", "conversation_id": 123}
		}`)).
		On("POST", "/api/v1/accounts/1/conversations/456/toggle_status", jsonResponse(200, `{
			"payload": {"success": true, "current_status": "resolved", "conversation_id": 456}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"close", "123", "456", "-o", "json"})
		if err != nil {
			t.Fatalf("close failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["closed"] != float64(2) {
		t.Fatalf("expected closed=2, got %#v", result["closed"])
	}
	if result["total"] != float64(2) {
		t.Fatalf("expected total=2, got %#v", result["total"])
	}
}

func TestReopenCommand_JSONSummary(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"payload": {"success": true, "current_status": "open", "conversation_id": 123}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"reopen", "123", "-o", "json"})
		if err != nil {
			t.Fatalf("reopen failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["reopened"] != float64(1) {
		t.Fatalf("expected reopened=1, got %#v", result["reopened"])
	}
	if result["total"] != float64(1) {
		t.Fatalf("expected total=1, got %#v", result["total"])
	}
}

func TestCloseCommand_AcceptsURLAndHashIDs(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"payload": {"success": true, "current_status": "resolved", "conversation_id": 123}
		}`)).
		On("POST", "/api/v1/accounts/1/conversations/456/toggle_status", jsonResponse(200, `{
			"payload": {"success": true, "current_status": "resolved", "conversation_id": 456}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"close",
			"https://app.chatwoot.com/app/accounts/1/conversations/123",
			"#456",
			"-o", "json",
		})
		if err != nil {
			t.Fatalf("close failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["closed"] != float64(2) {
		t.Fatalf("expected closed=2, got %#v", result["closed"])
	}
	if result["total"] != float64(2) {
		t.Fatalf("expected total=2, got %#v", result["total"])
	}
}
