package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListAllMessages(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		// mockResponses maps the "before" parameter to the response
		// key 0 means first page (no before param)
		mockResponses map[int][]Message
		expectError   bool
		expectedCount int
		validateFunc  func(*testing.T, []Message)
	}{
		{
			name:           "empty conversation - returns empty slice, no error",
			conversationID: 123,
			mockResponses: map[int][]Message{
				0: {}, // First page returns empty
			},
			expectError:   false,
			expectedCount: 0,
			validateFunc: func(t *testing.T, messages []Message) {
				if len(messages) != 0 {
					t.Errorf("Expected 0 messages for empty conversation, got %d", len(messages))
				}
			},
		},
		{
			name:           "single page - stops after detecting end",
			conversationID: 456,
			mockResponses: map[int][]Message{
				0: {
					{ID: 5, Content: "Message 5", ConversationID: 456},
					{ID: 4, Content: "Message 4", ConversationID: 456},
					{ID: 3, Content: "Message 3", ConversationID: 456},
					{ID: 2, Content: "Message 2", ConversationID: 456},
					{ID: 1, Content: "Message 1", ConversationID: 456},
				},
				1: {}, // Next page returns empty, signaling end
			},
			expectError:   false,
			expectedCount: 5,
			validateFunc: func(t *testing.T, messages []Message) {
				if len(messages) != 5 {
					t.Errorf("Expected 5 messages, got %d", len(messages))
				}
				// Verify messages are in the order returned by API
				if messages[0].ID != 5 {
					t.Errorf("Expected first message ID 5, got %d", messages[0].ID)
				}
				if messages[4].ID != 1 {
					t.Errorf("Expected last message ID 1, got %d", messages[4].ID)
				}
			},
		},
		{
			name:           "multi-page - aggregates messages correctly",
			conversationID: 789,
			mockResponses: map[int][]Message{
				0: { // First page (no before param)
					{ID: 20, Content: "Message 20", ConversationID: 789},
					{ID: 19, Content: "Message 19", ConversationID: 789},
					{ID: 18, Content: "Message 18", ConversationID: 789},
					{ID: 17, Content: "Message 17", ConversationID: 789},
					{ID: 16, Content: "Message 16", ConversationID: 789},
				},
				16: { // Second page (before=16, the minID from first page)
					{ID: 15, Content: "Message 15", ConversationID: 789},
					{ID: 14, Content: "Message 14", ConversationID: 789},
					{ID: 13, Content: "Message 13", ConversationID: 789},
					{ID: 12, Content: "Message 12", ConversationID: 789},
					{ID: 11, Content: "Message 11", ConversationID: 789},
				},
				11: { // Third page (before=11, the minID from second page)
					{ID: 10, Content: "Message 10", ConversationID: 789},
					{ID: 9, Content: "Message 9", ConversationID: 789},
					{ID: 8, Content: "Message 8", ConversationID: 789},
				},
				8: { // Fourth page (before=8, returns empty to signal end)
					// Empty response signals no more messages
				},
			},
			expectError:   false,
			expectedCount: 13, // 5 + 5 + 3
			validateFunc: func(t *testing.T, messages []Message) {
				if len(messages) != 13 {
					t.Errorf("Expected 13 messages, got %d", len(messages))
				}
				// Verify first message from first page
				if messages[0].ID != 20 {
					t.Errorf("Expected first message ID 20, got %d", messages[0].ID)
				}
				// Verify last message from third page
				if messages[12].ID != 8 {
					t.Errorf("Expected last message ID 8, got %d", messages[12].ID)
				}
				// Verify messages are aggregated in order
				expectedIDs := []int{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8}
				for i, msg := range messages {
					if msg.ID != expectedIDs[i] {
						t.Errorf("Message at index %d: expected ID %d, got %d", i, expectedIDs[i], msg.ID)
					}
				}
			},
		},
		{
			name:           "infinite loop protection - same minID returned twice",
			conversationID: 999,
			mockResponses: map[int][]Message{
				0: { // First page
					{ID: 100, Content: "Message 100", ConversationID: 999},
					{ID: 99, Content: "Message 99", ConversationID: 999},
					{ID: 98, Content: "Message 98", ConversationID: 999},
				},
				98: { // Second page - API bug returns messages with same minID=98
					{ID: 98, Content: "Message 98 again", ConversationID: 999}, // Same minID
					{ID: 99, Content: "Message 99 again", ConversationID: 999}, // Doesn't matter
					{ID: 100, Content: "Message 100 again", ConversationID: 999},
				},
				// Should not reach here because loop should break when minID=98 repeats
			},
			expectError:   false,
			expectedCount: 3, // Only first page, stops when detecting duplicate minID
			validateFunc: func(t *testing.T, messages []Message) {
				// Should only have first page messages
				if len(messages) != 3 {
					t.Errorf("Expected 3 messages (infinite loop protection), got %d", len(messages))
				}
				if messages[0].ID != 100 {
					t.Errorf("Expected first message ID 100, got %d", messages[0].ID)
				}
			},
		},
		{
			name:           "handles non-sequential message IDs",
			conversationID: 111,
			mockResponses: map[int][]Message{
				0: { // First page with non-sequential IDs
					{ID: 500, Content: "Message 500", ConversationID: 111},
					{ID: 250, Content: "Message 250", ConversationID: 111},
					{ID: 100, Content: "Message 100", ConversationID: 111},
				},
				100: { // Second page (minID was 100)
					{ID: 75, Content: "Message 75", ConversationID: 111},
					{ID: 50, Content: "Message 50", ConversationID: 111},
				},
				50: { // Empty to signal end
				},
			},
			expectError:   false,
			expectedCount: 5,
			validateFunc: func(t *testing.T, messages []Message) {
				if len(messages) != 5 {
					t.Errorf("Expected 5 messages, got %d", len(messages))
				}
				// Verify correct minID detection on first page
				if messages[2].ID != 100 {
					t.Errorf("Expected third message (minID of first page) to be 100, got %d", messages[2].ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track API call count for single-page test
			callCount := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++

				// Verify request method
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}

				// Verify path
				expectedPath := fmt.Sprintf("/conversations/%d/messages", tt.conversationID)
				if !strings.Contains(r.URL.Path, expectedPath) {
					t.Errorf("Expected path to contain %s, got %s", expectedPath, r.URL.Path)
				}

				// Verify headers
				if r.Header.Get("api_access_token") == "" {
					t.Error("Missing api_access_token header")
				}

				// Determine which page is being requested
				beforeParam := 0
				if beforeStr := r.URL.Query().Get("before"); beforeStr != "" {
					_, _ = fmt.Sscanf(beforeStr, "%d", &beforeParam)
				}

				// Get the mock response for this page
				mockMessages, exists := tt.mockResponses[beforeParam]
				if !exists {
					t.Errorf("Unexpected API call with before=%d", beforeParam)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				// Return the response
				response := struct {
					Payload []Message `json:"payload"`
				}{
					Payload: mockMessages,
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Errorf("Failed to encode response: %v", err)
				}
			}))
			defer server.Close()

			// Create client
			client := newTestClient(server.URL, "test-token", 1)

			// Execute
			messages, err := client.Messages().ListAll(context.Background(), tt.conversationID)

			// Verify error expectations
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Verify message count
			if len(messages) != tt.expectedCount {
				t.Errorf("Expected %d messages, got %d", tt.expectedCount, len(messages))
			}

			// For single-page test, verify only one API call was made
			if tt.name == "single page - should not make second API call" {
				// Should make exactly 1 call (first page returns data, then stops because next would be empty)
				// Actually, the implementation calls ListMessagesBefore once, gets messages,
				// then calls again with before=minID which returns empty, so 2 calls
				// Let me check the logic... it gets messages, if len > 0, it continues
				// So for single page: call 1 gets 5 messages, sets before=1 (minID)
				// call 2 with before=1 gets empty, breaks
				// So it will make 2 calls even for single page
				// The test description is misleading - let me verify the actual behavior
				if callCount > 2 {
					t.Errorf("Expected at most 2 API calls for single page, got %d", callCount)
				}
			}

			// Run custom validation
			if tt.validateFunc != nil {
				tt.validateFunc(t, messages)
			}
		})
	}
}

func TestListAllMessagesMaxPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"payload":[{"id":2,"conversation_id":1,"content":"one"},{"id":1,"conversation_id":1,"content":"two"}]}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	_, err := client.Messages().ListAllWithMaxPages(context.Background(), 1, 1)
	if err == nil {
		t.Fatal("expected error when max pages exceeded")
	}
}

func TestListMessages(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		statusCode     int
		responseBody   string
		expectError    bool
		validateFunc   func(*testing.T, []Message)
	}{
		{
			name:           "successful list",
			conversationID: 123,
			statusCode:     http.StatusOK,
			responseBody: `{
				"payload": [
					{
						"id": 1,
						"conversation_id": 123,
						"content": "Hello",
						"content_type": "text",
						"message_type": 0,
						"private": false,
						"created_at": 1700000000
					},
					{
						"id": 2,
						"conversation_id": 123,
						"content": "World",
						"content_type": "text",
						"message_type": 1,
						"private": false,
						"created_at": 1700001000
					}
				]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, messages []Message) {
				if len(messages) != 2 {
					t.Errorf("Expected 2 messages, got %d", len(messages))
				}
				if messages[0].ID != 1 {
					t.Errorf("Expected first message ID 1, got %d", messages[0].ID)
				}
				if messages[0].Content != "Hello" {
					t.Errorf("Expected first message content 'Hello', got %s", messages[0].Content)
				}
			},
		},
		{
			name:           "empty conversation",
			conversationID: 456,
			statusCode:     http.StatusOK,
			responseBody:   `{"payload": []}`,
			expectError:    false,
			validateFunc: func(t *testing.T, messages []Message) {
				if len(messages) != 0 {
					t.Errorf("Expected 0 messages, got %d", len(messages))
				}
			},
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
			result, err := client.Messages().List(context.Background(), tt.conversationID)

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

func TestListMessagesBefore(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		before         int
		expectPath     string
		responseBody   string
		expectError    bool
		validateFunc   func(*testing.T, []Message, *http.Request)
	}{
		{
			name:           "without before parameter",
			conversationID: 123,
			before:         0,
			expectPath:     "/conversations/123/messages",
			responseBody:   `{"payload": [{"id": 1, "content": "test", "conversation_id": 123, "created_at": 1700000000}]}`,
			expectError:    false,
			validateFunc: func(t *testing.T, messages []Message, r *http.Request) {
				// Verify no before param in query
				if r.URL.Query().Get("before") != "" {
					t.Errorf("Expected no before parameter, got %s", r.URL.Query().Get("before"))
				}
			},
		},
		{
			name:           "with before parameter",
			conversationID: 123,
			before:         50,
			expectPath:     "/conversations/123/messages?before=50",
			responseBody:   `{"payload": [{"id": 49, "content": "older message", "conversation_id": 123, "created_at": 1699999000}]}`,
			expectError:    false,
			validateFunc: func(t *testing.T, messages []Message, r *http.Request) {
				// Verify before param is present
				if r.URL.Query().Get("before") != "50" {
					t.Errorf("Expected before=50, got %s", r.URL.Query().Get("before"))
				}
				if len(messages) != 1 {
					t.Errorf("Expected 1 message, got %d", len(messages))
				}
				if messages[0].ID != 49 {
					t.Errorf("Expected message ID 49, got %d", messages[0].ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRequest *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequest = r
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Messages().ListBefore(context.Background(), tt.conversationID, tt.before)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validateFunc != nil && result != nil && capturedRequest != nil {
				tt.validateFunc(t, result, capturedRequest)
			}
		})
	}
}

func TestCreateMessageWithAttachments(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		content        string
		private        bool
		messageType    string
		attachments    map[string][]byte
		statusCode     int
		responseBody   string
		expectError    bool
		validateFunc   func(*testing.T, *Message, *http.Request)
	}{
		{
			name:           "message with single attachment",
			conversationID: 123,
			content:        "See attached",
			private:        false,
			messageType:    "outgoing",
			attachments: map[string][]byte{
				"receipt.pdf": []byte("pdf content"),
			},
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"conversation_id": 123,
				"content": "See attached",
				"message_type": 1,
				"private": false,
				"created_at": 1700000000,
				"attachments": [{"id": 10, "file_type": "file", "data_url": "https://example.com/file.pdf"}]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, msg *Message, r *http.Request) {
				if msg.ID != 1 {
					t.Errorf("Expected message ID 1, got %d", msg.ID)
				}
				if len(msg.Attachments) != 1 {
					t.Errorf("Expected 1 attachment, got %d", len(msg.Attachments))
				}
				// Verify multipart request
				contentType := r.Header.Get("Content-Type")
				if !strings.HasPrefix(contentType, "multipart/form-data") {
					t.Errorf("Expected multipart content type, got %s", contentType)
				}
			},
		},
		{
			name:           "message with multiple attachments",
			conversationID: 456,
			content:        "Documents attached",
			private:        true,
			messageType:    "outgoing",
			attachments: map[string][]byte{
				"doc1.pdf": []byte("pdf1"),
				"doc2.pdf": []byte("pdf2"),
			},
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 2,
				"conversation_id": 456,
				"content": "Documents attached",
				"message_type": 1,
				"private": true,
				"created_at": 1700000000,
				"attachments": [
					{"id": 11, "file_type": "file", "data_url": "https://example.com/doc1.pdf"},
					{"id": 12, "file_type": "file", "data_url": "https://example.com/doc2.pdf"}
				]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, msg *Message, r *http.Request) {
				if len(msg.Attachments) != 2 {
					t.Errorf("Expected 2 attachments, got %d", len(msg.Attachments))
				}
				if !msg.Private {
					t.Error("Expected private message")
				}
			},
		},
		{
			name:           "attachment without content",
			conversationID: 789,
			content:        "",
			private:        false,
			messageType:    "outgoing",
			attachments: map[string][]byte{
				"image.png": []byte("image data"),
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 3, "conversation_id": 789, "content": "", "message_type": 1, "private": false, "created_at": 1700000000}`,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRequest *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(body))
				capturedRequest = r.Clone(context.Background())
				capturedRequest.Body = io.NopCloser(bytes.NewBuffer(body))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			msg, err := client.Messages().CreateWithAttachments(
				context.Background(),
				tt.conversationID,
				tt.content,
				tt.private,
				tt.messageType,
				tt.attachments,
			)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validateFunc != nil && msg != nil && capturedRequest != nil {
				tt.validateFunc(t, msg, capturedRequest)
			}
		})
	}
}

