package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestPrintConversationsTable(t *testing.T) {
	tests := []struct {
		name           string
		conversations  []api.Conversation
		expectedOutput []string
	}{
		{
			name:          "empty conversations",
			conversations: []api.Conversation{},
			expectedOutput: []string{
				"ID",
				"INBOX",
				"STATUS",
				"PRIORITY",
				"UNREAD",
				"CREATED",
				"LAST_ACTIVITY",
			},
		},
		{
			name: "single conversation with all fields",
			conversations: []api.Conversation{
				{
					ID:        123,
					DisplayID: intPtr(456),
					InboxID:   1,
					Status:    "open",
					Priority:  strPtr("high"),
					Unread:    3,
					CreatedAt: 1700000000,
				},
			},
			expectedOutput: []string{
				"456", // DisplayID used when present
				"1",   // InboxID
				"open",
				"high",
				"3",
				"2023-11-14", // Date portion of formatted time
			},
		},
		{
			name: "conversation without optional fields",
			conversations: []api.Conversation{
				{
					ID:        789,
					InboxID:   2,
					Status:    "resolved",
					Priority:  nil, // No priority
					Unread:    0,
					CreatedAt: 1700000000,
				},
			},
			expectedOutput: []string{
				"789", // ID used when DisplayID is nil
				"2",
				"resolved",
				"-", // Priority placeholder
				"0",
			},
		},
		{
			name: "multiple conversations",
			conversations: []api.Conversation{
				{
					ID:        1,
					DisplayID: intPtr(10),
					InboxID:   1,
					Status:    "open",
					Priority:  strPtr("urgent"),
					Unread:    5,
					CreatedAt: 1700000000,
				},
				{
					ID:        2,
					InboxID:   2,
					Status:    "pending",
					Priority:  nil,
					Unread:    0,
					CreatedAt: 1700001000,
				},
				{
					ID:        3,
					DisplayID: intPtr(30),
					InboxID:   1,
					Status:    "resolved",
					Priority:  strPtr("low"),
					Unread:    1,
					CreatedAt: 1700002000,
				},
			},
			expectedOutput: []string{
				"10", "1", "open", "urgent", "5",
				"2", "2", "pending", "-", "0",
				"30", "1", "resolved", "low", "1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printConversationsTable(w, tt.conversations)

			// Restore stdout
			_ = w.Close()
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			output := buf.String()

			// Verify expected strings are present
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, output)
				}
			}

			// Verify header is always present
			if !strings.Contains(output, "ID") ||
				!strings.Contains(output, "INBOX") ||
				!strings.Contains(output, "STATUS") {
				t.Errorf("Output missing expected headers. Got:\n%s", output)
			}
		})
	}
}

