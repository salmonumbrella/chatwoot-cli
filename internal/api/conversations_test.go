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
			err := client.Conversations().MarkUnread(context.Background(), tt.conversationID)

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
			result, err := client.Conversations().List(context.Background(), ListConversationsParams{
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
			result, err := client.Conversations().Get(context.Background(), tt.conversationID)

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
			result, err := client.Conversations().ToggleStatus(context.Background(), tt.convID, tt.status, tt.snoozedUntil)

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
			err := client.Conversations().ToggleMute(context.Background(), tt.conversationID, tt.mute)

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

func TestCreateConversation(t *testing.T) {
	tests := []struct {
		name            string
		request         CreateConversationRequest
		statusCode      int
		responseBody    string
		expectError     bool
		validateFunc    func(*testing.T, *Conversation)
		validatePayload func(*testing.T, map[string]any)
	}{
		{
			name: "successful create with message",
			request: CreateConversationRequest{
				InboxID:   1,
				ContactID: 123,
				Message:   "Hello, starting a new conversation",
				Status:    "open",
			},
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 456,
				"account_id": 1,
				"inbox_id": 1,
				"contact_id": 123,
				"status": "open",
				"created_at": 1700000000
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, conv *Conversation) {
				if conv.ID != 456 {
					t.Errorf("Expected ID 456, got %d", conv.ID)
				}
				if conv.ContactID != 123 {
					t.Errorf("Expected contact ID 123, got %d", conv.ContactID)
				}
			},
			validatePayload: func(t *testing.T, payload map[string]any) {
				if payload["inbox_id"] != float64(1) {
					t.Errorf("Expected inbox_id 1, got %v", payload["inbox_id"])
				}
				if payload["contact_id"] != float64(123) {
					t.Errorf("Expected contact_id 123, got %v", payload["contact_id"])
				}
				if payload["message"] != "Hello, starting a new conversation" {
					t.Errorf("Expected message, got %v", payload["message"])
				}
			},
		},
		{
			name: "create with assignee and team",
			request: CreateConversationRequest{
				InboxID:   1,
				ContactID: 123,
				Assignee:  intPtr(5),
				TeamID:    intPtr(2),
			},
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 457,
				"account_id": 1,
				"inbox_id": 1,
				"contact_id": 123,
				"status": "open",
				"assignee_id": 5,
				"team_id": 2,
				"created_at": 1700000000
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, conv *Conversation) {
				if conv.AssigneeID == nil || *conv.AssigneeID != 5 {
					t.Errorf("Expected assignee ID 5, got %v", conv.AssigneeID)
				}
				if conv.TeamID == nil || *conv.TeamID != 2 {
					t.Errorf("Expected team ID 2, got %v", conv.TeamID)
				}
			},
		},
		{
			name: "create with custom attributes",
			request: CreateConversationRequest{
				InboxID:          1,
				ContactID:        123,
				CustomAttributes: map[string]any{"priority": "high", "source": "api"},
			},
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 458,
				"account_id": 1,
				"inbox_id": 1,
				"contact_id": 123,
				"status": "open",
				"custom_attributes": {"priority": "high", "source": "api"},
				"created_at": 1700000000
			}`,
			expectError: false,
		},
		{
			name: "error - inbox not found",
			request: CreateConversationRequest{
				InboxID:   999,
				ContactID: 123,
			},
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "Inbox not found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}

				if tt.validatePayload != nil {
					var payload map[string]any
					_ = json.NewDecoder(r.Body).Decode(&payload)
					tt.validatePayload(t, payload)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Conversations().Create(context.Background(), tt.request)

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

func TestFilterConversations(t *testing.T) {
	tests := []struct {
		name         string
		payload      map[string]any
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *ConversationList)
	}{
		{
			name: "successful filter",
			payload: map[string]any{
				"payload": []map[string]any{
					{"attribute_key": "status", "filter_operator": "equal_to", "values": []string{"open"}},
				},
			},
			statusCode: http.StatusOK,
			responseBody: `{
				"meta": {"current_page": 1, "total_count": 2},
				"payload": [
					{"id": 1, "status": "open", "inbox_id": 5, "created_at": 1700000000},
					{"id": 2, "status": "open", "inbox_id": 5, "created_at": 1700001000}
				]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *ConversationList) {
				if len(result.Data.Payload) != 2 {
					t.Errorf("Expected 2 conversations, got %d", len(result.Data.Payload))
				}
				if result.Data.Payload[0].Status != "open" {
					t.Errorf("Expected status 'open', got %s", result.Data.Payload[0].Status)
				}
			},
		},
		{
			name:         "empty filter results",
			payload:      map[string]any{},
			statusCode:   http.StatusOK,
			responseBody: `{"meta": {"current_page": 1, "total_count": 0}, "payload": []}`,
			expectError:  false,
			validateFunc: func(t *testing.T, result *ConversationList) {
				if len(result.Data.Payload) != 0 {
					t.Errorf("Expected 0 conversations, got %d", len(result.Data.Payload))
				}
			},
		},
		{
			name:         "null payload returns empty slice",
			payload:      map[string]any{},
			statusCode:   http.StatusOK,
			responseBody: `{"meta": {"current_page": 1, "total_count": 0}, "payload": null}`,
			expectError:  false,
			validateFunc: func(t *testing.T, result *ConversationList) {
				if result.Data.Payload == nil {
					t.Error("Expected non-nil payload slice, got nil")
				}
				if len(result.Data.Payload) != 0 {
					t.Errorf("Expected 0 conversations, got %d", len(result.Data.Payload))
				}
			},
		},
		{
			name:         "server error",
			payload:      map[string]any{},
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "Internal server error"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/filter") {
					t.Errorf("Expected path to contain /filter, got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Conversations().Filter(context.Background(), tt.payload, 0)

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

func TestGetConversationsMeta(t *testing.T) {
	tests := []struct {
		name         string
		params       ListConversationsParams
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, map[string]any)
		validatePath func(*testing.T, string)
	}{
		{
			name: "successful get meta",
			params: ListConversationsParams{
				Status:  "open",
				InboxID: "5",
			},
			statusCode: http.StatusOK,
			responseBody: `{
				"open": 10,
				"pending": 5,
				"resolved": 20,
				"all": 35
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result map[string]any) {
				if result["open"] != float64(10) {
					t.Errorf("Expected open count 10, got %v", result["open"])
				}
				if result["all"] != float64(35) {
					t.Errorf("Expected all count 35, got %v", result["all"])
				}
			},
			validatePath: func(t *testing.T, path string) {
				if !strings.Contains(path, "status=open") {
					t.Errorf("Expected path to contain status=open, got %s", path)
				}
				if !strings.Contains(path, "inbox_id=5") {
					t.Errorf("Expected path to contain inbox_id=5, got %s", path)
				}
			},
		},
		{
			name: "meta with labels filter",
			params: ListConversationsParams{
				Labels: []string{"bug", "urgent"},
			},
			statusCode: http.StatusOK,
			responseBody: `{
				"open": 2,
				"pending": 1,
				"resolved": 3,
				"all": 6
			}`,
			expectError: false,
			validatePath: func(t *testing.T, path string) {
				if !strings.Contains(path, "labels=bug") {
					t.Errorf("Expected path to contain labels filter, got %s", path)
				}
			},
		},
		{
			name: "meta with query",
			params: ListConversationsParams{
				Query: "test search",
			},
			statusCode: http.StatusOK,
			responseBody: `{
				"open": 1,
				"all": 1
			}`,
			expectError: false,
			validatePath: func(t *testing.T, path string) {
				if !strings.Contains(path, "q=test") {
					t.Errorf("Expected path to contain query, got %s", path)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/meta") {
					t.Errorf("Expected path to contain /meta, got %s", r.URL.Path)
				}

				if tt.validatePath != nil {
					tt.validatePath(t, r.URL.String())
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Conversations().Meta(context.Background(), tt.params)

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

func TestToggleConversationPriority(t *testing.T) {
	tests := []struct {
		name            string
		conversationID  int
		priority        string
		statusCode      int
		expectError     bool
		validatePayload func(*testing.T, map[string]any)
	}{
		{
			name:           "set priority to high",
			conversationID: 123,
			priority:       "high",
			statusCode:     http.StatusOK,
			expectError:    false,
			validatePayload: func(t *testing.T, payload map[string]any) {
				if payload["priority"] != "high" {
					t.Errorf("Expected priority 'high', got %v", payload["priority"])
				}
			},
		},
		{
			name:           "set priority to urgent",
			conversationID: 123,
			priority:       "urgent",
			statusCode:     http.StatusOK,
			expectError:    false,
		},
		{
			name:           "set priority to nil (remove)",
			conversationID: 123,
			priority:       "nil",
			statusCode:     http.StatusOK,
			expectError:    false,
		},
		{
			name:           "conversation not found",
			conversationID: 999,
			priority:       "high",
			statusCode:     http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/toggle_priority") {
					t.Errorf("Expected path to contain /toggle_priority, got %s", r.URL.Path)
				}

				if tt.validatePayload != nil {
					var payload map[string]any
					_ = json.NewDecoder(r.Body).Decode(&payload)
					tt.validatePayload(t, payload)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode >= 400 {
					_, _ = w.Write([]byte(`{"error": "error"}`))
				}
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.Conversations().TogglePriority(context.Background(), tt.conversationID, tt.priority)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestAssignConversation(t *testing.T) {
	tests := []struct {
		name            string
		conversationID  int
		agentID         int
		teamID          int
		statusCode      int
		responseBody    string
		expectError     bool
		validatePayload func(*testing.T, map[string]any)
	}{
		{
			name:           "assign to agent only",
			conversationID: 123,
			agentID:        5,
			teamID:         0,
			statusCode:     http.StatusOK,
			responseBody:   `{"id": 5, "name": "Agent Name"}`,
			expectError:    false,
			validatePayload: func(t *testing.T, payload map[string]any) {
				if payload["assignee_id"] != float64(5) {
					t.Errorf("Expected assignee_id 5, got %v", payload["assignee_id"])
				}
				if _, exists := payload["team_id"]; exists {
					t.Error("Expected team_id to not be in payload")
				}
			},
		},
		{
			name:           "assign to team only",
			conversationID: 123,
			agentID:        0,
			teamID:         2,
			statusCode:     http.StatusOK,
			responseBody:   `{"id": 2, "name": "Support Team"}`,
			expectError:    false,
			validatePayload: func(t *testing.T, payload map[string]any) {
				if _, exists := payload["assignee_id"]; exists {
					t.Error("Expected assignee_id to not be in payload")
				}
				if payload["team_id"] != float64(2) {
					t.Errorf("Expected team_id 2, got %v", payload["team_id"])
				}
			},
		},
		{
			name:           "assign to both agent and team",
			conversationID: 123,
			agentID:        5,
			teamID:         2,
			statusCode:     http.StatusOK,
			responseBody:   `{"id": 5, "name": "Agent Name"}`,
			expectError:    false,
			validatePayload: func(t *testing.T, payload map[string]any) {
				if payload["assignee_id"] != float64(5) {
					t.Errorf("Expected assignee_id 5, got %v", payload["assignee_id"])
				}
				if payload["team_id"] != float64(2) {
					t.Errorf("Expected team_id 2, got %v", payload["team_id"])
				}
			},
		},
		{
			name:           "error - conversation not found",
			conversationID: 999,
			agentID:        5,
			teamID:         0,
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error": "Conversation not found"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/assignments") {
					t.Errorf("Expected path to contain /assignments, got %s", r.URL.Path)
				}

				if tt.validatePayload != nil {
					var payload map[string]any
					_ = json.NewDecoder(r.Body).Decode(&payload)
					tt.validatePayload(t, payload)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Conversations().Assign(context.Background(), tt.conversationID, tt.agentID, tt.teamID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && result == nil {
				t.Error("Expected result but got nil")
			}
		})
	}
}

func TestGetConversationLabels(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		statusCode     int
		responseBody   string
		expectError    bool
		validateFunc   func(*testing.T, []string)
	}{
		{
			name:           "successful get labels",
			conversationID: 123,
			statusCode:     http.StatusOK,
			responseBody:   `{"payload": ["bug", "urgent", "customer-reported"]}`,
			expectError:    false,
			validateFunc: func(t *testing.T, result []string) {
				if len(result) != 3 {
					t.Errorf("Expected 3 labels, got %d", len(result))
				}
				if result[0] != "bug" {
					t.Errorf("Expected first label 'bug', got %s", result[0])
				}
			},
		},
		{
			name:           "no labels",
			conversationID: 123,
			statusCode:     http.StatusOK,
			responseBody:   `{"payload": []}`,
			expectError:    false,
			validateFunc: func(t *testing.T, result []string) {
				if len(result) != 0 {
					t.Errorf("Expected 0 labels, got %d", len(result))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, "/labels") {
					t.Errorf("Expected path to contain /labels, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Conversations().Labels(context.Background(), tt.conversationID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestAddConversationLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var payload map[string][]string
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if len(payload["labels"]) != 2 {
			t.Errorf("Expected 2 labels in payload, got %d", len(payload["labels"]))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"payload": ["existing", "new-label", "another"]}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Conversations().AddLabels(context.Background(), 123, []string{"new-label", "another"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("Expected 3 labels, got %d", len(result))
	}
}

func TestUpdateConversationCustomAttributes(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		attrs          map[string]any
		statusCode     int
		expectError    bool
	}{
		{
			name:           "successful update",
			conversationID: 123,
			attrs:          map[string]any{"priority": "high", "source": "api"},
			statusCode:     http.StatusOK,
			expectError:    false,
		},
		{
			name:           "conversation not found",
			conversationID: 999,
			attrs:          map[string]any{"key": "value"},
			statusCode:     http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/custom_attributes") {
					t.Errorf("Expected path to contain /custom_attributes, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode >= 400 {
					_, _ = w.Write([]byte(`{"error": "error"}`))
				}
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.Conversations().UpdateCustomAttributes(context.Background(), tt.conversationID, tt.attrs)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestSearchConversations(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		page         int
		statusCode   int
		responseBody string
		expectError  bool
		validatePath func(*testing.T, string)
		validateFunc func(*testing.T, *ConversationList)
	}{
		{
			name:       "successful search",
			query:      "billing issue",
			page:       1,
			statusCode: http.StatusOK,
			responseBody: `{
				"data": {
					"meta": {"current_page": 1, "total_count": 2},
					"payload": [
						{"id": 1, "status": "open", "created_at": 1700000000},
						{"id": 2, "status": "resolved", "created_at": 1700001000}
					]
				}
			}`,
			expectError: false,
			validatePath: func(t *testing.T, path string) {
				if !strings.Contains(path, "q=billing") {
					t.Errorf("Expected path to contain query, got %s", path)
				}
			},
			validateFunc: func(t *testing.T, result *ConversationList) {
				if len(result.Data.Payload) != 2 {
					t.Errorf("Expected 2 conversations, got %d", len(result.Data.Payload))
				}
			},
		},
		{
			name:       "search with pagination",
			query:      "test",
			page:       2,
			statusCode: http.StatusOK,
			responseBody: `{
				"data": {
					"meta": {"current_page": 2, "total_count": 25},
					"payload": []
				}
			}`,
			expectError: false,
			validatePath: func(t *testing.T, path string) {
				if !strings.Contains(path, "page=2") {
					t.Errorf("Expected path to contain page=2, got %s", path)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/search") {
					t.Errorf("Expected path to contain /search, got %s", r.URL.Path)
				}

				if tt.validatePath != nil {
					tt.validatePath(t, r.URL.String())
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Conversations().Search(context.Background(), tt.query, tt.page)

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

func TestGetConversationAttachments(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		statusCode     int
		responseBody   string
		expectError    bool
		validateFunc   func(*testing.T, []Attachment)
	}{
		{
			name:           "successful get attachments",
			conversationID: 123,
			statusCode:     http.StatusOK,
			responseBody: `{"meta": {"total_count": 2}, "payload": [
				{"id": 1, "file_type": "image", "data_url": "https://example.com/img1.jpg", "file_size": 12345},
				{"id": 2, "file_type": "document", "data_url": "https://example.com/doc.pdf", "file_size": 54321}
			]}`,
			expectError: false,
			validateFunc: func(t *testing.T, result []Attachment) {
				if len(result) != 2 {
					t.Errorf("Expected 2 attachments, got %d", len(result))
				}
				if result[0].FileType != "image" {
					t.Errorf("Expected file type 'image', got %s", result[0].FileType)
				}
			},
		},
		{
			name:           "no attachments",
			conversationID: 123,
			statusCode:     http.StatusOK,
			responseBody:   `{"meta": {"total_count": 0}, "payload": []}`,
			expectError:    false,
			validateFunc: func(t *testing.T, result []Attachment) {
				if len(result) != 0 {
					t.Errorf("Expected 0 attachments, got %d", len(result))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, "/attachments") {
					t.Errorf("Expected path to contain /attachments, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Conversations().Attachments(context.Background(), tt.conversationID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestLastActivityAtTime(t *testing.T) {
	conv := &Conversation{LastActivityAt: 1700000000}
	result := conv.LastActivityAtTime()

	if result.Unix() != 1700000000 {
		t.Errorf("Expected Unix timestamp 1700000000, got %d", result.Unix())
	}
}

// Helper function
func intPtr(i int) *int {
	return &i
}

func TestMuteConversation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/conversations/123/toggle_mute" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.Conversations().ToggleMute(context.Background(), 123, true)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestUnmuteConversation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/conversations/123/toggle_mute" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.Conversations().ToggleMute(context.Background(), 123, false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSendTranscript(t *testing.T) {
	var capturedBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.Conversations().Transcript(context.Background(), 123, "test@example.com")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if capturedBody["email"] != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %v", capturedBody["email"])
	}
}

func TestToggleTypingStatus(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.Conversations().ToggleTyping(context.Background(), 123, true, false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if capturedBody["typing_status"] != "on" {
		t.Errorf("Expected typing_status 'on', got %v", capturedBody["typing_status"])
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
			result, err := client.Conversations().Update(context.Background(), tt.conversationID, tt.priority, tt.slaPolicyID)

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
