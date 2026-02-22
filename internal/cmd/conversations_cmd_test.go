package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

func TestConversationsListCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{
						"id": 321,
						"inbox_id": 48,
						"status": "open",
						"unread_count": 2,
						"messages_count": 7,
						"last_activity_at": 1700005000,
						"contact_id": 123,
						"meta": {"sender": {"id": 123, "name": "Jane Doe", "email": "jane@example.com"}},
						"custom_attributes": {"tier": "gold"},
						"last_non_activity_message": {"content": "  Need refund status  "}
					}
				],
				"meta": {"total_pages": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "list", "--li"})
		if err != nil {
			t.Fatalf("conversations list --li failed: %v", err)
		}
	})
	if strings.Contains(output, "\n  ") {
		t.Fatalf("expected --li output to be compact by default, got pretty JSON:\n%s", output)
	}

	var payload struct {
		Items []struct {
			ID          int    `json:"id"`
			Status      string `json:"st"`
			InboxID     int    `json:"ib"`
			UnreadCount int    `json:"ur"`
			LastMessage string `json:"lm"`
			Contact     struct {
				ID   *int    `json:"id"`
				Name *string `json:"nm"`
			} `json:"ct"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light JSON: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	item := payload.Items[0]
	if item.ID != 321 || item.Status != "o" || item.InboxID != 48 {
		t.Fatalf("unexpected light item: %#v", item)
	}
	if item.LastMessage != "Need refund status" {
		t.Fatalf("expected trimmed last_message, got %q", item.LastMessage)
	}
	if item.Contact.ID == nil || *item.Contact.ID != 123 {
		t.Fatalf("expected contact id 123, got %#v", item.Contact.ID)
	}
	if item.Contact.Name == nil || *item.Contact.Name != "Jane Doe" {
		t.Fatalf("expected contact name Jane Doe, got %#v", item.Contact.Name)
	}
	if strings.Contains(output, `"custom_attributes"`) {
		t.Fatal("light output should not include custom_attributes")
	}
	if strings.Contains(output, `"jane@example.com"`) {
		t.Fatal("light output should not include sender email")
	}
}

func TestConversationsListCommand_LightNoEnvelope(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{
						"id": 1,
						"inbox_id": 48,
						"status": "open",
						"unread_count": 0,
						"last_activity_at": 1700000000,
						"meta": {"sender": {"id": 10, "name": "Alice"}}
					}
				],
				"meta": {"total_pages": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "list", "--light"})
		if err != nil {
			t.Fatalf("conversations list --light failed: %v", err)
		}
	})

	if strings.Contains(output, `"has_more"`) {
		t.Fatal("light output should not include has_more")
	}
	if strings.Contains(output, `"meta"`) {
		t.Fatal("light output should not include meta envelope")
	}
}

func TestConversationsListCommand_LightJSONLiteralQuery(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{
						"id": 1,
						"inbox_id": 48,
						"status": "open",
						"unread_count": 0,
						"last_activity_at": 1700000000
					}
				],
				"meta": {"total_pages": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"conversations", "list", "--light", "-o", "json", "--jq", ".items[0].st",
		})
		if err != nil {
			t.Fatalf("conversations list --light -o json --jq .items[0].st failed: %v", err)
		}
	})

	got := strings.TrimSpace(output)
	if got != `"o"` {
		t.Fatalf("expected jq output %q for short-key light status, got %q", `"o"`, got)
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

func TestConversationsGetCommand_EmitURL_SkipsAPICall(t *testing.T) {
	called := false
	setupTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	})

	out := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"conversations", "get", "123", "--emit", "url"}); err != nil {
			t.Fatalf("conversations get --emit url failed: %v", err)
		}
	})

	if called {
		t.Fatalf("expected no API call for --emit url")
	}
	if !strings.Contains(out, "/app/accounts/1/conversations/123") {
		t.Fatalf("unexpected output: %q", out)
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

func TestConversationsToggleStatusCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"meta": {},
			"payload": {
				"success": true,
				"conversation_id": 123,
				"current_status": "pending"
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "toggle-status", "123", "--status", "pending", "--light", "-o", "agent"})
		if err != nil {
			t.Fatalf("conversations toggle-status --light failed: %v", err)
		}
	})

	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"item"`) {
		t.Fatalf("light output should bypass agent envelope, got: %s", output)
	}
	var result struct {
		ID     int    `json:"id"`
		Status string `json:"st"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse light output: %v\noutput: %s", err, output)
	}
	if result.ID != 123 {
		t.Fatalf("expected id 123, got %d", result.ID)
	}
	if result.Status != "p" {
		t.Fatalf("expected short status p, got %q", result.Status)
	}
}

func TestConversationsToggleStatusCommand_AgentCompactAliases(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"meta": {},
			"payload": {
				"success": true,
				"conversation_id": 123,
				"current_status": "pending"
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "toggle-status", "123", "--status", "pending", "-o", "agent"})
		if err != nil {
			t.Fatalf("conversations toggle-status -o agent failed: %v", err)
		}
	})

	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"item"`) || strings.Contains(output, `"data"`) {
		t.Fatalf("agent output should be flat summary, got: %s", output)
	}
	if strings.Contains(output, "\n  ") {
		t.Fatalf("agent mutation output should be compact by default, got: %s", output)
	}
	var result struct {
		ID int    `json:"id"`
		OK bool   `json:"ok"`
		St string `json:"st"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse compact agent output: %v\noutput: %s", err, output)
	}
	if result.ID != 123 || !result.OK || result.St != "p" {
		t.Fatalf("unexpected compact status payload: %#v", result)
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

func TestConversationsAssignCommand_Light(t *testing.T) {
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
			"team_id": 2,
			"created_at": 1700000000
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "assign", "123", "--agent", "5", "--team", "2", "--light", "-o", "agent"})
		if err != nil {
			t.Fatalf("conversations assign --light failed: %v", err)
		}
	})

	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"item"`) {
		t.Fatalf("light output should bypass agent envelope, got: %s", output)
	}
	var result struct {
		ID      int  `json:"id"`
		AgentID *int `json:"ag"`
		TeamID  *int `json:"tm"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse light output: %v\noutput: %s", err, output)
	}
	if result.ID != 123 {
		t.Fatalf("expected id 123, got %d", result.ID)
	}
	if result.AgentID == nil || *result.AgentID != 5 {
		t.Fatalf("expected ag=5, got %#v", result.AgentID)
	}
	if result.TeamID == nil || *result.TeamID != 2 {
		t.Fatalf("expected tm=2, got %#v", result.TeamID)
	}
}

func TestConversationsAssignCommand_AgentCompactAliases(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/assignments", jsonResponse(200, `{
			"id": 5,
			"name": "Agent Name"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"status": "open",
			"assignee_id": 5,
			"team_id": 2
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "assign", "123", "--agent", "5", "--team", "2", "-o", "agent"})
		if err != nil {
			t.Fatalf("conversations assign -o agent failed: %v", err)
		}
	})

	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"item"`) || strings.Contains(output, `"data"`) {
		t.Fatalf("agent output should be flat summary, got: %s", output)
	}
	var result struct {
		ID int    `json:"id"`
		St string `json:"st"`
		Ag int    `json:"ag"`
		Tm int    `json:"tm"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse compact assign output: %v\noutput: %s", err, output)
	}
	if result.ID != 123 || result.St != "o" || result.Ag != 5 || result.Tm != 2 {
		t.Fatalf("unexpected compact assign payload: %#v", result)
	}
}

