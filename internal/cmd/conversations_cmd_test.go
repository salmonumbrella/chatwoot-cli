package cmd

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestConversationsListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 1, "inbox_id": 1, "status": "open", "unread_count": 2, "created_at": 1700000000},
					{"id": 2, "inbox_id": 2, "status": "resolved", "unread_count": 0, "created_at": 1700001000}
				],
				"meta": {"count": 2}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"list"})
		_ = cmd.Execute()
	})

	// Verify output contains expected headers
	if !strings.Contains(output, "ID") {
		t.Errorf("Expected ID header in output, got: %s", output)
	}
	if !strings.Contains(output, "STATUS") {
		t.Errorf("Expected STATUS header in output, got: %s", output)
	}
}

func TestConversationsListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 1, "inbox_id": 1, "status": "open", "unread_count": 2, "created_at": 1700000000}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "list", "--output", "json"})
		if err != nil {
			t.Errorf("conversations list --output json failed: %v", err)
		}
	})

	// Verify JSON output
	if !strings.Contains(output, `"id"`) {
		t.Error("Expected JSON with id field in output")
	}
	if !strings.Contains(output, `"status"`) {
		t.Error("Expected JSON with status field in output")
	}
}

func TestConversationsGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"contact_id": 456,
			"status": "open",
			"unread_count": 5,
			"muted": false,
			"created_at": 1700000000
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"get", "123"})
		_ = cmd.Execute()
	})

	if !strings.Contains(output, "Conversation #123") {
		t.Errorf("Expected conversation details in output, got: %s", output)
	}
}

func TestConversationsToggleStatusCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"meta": {},
			"payload": {
				"success": true,
				"conversation_id": 123,
				"current_status": "resolved"
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"toggle-status", "123", "--status", "resolved"})
		_ = cmd.Execute()
	})

	if !strings.Contains(output, "resolved") {
		t.Errorf("Expected 'resolved' in output, got: %s", output)
	}
}

func TestConversationsAssignCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/assignments", jsonResponse(200, `{
			"id": 5,
			"name": "Agent Name"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"contact_id": 456,
			"status": "open",
			"assignee_id": 5,
			"created_at": 1700000000
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"assign", "123", "--assignee-id", "5"})
		_ = cmd.Execute()
	})

	if !strings.Contains(output, "assigned") {
		t.Errorf("Expected 'assigned' in output, got: %s", output)
	}
}

func TestConversationsLabelsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/labels", jsonResponse(200, `{
			"payload": ["bug", "urgent"]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"labels", "123"})
		_ = cmd.Execute()
	})

	if !strings.Contains(output, "bug") {
		t.Errorf("Expected 'bug' label in output, got: %s", output)
	}
	if !strings.Contains(output, "urgent") {
		t.Errorf("Expected 'urgent' label in output, got: %s", output)
	}
}

func TestConversationsLabelsAddCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/labels", jsonResponse(200, `{
			"payload": ["bug", "urgent", "new-label"]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"labels-add", "123", "--labels", "new-label"})
		_ = cmd.Execute()
	})

	if !strings.Contains(output, "Labels updated") {
		t.Errorf("Expected 'Labels updated' in output, got: %s", output)
	}
}

func TestConversationsMuteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_mute", jsonResponse(200, `{}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"status": "open",
			"muted": true,
			"created_at": 1700000000
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"mute", "123"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("mute failed: %v", err)
		}
	})

	if !strings.Contains(output, "muted") {
		t.Errorf("Expected 'muted' in output, got: %s", output)
	}
}

func TestConversationsUnmuteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_mute", jsonResponse(200, `{}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"status": "open",
			"muted": false,
			"created_at": 1700000000
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"unmute", "123"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("unmute failed: %v", err)
		}
	})

	if !strings.Contains(output, "unmuted") {
		t.Errorf("Expected 'unmuted' in output, got: %s", output)
	}
}

