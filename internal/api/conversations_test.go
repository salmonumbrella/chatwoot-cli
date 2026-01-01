package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMarkConversationUnread(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		statusCode     int
		responseBody   string
		expectError    bool
	}{
		{
			name:           "successful mark unread",
			conversationID: 123,
			statusCode:     http.StatusOK,
			responseBody:   "",
			expectError:    false,
		},
		{
			name:           "conversation not found",
			conversationID: 999,
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error":"Conversation not found"}`,
			expectError:    true,
		},
		{
			name:           "unauthorized",
			conversationID: 123,
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error":"Unauthorized"}`,
			expectError:    true,
		},
		{
			name:           "server error",
			conversationID: 123,
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error":"Internal server error"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				// Verify request path contains conversation ID
				if !strings.Contains(r.URL.Path, "/conversations/") || !strings.HasSuffix(r.URL.Path, "/unread") {
					t.Errorf("Expected path to match /conversations/{id}/unread pattern, got %s", r.URL.Path)
				}

				// Verify headers
				if r.Header.Get("api_access_token") == "" {
					t.Error("Missing api_access_token header")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Send response
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create client
			client := newTestClient(server.URL, "test-token", 1)

			// Execute
			err := client.MarkConversationUnread(context.Background(), tt.conversationID)

			// Verify
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// For error cases, verify error contains status code
			if tt.expectError && err != nil {
				apiErr, ok := err.(*APIError)
				if !ok {
					t.Errorf("Expected APIError, got %T", err)
				} else if apiErr.StatusCode != tt.statusCode {
					t.Errorf("Expected status code %d, got %d", tt.statusCode, apiErr.StatusCode)
				}
			}
		})
	}
}