func TestPrintConversationsTable_Formatting(t *testing.T) {
	conversations := []api.Conversation{
		{
			ID:        1,
			DisplayID: intPtr(100),
			InboxID:   5,
			Status:    "open",
			Priority:  strPtr("high"),
			Unread:    10,
			CreatedAt: 1700000000,
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printConversationsTable(w, conversations)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Verify tabular format (multiple spaces/tabs between columns)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 lines (header + data), got %d", len(lines))
	}

	// Check that header and data rows have similar structure
	headerFields := strings.Fields(lines[0])
	dataFields := strings.Fields(lines[1])

	if len(headerFields) != 8 {
		t.Errorf("Expected 8 header fields, got %d: %v", len(headerFields), headerFields)
	}

	if len(dataFields) < 8 {
		t.Errorf("Expected at least 8 data fields, got %d: %v", len(dataFields), dataFields)
	}
}

// Helper functions for test data
func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func TestParseSnoozedUntil(t *testing.T) {
	// Get current time for relative tests
	now := time.Now().Unix()
	futureTimestamp := now + 3600                         // 1 hour from now
	farFutureTimestamp := now + (11 * 365 * 24 * 60 * 60) // 11 years from now

	// Create a timestamp far enough in the future (2026)
	futureDate := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	futureTimestampStr := strconv.FormatInt(futureDate.Unix(), 10)

	tests := []struct {
		name        string
		input       string
		expectError bool
		validate    func(*testing.T, int64)
	}{
		{
			name:        "valid unix timestamp",
			input:       futureTimestampStr,
			expectError: false,
			validate: func(t *testing.T, result int64) {
				if result != futureDate.Unix() {
					t.Errorf("Expected %d, got %d", futureDate.Unix(), result)
				}
			},
		},
		{
			name:        "valid RFC3339",
			input:       "2026-12-31T23:59:59Z",
			expectError: false,
			validate: func(t *testing.T, result int64) {
				expected := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC).Unix()
				if result != expected {
					t.Errorf("Expected %d, got %d", expected, result)
				}
			},
		},
		{
			name:        "valid RFC3339 with timezone",
			input:       "2027-01-15T10:00:00-05:00",
			expectError: false,
			validate: func(t *testing.T, result int64) {
				parsed, _ := time.Parse(time.RFC3339, "2027-01-15T10:00:00-05:00")
				expected := parsed.Unix()
				if result != expected {
					t.Errorf("Expected %d, got %d", expected, result)
				}
			},
		},
		{
			name:        "negative timestamp",
			input:       "-1",
			expectError: true,
		},
		{
			name:        "zero timestamp",
			input:       "0",
			expectError: true,
		},
		{
			name:        "timestamp in the past",
			input:       "1000000000",
			expectError: true,
		},
		{
			name:        "RFC3339 in the past",
			input:       "2020-01-01T00:00:00Z",
			expectError: true,
		},
		{
			name:        "invalid format",
			input:       "not-a-date",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "malformed RFC3339",
			input:       "2025-13-45T99:99:99Z",
			expectError: true,
		},
		{
			name:        "timestamp too far in future",
			input:       strconv.FormatInt(farFutureTimestamp, 10),
			expectError: true,
		},
		{
			name:        "RFC3339 too far in future",
			input:       "2040-01-01T00:00:00Z",
			expectError: true,
		},
		{
			name:        "valid future timestamp (1 hour from now)",
			input:       strconv.FormatInt(futureTimestamp, 10),
			expectError: false,
			validate: func(t *testing.T, result int64) {
				// Allow small timing differences
				if result < futureTimestamp-5 || result > futureTimestamp+5 {
					t.Errorf("Expected around %d, got %d", futureTimestamp, result)
				}
			},
		},
		{
			name:        "relative future time",
			input:       "30m",
			expectError: false,
			validate: func(t *testing.T, result int64) {
				expected := now + 1800
				if result < expected-10 || result > expected+10 {
					t.Errorf("Expected around %d, got %d", expected, result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSnoozedUntil(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestConversationsContextCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"contact_id": 456,
			"status": "open"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Hello, I have a question", "message_type": 0, "private": false},
				{"id": 2, "content": "Sure, how can I help?", "message_type": 1, "private": false}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {
				"id": 456,
				"name": "John Doe",
				"email": "john@example.com"
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "context", "123"})
		if err != nil {
			t.Errorf("conversations context failed: %v", err)
		}
	})

	if !strings.Contains(output, "Conversation #123") {
		t.Errorf("output missing conversation ID: %s", output)
	}
	if !strings.Contains(output, "John Doe") {
		t.Errorf("output missing contact name: %s", output)
	}
}

func TestConversationsContextCommand_URL(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{"id": 123, "contact_id": 0}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{"payload": []}`))

	setupTestEnvWithHandler(t, handler)

	// Should accept the pasted web UI URL and extract the conversation ID.
	err := Execute(context.Background(), []string{"conversations", "context", "https://app.chatwoot.com/app/accounts/1/conversations/123", "-o", "json"})
	if err != nil {
		t.Fatalf("conversations context with URL failed: %v", err)
	}
}

func TestConversationsContextCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{"id": 123, "contact_id": 0}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{"payload": []}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "context", "123", "-o", "json"})
		if err != nil {
			t.Errorf("conversations context --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"summary"`) {
		t.Errorf("JSON output missing summary: %s", output)
	}
}

func TestConversationsContextCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"contact_id": 456,
			"status": "open",
			"inbox_id": 48
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "  Hello  ", "message_type": 0, "private": false},
				{"id": 2, "content": "Sure", "message_type": 1, "private": false},
				{"id": 3, "content": "Agent assigned", "message_type": 2, "private": false},
				{"id": 4, "content": "Template", "message_type": 3, "private": false}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {
				"id": 456,
				"name": "TuTu",
				"email": "",
				"phone_number": "+15550001111"
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "context", "123", "--light"})
		if err != nil {
			t.Fatalf("conversations context --light failed: %v", err)
		}
	})

	var payload struct {
		ID      int    `json:"id"`
		St      string `json:"st"`
		Inbox   int    `json:"ib"`
		Contact struct {
			ID   *int    `json:"id"`
			Name *string `json:"nm"`
		} `json:"ct"`
		Msgs []string `json:"msgs"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	if payload.ID != 123 {
		t.Fatalf("expected id=123, got %d", payload.ID)
	}
	if payload.St != "o" {
		t.Fatalf("expected st=o, got %q", payload.St)
	}
	if payload.Inbox != 48 {
		t.Fatalf("expected ib=48, got %d", payload.Inbox)
	}
	if payload.Contact.ID == nil || *payload.Contact.ID != 456 {
		t.Fatalf("expected ct.id=456, got %#v", payload.Contact.ID)
	}
	if payload.Contact.Name == nil || *payload.Contact.Name != "TuTu" {
		t.Fatalf("expected ct.nm=TuTu, got %#v", payload.Contact.Name)
	}

	if len(payload.Msgs) != 2 {
		t.Fatalf("expected 2 non-activity messages, got %d (%#v)", len(payload.Msgs), payload.Msgs)
	}
	if payload.Msgs[0] != "Hello" || payload.Msgs[1] != "> Sure" {
		t.Fatalf("unexpected msgs payload: %#v", payload.Msgs)
	}
}

func TestConversationsContextCommand_Agent(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"contact_id": 456,
			"status": "open",
			"inbox_id": 1,
			"created_at": 1700000000
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Hello", "message_type": 0, "private": false, "created_at": 1700000001},
				{"id": 2, "content": "Internal", "message_type": 1, "private": true, "created_at": 1700000002}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {
				"id": 456,
				"name": "John Doe",
				"email": "john@example.com"
			}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456/labels", jsonResponse(200, `{
			"labels": ["vip", "trial"]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456/contactable_inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 9, "name": "Support", "channel_type": "Channel::Email"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "context", "123", "-o", "agent"})
		if err != nil {
			t.Errorf("conversations context --output agent failed: %v", err)
		}
	})

	var payload struct {
		Kind string         `json:"kind"`
		Item map[string]any `json:"item"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	if payload.Kind != "conversations.context" {
		t.Fatalf("expected kind conversations.context, got %q", payload.Kind)
	}
	messages, ok := payload.Item["messages"].([]any)
	if !ok {
		t.Fatalf("expected messages array, got %#v", payload.Item["messages"])
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	meta, ok := payload.Item["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected meta map, got %#v", payload.Item["meta"])
	}
	if meta["message_count"] != float64(2) {
		t.Fatalf("expected message_count 2, got %#v", meta["message_count"])
	}

	labels, ok := payload.Item["contact_labels"].([]any)
	if !ok || len(labels) != 2 {
		t.Fatalf("expected contact_labels with 2 entries, got %#v", payload.Item["contact_labels"])
	}
	inboxes, ok := payload.Item["contact_inboxes"].([]any)
	if !ok || len(inboxes) != 1 {
		t.Fatalf("expected contact_inboxes with 1 entry, got %#v", payload.Item["contact_inboxes"])
	}
}

func TestConversationsContextCommand_InvalidID(t *testing.T) {
	handler := newRouteHandler()
	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"conversations", "context", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestConversationsAssignCommand_WithAssignee(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/assignments", jsonResponse(200, `{
			"id": 123,
			"meta": {"assignee": {"id": 5, "name": "Agent Smith"}}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"meta": {"assignee": {"id": 5, "name": "Agent Smith"}}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "assign", "123", "--assignee-id", "5"})
		if err != nil {
			t.Errorf("conversations assign failed: %v", err)
		}
	})

	if !strings.Contains(output, "assigned") {
		t.Errorf("output missing assignment info: %s", output)
	}
}