func TestCreateMessage(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		content        string
		private        bool
		messageType    string
		statusCode     int
		responseBody   string
		expectError    bool
		validateFunc   func(*testing.T, *Message, map[string]any)
	}{
		{
			name:           "create outgoing message",
			conversationID: 123,
			content:        "Hello there!",
			private:        false,
			messageType:    "outgoing",
			statusCode:     http.StatusOK,
			responseBody:   `{"id": 1, "conversation_id": 123, "content": "Hello there!", "message_type": 1, "private": false, "created_at": 1700000000}`,
			expectError:    false,
			validateFunc: func(t *testing.T, msg *Message, body map[string]any) {
				if msg.ID != 1 {
					t.Errorf("Expected message ID 1, got %d", msg.ID)
				}
				if msg.Content != "Hello there!" {
					t.Errorf("Expected content 'Hello there!', got %s", msg.Content)
				}
				if body["content"] != "Hello there!" {
					t.Errorf("Expected content in body, got %v", body["content"])
				}
				if body["message_type"] != "outgoing" {
					t.Errorf("Expected message_type 'outgoing', got %v", body["message_type"])
				}
				if body["private"] != false {
					t.Errorf("Expected private false, got %v", body["private"])
				}
			},
		},
		{
			name:           "create private note",
			conversationID: 456,
			content:        "Internal note",
			private:        true,
			messageType:    "outgoing",
			statusCode:     http.StatusOK,
			responseBody:   `{"id": 2, "conversation_id": 456, "content": "Internal note", "message_type": 1, "private": true, "created_at": 1700000000}`,
			expectError:    false,
			validateFunc: func(t *testing.T, msg *Message, body map[string]any) {
				if !msg.Private {
					t.Error("Expected private message")
				}
				if body["private"] != true {
					t.Errorf("Expected private true in body, got %v", body["private"])
				}
			},
		},
		{
			name:           "server error",
			conversationID: 789,
			content:        "Test",
			private:        false,
			messageType:    "outgoing",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error": "internal error"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				_ = json.NewDecoder(r.Body).Decode(&capturedBody)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			msg, err := client.Messages().Create(context.Background(), tt.conversationID, tt.content, tt.private, tt.messageType)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.validateFunc != nil && msg != nil {
				tt.validateFunc(t, msg, capturedBody)
			}
		})
	}
}

