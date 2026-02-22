package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAccount(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Account)
	}{
		{
			name:         "successful get",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Test Account", "locale": "en", "domain": "example.com"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, account *Account) {
				if account.ID != 1 {
					t.Errorf("Expected ID 1, got %d", account.ID)
				}
				if account.Name != "Test Account" {
					t.Errorf("Expected name 'Test Account', got %s", account.Name)
				}
				if account.Locale != "en" {
					t.Errorf("Expected locale 'en', got %s", account.Locale)
				}
				if account.Domain != "example.com" {
					t.Errorf("Expected domain 'example.com', got %s", account.Domain)
				}
			},
		},
		{
			name:         "minimal response",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 2, "name": "Minimal", "locale": "fr"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, account *Account) {
				if account.ID != 2 {
					t.Errorf("Expected ID 2, got %d", account.ID)
				}
				if account.Domain != "" {
					t.Errorf("Expected empty domain, got %s", account.Domain)
				}
			},
		},
		{
			name:         "unauthorized",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error": "unauthorized"}`,
			expectError:  true,
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
			result, err := client.Account().Get(context.Background())

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

func TestUpdateAccount(t *testing.T) {
	tests := []struct {
		name         string
		newName      string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Account, map[string]any)
	}{
		{
			name:         "successful update",
			newName:      "Updated Account",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Updated Account", "locale": "en"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, account *Account, body map[string]any) {
				if account.Name != "Updated Account" {
					t.Errorf("Expected name 'Updated Account', got %s", account.Name)
				}
				if body["name"] != "Updated Account" {
					t.Errorf("Expected name in body, got %v", body["name"])
				}
			},
		},
		{
			name:         "update with empty name",
			newName:      "",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Original", "locale": "en"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, account *Account, body map[string]any) {
				if _, ok := body["name"]; ok {
					t.Error("Expected no name in body when empty")
				}
			},
		},
		{
			name:         "not found",
			newName:      "Test",
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
			result, err := client.Account().Update(context.Background(), tt.newName)

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