func TestConversationsBulkResolve(t *testing.T) {
	callCount := 0
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount++
			jsonResponse(200, `{"meta": {}, "payload": {"success": true, "conversation_id": 1, "current_status": "resolved"}}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount++
			jsonResponse(200, `{"meta": {}, "payload": {"success": true, "conversation_id": 2, "current_status": "resolved"}}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"bulk", "resolve", "--ids", "1,2"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("bulk resolve failed: %v", err)
		}
	})

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}

	if !strings.Contains(output, "Resolved 2 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsBulkAssign(t *testing.T) {
	callCount := 0
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/assignments", func(w http.ResponseWriter, r *http.Request) {
			callCount++
			jsonResponse(200, `{"id": 5, "name": "Agent"}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/assignments", func(w http.ResponseWriter, r *http.Request) {
			callCount++
			jsonResponse(200, `{"id": 5, "name": "Agent"}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"bulk", "assign", "--ids", "1,2", "--agent-id", "5"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("bulk assign failed: %v", err)
		}
	})

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}

	if !strings.Contains(output, "Assigned 2 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsBulkAddLabel(t *testing.T) {
	callCount := 0
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount++
			jsonResponse(200, `{"payload": ["urgent", "new-label"]}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount++
			jsonResponse(200, `{"payload": ["new-label"]}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"bulk", "add-label", "--ids", "1,2", "--labels", "new-label"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("bulk add-label failed: %v", err)
		}
	})

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}

	if !strings.Contains(output, "Added labels to 2 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsSearchCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 1, "inbox_id": 1, "status": "open", "created_at": 1700000000},
					{"id": 2, "inbox_id": 2, "status": "resolved", "created_at": 1700001000}
				],
				"meta": {"count": 2}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"search", "test query"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("search failed: %v", err)
		}
	})

	if !strings.Contains(output, "ID") {
		t.Errorf("Expected ID header in output, got: %s", output)
	}
}

func TestConversationsMarkUnreadCommand(t *testing.T) {
	// mark-unread first fetches the conversation to get initial state
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"status": "open",
			"unread_count": 1,
			"created_at": 1700000000
		}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/unread", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"mark-unread", "123"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("mark-unread failed: %v", err)
		}
	})

	if !strings.Contains(output, "marked as unread") {
		t.Errorf("Expected 'marked as unread' in output, got: %s", output)
	}
}

func TestConversationsToggleStatusCommand_RequiresStatus(t *testing.T) {
	setupTestEnvWithHandler(t, newRouteHandler())

	cmd := newConversationsCmd()
	cmd.SetArgs([]string{"toggle-status", "123"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when --status is missing")
	}
}

func TestConversationsBulkResolve_RequiresIDs(t *testing.T) {
	setupTestEnvWithHandler(t, newRouteHandler())

	cmd := newConversationsCmd()
	cmd.SetArgs([]string{"bulk", "resolve"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when --ids is missing")
	}
}

func TestConversationsBulkAssign_RequiresAgentOrTeam(t *testing.T) {
	setupTestEnvWithHandler(t, newRouteHandler())

	cmd := newConversationsCmd()
	cmd.SetArgs([]string{"bulk", "assign", "--ids", "1,2"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when neither --agent-id nor --team-id is provided")
	}
}

func TestConversationsCustomAttributesCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/custom_attributes", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"custom-attributes", "123", "--set", "priority=high", "--set", "source=web"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("custom-attributes failed: %v", err)
		}
	})

	if !strings.Contains(output, "Custom attributes updated") {
		t.Errorf("Expected 'Custom attributes updated' in output, got: %s", output)
	}
}

func TestConversationsFilterCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/filter", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 1, "inbox_id": 1, "status": "open", "created_at": 1700000000}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"filter", "--payload", `{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]}`})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("filter failed: %v", err)
		}
	})

	if !strings.Contains(output, "ID") {
		t.Errorf("Expected ID header in output, got: %s", output)
	}
}

func TestConversationsAttachmentsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/attachments", jsonResponse(200, `[
			{"id": 10, "file_type": "image", "file_size": 1024, "data_url": "https://example.com/file.png"}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"attachments", "123"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("attachments failed: %v", err)
		}
	})

	if !strings.Contains(output, "image") || !strings.Contains(output, "file.png") {
		t.Errorf("Expected attachment info in output, got: %s", output)
	}
}
