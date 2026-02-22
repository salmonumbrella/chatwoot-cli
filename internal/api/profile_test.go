package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetProfile(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Profile)
	}{
		{
			name:       "successful get",
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"name": "John Doe",
				"email": "john@example.com",
				"accounts": [
					{"id": 1, "name": "Account One", "locale": "en"},
					{"id": 2, "name": "Account Two", "locale": "fr"}
				]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, profile *Profile) {
				if profile.ID != 1 {
					t.Errorf("Expected ID 1, got %d", profile.ID)
				}
				if profile.Name != "John Doe" {
					t.Errorf("Expected name 'John Doe', got %s", profile.Name)
				}
				if profile.Email != "john@example.com" {
					t.Errorf("Expected email 'john@example.com', got %s", profile.Email)
				}
				if len(profile.AvailableAccounts) != 2 {
					t.Errorf("Expected 2 available accounts, got %d", len(profile.AvailableAccounts))
				}
				if profile.AvailableAccounts[0].Name != "Account One" {
					t.Errorf("Expected account name 'Account One', got %s", profile.AvailableAccounts[0].Name)
				}
			},
		},
		{
			name:         "minimal profile",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 2, "name": "Jane Doe", "email": "jane@example.com"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, profile *Profile) {
				if profile.ID != 2 {
					t.Errorf("Expected ID 2, got %d", profile.ID)
				}
				if len(profile.AvailableAccounts) != 0 {
					t.Errorf("Expected 0 available accounts, got %d", len(profile.AvailableAccounts))
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
				// Verify the path is /api/v1/profile (not account-scoped)
				if !strings.Contains(r.URL.Path, "/api/v1/profile") {
					t.Errorf("Expected profile path, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Profile().Get(context.Background())

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

func TestProfileUnmarshalPubsubToken(t *testing.T) {
	raw := `{
		"id": 42,
		"name": "Agent Smith",
		"email": "smith@example.com",
		"pubsub_token": "abc123token",
		"accounts": [{"id": 1, "name": "Acme"}]
	}`
	var p Profile
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.PubsubToken != "abc123token" {
		t.Errorf("PubsubToken = %q, want %q", p.PubsubToken, "abc123token")
	}
	if p.ID != 42 {
		t.Errorf("ID = %d, want 42", p.ID)
	}
}
