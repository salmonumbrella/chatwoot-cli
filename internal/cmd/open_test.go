package cmd

import (
	"context"
	"strings"
	"testing"
)

func TestOpenCommand_Conversation(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"display_id": 123,
			"inbox_id": 1,
			"contact_id": 456,
			"status": "open",
			"priority": "high",
			"unread_count": 5,
			"muted": false,
			"created_at": 1609459200,
			"labels": ["urgent", "vip"]
		}`))

	env := setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"open", env.server.URL + "/app/accounts/1/conversations/123"})
		if err != nil {
			t.Errorf("open conversation failed: %v", err)
		}
	})

	if !strings.Contains(output, "Conversation #123") {
		t.Errorf("output missing 'Conversation #123': %s", output)
	}
	if !strings.Contains(output, "Status:     open") {
		t.Errorf("output missing status: %s", output)
	}
	if !strings.Contains(output, "Priority:   high") {
		t.Errorf("output missing priority: %s", output)
	}
}

func TestOpenCommand_ConversationJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"status": "open"
		}`))

	env := setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"open", env.server.URL + "/app/accounts/1/conversations/123", "--output", "json"})
		if err != nil {
			t.Errorf("open conversation JSON failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing 'id': %s", output)
	}
	if !strings.Contains(output, `"status"`) {
		t.Errorf("JSON output missing 'status': %s", output)
	}
}

func TestOpenCommand_Contact(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {
				"id": 456,
				"name": "John Doe",
				"email": "john@example.com",
				"phone_number": "+1234567890"
			}
		}`))

	env := setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"open", env.server.URL + "/app/accounts/1/contacts/456"})
		if err != nil {
			t.Errorf("open contact failed: %v", err)
		}
	})

	if !strings.Contains(output, "Contact #456") {
		t.Errorf("output missing 'Contact #456': %s", output)
	}
	if !strings.Contains(output, "John Doe") {
		t.Errorf("output missing name: %s", output)
	}
	if !strings.Contains(output, "john@example.com") {
		t.Errorf("output missing email: %s", output)
	}
}

func TestOpenCommand_ContactByTypeArg(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {
				"id": 456,
				"name": "John Doe",
				"email": "john@example.com",
				"phone_number": "+1234567890"
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"open", "contact", "456"})
		if err != nil {
			t.Errorf("open contact by type arg failed: %v", err)
		}
	})

	if !strings.Contains(output, "Contact #456") {
		t.Errorf("output missing 'Contact #456': %s", output)
	}
}

func TestOpenCommand_ContactByTypeFlag(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/789", jsonResponse(200, `{
			"payload": {
				"id": 789,
				"name": "Jane Doe",
				"email": "jane@example.com",
				"phone_number": "+1234567890"
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"open", "789", "--type", "contact"})
		if err != nil {
			t.Errorf("open contact by type flag failed: %v", err)
		}
	})

	if !strings.Contains(output, "Contact #789") {
		t.Errorf("output missing 'Contact #789': %s", output)
	}
}

func TestOpenCommand_IDMissingType(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/456", jsonResponse(200, `{
			"id": 456,
			"display_id": 456,
			"inbox_id": 1,
			"status": "open",
			"muted": false,
			"unread_count": 0,
			"created_at": 1609459200
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"open", "456"}); err != nil {
			t.Fatalf("open bare ID failed: %v", err)
		}
	})
	if !strings.Contains(output, "Conversation #456") {
		t.Errorf("output missing 'Conversation #456': %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"open", "#456"}); err != nil {
			t.Fatalf("open hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "Conversation #456") {
		t.Errorf("output missing 'Conversation #456': %s", output2)
	}

	output3 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"open", "conv:456"}); err != nil {
			t.Fatalf("open prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output3, "Conversation #456") {
		t.Errorf("output missing 'Conversation #456': %s", output3)
	}
}

func TestOpenCommand_TypedIDPrefixes(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/789", jsonResponse(200, `{
			"payload": {
				"id": 789,
				"name": "Jane Doe",
				"email": "jane@example.com",
				"phone_number": "+1234567890"
			}
		}`)).
		On("GET", "/api/v1/accounts/1/teams/5", jsonResponse(200, `{
			"id": 5,
			"name": "Support Team",
			"description": "Primary support team"
		}`))

	setupTestEnvWithHandler(t, handler)

	out := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"open", "contact:789"}); err != nil {
			t.Fatalf("open contact:789 failed: %v", err)
		}
	})
	if !strings.Contains(out, "Contact #789") {
		t.Errorf("output missing 'Contact #789': %s", out)
	}

	out2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"open", "team:5"}); err != nil {
			t.Fatalf("open team:5 failed: %v", err)
		}
	})
	if !strings.Contains(out2, "Team #5") {
		t.Errorf("output missing 'Team #5': %s", out2)
	}
}

func TestOpenCommand_Inbox(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Website Chat",
			"channel_type": "Channel::WebWidget",
			"enable_auto_assignment": true,
			"greeting_enabled": false
		}`))

	env := setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"open", env.server.URL + "/app/accounts/1/inboxes/1"})
		if err != nil {
			t.Errorf("open inbox failed: %v", err)
		}
	})

	if !strings.Contains(output, "Inbox #1") {
		t.Errorf("output missing 'Inbox #1': %s", output)
	}
	if !strings.Contains(output, "Website Chat") {
		t.Errorf("output missing name: %s", output)
	}
}