func TestConversationsAssignCommand_WithAgentByName(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 5, "name": "Agent Smith", "email": "smith@example.com", "role": "agent"}
		]`)).
		On("POST", "/api/v1/accounts/1/conversations/123/assignments", jsonResponse(200, `{
			"id": 123,
			"meta": {"assignee": {"id": 5, "name": "Agent Smith"}}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"assignee_id": 5,
			"meta": {"assignee": {"id": 5, "name": "Agent Smith"}}
		}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"conversations", "assign", "123", "--agent", "Agent Smith", "--no-input"})
	if err != nil {
		t.Fatalf("conversations assign --agent name failed: %v", err)
	}
}

func TestConversationsAssignCommand_WithTeam(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/assignments", jsonResponse(200, `{
			"id": 123,
			"meta": {"team": {"id": 2, "name": "Support Team"}}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"meta": {"team": {"id": 2, "name": "Support Team"}}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "assign", "123", "--team-id", "2"})
		if err != nil {
			t.Errorf("conversations assign --team-id failed: %v", err)
		}
	})

	if !strings.Contains(output, "assigned") {
		t.Errorf("output missing assignment info: %s", output)
	}
}

func TestConversationsAssignCommand_InteractivePrompt(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 1, "name": "Agent One"}
		]`)).
		On("GET", "/api/v1/accounts/1/teams", jsonResponse(200, `[
			{"id": 2, "name": "Support Team"}
		]`)).
		On("POST", "/api/v1/accounts/1/conversations/123/assignments", jsonResponse(200, `{
			"id": 123
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"assignee_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)
	t.Setenv("CHATWOOT_FORCE_INTERACTIVE", "true")

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.Write([]byte("1\n0\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "assign", "123"})
		if err != nil {
			t.Errorf("conversations assign interactive failed: %v", err)
		}
	})

	if !strings.Contains(output, "Conversation #123 assigned") {
		t.Errorf("output missing assignment info: %s", output)
	}
	if !strings.Contains(output, "Agent: 1") {
		t.Errorf("output missing agent info: %s", output)
	}
}