func TestConversationsListCommand_AllIncludesLastPage(t *testing.T) {
	setupTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/accounts/1/conversations" {
			http.NotFound(w, r)
			return
		}
		page := r.URL.Query().Get("page")
		if page == "2" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data": {
					"payload": [
						{"id": 2, "inbox_id": 1, "status": "open", "created_at": 1700001000}
					],
					"meta": {"total_pages": 2}
				}
			}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"data": {
				"payload": [
					{"id": 1, "inbox_id": 1, "status": "open", "created_at": 1700000000}
				],
				"meta": {"total_pages": 2}
			}
		}`))
	})
	t.Setenv("CHATWOOT_TESTING", "1")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "list", "--all", "--output", "json"})
		if err != nil {
			t.Errorf("conversations list --all --output json failed: %v", err)
		}
	})

	items := decodeItems(t, output)
	if len(items) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(items))
	}
}

func TestConversationsSearchCommand_AllIncludesLastPage(t *testing.T) {
	setupTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/accounts/1/conversations/search" {
			http.NotFound(w, r)
			return
		}
		page := r.URL.Query().Get("page")
		if page == "2" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data": {
					"payload": [
						{"id": 11, "inbox_id": 1, "status": "open", "created_at": 1700001000}
					],
					"meta": {"total_pages": 2}
				}
			}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"data": {
				"payload": [
					{"id": 10, "inbox_id": 1, "status": "open", "created_at": 1700000000}
				],
				"meta": {"total_pages": 2}
			}
		}`))
	})
	t.Setenv("CHATWOOT_TESTING", "1")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "search", "test", "--all", "--output", "json"})
		if err != nil {
			t.Errorf("conversations search --all --output json failed: %v", err)
		}
	})

	items := decodeItems(t, output)
	if len(items) != 2 {
		t.Fatalf("expected 2 conversations, got %d", len(items))
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
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"meta": {}, "payload": {"success": true, "conversation_id": 1, "current_status": "resolved"}}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
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

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}

	if !strings.Contains(output, "Resolved 2 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsBulkResolve_AcceptsURLs(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"meta": {}, "payload": {"success": true, "conversation_id": 1, "current_status": "resolved"}}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"meta": {}, "payload": {"success": true, "conversation_id": 2, "current_status": "resolved"}}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{
			"bulk", "resolve",
			"--ids",
			"https://app.chatwoot.com/app/accounts/1/conversations/1,https://app.chatwoot.com/app/accounts/1/conversations/2",
		})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("bulk resolve failed: %v", err)
		}
	})

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}
	if !strings.Contains(output, "Resolved 2 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsBulkResolve_IdsFromStdin(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"meta": {}, "payload": {"success": true, "conversation_id": 1, "current_status": "resolved"}}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"meta": {}, "payload": {"success": true, "conversation_id": 2, "current_status": "resolved"}}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.Write([]byte("1\n2\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"bulk", "resolve", "--ids", "@-"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("bulk resolve failed: %v", err)
		}
	})

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}
	if !strings.Contains(output, "Resolved 2 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsBulkAssign(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/assignments", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"id": 5, "name": "Agent"}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/assignments", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
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

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}

	if !strings.Contains(output, "Assigned 2 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsBulkAssign_AgentByName(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 5, "name": "Agent", "email": "agent@example.com", "role": "agent"}
		]`)).
		On("POST", "/api/v1/accounts/1/conversations/1/assignments", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"id": 5, "name": "Agent"}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/assignments", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"id": 5, "name": "Agent"}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"bulk", "assign", "--ids", "1,2", "--agent", "Agent"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("bulk assign --agent name failed: %v", err)
		}
	})

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}
	if !strings.Contains(output, "Assigned 2 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsBulkAddLabel(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"payload": ["urgent", "new-label"]}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
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

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}

	if !strings.Contains(output, "Added labels to 2 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsBulkAddLabel_LabelsFromStdin(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"payload": ["urgent", "new-label"]}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.Write([]byte("urgent\nnew-label\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"bulk", "add-label", "--ids", "1", "--labels", "@-"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("bulk add-label failed: %v", err)
		}
	})

	if callCount.Load() != 1 {
		t.Errorf("expected 1 API call, got %d", callCount.Load())
	}
	if !strings.Contains(output, "Added labels to 1 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsBatchUpdate_InvalidConcurrency(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.Write([]byte(`[{"id":123,"status":"resolved"}]`))
		_ = w.Close()
	}()

	err = Execute(context.Background(), []string{"conversations", "bulk", "batch-update", "--concurrency", "0"})
	if err == nil {
		t.Fatal("expected error for invalid --concurrency")
	}
	if !strings.Contains(err.Error(), "--concurrency must be greater than 0") {
		t.Fatalf("unexpected error: %v", err)
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

func TestConversationsResolveMultiple(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"meta": {}, "payload": {"success": true, "conversation_id": 1, "current_status": "resolved"}}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"meta": {}, "payload": {"success": true, "conversation_id": 2, "current_status": "resolved"}}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/3/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"meta": {}, "payload": {"success": true, "conversation_id": 3, "current_status": "resolved"}}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		// Test space-separated IDs
		cmd.SetArgs([]string{"resolve", "1", "2", "3"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("resolve multiple failed: %v", err)
		}
	})

	if callCount.Load() != 3 {
		t.Errorf("expected 3 API calls, got %d", callCount.Load())
	}

	if !strings.Contains(output, "Resolved 3 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsResolveMultiple_CommaSeparated(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"meta": {}, "payload": {"success": true, "conversation_id": 1, "current_status": "resolved"}}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/toggle_status", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"meta": {}, "payload": {"success": true, "conversation_id": 2, "current_status": "resolved"}}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		// Test comma-separated IDs
		cmd.SetArgs([]string{"resolve", "1,2"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("resolve comma-separated failed: %v", err)
		}
	})

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}

	if !strings.Contains(output, "Resolved 2 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsResolveSingle(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{
			"meta": {},
			"payload": {"success": true, "conversation_id": 123, "current_status": "resolved"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		cmd.SetArgs([]string{"resolve", "123"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("resolve single failed: %v", err)
		}
	})

	if !strings.Contains(output, "Conversation #123 resolved") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsAssignMultiple(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/assignments", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"id": 5, "name": "Agent"}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/assignments", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"id": 5, "name": "Agent"}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/3/assignments", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"id": 5, "name": "Agent"}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		// Test space-separated IDs with --agent flag
		cmd.SetArgs([]string{"assign", "1", "2", "3", "--agent", "5"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("assign multiple failed: %v", err)
		}
	})

	if callCount.Load() != 3 {
		t.Errorf("expected 3 API calls, got %d", callCount.Load())
	}

	if !strings.Contains(output, "Assigned 3 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsAssignMultiple_CommaSeparated(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/1/assignments", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"id": 2, "name": "Team"}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/conversations/2/assignments", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"id": 2, "name": "Team"}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		cmd := newConversationsCmd()
		// Test comma-separated IDs with --team flag
		cmd.SetArgs([]string{"assign", "1,2", "--team", "2"})
		err := cmd.Execute()
		if err != nil {
			t.Errorf("assign comma-separated failed: %v", err)
		}
	})

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}

	if !strings.Contains(output, "Assigned 2 conversations") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestConversationsAssign_RequiresAgentOrTeam(t *testing.T) {
	setupTestEnvWithHandler(t, newRouteHandler())

	cmd := newConversationsCmd()
	cmd.SetArgs([]string{"assign", "1", "2"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error when neither --agent nor --team is provided")
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

func TestConversationsFilterCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/filter", jsonResponse(200, `{
			"payload": [
				{
					"id": 1,
					"inbox_id": 1,
					"status": "open",
					"unread_count": 5,
					"last_activity_at": 1700000000,
					"meta": {"sender": {"id": 10, "name": "Alice"}},
					"last_non_activity_message": {"content": "Still waiting"}
				}
			],
			"meta": {"count": 1}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"conversations", "filter",
			"--payload", `{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]}`,
			"--light",
		})
		if err != nil {
			t.Fatalf("conversations filter --light failed: %v", err)
		}
	})

	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse filter light output: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(payload.Items))
	}
	if payload.Items[0]["st"] != "o" {
		t.Fatalf("expected status o, got %#v", payload.Items[0]["st"])
	}
	if payload.Items[0]["lm"] != "Still waiting" {
		t.Fatalf("expected last_message, got %#v", payload.Items[0]["lm"])
	}
	if strings.Contains(output, `"meta"`) {
		t.Fatal("light output should not include full meta object")
	}
}

func TestConversationsFilterCommand_LightAll(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/filter", func(w http.ResponseWriter, r *http.Request) {
			page := r.URL.Query().Get("page")
			w.Header().Set("Content-Type", "application/json")
			if page == "" || page == "1" {
				_, _ = w.Write([]byte(`{
					"payload": [
						{"id": 1, "status": "open", "inbox_id": 1, "last_activity_at": 1700000000, "meta": {"sender": {"id": 10, "name": "Alice"}}}
					],
					"meta": {"count": 2, "total_pages": 2}
				}`))
			} else {
				_, _ = w.Write([]byte(`{
					"payload": [
						{"id": 2, "status": "resolved", "inbox_id": 2, "last_activity_at": 1700000100, "meta": {"sender": {"id": 20, "name": "Bob"}}}
					],
					"meta": {"count": 2, "total_pages": 2}
				}`))
			}
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"conversations", "filter",
			"--payload", `{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]}`,
			"--light", "--all",
		})
		if err != nil {
			t.Fatalf("filter --light --all failed: %v", err)
		}
	})

	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 2 {
		t.Fatalf("expected 2 conversations from 2 pages, got %d", len(payload.Items))
	}
	if payload.Items[0]["st"] != "o" || payload.Items[1]["st"] != "r" {
		t.Fatalf("unexpected items: %#v", payload.Items)
	}
}

func TestConversationsAttachmentsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/attachments", jsonResponse(200, `{"meta": {"total_count": 1}, "payload": [
			{"id": 10, "file_type": "image", "file_size": 1024, "data_url": "https://example.com/file.png"}
		]}`))

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

func TestConversationsCreateCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"id": 123,
			"status": "open",
			"inbox_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "create", "--inbox-id", "1", "--contact-id", "123"})
		if err != nil {
			t.Errorf("conversations create failed: %v", err)
		}
	})

	if !strings.Contains(output, "Created conversation") || !strings.Contains(output, "123") {
		t.Errorf("Expected creation confirmation, got: %s", output)
	}
}

func TestConversationsCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"id": 123,
			"status": "open"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "create", "--inbox-id", "1", "--contact-id", "123", "-o", "json"})
		if err != nil {
			t.Errorf("conversations create JSON failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing 'id' field: %s", output)
	}
}

func TestConversationsCreateCommand_MissingInboxID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"conversations", "create", "--contact-id", "123"})
	if err == nil {
		t.Error("expected error for missing --inbox-id")
	}
}

func TestConversationsMetaCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/meta", jsonResponse(200, `{
			"meta": {
				"all_count": 100,
				"assigned_count": 25,
				"unassigned_count": 30,
				"mine_count": 10
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "meta"})
		if err != nil {
			t.Errorf("conversations meta failed: %v", err)
		}
	})

	// Check for any of the count values in output
	if !strings.Contains(output, "100") {
		t.Errorf("Expected meta output with counts, got: %s", output)
	}
}

func TestConversationsMetaCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/meta", jsonResponse(200, `{
			"meta": {
				"all_count": 100
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "meta", "-o", "json"})
		if err != nil {
			t.Errorf("conversations meta JSON failed: %v", err)
		}
	})

	if !strings.Contains(output, `"all_count"`) {
		t.Errorf("JSON output missing 'all_count' field: %s", output)
	}
}

func TestConversationsCountsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/meta", jsonResponse(200, `{
			"meta": {
				"all_count": 105,
				"mine_count": 50,
				"assigned_count": 20,
				"unassigned_count": 30
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "counts"})
		if err != nil {
			t.Errorf("conversations counts failed: %v", err)
		}
	})

	if !strings.Contains(output, "50") {
		t.Errorf("Expected counts output, got: %s", output)
	}
}

func TestConversationsCountsCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/meta", jsonResponse(200, `{
			"meta": {
				"all_count": 105,
				"mine_count": 50
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "counts", "-o", "json"})
		if err != nil {
			t.Errorf("conversations counts JSON failed: %v", err)
		}
	})

	if !strings.Contains(output, "50") {
		t.Errorf("JSON output missing counts: %s", output)
	}
}

func TestConversationsTogglePriorityCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_priority", jsonResponse(200, ``)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"priority": "high"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "toggle-priority", "123", "--priority", "high"})
		if err != nil {
			t.Errorf("conversations toggle-priority failed: %v", err)
		}
	})

	if !strings.Contains(output, "priority updated") || !strings.Contains(output, "high") {
		t.Errorf("Expected priority update confirmation, got: %s", output)
	}
}

func TestConversationsTogglePriorityCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_priority", jsonResponse(200, ``)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"priority": "high"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "toggle-priority", "123", "--priority", "high", "-o", "json"})
		if err != nil {
			t.Errorf("conversations toggle-priority JSON failed: %v", err)
		}
	})

	if !strings.Contains(output, `"priority"`) {
		t.Errorf("JSON output missing 'priority' field: %s", output)
	}
}

func TestConversationsTogglePriorityCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_priority", jsonResponse(200, ``)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"priority": "urgent"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "toggle-priority", "123", "--priority", "urgent", "--light", "-o", "agent"})
		if err != nil {
			t.Fatalf("conversations toggle-priority --light failed: %v", err)
		}
	})

	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"item"`) {
		t.Fatalf("light output should bypass agent envelope, got: %s", output)
	}
	var result struct {
		ID       int    `json:"id"`
		Priority string `json:"pri"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse light output: %v\noutput: %s", err, output)
	}
	if result.ID != 123 {
		t.Fatalf("expected id 123, got %d", result.ID)
	}
	if result.Priority != "u" {
		t.Fatalf("expected short priority u, got %q", result.Priority)
	}
}

func TestConversationsTogglePriorityCommand_AgentCompactAliases(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_priority", jsonResponse(200, ``)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"status": "open",
			"priority": "medium",
			"inbox_id": 48,
			"unread_count": 3
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "toggle-priority", "123", "--priority", "medium", "-o", "agent"})
		if err != nil {
			t.Fatalf("conversations toggle-priority -o agent failed: %v", err)
		}
	})

	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"item"`) || strings.Contains(output, `"data"`) {
		t.Fatalf("agent output should be flat summary, got: %s", output)
	}
	if strings.Contains(output, "\n  ") {
		t.Fatalf("agent mutation output should be compact by default, got: %s", output)
	}
	var result struct {
		ID  int    `json:"id"`
		Pri string `json:"pri"`
		St  string `json:"st"`
		Ib  int    `json:"ib"`
		Ur  int    `json:"ur"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse compact priority output: %v\noutput: %s", err, output)
	}
	if result.ID != 123 || result.Pri != "m" || result.St != "o" || result.Ib != 48 || result.Ur != 3 {
		t.Fatalf("unexpected compact priority payload: %#v", result)
	}
}

func TestConversationsUpdateCommand(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"status": "open",
			"priority": "high"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "update", "123", "--priority", "high"})
		if err != nil {
			t.Errorf("conversations update failed: %v", err)
		}
	})

	if !strings.Contains(output, "updated") {
		t.Errorf("Expected update confirmation, got: %s", output)
	}
}

func TestConversationsUpdateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"priority": "high"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "update", "123", "--priority", "high", "-o", "json"})
		if err != nil {
			t.Errorf("conversations update JSON failed: %v", err)
		}
	})

	if !strings.Contains(output, `"priority"`) {
		t.Errorf("JSON output missing 'priority' field: %s", output)
	}
}

func TestConversationsUpdateCommand_NoFlags(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"conversations", "update", "123"})
	if err == nil {
		t.Error("expected error when no update flags provided")
	}
}

func TestConversationsLabelsRemoveCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/labels", jsonResponse(200, `{
			"payload": ["label1", "label2", "removed-label"]
		}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/labels", jsonResponse(200, `{
			"payload": ["label1", "label2"]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "labels-remove", "123", "--labels", "removed-label"})
		if err != nil {
			t.Errorf("conversations labels-remove failed: %v", err)
		}
	})

	if !strings.Contains(output, "Labels updated") {
		t.Errorf("Expected labels update confirmation, got: %s", output)
	}
}

// TestConversationsContextCommand is complex because it makes multiple
// sequential API calls with pagination. Covered by API tests.

// TestConversationsContextCommand_JSON is complex because it makes multiple
// sequential API calls with pagination. Covered by API tests.

