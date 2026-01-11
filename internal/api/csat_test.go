package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListCSATResponses(t *testing.T) {
	tests := []struct {
		name         string
		params       CSATListParams
		statusCode   int
		responseBody string
		expectError  bool
		expectCount  int
	}{
		{
			name:       "bare array format (actual Chatwoot API)",
			params:     CSATListParams{},
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "conversation_id": 100, "rating": 5, "feedback_message": "Great!", "created_at": 1700000000},
				{"id": 2, "conversation_id": 101, "rating": 3, "feedback_message": "OK", "created_at": 1700001000}
			]`,
			expectError: false,
			expectCount: 2,
		},
		{
			name:       "wrapped format fallback",
			params:     CSATListParams{},
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [
					{"id": 1, "conversation_id": 100, "rating": 5, "feedback_message": "Great!", "created_at": 1700000000},
					{"id": 2, "conversation_id": 101, "rating": 3, "feedback_message": "OK", "created_at": 1700001000}
				],
				"meta": {"total_count": 2}
			}`,
			expectError: false,
			expectCount: 2,
		},
		{
			name:       "filter by rating",
			params:     CSATListParams{Rating: "1,2"},
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 3, "conversation_id": 102, "rating": 1, "feedback_message": "Bad", "created_at": 1700002000}
			]`,
			expectError: false,
			expectCount: 1,
		},
		{
			name:         "empty bare array",
			params:       CSATListParams{InboxID: 999},
			statusCode:   http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			expectCount:  0,
		},
		{
			name:         "empty wrapped format",
			params:       CSATListParams{InboxID: 999},
			statusCode:   http.StatusOK,
			responseBody: `{"payload": [], "meta": {"total_count": 0}}`,
			expectError:  false,
			expectCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			responses, err := client.CSAT().List(context.Background(), tt.params)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && len(responses) != tt.expectCount {
				t.Errorf("Expected %d responses, got %d", tt.expectCount, len(responses))
			}
		})
	}
}

func TestGetConversationCSAT(t *testing.T) {
	tests := []struct {
		name           string
		conversationID int
		statusCode     int
		responseBody   string
		expectError    bool
		expectRating   int
	}{
		{
			name:           "bare array format (actual Chatwoot API)",
			conversationID: 123,
			statusCode:     http.StatusOK,
			responseBody:   `[{"id": 1, "conversation_id": 123, "rating": 5, "feedback_message": "Excellent!", "created_at": 1700000000}]`,
			expectError:    false,
			expectRating:   5,
		},
		{
			name:           "wrapped format fallback",
			conversationID: 123,
			statusCode:     http.StatusOK,
			responseBody:   `{"payload": [{"id": 1, "conversation_id": 123, "rating": 5, "feedback_message": "Excellent!", "created_at": 1700000000}], "meta": {}}`,
			expectError:    false,
			expectRating:   5,
		},
		{
			name:           "empty bare array",
			conversationID: 456,
			statusCode:     http.StatusOK,
			responseBody:   `[]`,
			expectError:    false,
			expectRating:   0,
		},
		{
			name:           "empty wrapped format",
			conversationID: 456,
			statusCode:     http.StatusOK,
			responseBody:   `{"payload": [], "meta": {}}`,
			expectError:    false,
			expectRating:   0,
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
			result, err := client.CSAT().Conversation(context.Background(), tt.conversationID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && tt.expectRating > 0 {
				if result == nil {
					t.Error("Expected CSAT response, got nil")
				} else if result.Rating != tt.expectRating {
					t.Errorf("Expected rating %d, got %d", tt.expectRating, result.Rating)
				}
			}
			if !tt.expectError && tt.expectRating == 0 && result != nil {
				t.Error("Expected nil for conversation without CSAT")
			}
		})
	}
}