func TestConversationsAssignCommand_WithInvalidID(t *testing.T) {
	handler := newRouteHandler()
	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"conversations", "assign", "invalid", "--assignee-id", "5"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestConversationsAssignCommand_MissingFlags(t *testing.T) {
	handler := newRouteHandler()
	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"conversations", "assign", "123", "--no-input"})
	// Should fail - either with validation error or API error when interactive mode tries to prompt
	if err == nil {
		t.Error("expected error for missing flags")
	}
}

func TestConversationsAssignCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/assignments", jsonResponse(200, `{
			"id": 123,
			"meta": {"assignee": {"id": 5, "name": "Agent Smith"}}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"status": "open",
			"assignee_id": 5,
			"meta": {"assignee": {"id": 5, "name": "Agent Smith"}}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "assign", "123", "--assignee-id", "5", "-o", "json"})
		if err != nil {
			t.Errorf("conversations assign --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing id: %s", output)
	}
}

func TestConversationsMuteCommand_Execute(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_mute", jsonResponse(200, `{}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"muted": true,
			"status": "open"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "mute", "123"})
		if err != nil {
			t.Errorf("conversations mute failed: %v", err)
		}
	})

	if !strings.Contains(output, "muted") {
		t.Errorf("output missing muted status: %s", output)
	}
}

func TestConversationsMuteCommand_Execute_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_mute", jsonResponse(200, `{}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"muted": true,
			"status": "open"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "mute", "123", "-o", "json"})
		if err != nil {
			t.Errorf("conversations mute --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"muted"`) {
		t.Errorf("JSON output missing muted field: %s", output)
	}
}

func TestConversationsUnmuteCommand_Execute(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_mute", jsonResponse(200, `{}`)).
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"muted": false,
			"status": "open"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "unmute", "123"})
		if err != nil {
			t.Errorf("conversations unmute failed: %v", err)
		}
	})

	if !strings.Contains(output, "unmuted") {
		t.Errorf("output missing unmuted status: %s", output)
	}
}

func TestConversationsMarkUnreadCommand_Execute(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"unread_count": 0,
			"status": "open"
		}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/unread", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "mark-unread", "123"})
		if err != nil {
			t.Errorf("conversations mark-unread failed: %v", err)
		}
	})

	if !strings.Contains(output, "marked as unread") {
		t.Errorf("output missing mark-unread confirmation: %s", output)
	}
}