func TestConversationsTranscriptCommand_JSONIncludesPrivate(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"contact_id": 99,
			"status": "open",
			"unread_count": 0,
			"created_at": 1700000000
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("before") != "" {
				_, _ = w.Write([]byte(`{"payload": []}`))
				return
			}
			_, _ = w.Write([]byte(`{"payload": [
				{"id": 2, "conversation_id": 123, "content": "Internal note", "message_type": 1, "private": true, "created_at": 1700000002},
				{"id": 1, "conversation_id": 123, "content": "Hello", "message_type": 0, "private": false, "created_at": 1700000001}
			]}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "transcript", "123", "--output", "json"})
		if err != nil {
			t.Errorf("conversations transcript --output json failed: %v", err)
		}
	})

	var payload struct {
		Messages []struct {
			Private bool `json:"private"`
		} `json:"messages"`
		Meta map[string]any `json:"meta"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	if len(payload.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(payload.Messages))
	}
	if payload.Messages[0].Private == false && payload.Messages[1].Private == false {
		t.Errorf("expected at least one private message in transcript")
	}
	if payload.Meta["public_only"] != false {
		t.Errorf("expected public_only false, got %v", payload.Meta["public_only"])
	}
}

func TestConversationsTranscriptCommand_PublicOnly(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"contact_id": 99,
			"status": "open",
			"unread_count": 0,
			"created_at": 1700000000
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Query().Get("before") != "" {
				_, _ = w.Write([]byte(`{"payload": []}`))
				return
			}
			_, _ = w.Write([]byte(`{"payload": [
				{"id": 2, "conversation_id": 123, "content": "Internal note", "message_type": 1, "private": true, "created_at": 1700000002},
				{"id": 1, "conversation_id": 123, "content": "Hello", "message_type": 0, "private": false, "created_at": 1700000001}
			]}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "transcript", "123", "--public-only", "--output", "json"})
		if err != nil {
			t.Errorf("conversations transcript --public-only failed: %v", err)
		}
	})

	var payload struct {
		Messages []struct {
			Private bool `json:"private"`
		} `json:"messages"`
		Meta map[string]any `json:"meta"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	if len(payload.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(payload.Messages))
	}
	if payload.Messages[0].Private {
		t.Errorf("expected public-only transcript to exclude private messages")
	}
	if payload.Meta["public_only"] != true {
		t.Errorf("expected public_only true, got %v", payload.Meta["public_only"])
	}
}

func runResolveNamesTest(t *testing.T, args []string) {
	t.Helper()

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 10, "inbox_id": 7, "contact_id": 42, "status": "open", "unread_count": 1, "created_at": 1700000000}
				],
				"meta": {"total_pages": 1}
			}
		}`)).
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 7, "name": "Support"}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/42", jsonResponse(200, `{
			"payload": {"id": 42, "name": "Jane Doe", "email": "jane@example.com"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), args)
		if err != nil {
			t.Errorf("Execute(%v) failed: %v", args, err)
		}
	})

	var payload struct {
		Items []struct {
			Path []struct {
				Type  string `json:"type"`
				ID    int    `json:"id"`
				Label string `json:"label"`
			} `json:"path"`
			Contact *struct {
				Name string `json:"name"`
			} `json:"contact"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(payload.Items))
	}
	if payload.Items[0].Contact == nil || payload.Items[0].Contact.Name != "Jane Doe" {
		t.Errorf("expected resolved contact name, got %#v", payload.Items[0].Contact)
	}
	foundInboxLabel := false
	for _, entry := range payload.Items[0].Path {
		if entry.Type == "inbox" && entry.ID == 7 && entry.Label == "Support" {
			foundInboxLabel = true
			break
		}
	}
	if !foundInboxLabel {
		t.Errorf("expected inbox label Support in path, got %#v", payload.Items[0].Path)
	}
}

func TestConversationsListCommand_AgentResolveNames(t *testing.T) {
	runResolveNamesTest(t, []string{"conversations", "list", "--output", "agent", "--resolve-names"})
}

func TestConversationsListCommand_AgentResolveNamesFromEnv(t *testing.T) {
	t.Setenv("CHATWOOT_RESOLVE_NAMES", "1")
	runResolveNamesTest(t, []string{"conversations", "list", "--output", "agent"})
}

func TestConversationsListUnreadOnly(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"meta": {"total_pages": 1},
				"payload": [
					{"id": 1, "status": "open", "inbox_id": 1, "unread_count": 5},
					{"id": 2, "status": "open", "inbox_id": 1, "unread_count": 0},
					{"id": 3, "status": "open", "inbox_id": 1, "unread_count": 3}
				]
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "list", "--unread-only", "--output", "json"})
		if err != nil {
			t.Errorf("conversations list --unread-only failed: %v", err)
		}
	})

	var result struct {
		Items []struct{ ID int } `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 unread conversations, got %d", len(result.Items))
	}
}

func TestConversationsListSinceFlag(t *testing.T) {
	now := time.Now().Unix()
	yesterday := now - 86400
	lastWeek := now - 86400*7

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"meta": map[string]any{"total_pages": 1},
					"payload": []map[string]any{
						{"id": 1, "status": "open", "inbox_id": 1, "last_activity_at": now},
						{"id": 2, "status": "open", "inbox_id": 1, "last_activity_at": yesterday},
						{"id": 3, "status": "open", "inbox_id": 1, "last_activity_at": lastWeek},
					},
				},
			})
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "list", "--since", "2d ago", "--output", "json"})
		if err != nil {
			t.Errorf("conversations list --since failed: %v", err)
		}
	})

	var result struct {
		Items []struct{ ID int } `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 conversations since 2d ago, got %d", len(result.Items))
	}
}

func TestConversationsListWaiting(t *testing.T) {
	// Conversations with different last_activity_at values
	// ID 1: most recent activity (should be last after sorting)
	// ID 2: middle activity
	// ID 3: oldest activity (should be first after sorting - longest waiting)
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"meta": {"total_pages": 1},
				"payload": [
					{"id": 1, "status": "open", "inbox_id": 1, "last_activity_at": 1700003000},
					{"id": 2, "status": "open", "inbox_id": 1, "last_activity_at": 1700002000},
					{"id": 3, "status": "open", "inbox_id": 1, "last_activity_at": 1700001000}
				]
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "list", "--waiting", "--output", "json"})
		if err != nil {
			t.Errorf("conversations list --waiting failed: %v", err)
		}
	})

	var result struct {
		Items []struct {
			ID             int   `json:"id"`
			LastActivityAt int64 `json:"last_activity_at"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 conversations, got %d", len(result.Items))
	}
	// Verify sorted by oldest last_activity_at first (longest waiting)
	if result.Items[0].ID != 3 {
		t.Errorf("expected conversation 3 (oldest activity) first, got %d", result.Items[0].ID)
	}
	if result.Items[1].ID != 2 {
		t.Errorf("expected conversation 2 (middle activity) second, got %d", result.Items[1].ID)
	}
	if result.Items[2].ID != 1 {
		t.Errorf("expected conversation 1 (newest activity) last, got %d", result.Items[2].ID)
	}
}

func TestConversationsListWaitingAlias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"meta": {"total_pages": 1},
				"payload": [
					{"id": 1, "status": "open", "inbox_id": 1, "last_activity_at": 1700003000},
					{"id": 2, "status": "open", "inbox_id": 1, "last_activity_at": 1700002000},
					{"id": 3, "status": "open", "inbox_id": 1, "last_activity_at": 1700001000}
				]
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "list", "--wt", "--output", "json"})
		if err != nil {
			t.Errorf("conversations list --wt failed: %v", err)
		}
	})

	var result struct {
		Items []struct {
			ID int `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 conversations, got %d", len(result.Items))
	}
	if result.Items[0].ID != 3 {
		t.Errorf("expected conversation 3 (oldest activity) first, got %d", result.Items[0].ID)
	}
}

func TestConversationsGetWithMessages(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"contact_id": 456,
			"status": "open",
			"unread_count": 2,
			"muted": false,
			"created_at": 1700000000
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "conversation_id": 123, "message_type": 0, "content": "Hello", "created_at": 1700000100},
				{"id": 2, "conversation_id": 123, "message_type": 1, "content": "Hi there", "created_at": 1700000200},
				{"id": 3, "conversation_id": 123, "message_type": 0, "content": "How can I help?", "created_at": 1700000300}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "--with-messages", "--output", "agent"})
		if err != nil {
			t.Errorf("conversations get --with-messages failed: %v", err)
		}
	})

	// Verify it's valid JSON with expected structure
	var result struct {
		Kind string `json:"kind"`
		Item struct {
			ID       int `json:"id"`
			Messages []struct {
				ID      int    `json:"id"`
				Content string `json:"content"`
			} `json:"messages"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput: %s", err, output)
	}

	if result.Kind != "conversations.get" {
		t.Errorf("expected kind 'conversations.get', got %q", result.Kind)
	}
	if result.Item.ID != 123 {
		t.Errorf("expected conversation ID 123, got %d", result.Item.ID)
	}
	if len(result.Item.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(result.Item.Messages))
	}
	if result.Item.Messages[0].Content != "Hello" {
		t.Errorf("expected first message content 'Hello', got %q", result.Item.Messages[0].Content)
	}
}

