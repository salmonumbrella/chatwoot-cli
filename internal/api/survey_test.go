package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetSurveyResponse(t *testing.T) {
	tests := []struct {
		name             string
		conversationUUID string
		statusCode       int
		responseBody     string
		expectError      bool
		validateFunc     func(*testing.T, *SurveyResponse)
	}{
		{
			name:             "successful get",
			conversationUUID: "uuid-123-456",
			statusCode:       http.StatusOK,
			responseBody: `{
				"conversation_id": 1,
				"rating": 5,
				"message": "Great service!",
				"feedback_message": "Very helpful",
				"contact_id": 10,
				"assigned_agent_id": 20
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, resp *SurveyResponse) {
				if resp.ConversationID != 1 {
					t.Errorf("Expected conversation_id 1, got %d", resp.ConversationID)
				}
				if resp.Rating != 5 {
					t.Errorf("Expected rating 5, got %d", resp.Rating)
				}
				if resp.Message != "Great service!" {
					t.Errorf("Expected message 'Great service!', got %s", resp.Message)
				}
				if resp.FeedbackMessage != "Very helpful" {
					t.Errorf("Expected feedback_message 'Very helpful', got %s", resp.FeedbackMessage)
				}
				if resp.ContactID != 10 {
					t.Errorf("Expected contact_id 10, got %d", resp.ContactID)
				}
				if resp.AssignedAgentID != 20 {
					t.Errorf("Expected assigned_agent_id 20, got %d", resp.AssignedAgentID)
				}
			},
		},
		{
			name:             "minimal response",
			conversationUUID: "uuid-789",
			statusCode:       http.StatusOK,
			responseBody:     `{"conversation_id": 2, "rating": 3}`,
			expectError:      false,
			validateFunc: func(t *testing.T, resp *SurveyResponse) {
				if resp.ConversationID != 2 {
					t.Errorf("Expected conversation_id 2, got %d", resp.ConversationID)
				}
				if resp.Rating != 3 {
					t.Errorf("Expected rating 3, got %d", resp.Rating)
				}
			},
		},
		{
			name:             "not found",
			conversationUUID: "nonexistent-uuid",
			statusCode:       http.StatusNotFound,
			responseBody:     `{"error": "not found"}`,
			expectError:      true,
		},
		{
			name:             "server error",
			conversationUUID: "uuid-error",
			statusCode:       http.StatusInternalServerError,
			responseBody:     `{"error": "internal error"}`,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				// Verify the path includes /survey/responses/
				if !strings.Contains(r.URL.Path, "/survey/responses/") {
					t.Errorf("Expected survey responses path, got %s", r.URL.Path)
				}
				// Verify the UUID is in the path
				if !strings.Contains(r.URL.Path, tt.conversationUUID) {
					t.Errorf("Expected UUID in path, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Survey().GetResponse(context.Background(), tt.conversationUUID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestSurveyPath(t *testing.T) {
	client := newTestClient("https://example.com", "token", 1)

	tests := []struct {
		path     string
		expected string
	}{
		{"/responses/uuid-123", "https://example.com/survey/responses/uuid-123"},
		{"", "https://example.com/survey"},
	}

	for _, tt := range tests {
		result := client.surveyPath(tt.path)
		if result != tt.expected {
			t.Errorf("surveyPath(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}