func TestConversationsMarkUnreadCommand_Execute_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"unread_count": 1,
			"status": "open"
		}`)).
		On("POST", "/api/v1/accounts/1/conversations/123/unread", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "mark-unread", "123", "-o", "json"})
		if err != nil {
			t.Errorf("conversations mark-unread --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing id: %s", output)
	}
}

func TestConversationsSearchCommand_Execute(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 123, "inbox_id": 1, "status": "open", "unread_count": 2, "created_at": 1700000000}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "search", "test query"})
		if err != nil {
			t.Errorf("conversations search failed: %v", err)
		}
	})

	if !strings.Contains(output, "123") {
		t.Errorf("output missing conversation ID: %s", output)
	}
}

func TestConversationsSearchCommand_Execute_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
			"data": {
				"payload": [
					{"id": 123, "inbox_id": 1, "status": "open"}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "search", "test", "-o", "json"})
		if err != nil {
			t.Errorf("conversations search --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing id: %s", output)
	}
}

func TestConversationsSearchCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/search", jsonResponse(200, `{
			"data": {
				"payload": [
					{
						"id": 123,
						"inbox_id": 48,
						"status": "open",
						"unread_count": 1,
						"last_activity_at": 1700000000,
						"meta": {"sender": {"id": 456, "name": "Jane"}},
						"last_non_activity_message": {"content": "Refund please"},
						"custom_attributes": {"debug": true}
					}
				],
				"meta": {"count": 1}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "search", "refund", "--li"})
		if err != nil {
			t.Fatalf("conversations search --li failed: %v", err)
		}
	})

	var payload struct {
		Items []struct {
			ID          int    `json:"id"`
			Status      string `json:"st"`
			InboxID     int    `json:"ib"`
			LastMessage string `json:"lm"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light search output: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 result, got %d", len(payload.Items))
	}
	if payload.Items[0].Status != "o" || payload.Items[0].InboxID != 48 {
		t.Fatalf("unexpected payload: %#v", payload.Items[0])
	}
	if payload.Items[0].LastMessage != "Refund please" {
		t.Fatalf("expected last message, got %q", payload.Items[0].LastMessage)
	}
	if strings.Contains(output, `"custom_attributes"`) {
		t.Fatal("light output should not include custom_attributes")
	}
}

func TestConversationsSearchCommand_LightAll(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/search", func(w http.ResponseWriter, r *http.Request) {
			page := r.URL.Query().Get("page")
			w.Header().Set("Content-Type", "application/json")
			if page == "" || page == "1" {
				_, _ = w.Write([]byte(`{
					"data": {
						"payload": [
							{"id": 1, "status": "open", "inbox_id": 48, "last_activity_at": 1700000000, "meta": {"sender": {"id": 10, "name": "Jane"}}}
						],
						"meta": {"count": 2, "total_pages": 2}
					}
				}`))
			} else {
				_, _ = w.Write([]byte(`{
					"data": {
						"payload": [
							{"id": 2, "status": "resolved", "inbox_id": 48, "last_activity_at": 1700000100, "meta": {"sender": {"id": 20, "name": "Bob"}}}
						],
						"meta": {"count": 2, "total_pages": 2}
					}
				}`))
			}
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "search", "refund", "--light", "--all"})
		if err != nil {
			t.Fatalf("search --light --all failed: %v", err)
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
}

func TestConversationsAttachmentsCommand_Execute(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/attachments", jsonResponse(200, `{"meta": {"total_count": 1}, "payload": [
			{"id": 10, "file_type": "file", "data_url": "https://example.com/file.pdf", "file_size": 1024}
		]}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "attachments", "123"})
		if err != nil {
			t.Errorf("conversations attachments failed: %v", err)
		}
	})

	if !strings.Contains(output, "file.pdf") {
		t.Errorf("output missing attachment filename: %s", output)
	}
}

func TestConversationsAttachmentsCommand_Execute_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/attachments", jsonResponse(200, `{"meta": {"total_count": 1}, "payload": [
			{"id": 10, "file_type": "image", "data_url": "https://example.com/image.png"}
		]}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "attachments", "123", "-o", "json"})
		if err != nil {
			t.Errorf("conversations attachments --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"file_type"`) {
		t.Errorf("JSON output missing file_type: %s", output)
	}
}