func TestConversationsGetWithMessagesLimit(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"contact_id": 456,
			"status": "open",
			"created_at": 1700000000
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "conversation_id": 123, "message_type": 0, "content": "Message 1", "created_at": 1700000100},
				{"id": 2, "conversation_id": 123, "message_type": 0, "content": "Message 2", "created_at": 1700000200},
				{"id": 3, "conversation_id": 123, "message_type": 0, "content": "Message 3", "created_at": 1700000300},
				{"id": 4, "conversation_id": 123, "message_type": 0, "content": "Message 4", "created_at": 1700000400},
				{"id": 5, "conversation_id": 123, "message_type": 0, "content": "Message 5", "created_at": 1700000500}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "--with-messages", "--message-limit", "2", "--output", "agent"})
		if err != nil {
			t.Errorf("conversations get --with-messages --message-limit 2 failed: %v", err)
		}
	})

	var result struct {
		Item struct {
			Messages []struct {
				ID int `json:"id"`
			} `json:"messages"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput: %s", err, output)
	}

	if len(result.Item.Messages) != 2 {
		t.Errorf("expected 2 messages (limited), got %d", len(result.Item.Messages))
	}
}

func TestConversationsGetWithMessagesNoAgent(t *testing.T) {
	// --with-messages should only work in agent mode
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"contact_id": 456,
			"status": "open",
			"created_at": 1700000000
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "--with-messages", "--output", "json"})
		if err != nil {
			t.Errorf("conversations get --with-messages failed: %v", err)
		}
	})

	// Should return normal JSON output without messages field
	var result struct {
		ID       int   `json:"id"`
		Messages []any `json:"messages"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput: %s", err, output)
	}

	if result.ID != 123 {
		t.Errorf("expected ID 123, got %d", result.ID)
	}
	// messages field should not be present in regular JSON output
	if result.Messages != nil {
		t.Errorf("expected no messages field in non-agent output, got %v", result.Messages)
	}
}

func TestConversationsGetContext(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"contact_id": 456,
			"status": "open",
			"created_at": 1700000000
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "conversation_id": 123, "message_type": 0, "content": "Hello", "created_at": 1700000100},
				{"id": 2, "conversation_id": 123, "message_type": 1, "content": "Hi there", "created_at": 1700000200},
				{"id": 3, "conversation_id": 123, "message_type": 0, "content": "How can I help?", "created_at": 1700000300}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {
				"id": 456,
				"name": "John Doe",
				"email": "john@example.com",
				"phone_number": "+1234567890"
			}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456/conversations", jsonResponse(200, `{
			"payload": [
				{"id": 123, "status": "open", "created_at": 1700000000, "last_activity_at": 1700000300},
				{"id": 100, "status": "resolved", "created_at": 1699000000, "last_activity_at": 1699500000}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "--context", "--output", "agent"})
		if err != nil {
			t.Errorf("conversations get --context failed: %v", err)
		}
	})

	var result struct {
		Kind string `json:"kind"`
		Item struct {
			Conversation struct {
				ID     int    `json:"id"`
				Status string `json:"status"`
			} `json:"conversation"`
			Messages []struct {
				ID      int    `json:"id"`
				Content string `json:"content"`
			} `json:"messages"`
			Contact *struct {
				ID           int    `json:"id"`
				Name         string `json:"name"`
				Email        string `json:"email"`
				Relationship *struct {
					TotalConversations int `json:"total_conversations"`
					OpenConversations  int `json:"open_conversations"`
				} `json:"relationship"`
			} `json:"contact"`
		} `json:"item"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput: %s", err, output)
	}

	// Verify conversation
	if result.Item.Conversation.ID != 123 {
		t.Errorf("expected conversation ID 123, got %d", result.Item.Conversation.ID)
	}
	if result.Item.Conversation.Status != "open" {
		t.Errorf("expected status open, got %s", result.Item.Conversation.Status)
	}

	// Verify messages
	if len(result.Item.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(result.Item.Messages))
	}

	// Verify contact
	if result.Item.Contact == nil {
		t.Fatal("expected contact, got nil")
	}
	if result.Item.Contact.ID != 456 {
		t.Errorf("expected contact ID 456, got %d", result.Item.Contact.ID)
	}
	if result.Item.Contact.Name != "John Doe" {
		t.Errorf("expected contact name 'John Doe', got %q", result.Item.Contact.Name)
	}

	// Verify relationship
	if result.Item.Contact.Relationship == nil {
		t.Fatal("expected relationship, got nil")
	}
	if result.Item.Contact.Relationship.TotalConversations != 2 {
		t.Errorf("expected 2 total conversations, got %d", result.Item.Contact.Relationship.TotalConversations)
	}
	if result.Item.Contact.Relationship.OpenConversations != 1 {
		t.Errorf("expected 1 open conversation, got %d", result.Item.Contact.Relationship.OpenConversations)
	}
}

func TestConversationsGetContextWithMessageLimit(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"contact_id": 456,
			"status": "open",
			"created_at": 1700000000
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "conversation_id": 123, "message_type": 0, "content": "Message 1", "created_at": 1700000100},
				{"id": 2, "conversation_id": 123, "message_type": 0, "content": "Message 2", "created_at": 1700000200},
				{"id": 3, "conversation_id": 123, "message_type": 0, "content": "Message 3", "created_at": 1700000300},
				{"id": 4, "conversation_id": 123, "message_type": 0, "content": "Message 4", "created_at": 1700000400},
				{"id": 5, "conversation_id": 123, "message_type": 0, "content": "Message 5", "created_at": 1700000500}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {"id": 456, "name": "Jane Doe"}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456/conversations", jsonResponse(200, `{
			"payload": [{"id": 123, "status": "open", "created_at": 1700000000}]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "--context", "--message-limit", "2", "--output", "agent"})
		if err != nil {
			t.Errorf("conversations get --context --message-limit 2 failed: %v", err)
		}
	})

	var result struct {
		Item struct {
			Messages []struct {
				ID int `json:"id"`
			} `json:"messages"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput: %s", err, output)
	}

	if len(result.Item.Messages) != 2 {
		t.Errorf("expected 2 messages (limited), got %d", len(result.Item.Messages))
	}

	// Should return the most recent messages (IDs 4 and 5)
	if len(result.Item.Messages) >= 2 {
		if result.Item.Messages[0].ID != 4 {
			t.Errorf("expected first message ID 4, got %d", result.Item.Messages[0].ID)
		}
		if result.Item.Messages[1].ID != 5 {
			t.Errorf("expected second message ID 5, got %d", result.Item.Messages[1].ID)
		}
	}
}

func TestConversationsGetContextNoAgent(t *testing.T) {
	// --context should only work in agent mode
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"status": "open",
			"created_at": 1700000000
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "--context", "--output", "json"})
		if err != nil {
			t.Errorf("conversations get --context failed: %v", err)
		}
	})

	// Should return normal JSON output without context fields
	var result struct {
		ID       int   `json:"id"`
		Messages []any `json:"messages"`
		Contact  any   `json:"contact"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput: %s", err, output)
	}

	if result.ID != 123 {
		t.Errorf("expected ID 123, got %d", result.ID)
	}
	// messages and contact fields should not be present in regular JSON output
	if result.Messages != nil {
		t.Errorf("expected no messages field in non-agent output, got %v", result.Messages)
	}
	if result.Contact != nil {
		t.Errorf("expected no contact field in non-agent output, got %v", result.Contact)
	}
}

