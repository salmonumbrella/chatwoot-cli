package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListIntegrationApps(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []Integration)
	}{
		{
			name:       "successful list",
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [
					{"id": "slack", "name": "Slack", "description": "Slack integration", "hook_type": "account", "enabled": true, "allow_multiple_hooks": false, "hooks": []},
					{"id": "dialogflow", "name": "Dialogflow", "description": "Dialogflow integration", "hook_type": "inbox", "enabled": true, "allow_multiple_hooks": true, "hooks": [{"id": 1, "app_id": "dialogflow", "inbox_id": 10, "account_id": 1}]}
				]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, integrations []Integration) {
				if len(integrations) != 2 {
					t.Errorf("Expected 2 integrations, got %d", len(integrations))
				}
				if integrations[0].ID != "slack" {
					t.Errorf("Expected ID 'slack', got %s", integrations[0].ID)
				}
				if integrations[0].Name != "Slack" {
					t.Errorf("Expected name 'Slack', got %s", integrations[0].Name)
				}
				if integrations[1].AllowMultipleHooks != true {
					t.Error("Expected allow_multiple_hooks to be true")
				}
				if len(integrations[1].Hooks) != 1 {
					t.Errorf("Expected 1 hook, got %d", len(integrations[1].Hooks))
				}
			},
		},
		{
			name:         "empty list",
			statusCode:   http.StatusOK,
			responseBody: `{"payload": []}`,
			expectError:  false,
			validateFunc: func(t *testing.T, integrations []Integration) {
				if len(integrations) != 0 {
					t.Errorf("Expected 0 integrations, got %d", len(integrations))
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
			result, err := client.Integrations().ListApps(context.Background())

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

func TestListIntegrationHooks(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []IntegrationHook)
	}{
		{
			name:       "successful list - hooks from apps",
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [
					{"id": "app1", "name": "App One", "hooks": [
						{"id": 1, "app_id": "app1", "inbox_id": 10, "account_id": 1},
						{"id": 2, "app_id": "app1", "inbox_id": 20, "account_id": 1}
					]},
					{"id": "app2", "name": "App Two", "hooks": [
						{"id": 3, "app_id": "app2", "account_id": 1}
					]}
				]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, hooks []IntegrationHook) {
				if len(hooks) != 3 {
					t.Errorf("Expected 3 hooks, got %d", len(hooks))
				}
				if hooks[0].ID != 1 {
					t.Errorf("Expected first hook ID 1, got %d", hooks[0].ID)
				}
				if hooks[2].AppID != "app2" {
					t.Errorf("Expected app_id 'app2', got %s", hooks[2].AppID)
				}
			},
		},
		{
			name:         "no hooks",
			statusCode:   http.StatusOK,
			responseBody: `{"payload": [{"id": "app1", "name": "App One", "hooks": []}]}`,
			expectError:  false,
			validateFunc: func(t *testing.T, hooks []IntegrationHook) {
				if len(hooks) != 0 {
					t.Errorf("Expected 0 hooks, got %d", len(hooks))
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
			result, err := client.Integrations().ListHooks(context.Background())

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

func TestCreateIntegrationHook(t *testing.T) {
	tests := []struct {
		name         string
		appID        string
		inboxID      int
		settings     map[string]any
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *IntegrationHook, map[string]any)
	}{
		{
			name:         "successful create with inbox",
			appID:        "dialogflow",
			inboxID:      10,
			settings:     map[string]any{"project_id": "my-project"},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "app_id": "dialogflow", "inbox_id": 10, "account_id": 1, "settings": {"project_id": "my-project"}}`,
			expectError:  false,
			validateFunc: func(t *testing.T, hook *IntegrationHook, body map[string]any) {
				if hook.ID != 1 {
					t.Errorf("Expected ID 1, got %d", hook.ID)
				}
				if hook.InboxID != 10 {
					t.Errorf("Expected inbox_id 10, got %d", hook.InboxID)
				}
				if body["app_id"] != "dialogflow" {
					t.Errorf("Expected app_id in body, got %v", body["app_id"])
				}
				if body["inbox_id"] != float64(10) {
					t.Errorf("Expected inbox_id in body, got %v", body["inbox_id"])
				}
			},
		},
		{
			name:         "create without inbox",
			appID:        "slack",
			inboxID:      0,
			settings:     nil,
			statusCode:   http.StatusOK,
			responseBody: `{"id": 2, "app_id": "slack", "account_id": 1}`,
			expectError:  false,
			validateFunc: func(t *testing.T, hook *IntegrationHook, body map[string]any) {
				if _, ok := body["inbox_id"]; ok {
					t.Error("Expected no inbox_id in body when 0")
				}
				if _, ok := body["settings"]; ok {
					t.Error("Expected no settings in body when nil")
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
			result, err := client.Integrations().CreateHook(context.Background(), tt.appID, tt.inboxID, tt.settings)

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

func TestUpdateIntegrationHook(t *testing.T) {
	tests := []struct {
		name         string
		hookID       int
		settings     map[string]any
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *IntegrationHook, map[string]any)
	}{
		{
			name:         "successful update",
			hookID:       1,
			settings:     map[string]any{"api_key": "new-key"},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "app_id": "slack", "account_id": 1, "settings": {"api_key": "new-key"}}`,
			expectError:  false,
			validateFunc: func(t *testing.T, hook *IntegrationHook, body map[string]any) {
				if hook.ID != 1 {
					t.Errorf("Expected ID 1, got %d", hook.ID)
				}
				settings := body["settings"].(map[string]any)
				if settings["api_key"] != "new-key" {
					t.Errorf("Expected api_key in settings, got %v", settings["api_key"])
				}
			},
		},
		{
			name:         "update with nil settings",
			hookID:       1,
			settings:     nil,
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "app_id": "slack", "account_id": 1}`,
			expectError:  false,
			validateFunc: func(t *testing.T, hook *IntegrationHook, body map[string]any) {
				if _, ok := body["settings"]; ok {
					t.Error("Expected no settings in body when nil")
				}
			},
		},
		{
			name:         "not found",
			hookID:       999,
			settings:     map[string]any{},
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
			result, err := client.Integrations().UpdateHook(context.Background(), tt.hookID, tt.settings)

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

func TestDeleteIntegrationHook(t *testing.T) {
	tests := []struct {
		name        string
		hookID      int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			hookID:      1,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			hookID:      999,
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
			err := client.Integrations().DeleteHook(context.Background(), tt.hookID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
