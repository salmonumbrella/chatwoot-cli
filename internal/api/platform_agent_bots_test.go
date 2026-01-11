package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListPlatformAgentBots(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []PlatformAgentBot)
	}{
		{
			name:       "successful list",
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "name": "Bot 1", "description": "First bot", "bot_type": "webhook"},
				{"id": 2, "name": "Bot 2", "description": "Second bot", "bot_type": "csml"}
			]`,
			expectError: false,
			validateFunc: func(t *testing.T, result []PlatformAgentBot) {
				if len(result) != 2 {
					t.Errorf("Expected 2 bots, got %d", len(result))
				}
				if result[0].Name != "Bot 1" {
					t.Errorf("Expected name 'Bot 1', got %s", result[0].Name)
				}
			},
		},
		{
			name:         "empty list",
			statusCode:   http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			validateFunc: func(t *testing.T, result []PlatformAgentBot) {
				if len(result) != 0 {
					t.Errorf("Expected 0 bots, got %d", len(result))
				}
			},
		},
		{
			name:         "unauthorized",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error": "Unauthorized"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/platform/api/v1/agent_bots") {
					t.Errorf("Expected path to contain /platform/api/v1/agent_bots, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.PlatformAgentBots().List(context.Background())

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

func TestGetPlatformAgentBot(t *testing.T) {
	tests := []struct {
		name         string
		botID        int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *PlatformAgentBot)
	}{
		{
			name:       "successful get",
			botID:      1,
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"name": "Test Bot",
				"description": "A test bot",
				"outgoing_url": "https://example.com/webhook",
				"bot_type": "webhook",
				"access_token": "token123"
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *PlatformAgentBot) {
				if result.ID != 1 {
					t.Errorf("Expected ID 1, got %d", result.ID)
				}
				if result.Name != "Test Bot" {
					t.Errorf("Expected name 'Test Bot', got %s", result.Name)
				}
				if result.AccessToken != "token123" {
					t.Errorf("Expected access_token 'token123', got %s", result.AccessToken)
				}
			},
		},
		{
			name:         "not found",
			botID:        999,
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "Bot not found"}`,
			expectError:  true,
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
			result, err := client.PlatformAgentBots().Get(context.Background(), tt.botID)

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

func TestCreatePlatformAgentBot(t *testing.T) {
	tests := []struct {
		name            string
		request         CreatePlatformAgentBotRequest
		statusCode      int
		responseBody    string
		expectError     bool
		validateFunc    func(*testing.T, *PlatformAgentBot)
		validatePayload func(*testing.T, map[string]any)
	}{
		{
			name: "successful create",
			request: CreatePlatformAgentBotRequest{
				Name:        "New Bot",
				Description: "A new bot",
				OutgoingURL: "https://example.com/webhook",
			},
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"name": "New Bot",
				"description": "A new bot",
				"outgoing_url": "https://example.com/webhook",
				"access_token": "new-token"
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *PlatformAgentBot) {
				if result.ID != 1 {
					t.Errorf("Expected ID 1, got %d", result.ID)
				}
				if result.Name != "New Bot" {
					t.Errorf("Expected name 'New Bot', got %s", result.Name)
				}
			},
			validatePayload: func(t *testing.T, payload map[string]any) {
				if payload["name"] != "New Bot" {
					t.Errorf("Expected name 'New Bot', got %v", payload["name"])
				}
			},
		},
		{
			name: "validation error",
			request: CreatePlatformAgentBotRequest{
				Name: "",
			},
			statusCode:   http.StatusUnprocessableEntity,
			responseBody: `{"error": "Name is required"}`,
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
			result, err := client.PlatformAgentBots().Create(context.Background(), tt.request)

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

func TestUpdatePlatformAgentBot(t *testing.T) {
	tests := []struct {
		name         string
		botID        int
		request      UpdatePlatformAgentBotRequest
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *PlatformAgentBot)
	}{
		{
			name:  "successful update",
			botID: 1,
			request: UpdatePlatformAgentBotRequest{
				Name:        "Updated Bot",
				Description: "Updated description",
			},
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"name": "Updated Bot",
				"description": "Updated description"
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *PlatformAgentBot) {
				if result.Name != "Updated Bot" {
					t.Errorf("Expected name 'Updated Bot', got %s", result.Name)
				}
			},
		},
		{
			name:  "not found",
			botID: 999,
			request: UpdatePlatformAgentBotRequest{
				Name: "Updated",
			},
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "Bot not found"}`,
			expectError:  true,
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
			result, err := client.PlatformAgentBots().Update(context.Background(), tt.botID, tt.request)

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

func TestDeletePlatformAgentBot(t *testing.T) {
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
				if tt.statusCode >= 400 {
					_, _ = w.Write([]byte(`{"error": "error"}`))
				}
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.PlatformAgentBots().Delete(context.Background(), tt.botID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