func TestConversationsGetSuggestedActions(t *testing.T) {
	// Set last_activity_at to more than 24 hours ago to trigger "high" priority reply suggestion
	oldActivityAt := time.Now().Add(-48 * time.Hour).Unix()

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"contact_id": 456,
			"status": "open",
			"unread_count": 3,
			"muted": false,
			"created_at": 1700000000,
			"last_activity_at": `+strconv.FormatInt(oldActivityAt, 10)+`
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "conversation_id": 123, "message_type": 0, "content": "Hello", "created_at": 1700000100},
				{"id": 2, "conversation_id": 123, "message_type": 1, "content": "Hi there", "created_at": 1700000200}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456/conversations", jsonResponse(200, `{
			"payload": [
				{"id": 123, "status": "open", "created_at": 1700000000, "last_activity_at": 1700000300}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "--suggested-actions", "--output", "agent"})
		if err != nil {
			t.Errorf("conversations get --suggested-actions failed: %v", err)
		}
	})

	// Verify it's valid JSON with expected structure including suggested_actions
	var result struct {
		Kind string `json:"kind"`
		Item struct {
			ID               int `json:"id"`
			SuggestedActions []struct {
				Action   string `json:"action"`
				Reason   string `json:"reason"`
				Priority string `json:"priority"`
			} `json:"suggested_actions"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput: %s", err, output)
	}

	if result.Kind != "conversations.get" {
		t.Errorf("expected kind 'conversations.get', got %q", result.Kind)
	}
	if result.Item.ID != 123 {
		t.Errorf("expected conversation ID 123, got %d", result.Item.ID)
	}
	if len(result.Item.SuggestedActions) == 0 {
		t.Error("expected at least one suggested action, got none")
	}

	// Verify at least one action is "reply" (for unread conversation)
	hasReplyAction := false
	for _, action := range result.Item.SuggestedActions {
		if action.Action == "reply" {
			hasReplyAction = true
			break
		}
	}
	if !hasReplyAction {
		t.Errorf("expected 'reply' action for unread conversation, got actions: %+v", result.Item.SuggestedActions)
	}
}

func TestConversationsGetExplain(t *testing.T) {
	// Set last_activity_at to more than 72 hours ago and >3 unread messages to trigger high urgency
	oldActivityAt := time.Now().Add(-96 * time.Hour).Unix()

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"contact_id": 456,
			"status": "open",
			"unread_count": 5,
			"muted": false,
			"created_at": 1700000000,
			"last_activity_at": `+strconv.FormatInt(oldActivityAt, 10)+`
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "conversation_id": 123, "message_type": 0, "content": "Hello", "created_at": 1700000100},
				{"id": 2, "conversation_id": 123, "message_type": 1, "content": "Hi there", "created_at": 1700000200},
				{"id": 3, "conversation_id": 123, "message_type": 0, "content": "這個很急，請盡快回覆", "created_at": 1700000300}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456/conversations", jsonResponse(200, `{
			"payload": [
				{"id": 123, "status": "open", "created_at": 1700000000, "last_activity_at": 1700000300},
				{"id": 100, "status": "resolved", "created_at": 1600000000, "last_activity_at": 1600000300}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "--explain", "--output", "agent"})
		if err != nil {
			t.Errorf("conversations get --explain failed: %v", err)
		}
	})

	// Verify it's valid JSON with expected structure including _explanation
	var result struct {
		Kind string `json:"kind"`
		Item struct {
			ID          int `json:"id"`
			Explanation *struct {
				Urgency       string   `json:"urgency"`
				Reasons       []string `json:"reasons"`
				SentimentHint string   `json:"sentiment_hint"`
				Context       string   `json:"context"`
			} `json:"_explanation"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput: %s", err, output)
	}

	if result.Kind != "conversations.get" {
		t.Errorf("expected kind 'conversations.get', got %q", result.Kind)
	}
	if result.Item.ID != 123 {
		t.Errorf("expected conversation ID 123, got %d", result.Item.ID)
	}
	if result.Item.Explanation == nil {
		t.Fatal("expected _explanation object, got nil")
	}

	// Verify urgency is "high" due to long wait time and urgency keywords in Chinese
	if result.Item.Explanation.Urgency != "high" {
		t.Errorf("expected urgency 'high', got %q", result.Item.Explanation.Urgency)
	}

	// Verify reasons are populated
	if len(result.Item.Explanation.Reasons) == 0 {
		t.Error("expected at least one reason, got none")
	}

	// Verify context mentions returning customer (has 2 conversations)
	if result.Item.Explanation.Context == "" {
		t.Error("expected non-empty context")
	}
}

func TestConversationsTriage(t *testing.T) {
	// Set up current time for wait time calculation
	now := time.Now()
	// Conversation 1: older (longer wait), last activity 2 hours ago
	oldActivityTime := now.Add(-2 * time.Hour).Unix()
	// Conversation 2: newer (shorter wait), last activity 30 minutes ago
	newActivityTime := now.Add(-30 * time.Minute).Unix()

	handler := newRouteHandler().
		// First call: list open conversations with status=open
		On("GET", "/api/v1/accounts/1/conversations", func(w http.ResponseWriter, r *http.Request) {
			status := r.URL.Query().Get("status")
			if status != "open" && status != "pending" {
				// Return all for other status queries
				jsonResponse(200, `{
					"data": {
						"payload": [],
						"meta": {"total_pages": 0}
					}
				}`)(w, r)
				return
			}
			jsonResponse(200, `{
				"data": {
					"payload": [
						{
							"id": 1,
							"inbox_id": 10,
							"status": "open",
							"unread_count": 2,
							"last_activity_at": `+strconv.FormatInt(newActivityTime, 10)+`,
							"meta": {"sender": {"name": "Alice Customer"}}
						},
						{
							"id": 2,
							"inbox_id": 20,
							"status": "open",
							"unread_count": 1,
							"last_activity_at": `+strconv.FormatInt(oldActivityTime, 10)+`,
							"meta": {"sender": {"name": "Bob Waiting"}}
						}
					],
					"meta": {"total_pages": 1}
				}
			}`)(w, r)
		}).
		// Messages for conversation 1 (newer)
		On("GET", "/api/v1/accounts/1/conversations/1/messages", jsonResponse(200, `{
			"payload": [
				{"id": 101, "content": "Hello from Alice, this is a test message", "message_type": 0, "created_at": `+strconv.FormatInt(newActivityTime, 10)+`}
			]
		}`)).
		// Messages for conversation 2 (older)
		On("GET", "/api/v1/accounts/1/conversations/2/messages", jsonResponse(200, `{
			"payload": [
				{"id": 201, "content": "Bob has been waiting for a response please help", "message_type": 0, "created_at": `+strconv.FormatInt(oldActivityTime, 10)+`}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "triage"})
		if err != nil {
			t.Fatalf("conversations triage failed: %v", err)
		}
	})

	// Verify older conversation (ID 2, Bob) appears first in text output
	// The output should have Bob before Alice since Bob has been waiting longer
	bobPos := strings.Index(output, "Bob")
	alicePos := strings.Index(output, "Alice")

	if bobPos == -1 {
		t.Errorf("expected Bob Waiting in output, got: %s", output)
	}
	if alicePos == -1 {
		t.Errorf("expected Alice Customer in output, got: %s", output)
	}
	if bobPos > alicePos && alicePos != -1 {
		t.Errorf("expected Bob (older) to appear before Alice (newer), but Alice at %d, Bob at %d\nOutput: %s", alicePos, bobPos, output)
	}
}