func TestOpenCommand_Team(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams/5", jsonResponse(200, `{
			"id": 5,
			"name": "Support Team",
			"description": "Primary support team"
		}`))

	env := setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"open", env.server.URL + "/app/accounts/1/teams/5"})
		if err != nil {
			t.Errorf("open team failed: %v", err)
		}
	})

	if !strings.Contains(output, "Team #5") {
		t.Errorf("output missing 'Team #5': %s", output)
	}
	if !strings.Contains(output, "Support Team") {
		t.Errorf("output missing name: %s", output)
	}
}

func TestOpenCommand_Agent(t *testing.T) {
	// GetAgent calls ListAgents and finds the agent by ID
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 10, "name": "Jane Agent", "email": "jane@example.com", "role": "agent"},
			{"id": 20, "name": "Other Agent", "email": "other@example.com", "role": "admin"}
		]`))

	env := setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"open", env.server.URL + "/app/accounts/1/agents/10"})
		if err != nil {
			t.Errorf("open agent failed: %v", err)
		}
	})

	if !strings.Contains(output, "Agent #10") {
		t.Errorf("output missing 'Agent #10': %s", output)
	}
	if !strings.Contains(output, "Jane Agent") {
		t.Errorf("output missing name: %s", output)
	}
}

func TestOpenCommand_Campaign(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/campaigns/7", jsonResponse(200, `{
			"id": 7,
			"title": "Welcome Campaign",
			"message": "Welcome to our service!",
			"inbox_id": 1,
			"enabled": true
		}`))

	env := setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"open", env.server.URL + "/app/accounts/1/campaigns/7"})
		if err != nil {
			t.Errorf("open campaign failed: %v", err)
		}
	})

	if !strings.Contains(output, "Campaign #7") {
		t.Errorf("output missing 'Campaign #7': %s", output)
	}
	if !strings.Contains(output, "Welcome Campaign") {
		t.Errorf("output missing title: %s", output)
	}
}

func TestOpenCommand_InvalidURL(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{
			name:    "missing scheme",
			url:     "app.chatwoot.com/app/accounts/1/conversations/123",
			wantErr: "missing scheme",
		},
		{
			name:    "invalid format",
			url:     "https://app.chatwoot.com/some/random/path",
			wantErr: "invalid Chatwoot URL format",
		},
		{
			name:    "unsupported resource type",
			url:     "https://app.chatwoot.com/app/accounts/1/widgets/123",
			wantErr: "unsupported resource type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Execute(context.Background(), []string{"open", tt.url})
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want error containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestOpenCommand_AccountMismatch(t *testing.T) {
	// The test environment sets CHATWOOT_ACCOUNT_ID=1
	// Try to open a URL with account_id=99
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"open", "https://app.chatwoot.com/app/accounts/99/conversations/123"})
	if err == nil {
		t.Fatal("expected error for account mismatch")
	}
	if !strings.Contains(err.Error(), "does not match authenticated account ID") {
		t.Errorf("error = %q, want error about account mismatch", err.Error())
	}
}

func TestOpenCommand_NoResourceID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"open", "https://app.chatwoot.com/app/accounts/1/conversations"})
	if err == nil {
		t.Fatal("expected error for missing resource ID")
	}
	if !strings.Contains(err.Error(), "must include a resource ID") {
		t.Errorf("error = %q, want error about missing resource ID", err.Error())
	}
}

func TestOpenCommand_NoArgs(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"open"})
	if err == nil {
		t.Fatal("expected error for missing URL argument")
	}
}

func TestOpenCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/999", jsonResponse(404, `{"error": "Conversation not found"}`))

	env := setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"open", env.server.URL + "/app/accounts/1/conversations/999"})
	if err == nil {
		t.Fatal("expected error for API 404")
	}
	if !strings.Contains(err.Error(), "failed to get conversation") {
		t.Errorf("error = %q, want error about failed to get conversation", err.Error())
	}
}

func TestOpenCommand_URLWithQueryAndFragment(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"status": "open"
		}`))

	env := setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		// URL with query string and fragment should still work
		err := Execute(context.Background(), []string{"open", env.server.URL + "/app/accounts/1/conversations/123?tab=messages#bottom"})
		if err != nil {
			t.Errorf("open with query/fragment failed: %v", err)
		}
	})

	if !strings.Contains(output, "Conversation #123") {
		t.Errorf("output missing 'Conversation #123': %s", output)
	}
}