func TestListConversations(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		inboxID      string
		page         int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *ConversationList)
	}{
		{
			name:    "successful list with results",
			status:  "open",
			inboxID: "1",
			page:    1,
			responseBody: `{
				"data": {
					"meta": {
						"current_page": 1,
						"per_page": 15,
						"total_pages": 3,
						"total_count": 42
					},
					"payload": [
						{
							"id": 1,
							"account_id": 1,
							"inbox_id": 1,
							"status": "open",
							"priority": "high",
							"display_id": 100,
							"unread_count": 5,
							"created_at": 1700000000
						},
						{
							"id": 2,
							"account_id": 1,
							"inbox_id": 1,
							"status": "open",
							"display_id": 101,
							"unread_count": 0,
							"created_at": 1700001000
						}
					]
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *ConversationList) {
				if len(result.Data.Payload) != 2 {
					t.Errorf("Expected 2 conversations, got %d", len(result.Data.Payload))
				}
				if int(result.Data.Meta.TotalPages) != 3 {
					t.Errorf("Expected 3 total pages, got %d", result.Data.Meta.TotalPages)
				}
				if int(result.Data.Meta.TotalCount) != 42 {
					t.Errorf("Expected 42 total count, got %d", result.Data.Meta.TotalCount)
				}
			},
		},
		{
			name:    "empty results",
			status:  "resolved",
			inboxID: "",
			page:    1,
			responseBody: `{
				"data": {
					"meta": {
						"current_page": 1,
						"per_page": 15,
						"total_pages": 0,
						"total_count": 0
					},
					"payload": []
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *ConversationList) {
				if len(result.Data.Payload) != 0 {
					t.Errorf("Expected 0 conversations, got %d", len(result.Data.Payload))
				}
				if int(result.Data.Meta.TotalPages) != 0 {
					t.Errorf("Expected 0 total pages, got %d", result.Data.Meta.TotalPages)
				}
			},
		},
		{
			name:    "single page result",
			status:  "all",
			inboxID: "",
			page:    1,
			responseBody: `{
				"data": {
					"meta": {
						"current_page": 1,
						"per_page": 15,
						"total_pages": 1,
						"total_count": 3
					},
					"payload": [
						{
							"id": 1,
							"account_id": 1,
							"inbox_id": 1,
							"status": "open",
							"unread_count": 0,
							"created_at": 1700000000
						}
					]
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *ConversationList) {
				if int(result.Data.Meta.TotalPages) != 1 {
					t.Errorf("Expected 1 total page, got %d", result.Data.Meta.TotalPages)
				}
				if int(result.Data.Meta.CurrentPage) != 1 {
					t.Errorf("Expected current page 1, got %d", result.Data.Meta.CurrentPage)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}

				// Verify query parameters
				query := r.URL.Query()
				if tt.status != "" && tt.status != "all" {
					if query.Get("status") != tt.status {
						t.Errorf("Expected status param %s, got %s", tt.status, query.Get("status"))
					}
				}
				if tt.inboxID != "" {
					if query.Get("inbox_id") != tt.inboxID {
						t.Errorf("Expected inbox_id param %s, got %s", tt.inboxID, query.Get("inbox_id"))
					}
				}

				// Send response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create client
			client := newTestClient(server.URL, "test-token", 1)

			// Execute
			result, err := client.ListConversations(context.Background(), ListConversationsParams{
				Status:  tt.status,
				InboxID: tt.inboxID,
				Page:    tt.page,
			})

			// Verify
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Run custom validation
			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestGetConversation(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		statusCode     int
		responseBody   string
		expectError    bool
		validateFunc   func(*testing.T, *Conversation)
	}{
		{
			name:           "successful get",
			conversationID: 123,
			statusCode:     http.StatusOK,
			responseBody: `{
				"id": 123,
				"account_id": 1,
				"inbox_id": 5,
				"status": "open",
				"priority": "high",
				"display_id": 456,
				"contact_id": 789,
				"unread_count": 3,
				"muted": false,
				"created_at": 1700000000,
				"labels": ["bug", "urgent"]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, conv *Conversation) {
				if conv.ID != 123 {
					t.Errorf("Expected ID 123, got %d", conv.ID)
				}
				if conv.Status != "open" {
					t.Errorf("Expected status open, got %s", conv.Status)
				}
				if conv.Priority == nil || *conv.Priority != "high" {
					t.Errorf("Expected priority high, got %v", conv.Priority)
				}
				if len(conv.Labels) != 2 {
					t.Errorf("Expected 2 labels, got %d", len(conv.Labels))
				}
			},
		},
		{
			name:           "conversation not found",
			conversationID: 999,
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error":"Not found"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.GetConversation(context.Background(), tt.conversationID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestToggleConversationStatus(t *testing.T) {
	tests := []struct {
		name            string
		convID          int
		status          string
		snoozedUntil    int64
		responseBody    string
		expectError     bool
		validateFunc    func(*testing.T, *ToggleStatusResponse)
		validatePayload func(*testing.T, map[string]any)
	}{
		{
			name:         "successful status toggle",
			convID:       123,
			status:       "resolved",
			snoozedUntil: 0,
			responseBody: `{
				"meta": {},
				"payload": {
					"success": true,
					"conversation_id": 123,
					"current_status": "resolved"
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, resp *ToggleStatusResponse) {
				if !resp.Payload.Success {
					t.Error("Expected success to be true")
				}
				if resp.Payload.CurrentStatus != "resolved" {
					t.Errorf("Expected status resolved, got %s", resp.Payload.CurrentStatus)
				}
				if resp.Payload.ConversationID != 123 {
					t.Errorf("Expected conversation ID 123, got %d", resp.Payload.ConversationID)
				}
			},
			validatePayload: func(t *testing.T, payload map[string]any) {
				if payload["status"] != "resolved" {
					t.Errorf("Expected status resolved in payload, got %v", payload["status"])
				}
				if _, exists := payload["snoozed_until"]; exists {
					t.Error("Expected snoozed_until to not be in payload when snoozedUntil=0")
				}
			},
		},
		{
			name:         "snooze without snoozed_until",
			convID:       123,
			status:       "snoozed",
			snoozedUntil: 0,
			responseBody: `{
				"meta": {},
				"payload": {
					"success": true,
					"conversation_id": 123,
					"current_status": "snoozed"
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, resp *ToggleStatusResponse) {
				if resp.Payload.CurrentStatus != "snoozed" {
					t.Errorf("Expected status snoozed, got %s", resp.Payload.CurrentStatus)
				}
				if resp.Payload.SnoozedUntil != nil {
					t.Errorf("Expected SnoozedUntil to be nil, got %v", *resp.Payload.SnoozedUntil)
				}
			},
			validatePayload: func(t *testing.T, payload map[string]any) {
				if payload["status"] != "snoozed" {
					t.Errorf("Expected status snoozed in payload, got %v", payload["status"])
				}
				if _, exists := payload["snoozed_until"]; exists {
					t.Error("Expected snoozed_until to not be in payload when snoozedUntil=0")
				}
			},
		},
		{
			name:         "snooze with snoozed_until",
			convID:       123,
			status:       "snoozed",
			snoozedUntil: 1735689600,
			responseBody: `{
				"meta": {},
				"payload": {
					"success": true,
					"conversation_id": 123,
					"current_status": "snoozed",
					"snoozed_until": 1735689600
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, resp *ToggleStatusResponse) {
				if resp.Payload.CurrentStatus != "snoozed" {
					t.Errorf("Expected status snoozed, got %s", resp.Payload.CurrentStatus)
				}
				if resp.Payload.SnoozedUntil == nil {
					t.Error("Expected SnoozedUntil to be set")
				} else if *resp.Payload.SnoozedUntil != 1735689600 {
					t.Errorf("Expected SnoozedUntil 1735689600, got %d", *resp.Payload.SnoozedUntil)
				}
			},
			validatePayload: func(t *testing.T, payload map[string]any) {
				if payload["status"] != "snoozed" {
					t.Errorf("Expected status snoozed in payload, got %v", payload["status"])
				}
				snoozedUntil, exists := payload["snoozed_until"]
				if !exists {
					t.Error("Expected snoozed_until in payload")
				} else {
					// JSON numbers decode to float64
					if val, ok := snoozedUntil.(float64); !ok || int64(val) != 1735689600 {
						t.Errorf("Expected snoozed_until 1735689600, got %v", snoozedUntil)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request payload
				var payload map[string]any
				_ = json.NewDecoder(r.Body).Decode(&payload)

				if tt.validatePayload != nil {
					tt.validatePayload(t, payload)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.ToggleConversationStatus(context.Background(), tt.convID, tt.status, tt.snoozedUntil)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestFlexIntUnmarshal(t *testing.T) {
	tests := []struct {
		name        string
		jsonInput   string
		expected    int
		expectError bool
	}{
		{
			name:        "integer value",
			jsonInput:   `{"value": 42}`,
			expected:    42,
			expectError: false,
		},
		{
			name:        "string value",
			jsonInput:   `{"value": "42"}`,
			expected:    42,
			expectError: false,
		},
		{
			name:        "empty string",
			jsonInput:   `{"value": ""}`,
			expected:    0,
			expectError: false,
		},
		{
			name:        "zero value",
			jsonInput:   `{"value": 0}`,
			expected:    0,
			expectError: false,
		},
		{
			name:        "invalid string",
			jsonInput:   `{"value": "not-a-number"}`,
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Value FlexInt `json:"value"`
			}

			err := json.Unmarshal([]byte(tt.jsonInput), &result)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && int(result.Value) != tt.expected {
				t.Errorf("Expected value %d, got %d", tt.expected, int(result.Value))
			}
		})
	}
}

func TestFlexStringUnmarshal(t *testing.T) {
	tests := []struct {
		name        string
		jsonInput   string
		expected    string
		expectError bool
	}{
		{
			name:        "string value",
			jsonInput:   `{"value": "hello"}`,
			expected:    "hello",
			expectError: false,
		},
		{
			name:        "integer value",
			jsonInput:   `{"value": 42}`,
			expected:    "42",
			expectError: false,
		},
		{
			name:        "float value as whole number",
			jsonInput:   `{"value": 100.0}`,
			expected:    "100",
			expectError: false,
		},
		{
			name:        "float value with decimals",
			jsonInput:   `{"value": 3.14159}`,
			expected:    "3.14159",
			expectError: false,
		},
		{
			name:        "empty string",
			jsonInput:   `{"value": ""}`,
			expected:    "",
			expectError: false,
		},
		{
			name:        "zero number",
			jsonInput:   `{"value": 0}`,
			expected:    "0",
			expectError: false,
		},
		{
			name:        "negative number",
			jsonInput:   `{"value": -42}`,
			expected:    "-42",
			expectError: false,
		},
		{
			name:        "boolean value",
			jsonInput:   `{"value": true}`,
			expected:    "",
			expectError: true,
		},
		{
			name:        "null value becomes empty string",
			jsonInput:   `{"value": null}`,
			expected:    "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Value FlexString `json:"value"`
			}

			err := json.Unmarshal([]byte(tt.jsonInput), &result)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && string(result.Value) != tt.expected {
				t.Errorf("Expected value %q, got %q", tt.expected, string(result.Value))
			}
		})
	}
}

func TestFlexStringString(t *testing.T) {
	fs := FlexString("test value")
	if fs.String() != "test value" {
		t.Errorf("Expected 'test value', got %q", fs.String())
	}
}

func TestConversationCreatedAtTime(t *testing.T) {
	conv := &Conversation{CreatedAt: 1700000000}
	result := conv.CreatedAtTime()

	if result.Unix() != 1700000000 {
		t.Errorf("Expected Unix timestamp 1700000000, got %d", result.Unix())
	}
}

func TestMessageCreatedAtTime(t *testing.T) {
	msg := &Message{CreatedAt: 1700000000}
	result := msg.CreatedAtTime()

	if result.Unix() != 1700000000 {
		t.Errorf("Expected Unix timestamp 1700000000, got %d", result.Unix())
	}
}

func TestMessageTypeName(t *testing.T) {
	tests := []struct {
		messageType int
		expected    string
	}{
		{0, "incoming"},
		{1, "outgoing"},
		{2, "activity"},
		{3, "template"},
		{99, "unknown"},
	}

	for _, tt := range tests {
		msg := &Message{MessageType: tt.messageType}
		if msg.MessageTypeName() != tt.expected {
			t.Errorf("MessageType %d: expected %q, got %q", tt.messageType, tt.expected, msg.MessageTypeName())
		}
	}
}

func TestContactCreatedAtTime(t *testing.T) {
	contact := &Contact{CreatedAt: 1700000000}
	result := contact.CreatedAtTime()

	if result.Unix() != 1700000000 {
		t.Errorf("Expected Unix timestamp 1700000000, got %d", result.Unix())
	}
}

func TestCampaignScheduledAtTime(t *testing.T) {
	// Zero value
	c1 := &Campaign{ScheduledAt: 0}
	if !c1.ScheduledAtTime().IsZero() {
		t.Error("Expected zero time for ScheduledAt=0")
	}

	// Non-zero value
	c2 := &Campaign{ScheduledAt: 1700000000}
	if c2.ScheduledAtTime().Unix() != 1700000000 {
		t.Errorf("Expected Unix timestamp 1700000000, got %d", c2.ScheduledAtTime().Unix())
	}
}

func TestToggleMuteConversation(t *testing.T) {
	tests := []struct {
		name            string
		conversationID  int
		mute            bool
		statusCode      int
		responseBody    string
		expectError     bool
		validatePayload func(*testing.T, map[string]any)
	}{
		{
			name:           "successful mute",
			conversationID: 123,
			mute:           true,
			statusCode:     http.StatusOK,
			responseBody:   "",
			expectError:    false,
			validatePayload: func(t *testing.T, payload map[string]any) {
				status, ok := payload["status"]
				if !ok {
					t.Error("Expected status in payload")
					return
				}
				if status != true {
					t.Errorf("Expected status true, got %v", status)
				}
			},
		},
		{
			name:           "successful unmute",
			conversationID: 123,
			mute:           false,
			statusCode:     http.StatusOK,
			responseBody:   "",
			expectError:    false,
			validatePayload: func(t *testing.T, payload map[string]any) {
				status, ok := payload["status"]
				if !ok {
					t.Error("Expected status in payload")
					return
				}
				if status != false {
					t.Errorf("Expected status false, got %v", status)
				}
			},
		},
		{
			name:           "conversation not found",
			conversationID: 999,
			mute:           true,
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error":"Conversation not found"}`,
			expectError:    true,
		},
		{
			name:           "unauthorized",
			conversationID: 123,
			mute:           true,
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error":"Unauthorized"}`,
			expectError:    true,
		},
		{
			name:           "server error",
			conversationID: 123,
			mute:           false,
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error":"Internal server error"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				// Verify request path
				expectedPath := fmt.Sprintf("/api/v1/accounts/1/conversations/%d/toggle_mute", tt.conversationID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				// Verify headers
				if r.Header.Get("api_access_token") == "" {
					t.Error("Missing api_access_token header")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Verify request payload
				if tt.validatePayload != nil {
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Errorf("Failed to decode request body: %v", err)
					} else {
						tt.validatePayload(t, payload)
					}
				}

				// Send response
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create client
			client := newTestClient(server.URL, "test-token", 1)

			// Execute
			err := client.ToggleMuteConversation(context.Background(), tt.conversationID, tt.mute)

			// Verify
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// For error cases, verify error contains status code
			if tt.expectError && err != nil {
				apiErr, ok := err.(*APIError)
				if !ok {
					t.Errorf("Expected APIError, got %T", err)
				} else if apiErr.StatusCode != tt.statusCode {
					t.Errorf("Expected status code %d, got %d", tt.statusCode, apiErr.StatusCode)
				}
			}
		})
	}
}

func TestUpdateConversation(t *testing.T) {
	tests := []struct {
		name            string
		conversationID  int
		priority        string
		slaPolicyID     int
		statusCode      int
		responseBody    string
		expectError     bool
		validateFunc    func(*testing.T, *Conversation)
		validatePayload func(*testing.T, map[string]any)
	}{
		{
			name:           "update priority only",
			conversationID: 123,
			priority:       "high",
			slaPolicyID:    0,
			statusCode:     http.StatusOK,
			responseBody: `{
				"id": 123,
				"account_id": 1,
				"inbox_id": 5,
				"status": "open",
				"priority": "high",
				"display_id": 456,
				"contact_id": 789,
				"unread_count": 3,
				"muted": false,
				"created_at": 1700000000
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, conv *Conversation) {
				if conv.ID != 123 {
					t.Errorf("Expected ID 123, got %d", conv.ID)
				}
				if conv.Priority == nil || *conv.Priority != "high" {
					t.Errorf("Expected priority high, got %v", conv.Priority)
				}
			},
			validatePayload: func(t *testing.T, payload map[string]any) {
				if payload["priority"] != "high" {
					t.Errorf("Expected priority high in payload, got %v", payload["priority"])
				}
				if _, exists := payload["sla_policy_id"]; exists {
					t.Error("Expected sla_policy_id to not be in payload when slaPolicyID=0")
				}
			},
		},
		{
			name:           "update SLA policy only",
			conversationID: 123,
			priority:       "",
			slaPolicyID:    5,
			statusCode:     http.StatusOK,
			responseBody: `{
				"id": 123,
				"account_id": 1,
				"inbox_id": 5,
				"status": "open",
				"display_id": 456,
				"contact_id": 789,
				"unread_count": 3,
				"muted": false,
				"created_at": 1700000000
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, conv *Conversation) {
				if conv.ID != 123 {
					t.Errorf("Expected ID 123, got %d", conv.ID)
				}
			},
			validatePayload: func(t *testing.T, payload map[string]any) {
				if _, exists := payload["priority"]; exists {
					t.Error("Expected priority to not be in payload when priority is empty")
				}
				slaPolicyID, exists := payload["sla_policy_id"]
				if !exists {
					t.Error("Expected sla_policy_id in payload")
				} else {
					// JSON numbers decode to float64
					if val, ok := slaPolicyID.(float64); !ok || int(val) != 5 {
						t.Errorf("Expected sla_policy_id 5, got %v", slaPolicyID)
					}
				}
			},
		},
		{
			name:           "update both priority and SLA policy",
			conversationID: 123,
			priority:       "urgent",
			slaPolicyID:    5,
			statusCode:     http.StatusOK,
			responseBody: `{
				"id": 123,
				"account_id": 1,
				"inbox_id": 5,
				"status": "open",
				"priority": "urgent",
				"display_id": 456,
				"contact_id": 789,
				"unread_count": 3,
				"muted": false,
				"created_at": 1700000000
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, conv *Conversation) {
				if conv.ID != 123 {
					t.Errorf("Expected ID 123, got %d", conv.ID)
				}
				if conv.Priority == nil || *conv.Priority != "urgent" {
					t.Errorf("Expected priority urgent, got %v", conv.Priority)
				}
			},
			validatePayload: func(t *testing.T, payload map[string]any) {
				if payload["priority"] != "urgent" {
					t.Errorf("Expected priority urgent in payload, got %v", payload["priority"])
				}
				slaPolicyID, exists := payload["sla_policy_id"]
				if !exists {
					t.Error("Expected sla_policy_id in payload")
				} else {
					// JSON numbers decode to float64
					if val, ok := slaPolicyID.(float64); !ok || int(val) != 5 {
						t.Errorf("Expected sla_policy_id 5, got %v", slaPolicyID)
					}
				}
			},
		},
		{
			name:           "conversation not found",
			conversationID: 999,
			priority:       "high",
			slaPolicyID:    0,
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error":"Conversation not found"}`,
			expectError:    true,
		},
		{
			name:           "unauthorized",
			conversationID: 123,
			priority:       "high",
			slaPolicyID:    0,
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error":"Unauthorized"}`,
			expectError:    true,
		},
		{
			name:           "server error",
			conversationID: 123,
			priority:       "high",
			slaPolicyID:    0,
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error":"Internal server error"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodPatch {
					t.Errorf("Expected PATCH request, got %s", r.Method)
				}

				// Verify request path
				expectedPath := fmt.Sprintf("/api/v1/accounts/1/conversations/%d", tt.conversationID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				// Verify headers
				if r.Header.Get("api_access_token") == "" {
					t.Error("Missing api_access_token header")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Verify request payload
				if tt.validatePayload != nil {
					var payload map[string]any
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Errorf("Failed to decode request body: %v", err)
					} else {
						tt.validatePayload(t, payload)
					}
				}

				// Send response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create client
			client := newTestClient(server.URL, "test-token", 1)

			// Execute
			result, err := client.UpdateConversation(context.Background(), tt.conversationID, tt.priority, tt.slaPolicyID)

			// Verify
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// For error cases, verify error contains status code
			if tt.expectError && err != nil {
				apiErr, ok := err.(*APIError)
				if !ok {
					t.Errorf("Expected APIError, got %T", err)
				} else if apiErr.StatusCode != tt.statusCode {
					t.Errorf("Expected status code %d, got %d", tt.statusCode, apiErr.StatusCode)
				}
			}

			// Run custom validation
			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}