func TestConversationsTriage_JSON(t *testing.T) {
	now := time.Now()
	oldActivityTime := now.Add(-2 * time.Hour).Unix()
	newActivityTime := now.Add(-30 * time.Minute).Unix()

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", func(w http.ResponseWriter, r *http.Request) {
			status := r.URL.Query().Get("status")
			if status != "open" && status != "pending" {
				jsonResponse(200, `{
					"data": {
						"payload": [],
						"meta": {"total_pages": 0}
					}
				}`)(w, r)
				return
			}
			jsonResponse(200, `{
				"data": {
					"payload": [
						{
							"id": 1,
							"inbox_id": 10,
							"status": "open",
							"unread_count": 2,
							"last_activity_at": `+strconv.FormatInt(newActivityTime, 10)+`,
							"meta": {"sender": {"name": "Alice Customer"}}
						},
						{
							"id": 2,
							"inbox_id": 20,
							"status": "open",
							"unread_count": 1,
							"last_activity_at": `+strconv.FormatInt(oldActivityTime, 10)+`,
							"meta": {"sender": {"name": "Bob Waiting"}}
						}
					],
					"meta": {"total_pages": 1}
				}
			}`)(w, r)
		}).
		On("GET", "/api/v1/accounts/1/conversations/1/messages", jsonResponse(200, `{
			"payload": [
				{"id": 101, "content": "Hello from Alice", "message_type": 0, "created_at": `+strconv.FormatInt(newActivityTime, 10)+`}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/2/messages", jsonResponse(200, `{
			"payload": [
				{"id": 201, "content": "Bob has been waiting", "message_type": 0, "created_at": `+strconv.FormatInt(oldActivityTime, 10)+`}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "triage", "--output", "json"})
		if err != nil {
			t.Fatalf("conversations triage --output json failed: %v", err)
		}
	})

	// Verify JSON structure
	var result struct {
		Items []struct {
			ID          int    `json:"id"`
			ContactName string `json:"contact_name"`
			WaitTime    string `json:"wait_time"`
			LastMessage string `json:"last_message"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput: %s", err, output)
	}

	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}

	// First item should be Bob (older, longer wait)
	if result.Items[0].ID != 2 {
		t.Errorf("expected first item to be conversation 2 (Bob), got %d", result.Items[0].ID)
	}
	if result.Items[0].ContactName != "Bob Waiting" {
		t.Errorf("expected first contact name 'Bob Waiting', got %q", result.Items[0].ContactName)
	}
	if result.Items[0].WaitTime == "" {
		t.Error("expected wait_time to be populated")
	}

	// Second item should be Alice (newer, shorter wait)
	if result.Items[1].ID != 1 {
		t.Errorf("expected second item to be conversation 1 (Alice), got %d", result.Items[1].ID)
	}
	if result.Items[1].ContactName != "Alice Customer" {
		t.Errorf("expected second contact name 'Alice Customer', got %q", result.Items[1].ContactName)
	}
}

func TestConversationsTriage_Light_FilteredInbox(t *testing.T) {
	activityTime := time.Now().Add(-90 * time.Minute).Unix()

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", func(w http.ResponseWriter, r *http.Request) {
			if got := r.URL.Query().Get("inbox_id"); got != "48" {
				t.Fatalf("expected inbox_id=48, got %q", got)
			}
			switch r.URL.Query().Get("status") {
			case "open":
				jsonResponse(200, `{"data":{"payload":[],"meta":{"total_pages":1}}}`)(w, r)
			case "pending":
				jsonResponse(200, `{
					"data": {
						"payload": [
							{
								"id": 43470,
								"inbox_id": 48,
								"status": "pending",
								"unread_count": 10,
								"last_activity_at": `+strconv.FormatInt(activityTime, 10)+`,
								"last_non_activity_message": {"content": "能協助看是否還有庫存"},
								"meta": {"sender": {"name": "Shani Chiang"}}
							}
						],
						"meta": {"total_pages": 1}
					}
				}`)(w, r)
			default:
				jsonResponse(200, `{"data":{"payload":[],"meta":{"total_pages":1}}}`)(w, r)
			}
		})

	setupTestEnvWithHandler(t, handler)
	t.Setenv("CHATWOOT_OUTPUT", "agent")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "triage", "--inbox", "48", "--light", "--brief"})
		if err != nil {
			t.Fatalf("conversations triage --inbox 48 --light --brief failed: %v", err)
		}
	})
	if strings.Contains(output, "\n  ") {
		t.Fatalf("expected --light output to be compact by default, got pretty JSON:\n%s", output)
	}

	if strings.Contains(output, `"kind"`) || strings.Contains(output, `"data"`) {
		t.Fatalf("light triage should bypass agent envelope, got: %s", output)
	}
	if strings.Contains(output, `"wt"`) {
		t.Fatalf("light triage should omit wait time in light mode, got: %s", output)
	}

	var result struct {
		Items []struct {
			ID          int    `json:"id"`
			Status      string `json:"st"`
			InboxID     *int   `json:"ib,omitempty"`
			Unread      int    `json:"ur"`
			LastMessage string `json:"lm"`
			ContactName string `json:"cnm"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to decode light triage output: %v\noutput: %s", err, output)
	}

	if len(result.Items) != 1 {
		t.Fatalf("expected 1 triage item, got %d", len(result.Items))
	}

	item := result.Items[0]
	if item.ID != 43470 {
		t.Fatalf("expected id 43470, got %d", item.ID)
	}
	if item.Status != "p" {
		t.Fatalf("expected short status p, got %q", item.Status)
	}
	if item.InboxID != nil {
		t.Fatalf("expected ib to be omitted when --inbox is provided, got %v", *item.InboxID)
	}
	if item.Unread != 10 {
		t.Fatalf("expected unread 10, got %d", item.Unread)
	}
	if item.LastMessage != "能協助看是否還有庫存" {
		t.Fatalf("expected last message from last_non_activity_message, got %q", item.LastMessage)
	}
	if item.ContactName != "Shani Chiang" {
		t.Fatalf("expected contact name Shani Chiang, got %q", item.ContactName)
	}
}

func TestConversationsTriage_Light_CompactCanBeDisabled(t *testing.T) {
	activityTime := time.Now().Add(-90 * time.Minute).Unix()

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Query().Get("status") {
			case "open":
				jsonResponse(200, `{"data":{"payload":[],"meta":{"total_pages":1}}}`)(w, r)
			case "pending":
				jsonResponse(200, `{
					"data": {
						"payload": [
							{
								"id": 43470,
								"inbox_id": 48,
								"status": "pending",
								"unread_count": 10,
								"last_activity_at": `+strconv.FormatInt(activityTime, 10)+`,
								"last_non_activity_message": {"content": "Need help"},
								"meta": {"sender": {"name": "Shani Chiang"}}
							}
						],
						"meta": {"total_pages": 1}
					}
				}`)(w, r)
			default:
				jsonResponse(200, `{"data":{"payload":[],"meta":{"total_pages":1}}}`)(w, r)
			}
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"conversations", "triage", "--inbox", "48", "--light", "--brief", "--compact-json=false",
		})
		if err != nil {
			t.Fatalf("conversations triage --light --compact-json=false failed: %v", err)
		}
	})

	if !strings.Contains(output, "\n  \"items\"") {
		t.Fatalf("expected pretty JSON when compact is explicitly disabled, got:\n%s", output)
	}
}

func TestConversationsTriage_Light_UnfilteredIncludesInbox(t *testing.T) {
	openActivityTime := time.Now().Add(-2 * time.Hour).Unix()
	pendingActivityTime := time.Now().Add(-1 * time.Hour).Unix()

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Query().Get("status") {
			case "open":
				jsonResponse(200, `{
					"data": {
						"payload": [
							{
								"id": 99,
								"inbox_id": 48,
								"status": "open",
								"unread_count": 2,
								"last_activity_at": `+strconv.FormatInt(openActivityTime, 10)+`,
								"last_non_activity_message": {"content": "Need help"},
								"meta": {"sender": {"name": "Alice"}}
							}
						],
						"meta": {"total_pages": 1}
					}
				}`)(w, r)
			case "pending":
				jsonResponse(200, `{
					"data": {
						"payload": [
							{
								"id": 100,
								"inbox_id": 77,
								"status": "pending",
								"unread_count": 1,
								"last_activity_at": `+strconv.FormatInt(pendingActivityTime, 10)+`,
								"last_non_activity_message": {"content": "Still waiting"},
								"meta": {"sender": {"name": "Bob"}}
							}
						],
						"meta": {"total_pages": 1}
					}
				}`)(w, r)
			default:
				jsonResponse(200, `{"data":{"payload":[],"meta":{"total_pages":1}}}`)(w, r)
			}
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "triage", "--light", "--brief", "--output", "json"})
		if err != nil {
			t.Fatalf("conversations triage --light --brief --output json failed: %v", err)
		}
	})

	var result struct {
		Items []struct {
			ID      int    `json:"id"`
			InboxID *int   `json:"ib,omitempty"`
			Status  string `json:"st"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to decode light triage output: %v\noutput: %s", err, output)
	}

	if len(result.Items) != 2 {
		t.Fatalf("expected 2 triage items, got %d", len(result.Items))
	}
	if strings.Contains(output, `"wt"`) {
		t.Fatalf("light triage should omit wait time in light mode, got: %s", output)
	}

	inboxesByID := map[int]int{}
	statusByID := map[int]string{}
	for _, item := range result.Items {
		statusByID[item.ID] = item.Status
		if item.InboxID == nil {
			t.Fatalf("expected ib to be present when no inbox filter is provided for item %d", item.ID)
		}
		inboxesByID[item.ID] = *item.InboxID
	}

	if statusByID[99] != "o" {
		t.Fatalf("expected id 99 to have short status o, got %q", statusByID[99])
	}
	if statusByID[100] != "p" {
		t.Fatalf("expected id 100 to have short status p, got %q", statusByID[100])
	}
	if inboxesByID[99] != 48 {
		t.Fatalf("expected id 99 to have ib=48, got %d", inboxesByID[99])
	}
	if inboxesByID[100] != 77 {
		t.Fatalf("expected id 100 to have ib=77, got %d", inboxesByID[100])
	}
}

func TestConversationsTriage_IncludesPendingStatus(t *testing.T) {
	now := time.Now().Unix()
	var openCalls atomic.Int32
	var pendingCalls atomic.Int32

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Query().Get("status") {
			case "open":
				openCalls.Add(1)
				jsonResponse(200, `{
					"data": {
						"payload": [
							{"id": 1, "inbox_id": 10, "status": "open", "unread_count": 1, "last_activity_at": `+strconv.FormatInt(now-300, 10)+`, "meta": {"sender": {"name": "Open Customer"}}}
						],
						"meta": {"total_pages": 1}
					}
				}`)(w, r)
			case "pending":
				pendingCalls.Add(1)
				jsonResponse(200, `{
					"data": {
						"payload": [
							{"id": 2, "inbox_id": 20, "status": "pending", "unread_count": 1, "last_activity_at": `+strconv.FormatInt(now-600, 10)+`, "meta": {"sender": {"name": "Pending Customer"}}}
						],
						"meta": {"total_pages": 1}
					}
				}`)(w, r)
			default:
				jsonResponse(200, `{"data":{"payload":[],"meta":{"total_pages":0}}}`)(w, r)
			}
		}).
		On("GET", "/api/v1/accounts/1/conversations/1/messages", jsonResponse(200, `{"payload":[{"id":101,"content":"Open message","message_type":0}]}`)).
		On("GET", "/api/v1/accounts/1/conversations/2/messages", jsonResponse(200, `{"payload":[{"id":201,"content":"Pending message","message_type":0}]}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "triage", "--output", "json"})
		if err != nil {
			t.Fatalf("conversations triage failed: %v", err)
		}
	})

	var result struct {
		Items []struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to decode triage output: %v\noutput: %s", err, output)
	}

	if openCalls.Load() == 0 {
		t.Fatal("expected open conversations query to be called")
	}
	if pendingCalls.Load() == 0 {
		t.Fatal("expected pending conversations query to be called")
	}

	if len(result.Items) != 2 {
		t.Fatalf("expected 2 triage items, got %d", len(result.Items))
	}

	statuses := map[int]string{}
	for _, item := range result.Items {
		statuses[item.ID] = item.Status
	}
	if statuses[1] != "open" {
		t.Fatalf("expected conversation 1 status=open, got %q", statuses[1])
	}
	if statuses[2] != "pending" {
		t.Fatalf("expected conversation 2 status=pending, got %q", statuses[2])
	}
}