func TestDeleteMessage(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		messageID      int
		statusCode     int
		expectError    bool
	}{
		{
			name:           "successful delete",
			conversationID: 123,
			messageID:      456,
			statusCode:     http.StatusOK,
			expectError:    false,
		},
		{
			name:           "not found",
			conversationID: 123,
			messageID:      999,
			statusCode:     http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "server error",
			conversationID: 123,
			messageID:      456,
			statusCode:     http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/v1/accounts/1/conversations/%d/messages/%d", tt.conversationID, tt.messageID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.Messages().Delete(context.Background(), tt.conversationID, tt.messageID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestUpdateMessage(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		messageID      int
		content        string
		statusCode     int
		responseBody   string
		expectError    bool
		validateFunc   func(*testing.T, *Message, map[string]any)
	}{
		{
			name:           "successful update",
			conversationID: 123,
			messageID:      456,
			content:        "Updated content",
			statusCode:     http.StatusOK,
			responseBody:   `{"id": 456, "conversation_id": 123, "content": "Updated content", "message_type": 1, "created_at": 1700000000}`,
			expectError:    false,
			validateFunc: func(t *testing.T, msg *Message, body map[string]any) {
				if msg.Content != "Updated content" {
					t.Errorf("Expected content 'Updated content', got %s", msg.Content)
				}
				if body["content"] != "Updated content" {
					t.Errorf("Expected content in body, got %v", body["content"])
				}
			},
		},
		{
			name:           "empty content fails",
			conversationID: 123,
			messageID:      456,
			content:        "",
			statusCode:     http.StatusOK,
			responseBody:   `{}`,
			expectError:    true,
		},
		{
			name:           "not found",
			conversationID: 123,
			messageID:      999,
			content:        "Test",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error": "not found"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Errorf("Expected PATCH, got %s", r.Method)
				}
				_ = json.NewDecoder(r.Body).Decode(&capturedBody)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			msg, err := client.Messages().Update(context.Background(), tt.conversationID, tt.messageID, tt.content)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.validateFunc != nil && msg != nil {
				tt.validateFunc(t, msg, capturedBody)
			}
		})
	}
}

