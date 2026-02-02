package cmd

import (
	"context"
	"strings"
	"testing"
)

func TestInboxesListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Email Inbox", "channel_type": "Channel::Email", "enable_auto_assignment": true},
				{"id": 2, "name": "Web Chat", "channel_type": "Channel::WebWidget", "enable_auto_assignment": false}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "list"})
		if err != nil {
			t.Errorf("inboxes list failed: %v", err)
		}
	})

	if !strings.Contains(output, "Email Inbox") {
		t.Errorf("output missing 'Email Inbox': %s", output)
	}
	if !strings.Contains(output, "Web Chat") {
		t.Errorf("output missing 'Web Chat': %s", output)
	}
	if !strings.Contains(output, "Channel::Email") {
		t.Errorf("output missing channel type: %s", output)
	}
}

func TestInboxesListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Email Inbox", "channel_type": "Channel::Email"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "list", "--output", "json"})
		if err != nil {
			t.Errorf("inboxes list --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing 'id' field: %s", output)
	}
	if !strings.Contains(output, `"name"`) {
		t.Errorf("JSON output missing 'name' field: %s", output)
	}
}

func TestInboxesListCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": []
		}`))

	setupTestEnvWithHandler(t, handler)

	// The empty message goes to stderr via the formatter, stdout should be empty
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "list"})
		if err != nil {
			t.Errorf("inboxes list failed: %v", err)
		}
	})

	// With an empty list, stdout should not contain any inbox data
	if strings.Contains(output, "Email Inbox") {
		t.Errorf("output should not contain inbox data when list is empty: %s", output)
	}
}

func TestInboxesGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Email Inbox",
			"channel_type": "Channel::Email",
			"enable_auto_assignment": true,
			"greeting_enabled": true,
			"greeting_message": "Welcome!",
			"website_url": "https://example.com"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "get", "1"})
		if err != nil {
			t.Errorf("inboxes get failed: %v", err)
		}
	})

	if !strings.Contains(output, "Email Inbox") {
		t.Errorf("output missing 'Email Inbox': %s", output)
	}
	if !strings.Contains(output, "Channel::Email") {
		t.Errorf("output missing channel type: %s", output)
	}
	if !strings.Contains(output, "Welcome!") {
		t.Errorf("output missing greeting message: %s", output)
	}
}

func TestInboxesGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Email Inbox",
			"channel_type": "Channel::Email"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "get", "1", "--output", "json"})
		if err != nil {
			t.Errorf("inboxes get --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing 'id' field: %s", output)
	}
}

func TestInboxesGetCommand_InvalidID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "get", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestInboxesGetCommand_MissingID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "get"})
	if err == nil {
		t.Error("expected error for missing ID argument")
	}
}

func TestInboxesCreateCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"id": 3,
			"name": "New Inbox",
			"channel_type": "Channel::Api"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "create", "--name", "New Inbox", "--channel-type", "Channel::Api"})
		if err != nil {
			t.Errorf("inboxes create failed: %v", err)
		}
	})

	if !strings.Contains(output, "Created inbox 3") {
		t.Errorf("output missing success message: %s", output)
	}
	if !strings.Contains(output, "New Inbox") {
		t.Errorf("output missing inbox name: %s", output)
	}
}

func TestInboxesCreateCommand_WithOptions(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"id": 4,
			"name": "Configured Inbox",
			"channel_type": "Channel::WebWidget",
			"greeting_enabled": true,
			"greeting_message": "Hello!"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"inboxes", "create",
			"--name", "Configured Inbox",
			"--channel-type", "Channel::WebWidget",
			"--greeting-enabled",
			"--greeting-message", "Hello!",
		})
		if err != nil {
			t.Errorf("inboxes create with options failed: %v", err)
		}
	})

	if !strings.Contains(output, "Created inbox 4") {
		t.Errorf("output missing success message: %s", output)
	}
}

func TestInboxesCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"id": 5,
			"name": "JSON Inbox",
			"channel_type": "Channel::Api"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "create", "--name", "JSON Inbox", "--channel-type", "Channel::Api", "--output", "json"})
		if err != nil {
			t.Errorf("inboxes create --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing 'id' field: %s", output)
	}
}

func TestInboxesCreateCommand_MissingName(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "create", "--channel-type", "Channel::Api"})
	if err == nil {
		t.Error("expected error for missing --name flag")
	}
}

func TestInboxesCreateCommand_MissingChannelType(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "create", "--name", "Test Inbox"})
	if err == nil {
		t.Error("expected error for missing --channel-type flag")
	}
}

func TestInboxesCreateCommand_InvalidAutoAssignmentConfig(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{
		"inboxes", "create",
		"--name", "Test",
		"--channel-type", "Channel::Api",
		"--auto-assignment-config", "not-valid-json",
	})
	if err == nil {
		t.Error("expected error for invalid JSON in auto-assignment-config")
	}
}

func TestInboxesUpdateCommand(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Updated Inbox",
			"channel_type": "Channel::Email"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "update", "1", "--name", "Updated Inbox"})
		if err != nil {
			t.Errorf("inboxes update failed: %v", err)
		}
	})

	if !strings.Contains(output, "Updated inbox 1") {
		t.Errorf("output missing success message: %s", output)
	}
}

func TestInboxesUpdateCommand_WithOptions(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/inboxes/2", jsonResponse(200, `{
			"id": 2,
			"name": "Full Update",
			"channel_type": "Channel::WebWidget",
			"greeting_enabled": true
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"inboxes", "update", "2",
			"--name", "Full Update",
			"--greeting-enabled",
			"--greeting-message", "Welcome!",
			"--timezone", "America/New_York",
		})
		if err != nil {
			t.Errorf("inboxes update with options failed: %v", err)
		}
	})

	if !strings.Contains(output, "Updated inbox 2") {
		t.Errorf("output missing success message: %s", output)
	}
}

func TestInboxesUpdateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Updated",
			"channel_type": "Channel::Email"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "update", "1", "--name", "Updated", "--output", "json"})
		if err != nil {
			t.Errorf("inboxes update --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing 'id' field: %s", output)
	}
}

func TestInboxesUpdateCommand_NoFlags(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "update", "1"})
	if err == nil {
		t.Error("expected error when no update flags provided")
	}
}

func TestInboxesUpdateCommand_InvalidID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "update", "invalid", "--name", "Test"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestInboxesUpdateCommand_InvalidAutoAssignmentConfig(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{
		"inboxes", "update", "1",
		"--auto-assignment-config", "not-valid-json",
	})
	if err == nil {
		t.Error("expected error for invalid JSON in auto-assignment-config")
	}
}

func TestInboxesDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "delete", "1"})
		if err != nil {
			t.Errorf("inboxes delete failed: %v", err)
		}
	})

	if !strings.Contains(output, "Deleted inbox 1") {
		t.Errorf("output missing success message: %s", output)
	}
}

func TestInboxesDeleteCommand_InvalidID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "delete", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestInboxesDeleteCommand_MissingID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "delete"})
	if err == nil {
		t.Error("expected error for missing ID argument")
	}
}

func TestInboxesAgentBotCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1/agent_bot", jsonResponse(200, `{
			"id": 10,
			"name": "Support Bot",
			"description": "Handles initial queries",
			"outgoing_url": "https://bot.example.com/webhook"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "agent-bot", "1"})
		if err != nil {
			t.Errorf("inboxes agent-bot failed: %v", err)
		}
	})

	if !strings.Contains(output, "Support Bot") {
		t.Errorf("output missing bot name: %s", output)
	}
	if !strings.Contains(output, "Handles initial queries") {
		t.Errorf("output missing description: %s", output)
	}
}

func TestInboxesAgentBotCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1/agent_bot", jsonResponse(200, `{
			"id": 10,
			"name": "Support Bot"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "agent-bot", "1", "--output", "json"})
		if err != nil {
			t.Errorf("inboxes agent-bot --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing 'id' field: %s", output)
	}
}

func TestInboxesAgentBotCommand_InvalidID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "agent-bot", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestInboxesSetAgentBotCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/inboxes/1/set_agent_bot", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "set-agent-bot", "1", "--bot-id", "10"})
		if err != nil {
			t.Errorf("inboxes set-agent-bot failed: %v", err)
		}
	})

	if !strings.Contains(output, "Assigned agent bot 10 to inbox 1") {
		t.Errorf("output missing success message: %s", output)
	}
}