func TestConversationsWatch_InvalidInterval(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"conversations", "watch", "--interval", "0"})
	if err == nil {
		t.Fatal("expected error for invalid interval")
	}
	if !strings.Contains(err.Error(), "--interval must be greater than 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExpandFilterPayload(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "shortcode keys and values expand",
			input:    `{"payload":[{"ak":"st","fo":"eq","v":["open"]}]}`,
			expected: `{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]}`,
		},
		{
			name:     "bare array gets wrapped and expanded",
			input:    `[{"ak":"ii","fo":"ne","v":[48]}]`,
			expected: `{"payload":[{"attribute_key":"inbox_id","filter_operator":"not_equal_to","values":[48]}]}`,
		},
		{
			name:     "query_operator uppercased",
			input:    `{"payload":[{"ak":"st","fo":"eq","v":["open"],"qo":"and"}]}`,
			expected: `{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"],"query_operator":"AND"}]}`,
		},
		{
			name:     "already uppercase query_operator preserved",
			input:    `{"payload":[{"ak":"st","fo":"eq","v":["open"],"qo":"OR"}]}`,
			expected: `{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"],"query_operator":"OR"}]}`,
		},
		{
			name:     "unknown shortcodes passed through",
			input:    `{"payload":[{"ak":"custom_field","fo":"zz","v":["x"],"extra":"y"}]}`,
			expected: `{"payload":[{"attribute_key":"custom_field","filter_operator":"zz","values":["x"],"extra":"y"}]}`,
		},
		{
			name:     "full keys not double-expanded",
			input:    `{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]}`,
			expected: `{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]}`,
		},
		{
			name:     "pl shortcode expands to payload",
			input:    `{"pl":[{"ak":"st","fo":"eq","v":["open"]}]}`,
			expected: `{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]}`,
		},
		{
			name:     "multiple conditions",
			input:    `{"payload":[{"ak":"st","fo":"eq","v":["open"],"qo":"and"},{"ak":"ii","fo":"eq","v":[15,20,28]}]}`,
			expected: `{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"],"query_operator":"AND"},{"attribute_key":"inbox_id","filter_operator":"equal_to","values":[15,20,28]}]}`,
		},
		{
			name:     "all attribute shortcodes",
			input:    `{"payload":[{"ak":"ai","fo":"eq","v":[1]},{"ak":"ti","fo":"eq","v":[2]},{"ak":"pr","fo":"eq","v":["urgent"]},{"ak":"lb","fo":"co","v":["vip"]}]}`,
			expected: `{"payload":[{"attribute_key":"assignee_id","filter_operator":"equal_to","values":[1]},{"attribute_key":"team_id","filter_operator":"equal_to","values":[2]},{"attribute_key":"priority","filter_operator":"equal_to","values":["urgent"]},{"attribute_key":"label_list","filter_operator":"contains","values":["vip"]}]}`,
		},
		{
			name:     "all operator shortcodes",
			input:    `{"payload":[{"ak":"st","fo":"co","v":["a"]},{"ak":"st","fo":"nc","v":["b"]},{"ak":"st","fo":"ip","v":[]},{"ak":"st","fo":"np","v":[]}]}`,
			expected: `{"payload":[{"attribute_key":"status","filter_operator":"contains","values":["a"]},{"attribute_key":"status","filter_operator":"does_not_contain","values":["b"]},{"attribute_key":"status","filter_operator":"is_present","values":[]},{"attribute_key":"status","filter_operator":"is_not_present","values":[]}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input map[string]any
			raw := tt.input

			// Simulate the same bare-array wrapping that the command does
			if err := json.Unmarshal([]byte(raw), &input); err != nil {
				var arr []any
				if arrErr := json.Unmarshal([]byte(raw), &arr); arrErr != nil {
					t.Fatalf("invalid test input JSON: %v", err)
				}
				input = map[string]any{"payload": arr}
			}

			result := expandFilterPayload(input)

			var expected map[string]any
			if err := json.Unmarshal([]byte(tt.expected), &expected); err != nil {
				t.Fatalf("invalid expected JSON: %v", err)
			}

			// Marshal and re-unmarshal both to normalize number types (float64)
			resultJSON, _ := json.Marshal(result)
			expectedJSON, _ := json.Marshal(expected)

			var resultNorm, expectedNorm any
			_ = json.Unmarshal(resultJSON, &resultNorm)
			_ = json.Unmarshal(expectedJSON, &expectedNorm)

			if !reflect.DeepEqual(resultNorm, expectedNorm) {
				t.Errorf("expandFilterPayload mismatch\n  got:  %s\n  want: %s", resultJSON, expectedJSON)
			}
		})
	}
}

func TestConversationsTriage_Brief(t *testing.T) {
	now := time.Now()
	activityTime := now.Add(-1 * time.Hour).Unix()

	var callPaths []string
	var mu sync.Mutex

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			callPaths = append(callPaths, r.URL.Path+"?status="+r.URL.Query().Get("status"))
			mu.Unlock()

			status := r.URL.Query().Get("status")
			if status != "open" && status != "pending" {
				jsonResponse(200, `{"data":{"payload":[],"meta":{"total_pages":0}}}`)(w, r)
				return
			}
			if status == "pending" {
				jsonResponse(200, `{"data":{"payload":[],"meta":{"total_pages":0}}}`)(w, r)
				return
			}
			jsonResponse(200, `{
				"data": {
					"payload": [{
						"id": 1,
						"inbox_id": 48,
						"status": "open",
						"unread_count": 2,
						"last_activity_at": `+strconv.FormatInt(activityTime, 10)+`,
						"last_non_activity_message": {"content": "Need help with order"},
						"meta": {"sender": {"id": 10, "name": "Alice"}}
					}],
					"meta": {"total_pages": 1}
				}
			}`)(w, r)
		}).
		On("GET", "/api/v1/accounts/1/conversations/1/messages", func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			callPaths = append(callPaths, r.URL.Path)
			mu.Unlock()
			jsonResponse(200, `{"payload": [{"id": 101, "content": "Should not be fetched", "message_type": 0}]}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "triage", "--brief", "-o", "json"})
		if err != nil {
			t.Fatalf("triage --brief failed: %v", err)
		}
	})

	// Verify messages endpoint was NOT called
	mu.Lock()
	for _, path := range callPaths {
		if strings.Contains(path, "/messages") {
			t.Errorf("--brief should skip Messages().List calls, but got request to %s", path)
		}
	}
	mu.Unlock()

	// Verify last_non_activity_message content appears in output
	if !strings.Contains(output, "Need help with order") {
		t.Errorf("expected last_non_activity_message in output, got: %s", output)
	}
}
