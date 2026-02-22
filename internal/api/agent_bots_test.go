package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAgentBots(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []AgentBot)
	}{
		{
			name:       "successful list",
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "name": "Bot One", "description": "First bot", "outgoing_url": "https://bot1.example.com/webhook", "account_id": 1},
				{"id": 2, "name": "Bot Two", "outgoing_url": "https://bot2.example.com/webhook", "account_id": 1}
			]`,
			expectError: false,
			validateFunc: func(t *testing.T, bots []AgentBot) {
				if len(bots) != 2 {
					t.Errorf("Expected 2 bots, got %d", len(bots))
				}
				if bots[0].Name != "Bot One" {
					t.Errorf("Expected name 'Bot One', got %s", bots[0].Name)
				}
				if bots[0].OutgoingURL != "https://bot1.example.com/webhook" {
					t.Errorf("Expected outgoing URL, got %s", bots[0].OutgoingURL)
				}
			},
		},
		{
			name:         "empty list",
			statusCode:   http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			validateFunc: func(t *testing.T, bots []AgentBot) {
				if len(bots) != 0 {
					t.Errorf("Expected 0 bots, got %d", len(bots))
				}
			},
		},
		{
			name:         "server error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "internal error"}`,
			expectError:  true,
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
			result, err := client.AgentBots().List(context.Background())

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

func TestGetAgentBot(t *testing.T) {
	tests := []struct {
		name         string
		botID        int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *AgentBot)
	}{
		{
			name:         "successful get",
			botID:        1,
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Bot One", "description": "First bot", "outgoing_url": "https://bot.example.com/webhook", "account_id": 1}`,
			expectError:  false,
			validateFunc: func(t *testing.T, bot *AgentBot) {
				if bot.ID != 1 {
					t.Errorf("Expected ID 1, got %d", bot.ID)
				}
				if bot.Description != "First bot" {
					t.Errorf("Expected description 'First bot', got %s", bot.Description)
				}
			},
		},
		{
			name:         "not found",
			botID:        999,
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
			expectError:  true,
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
			result, err := client.AgentBots().Get(context.Background(), tt.botID)

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

func TestCreateAgentBot(t *testing.T) {
	tests := []struct {
		name         string
		botName      string
		outgoingURL  string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *AgentBot, map[string]any)
	}{
		{
			name:         "successful create",
			botName:      "New Bot",
			outgoingURL:  "https://newbot.example.com/webhook",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "New Bot", "outgoing_url": "https://newbot.example.com/webhook", "account_id": 1}`,
			expectError:  false,
			validateFunc: func(t *testing.T, bot *AgentBot, body map[string]any) {
				if bot.Name != "New Bot" {
					t.Errorf("Expected name 'New Bot', got %s", bot.Name)
				}
				if body["name"] != "New Bot" {
					t.Errorf("Expected name in body, got %v", body["name"])
				}
				if body["outgoing_url"] != "https://newbot.example.com/webhook" {
					t.Errorf("Expected outgoing_url in body, got %v", body["outgoing_url"])
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
			result, err := client.AgentBots().Create(context.Background(), tt.botName, tt.outgoingURL)

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

func TestUpdateAgentBot(t *testing.T) {
	tests := []struct {
		name         string
		botID        int
		botName      string
		outgoingURL  string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *AgentBot, map[string]any)
	}{
		{
			name:         "update all fields",
			botID:        1,
			botName:      "Updated Bot",
			outgoingURL:  "https://updated.example.com/webhook",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Updated Bot", "outgoing_url": "https://updated.example.com/webhook", "account_id": 1}`,
			expectError:  false,
			validateFunc: func(t *testing.T, bot *AgentBot, body map[string]any) {
				if bot.Name != "Updated Bot" {
					t.Errorf("Expected name 'Updated Bot', got %s", bot.Name)
				}
				if body["name"] != "Updated Bot" {
					t.Errorf("Expected name in body, got %v", body["name"])
				}
			},
		},
		{
			name:         "partial update - name only",
			botID:        1,
			botName:      "Only Name",
			outgoingURL:  "",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Only Name", "outgoing_url": "https://original.example.com/webhook", "account_id": 1}`,
			expectError:  false,
			validateFunc: func(t *testing.T, bot *AgentBot, body map[string]any) {
				if _, ok := body["outgoing_url"]; ok {
					t.Error("Expected no outgoing_url in body when empty")
				}
			},
		},
		{
			name:         "not found",
			botID:        999,
			botName:      "Test",
			outgoingURL:  "https://test.example.com",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
			expectError:  true,
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
			result, err := client.AgentBots().Update(context.Background(), tt.botID, tt.botName, tt.outgoingURL)

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

func TestDeleteAgentBot(t *testing.T) {
	tests := []struct {
		name        string
		botID       int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			botID:       1,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			botID:       999,
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE, got %s", r.Method)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.AgentBots().Delete(context.Background(), tt.botID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDeleteAgentBotAvatar(t *testing.T) {
	tests := []struct {
		name        string
		botID       int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete avatar",
			botID:       1,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			botID:       999,
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE, got %s", r.Method)
				}
				expectedPath := "/api/v1/accounts/1/agent_bots/1/avatar"
				if tt.botID == 999 {
					expectedPath = "/api/v1/accounts/1/agent_bots/999/avatar"
				}
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.AgentBots().DeleteAvatar(context.Background(), tt.botID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestResetAgentBotAccessToken(t *testing.T) {
	tests := []struct {
		name         string
		botID        int
		statusCode   int
		responseBody string
		expectError  bool
		expectToken  string
	}{
		{
			name:         "successful reset",
			botID:        1,
			statusCode:   http.StatusOK,
			responseBody: `{"access_token": "new_test_token_123"}`,
			expectError:  false,
			expectToken:  "new_test_token_123",
		},
		{
			name:         "not found",
			botID:        999,
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			token, err := client.AgentBots().ResetAccessToken(context.Background(), tt.botID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if token != tt.expectToken {
				t.Errorf("Expected token %s, got %s", tt.expectToken, token)
			}
		})
	}
}
