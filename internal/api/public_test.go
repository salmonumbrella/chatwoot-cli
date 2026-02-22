package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicCreateContact(t *testing.T) {
	tests := []struct {
		name            string
		inboxIdentifier string
		req             PublicCreateContactRequest
		statusCode      int
		responseBody    string
		expectError     bool
		validateFunc    func(*testing.T, *PublicContact)
	}{
		{
			name:            "successful create",
			inboxIdentifier: "inbox123",
			req: PublicCreateContactRequest{
				Name:  "Test Contact",
				Email: "test@example.com",
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "source_id": "src123", "name": "Test Contact", "email": "test@example.com", "pubsub_token": "token123"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, contact *PublicContact) {
				if contact.ID != 1 {
					t.Errorf("Expected ID 1, got %d", contact.ID)
				}
				if contact.SourceID != "src123" {
					t.Errorf("Expected source_id 'src123', got %s", contact.SourceID)
				}
				if contact.PubsubToken != "token123" {
					t.Errorf("Expected pubsub_token 'token123', got %s", contact.PubsubToken)
				}
			},
		},
		{
			name:            "validation error",
			inboxIdentifier: "inbox123",
			req:             PublicCreateContactRequest{},
			statusCode:      http.StatusBadRequest,
			responseBody:    `{"error": "validation failed"}`,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/public/api/v1/inboxes/"+tt.inboxIdentifier+"/contacts") {
					t.Errorf("Expected public inbox path, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Public().CreateContact(context.Background(), tt.inboxIdentifier, tt.req)

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

func TestPublicGetContact(t *testing.T) {
	tests := []struct {
		name              string
		inboxIdentifier   string
		contactIdentifier string
		statusCode        int
		responseBody      string
		expectError       bool
		validateFunc      func(*testing.T, *PublicContact)
	}{
		{
			name:              "successful get",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			statusCode:        http.StatusOK,
			responseBody:      `{"id": 1, "source_id": "src123", "name": "Test Contact", "email": "test@example.com"}`,
			expectError:       false,
			validateFunc: func(t *testing.T, contact *PublicContact) {
				if contact.ID != 1 {
					t.Errorf("Expected ID 1, got %d", contact.ID)
				}
			},
		},
		{
			name:              "not found",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "notexist",
			statusCode:        http.StatusNotFound,
			responseBody:      `{"error": "not found"}`,
			expectError:       true,
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
			result, err := client.Public().GetContact(context.Background(), tt.inboxIdentifier, tt.contactIdentifier)

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

func TestPublicUpdateContact(t *testing.T) {
	tests := []struct {
		name              string
		inboxIdentifier   string
		contactIdentifier string
		req               PublicUpdateContactRequest
		statusCode        int
		responseBody      string
		expectError       bool
		validateFunc      func(*testing.T, *PublicContact)
	}{
		{
			name:              "successful update",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			req: PublicUpdateContactRequest{
				Name: "Updated Name",
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "source_id": "src123", "name": "Updated Name", "email": "test@example.com"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, contact *PublicContact) {
				if contact.Name != "Updated Name" {
					t.Errorf("Expected name 'Updated Name', got %s", contact.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Errorf("Expected PATCH, got %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Public().UpdateContact(context.Background(), tt.inboxIdentifier, tt.contactIdentifier, tt.req)

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

func TestPublicCreateConversation(t *testing.T) {
	tests := []struct {
		name              string
		inboxIdentifier   string
		contactIdentifier string
		customAttributes  map[string]any
		statusCode        int
		responseBody      string
		expectError       bool
		validateFunc      func(*testing.T, map[string]any, map[string]any)
	}{
		{
			name:              "successful create",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			customAttributes:  map[string]any{"source": "web"},
			statusCode:        http.StatusOK,
			responseBody:      `{"id": 1, "status": "open"}`,
			expectError:       false,
			validateFunc: func(t *testing.T, result map[string]any, body map[string]any) {
				if result["id"] != float64(1) {
					t.Errorf("Expected id 1, got %v", result["id"])
				}
				if body["custom_attributes"] == nil {
					t.Error("Expected custom_attributes in body")
				}
			},
		},
		{
			name:              "create without custom attributes",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			customAttributes:  nil,
			statusCode:        http.StatusOK,
			responseBody:      `{"id": 2, "status": "open"}`,
			expectError:       false,
			validateFunc: func(t *testing.T, result map[string]any, body map[string]any) {
				if result["id"] != float64(2) {
					t.Errorf("Expected id 2, got %v", result["id"])
				}
			},
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
			result, err := client.Public().CreateConversation(context.Background(), tt.inboxIdentifier, tt.contactIdentifier, tt.customAttributes)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result, capturedBody)
			}
		})
	}
}

func TestPublicListConversations(t *testing.T) {
	tests := []struct {
		name              string
		inboxIdentifier   string
		contactIdentifier string
		statusCode        int
		responseBody      string
		expectError       bool
		validateFunc      func(*testing.T, []map[string]any)
	}{
		{
			name:              "successful list",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			statusCode:        http.StatusOK,
			responseBody:      `[{"id": 1, "status": "open"}, {"id": 2, "status": "resolved"}]`,
			expectError:       false,
			validateFunc: func(t *testing.T, result []map[string]any) {
				if len(result) != 2 {
					t.Errorf("Expected 2 conversations, got %d", len(result))
				}
			},
		},
		{
			name:              "empty list",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			statusCode:        http.StatusOK,
			responseBody:      `[]`,
			expectError:       false,
			validateFunc: func(t *testing.T, result []map[string]any) {
				if len(result) != 0 {
					t.Errorf("Expected 0 conversations, got %d", len(result))
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
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Public().ListConversations(context.Background(), tt.inboxIdentifier, tt.contactIdentifier)

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

func TestPublicGetConversation(t *testing.T) {
	tests := []struct {
		name              string
		inboxIdentifier   string
		contactIdentifier string
		conversationID    int
		statusCode        int
		responseBody      string
		expectError       bool
		validateFunc      func(*testing.T, map[string]any)
	}{
		{
			name:              "successful get",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    1,
			statusCode:        http.StatusOK,
			responseBody:      `{"id": 1, "status": "open", "messages": []}`,
			expectError:       false,
			validateFunc: func(t *testing.T, result map[string]any) {
				if result["id"] != float64(1) {
					t.Errorf("Expected id 1, got %v", result["id"])
				}
			},
		},
		{
			name:              "not found",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    999,
			statusCode:        http.StatusNotFound,
			responseBody:      `{"error": "not found"}`,
			expectError:       true,
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
			result, err := client.Public().GetConversation(context.Background(), tt.inboxIdentifier, tt.contactIdentifier, tt.conversationID)

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

func TestPublicResolveConversation(t *testing.T) {
	tests := []struct {
		name              string
		inboxIdentifier   string
		contactIdentifier string
		conversationID    int
		statusCode        int
		responseBody      string
		expectError       bool
		validateFunc      func(*testing.T, map[string]any)
	}{
		{
			name:              "successful resolve",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    1,
			statusCode:        http.StatusOK,
			responseBody:      `{"id": 1, "status": "resolved"}`,
			expectError:       false,
			validateFunc: func(t *testing.T, result map[string]any) {
				if result["status"] != "resolved" {
					t.Errorf("Expected status 'resolved', got %v", result["status"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/toggle_status") {
					t.Errorf("Expected toggle_status path, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Public().ResolveConversation(context.Background(), tt.inboxIdentifier, tt.contactIdentifier, tt.conversationID)

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

func TestPublicToggleTyping(t *testing.T) {
	tests := []struct {
		name              string
		inboxIdentifier   string
		contactIdentifier string
		conversationID    int
		status            string
		statusCode        int
		expectError       bool
		validateBody      func(*testing.T, map[string]any)
	}{
		{
			name:              "toggle typing on",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    1,
			status:            "on",
			statusCode:        http.StatusOK,
			expectError:       false,
			validateBody: func(t *testing.T, body map[string]any) {
				if body["typing_status"] != "on" {
					t.Errorf("Expected typing_status 'on', got %v", body["typing_status"])
				}
			},
		},
		{
			name:              "toggle typing off",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    1,
			status:            "off",
			statusCode:        http.StatusOK,
			expectError:       false,
			validateBody: func(t *testing.T, body map[string]any) {
				if body["typing_status"] != "off" {
					t.Errorf("Expected typing_status 'off', got %v", body["typing_status"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/toggle_typing") {
					t.Errorf("Expected toggle_typing path, got %s", r.URL.Path)
				}
				_ = json.NewDecoder(r.Body).Decode(&capturedBody)
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.Public().ToggleTyping(context.Background(), tt.inboxIdentifier, tt.contactIdentifier, tt.conversationID, tt.status)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateBody != nil {
				tt.validateBody(t, capturedBody)
			}
		})
	}
}

func TestPublicUpdateLastSeen(t *testing.T) {
	tests := []struct {
		name              string
		inboxIdentifier   string
		contactIdentifier string
		conversationID    int
		statusCode        int
		expectError       bool
	}{
		{
			name:              "successful update",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    1,
			statusCode:        http.StatusOK,
			expectError:       false,
		},
		{
			name:              "not found",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    999,
			statusCode:        http.StatusNotFound,
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/update_last_seen") {
					t.Errorf("Expected update_last_seen path, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.Public().UpdateLastSeen(context.Background(), tt.inboxIdentifier, tt.contactIdentifier, tt.conversationID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestPublicCreateMessage(t *testing.T) {
	tests := []struct {
		name              string
		inboxIdentifier   string
		contactIdentifier string
		conversationID    int
		content           string
		echoID            string
		statusCode        int
		responseBody      string
		expectError       bool
		validateFunc      func(*testing.T, map[string]any, map[string]any)
	}{
		{
			name:              "successful create with echo_id",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    1,
			content:           "Hello!",
			echoID:            "echo123",
			statusCode:        http.StatusOK,
			responseBody:      `{"id": 1, "content": "Hello!"}`,
			expectError:       false,
			validateFunc: func(t *testing.T, result map[string]any, body map[string]any) {
				if body["content"] != "Hello!" {
					t.Errorf("Expected content 'Hello!', got %v", body["content"])
				}
				if body["echo_id"] != "echo123" {
					t.Errorf("Expected echo_id 'echo123', got %v", body["echo_id"])
				}
			},
		},
		{
			name:              "create without echo_id",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    1,
			content:           "Hi!",
			echoID:            "",
			statusCode:        http.StatusOK,
			responseBody:      `{"id": 2, "content": "Hi!"}`,
			expectError:       false,
			validateFunc: func(t *testing.T, result map[string]any, body map[string]any) {
				if _, ok := body["echo_id"]; ok {
					t.Error("Expected no echo_id in body")
				}
			},
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
			result, err := client.Public().CreateMessage(context.Background(), tt.inboxIdentifier, tt.contactIdentifier, tt.conversationID, tt.content, tt.echoID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result, capturedBody)
			}
		})
	}
}

func TestPublicGetInbox(t *testing.T) {
	tests := []struct {
		name            string
		inboxIdentifier string
		statusCode      int
		responseBody    string
		expectError     bool
		validateFunc    func(*testing.T, *PublicInbox)
	}{
		{
			name:            "successful get",
			inboxIdentifier: "inbox123",
			statusCode:      http.StatusOK,
			responseBody:    `{"name": "Support", "working_hours_enabled": true, "timezone": "UTC", "csat_survey_enabled": true}`,
			expectError:     false,
			validateFunc: func(t *testing.T, inbox *PublicInbox) {
				if inbox.Name != "Support" {
					t.Errorf("Expected name 'Support', got %s", inbox.Name)
				}
				if !inbox.WorkingHoursEnabled {
					t.Error("Expected working_hours_enabled to be true")
				}
				if !inbox.CsatSurveyEnabled {
					t.Error("Expected csat_survey_enabled to be true")
				}
			},
		},
		{
			name:            "not found",
			inboxIdentifier: "notexist",
			statusCode:      http.StatusNotFound,
			responseBody:    `{"error": "not found"}`,
			expectError:     true,
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
			result, err := client.Public().GetInbox(context.Background(), tt.inboxIdentifier)

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

func TestPublicListMessages(t *testing.T) {
	tests := []struct {
		name              string
		inboxIdentifier   string
		contactIdentifier string
		conversationID    int
		statusCode        int
		responseBody      string
		expectError       bool
		validateFunc      func(*testing.T, []map[string]any)
	}{
		{
			name:              "successful list",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    1,
			statusCode:        http.StatusOK,
			responseBody:      `[{"id": 1, "content": "Hello"}, {"id": 2, "content": "Hi there"}]`,
			expectError:       false,
			validateFunc: func(t *testing.T, result []map[string]any) {
				if len(result) != 2 {
					t.Errorf("Expected 2 messages, got %d", len(result))
				}
			},
		},
		{
			name:              "empty list",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    1,
			statusCode:        http.StatusOK,
			responseBody:      `[]`,
			expectError:       false,
			validateFunc: func(t *testing.T, result []map[string]any) {
				if len(result) != 0 {
					t.Errorf("Expected 0 messages, got %d", len(result))
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
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Public().ListMessages(context.Background(), tt.inboxIdentifier, tt.contactIdentifier, tt.conversationID)

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

func TestPublicUpdateMessage(t *testing.T) {
	tests := []struct {
		name              string
		inboxIdentifier   string
		contactIdentifier string
		conversationID    int
		messageID         int
		content           string
		statusCode        int
		responseBody      string
		expectError       bool
		validateFunc      func(*testing.T, map[string]any)
	}{
		{
			name:              "successful update",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    1,
			messageID:         1,
			content:           "Updated content",
			statusCode:        http.StatusOK,
			responseBody:      `{"id": 1, "content": "Updated content"}`,
			expectError:       false,
			validateFunc: func(t *testing.T, result map[string]any) {
				if result["content"] != "Updated content" {
					t.Errorf("Expected content 'Updated content', got %v", result["content"])
				}
			},
		},
		{
			name:              "not found",
			inboxIdentifier:   "inbox123",
			contactIdentifier: "contact456",
			conversationID:    1,
			messageID:         999,
			content:           "Updated",
			statusCode:        http.StatusNotFound,
			responseBody:      `{"error": "not found"}`,
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Errorf("Expected PATCH, got %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Public().UpdateMessage(context.Background(), tt.inboxIdentifier, tt.contactIdentifier, tt.conversationID, tt.messageID, tt.content)

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