func TestInboxesSetAgentBotCommand_MissingBotID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "set-agent-bot", "1"})
	if err == nil {
		t.Error("expected error for missing --bot-id flag")
	}
}

func TestInboxesSetAgentBotCommand_InvalidID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "set-agent-bot", "invalid", "--bot-id", "10"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestInboxesTriageCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Support Inbox",
			"channel_type": "Channel::Email"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 101, "status": "open", "contact_id": 1, "unread_count": 3},
					{"id": 102, "status": "pending", "contact_id": 2, "unread_count": 0}
				],
				"meta": {"count": 2}
			}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/1", jsonResponse(200, `{
			"payload": {"id": 1, "name": "John Doe", "email": "john@example.com"}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/2", jsonResponse(200, `{
			"payload": {"id": 2, "name": "Jane Doe", "email": "jane@example.com"}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/101/messages", jsonResponse(200, `[
			{"id": 1, "content": "Hello, I need help", "message_type": 0}
		]`)).
		On("GET", "/api/v1/accounts/1/conversations/102/messages", jsonResponse(200, `[
			{"id": 2, "content": "Thank you", "message_type": 1}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "triage", "1"})
		if err != nil {
			t.Errorf("inboxes triage failed: %v", err)
		}
	})

	if !strings.Contains(output, "Support Inbox") {
		t.Errorf("output missing inbox name: %s", output)
	}
	if !strings.Contains(output, "open") {
		t.Errorf("output missing 'open' status: %s", output)
	}
}

func TestInboxesTriageCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Support Inbox",
			"channel_type": "Channel::Email"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [],
				"meta": {"count": 0}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "triage", "1", "--output", "json"})
		if err != nil {
			t.Errorf("inboxes triage --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"inbox_id"`) {
		t.Errorf("JSON output missing 'inbox_id' field: %s", output)
	}
}

func TestInboxesTriageCommand_WithStatusFilter(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Support Inbox",
			"channel_type": "Channel::Email"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 103, "status": "pending", "contact_id": 3, "unread_count": 1}
				],
				"meta": {"count": 1}
			}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/3", jsonResponse(200, `{
			"payload": {"id": 3, "name": "Bob Smith"}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/103/messages", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "triage", "1", "--status", "pending"})
		if err != nil {
			t.Errorf("inboxes triage --status failed: %v", err)
		}
	})

	if !strings.Contains(output, "pending") {
		t.Errorf("output missing 'pending' status: %s", output)
	}
}

func TestInboxesTriageCommand_InvalidID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "triage", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestInboxesTriageCommand_EmptyConversations(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Empty Inbox",
			"channel_type": "Channel::Email"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [],
				"meta": {"count": 0}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "triage", "1"})
		if err != nil {
			t.Errorf("inboxes triage failed: %v", err)
		}
	})

	if !strings.Contains(output, "No conversations found") {
		t.Errorf("output missing empty message: %s", output)
	}
}

func TestInboxesStatsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Support Inbox",
			"channel_type": "Channel::Email"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 101, "status": "open", "unread_count": 3, "last_activity_at": 1700000000},
					{"id": 102, "status": "pending", "unread_count": 1, "last_activity_at": 1700000500},
					{"id": 103, "status": "resolved", "unread_count": 0, "last_activity_at": 1700001000}
				],
				"meta": {"count": 3}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "stats", "1"})
		if err != nil {
			t.Errorf("inboxes stats failed: %v", err)
		}
	})

	if !strings.Contains(output, "Support Inbox") {
		t.Errorf("output missing inbox name: %s", output)
	}
	if !strings.Contains(output, "Open: 1") {
		t.Errorf("output missing open count: %s", output)
	}
	if !strings.Contains(output, "Pending: 1") {
		t.Errorf("output missing pending count: %s", output)
	}
	if !strings.Contains(output, "Unread: 4") {
		t.Errorf("output missing unread count: %s", output)
	}
}

func TestInboxesStatsCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Support Inbox",
			"channel_type": "Channel::Email"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 101, "status": "open", "unread_count": 2, "last_activity_at": 1700000000}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "stats", "1", "--output", "json"})
		if err != nil {
			t.Errorf("inboxes stats --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"inbox_id"`) {
		t.Errorf("JSON output missing 'inbox_id' field: %s", output)
	}
	if !strings.Contains(output, `"open_count"`) {
		t.Errorf("JSON output missing 'open_count' field: %s", output)
	}
	if !strings.Contains(output, `"pending_count"`) {
		t.Errorf("JSON output missing 'pending_count' field: %s", output)
	}
	if !strings.Contains(output, `"unread_count"`) {
		t.Errorf("JSON output missing 'unread_count' field: %s", output)
	}
	if !strings.Contains(output, `"avg_wait_seconds"`) {
		t.Errorf("JSON output missing 'avg_wait_seconds' field: %s", output)
	}
}

func TestInboxesStatsCommand_InvalidID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "stats", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestInboxesStatsCommand_MissingID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"inboxes", "stats"})
	if err == nil {
		t.Error("expected error for missing ID argument")
	}
}

func TestInboxesStatsCommand_EmptyConversations(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Empty Inbox",
			"channel_type": "Channel::Email"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [],
				"meta": {"count": 0}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "stats", "1"})
		if err != nil {
			t.Errorf("inboxes stats failed: %v", err)
		}
	})

	if !strings.Contains(output, "Empty Inbox") {
		t.Errorf("output missing inbox name: %s", output)
	}
	if !strings.Contains(output, "Open: 0") {
		t.Errorf("output missing zero open count: %s", output)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int64
		expected string
	}{
		{0, "0s"},
		{30, "30s"},
		{59, "59s"},
		{60, "1m"},
		{90, "1m"},
		{120, "2m"},
		{3599, "59m"},
		{3600, "1h"},
		{3660, "1h1m"},
		{7200, "2h"},
		{7320, "2h2m"},
		{86400, "24h"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.seconds)
		if result != tt.expected {
			t.Errorf("formatDuration(%d) = %s, expected %s", tt.seconds, result, tt.expected)
		}
	}
}

func TestInboxesTriage_Overview(t *testing.T) {
	// Cross-inbox triage overview: when no ID is provided, show all inboxes
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Support Inbox", "channel_type": "Channel::Email"},
				{"id": 2, "name": "Sales Chat", "channel_type": "Channel::WebWidget"}
			]
		}`)).
		// Conversations for inbox 1
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 101, "status": "open", "inbox_id": 1, "unread_count": 3, "last_activity_at": 1700000000},
					{"id": 102, "status": "pending", "inbox_id": 1, "unread_count": 1, "last_activity_at": 1700000500}
				],
				"meta": {"count": 2}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "triage"})
		if err != nil {
			t.Errorf("inboxes triage (no args) failed: %v", err)
		}
	})

	// Should show overview with all inboxes
	if !strings.Contains(output, "Support Inbox") {
		t.Errorf("output missing 'Support Inbox': %s", output)
	}
	if !strings.Contains(output, "Sales Chat") {
		t.Errorf("output missing 'Sales Chat': %s", output)
	}
	// Should show tabular headers
	if !strings.Contains(output, "INBOX") {
		t.Errorf("output missing 'INBOX' header: %s", output)
	}
}

func TestInboxesTriage_Overview_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Support Inbox", "channel_type": "Channel::Email"},
				{"id": 2, "name": "Sales Chat", "channel_type": "Channel::WebWidget"}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 101, "status": "open", "inbox_id": 1, "unread_count": 3, "last_activity_at": 1700000000}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "triage", "--output", "json"})
		if err != nil {
			t.Errorf("inboxes triage --json (no args) failed: %v", err)
		}
	})

	if !strings.Contains(output, `"items"`) {
		t.Errorf("JSON output missing 'items' field: %s", output)
	}
	if !strings.Contains(output, `"inbox_id"`) {
		t.Errorf("JSON output missing 'inbox_id' field: %s", output)
	}
	if !strings.Contains(output, `"inbox_name"`) {
		t.Errorf("JSON output missing 'inbox_name' field: %s", output)
	}
}
