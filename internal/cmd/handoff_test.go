package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
)

func TestHandoffCommandExists(t *testing.T) {
	err := Execute(context.Background(), []string{"handoff", "--help"})
	if err != nil {
		t.Fatalf("handoff --help failed: %v", err)
	}
}

func TestHandoffRequiresAssignment(t *testing.T) {
	err := Execute(context.Background(), []string{"handoff", "123", "--reason", "test"})
	if err == nil {
		t.Fatal("expected error when no --agent or --team specified")
	}
	if !strings.Contains(err.Error(), "--agent or --team") {
		t.Fatalf("expected agent/team validation error, got: %v", err)
	}
}

func TestHandoffRejectsInvalidReason(t *testing.T) {
	longReason := strings.Repeat("a", validation.MaxMessageLength+1)

	err := Execute(context.Background(), []string{"handoff", "123", "--team", "2", "--reason", longReason})
	if err == nil {
		t.Fatal("expected message validation error")
	}
	if !strings.Contains(err.Error(), "message content exceeds maximum size") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandoffDryRunWithAgentNameAvoidsLookups(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	t.Setenv("CHATWOOT_BASE_URL", server.URL)
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_NO_KEYCHAIN", "1")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"handoff", "123",
			"--agent", "Lily Hu",
			"--team", "Billing",
			"--priority", "urgent",
			"--reason", "Escalate to billing",
			"--dry-run", "-o", "json",
		})
		if err != nil {
			t.Fatalf("handoff --dry-run failed: %v", err)
		}
	})

	if requestCount != 0 {
		t.Fatalf("dry-run should not make lookup or mutation HTTP requests, got %d", requestCount)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	details, ok := payload["details"].(map[string]any)
	if !ok {
		t.Fatalf("expected details object, got %#v", payload["details"])
	}
	if details["agent"] != "Lily Hu" {
		t.Fatalf("expected agent name preview, got %#v", details["agent"])
	}
	if details["team"] != "Billing" {
		t.Fatalf("expected team name preview, got %#v", details["team"])
	}
	if details["priority"] != "urgent" {
		t.Fatalf("expected priority preview, got %#v", details["priority"])
	}
	if details["reason"] != "Escalate to billing" {
		t.Fatalf("expected reason preview, got %#v", details["reason"])
	}
}

func TestHandoffLightOutput(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"id": 700,
			"content": "Escalate to billing",
			"message_type": 1,
			"private": true,
			"created_at": 1704067200
		}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/assignments", jsonResponse(200, `{"success": true}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_priority", jsonResponse(200, `{}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"status": "open",
			"inbox_id": 48,
			"unread_count": 0,
			"assignee_id": 5,
			"team_id": 2,
			"priority": "high",
			"created_at": 1704067200
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"handoff", "123", "--agent", "5", "--team", "2", "--priority", "urgent", "--reason", "Escalate to billing", "--li"})
		if err != nil {
			t.Fatalf("handoff --li failed: %v", err)
		}
	})

	var payload struct {
		ID        int    `json:"id"`
		MessageID int    `json:"mid,omitempty"`
		AgentID   *int   `json:"ag,omitempty"`
		TeamID    *int   `json:"tm,omitempty"`
		Priority  string `json:"pri,omitempty"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	if payload.ID != 123 {
		t.Fatalf("expected conversation id 123, got %d", payload.ID)
	}
	if payload.AgentID == nil || *payload.AgentID != 5 {
		t.Fatalf("expected agent id 5, got %#v", payload.AgentID)
	}
	if payload.TeamID == nil || *payload.TeamID != 2 {
		t.Fatalf("expected team id 2, got %#v", payload.TeamID)
	}
	if payload.Priority != "h" {
		t.Fatalf("expected short priority h, got %q", payload.Priority)
	}
	if payload.MessageID != 700 {
		t.Fatalf("expected message id 700, got %d", payload.MessageID)
	}
}