func TestTranslateMessage(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		messageID      int
		targetLanguage string
		statusCode     int
		responseBody   string
		expectError    bool
		expectedResult string
	}{
		{
			name:           "successful translation",
			conversationID: 1,
			messageID:      100,
			targetLanguage: "es",
			statusCode:     http.StatusOK,
			responseBody:   `{"content": "Texto traducido"}`,
			expectError:    false,
			expectedResult: "Texto traducido",
		},
		{
			name:           "translation to French",
			conversationID: 2,
			messageID:      200,
			targetLanguage: "fr",
			statusCode:     http.StatusOK,
			responseBody:   `{"content": "Texte traduit"}`,
			expectError:    false,
			expectedResult: "Texte traduit",
		},
		{
			name:           "server error",
			conversationID: 1,
			messageID:      100,
			targetLanguage: "es",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error": "translation service unavailable"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/v1/accounts/1/conversations/%d/messages/%d/translate", tt.conversationID, tt.messageID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				_ = json.NewDecoder(r.Body).Decode(&capturedBody)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			content, err := client.Messages().Translate(context.Background(), tt.conversationID, tt.messageID, tt.targetLanguage)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && content != tt.expectedResult {
				t.Errorf("Expected '%s', got '%s'", tt.expectedResult, content)
			}
			if !tt.expectError && capturedBody["target_language"] != tt.targetLanguage {
				t.Errorf("Expected target_language '%s' in body, got %v", tt.targetLanguage, capturedBody["target_language"])
			}
		})
	}
}

func TestRetryMessage(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		messageID      int
		statusCode     int
		responseBody   string
		expectError    bool
		expectedID     int
	}{
		{
			name:           "successful retry",
			conversationID: 1,
			messageID:      100,
			statusCode:     http.StatusOK,
			responseBody:   `{"id": 100, "content": "Retried message", "message_type": 1, "conversation_id": 1, "created_at": 1700000000}`,
			expectError:    false,
			expectedID:     100,
		},
		{
			name:           "retry different message",
			conversationID: 2,
			messageID:      200,
			statusCode:     http.StatusOK,
			responseBody:   `{"id": 200, "content": "Another retried message", "message_type": 1, "conversation_id": 2, "created_at": 1700001000}`,
			expectError:    false,
			expectedID:     200,
		},
		{
			name:           "not found",
			conversationID: 1,
			messageID:      999,
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error": "message not found"}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/v1/accounts/1/conversations/%d/messages/%d/retry", tt.conversationID, tt.messageID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			msg, err := client.Messages().Retry(context.Background(), tt.conversationID, tt.messageID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && msg.ID != tt.expectedID {
				t.Errorf("Expected ID %d, got %d", tt.expectedID, msg.ID)
			}
		})
	}
}
