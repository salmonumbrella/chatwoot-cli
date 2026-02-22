package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreatePlatformAccount(t *testing.T) {
	tests := []struct {
		name         string
		req          CreatePlatformAccountRequest
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *PlatformAccount)
	}{
		{
			name: "successful create",
			req: CreatePlatformAccountRequest{
				Name:   "Test Account",
				Locale: "en",
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Test Account", "locale": "en"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, acc *PlatformAccount) {
				if acc.ID != 1 {
					t.Errorf("Expected ID 1, got %d", acc.ID)
				}
				if acc.Name != "Test Account" {
					t.Errorf("Expected name 'Test Account', got %s", acc.Name)
				}
			},
		},
		{
			name: "server error",
			req: CreatePlatformAccountRequest{
				Name: "Test",
			},
			statusCode:   http.StatusBadRequest,
			responseBody: `{"error": "validation failed"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/platform/api/v1/accounts") {
					t.Errorf("Expected platform path, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Platform().CreateAccount(context.Background(), tt.req)

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

func TestGetPlatformAccount(t *testing.T) {
	tests := []struct {
		name         string
		accountID    int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *PlatformAccount)
	}{
		{
			name:         "successful get",
			accountID:    1,
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Test Account", "locale": "en", "status": "active"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, acc *PlatformAccount) {
				if acc.ID != 1 {
					t.Errorf("Expected ID 1, got %d", acc.ID)
				}
				if acc.Status != "active" {
					t.Errorf("Expected status 'active', got %s", acc.Status)
				}
			},
		},
		{
			name:         "not found",
			accountID:    999,
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
			result, err := client.Platform().GetAccount(context.Background(), tt.accountID)

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

func TestUpdatePlatformAccount(t *testing.T) {
	tests := []struct {
		name         string
		accountID    int
		req          UpdatePlatformAccountRequest
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *PlatformAccount, map[string]any)
	}{
		{
			name:      "update name",
			accountID: 1,
			req: UpdatePlatformAccountRequest{
				Name: "Updated Name",
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Updated Name"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, acc *PlatformAccount, body map[string]any) {
				if acc.Name != "Updated Name" {
					t.Errorf("Expected name 'Updated Name', got %s", acc.Name)
				}
			},
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
			result, err := client.Platform().UpdateAccount(context.Background(), tt.accountID, tt.req)

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

func TestDeletePlatformAccount(t *testing.T) {
	tests := []struct {
		name        string
		accountID   int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			accountID:   1,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			accountID:   999,
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
			err := client.Platform().DeleteAccount(context.Background(), tt.accountID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestCreatePlatformUser(t *testing.T) {
	tests := []struct {
		name         string
		req          CreatePlatformUserRequest
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *PlatformUser)
	}{
		{
			name: "successful create",
			req: CreatePlatformUserRequest{
				Name:     "Test User",
				Email:    "test@example.com",
				Password: "securepassword",
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Test User", "email": "test@example.com"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, user *PlatformUser) {
				if user.ID != 1 {
					t.Errorf("Expected ID 1, got %d", user.ID)
				}
				if user.Email != "test@example.com" {
					t.Errorf("Expected email 'test@example.com', got %s", user.Email)
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
				if !strings.Contains(r.URL.Path, "/platform/api/v1/users") {
					t.Errorf("Expected platform users path, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Platform().CreateUser(context.Background(), tt.req)

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

func TestGetPlatformUser(t *testing.T) {
	tests := []struct {
		name         string
		userID       int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *PlatformUser)
	}{
		{
			name:         "successful get",
			userID:       1,
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Test User", "email": "test@example.com", "display_name": "Testy"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, user *PlatformUser) {
				if user.ID != 1 {
					t.Errorf("Expected ID 1, got %d", user.ID)
				}
				if user.DisplayName != "Testy" {
					t.Errorf("Expected display_name 'Testy', got %s", user.DisplayName)
				}
			},
		},
		{
			name:         "not found",
			userID:       999,
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
			result, err := client.Platform().GetUser(context.Background(), tt.userID)

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

func TestUpdatePlatformUser(t *testing.T) {
	tests := []struct {
		name         string
		userID       int
		req          UpdatePlatformUserRequest
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *PlatformUser)
	}{
		{
			name:   "update name",
			userID: 1,
			req: UpdatePlatformUserRequest{
				Name: "Updated Name",
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Updated Name", "email": "test@example.com"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, user *PlatformUser) {
				if user.Name != "Updated Name" {
					t.Errorf("Expected name 'Updated Name', got %s", user.Name)
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
			result, err := client.Platform().UpdateUser(context.Background(), tt.userID, tt.req)

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

func TestDeletePlatformUser(t *testing.T) {
	tests := []struct {
		name        string
		userID      int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			userID:      1,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			userID:      999,
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
			err := client.Platform().DeleteUser(context.Background(), tt.userID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGetPlatformUserLogin(t *testing.T) {
	tests := []struct {
		name         string
		userID       int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *PlatformUserLogin)
	}{
		{
			name:         "successful get login",
			userID:       1,
			statusCode:   http.StatusOK,
			responseBody: `{"url": "https://example.com/sso?token=abc123"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, login *PlatformUserLogin) {
				if login.URL != "https://example.com/sso?token=abc123" {
					t.Errorf("Expected SSO URL, got %s", login.URL)
				}
			},
		},
		{
			name:         "not found",
			userID:       999,
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
				if !strings.Contains(r.URL.Path, "/login") {
					t.Errorf("Expected login path, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Platform().GetUserLogin(context.Background(), tt.userID)

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

func TestListPlatformAccountUsers(t *testing.T) {
	tests := []struct {
		name         string
		accountID    int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []PlatformAccountUser)
	}{
		{
			name:       "successful list",
			accountID:  1,
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "account_id": 1, "user_id": 10, "role": "administrator"},
				{"id": 2, "account_id": 1, "user_id": 20, "role": "agent"}
			]`,
			expectError: false,
			validateFunc: func(t *testing.T, users []PlatformAccountUser) {
				if len(users) != 2 {
					t.Errorf("Expected 2 users, got %d", len(users))
				}
				if users[0].Role != "administrator" {
					t.Errorf("Expected role 'administrator', got %s", users[0].Role)
				}
			},
		},
		{
			name:         "empty list",
			accountID:    1,
			statusCode:   http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			validateFunc: func(t *testing.T, users []PlatformAccountUser) {
				if len(users) != 0 {
					t.Errorf("Expected 0 users, got %d", len(users))
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
				if !strings.Contains(r.URL.Path, "/account_users") {
					t.Errorf("Expected account_users path, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Platform().ListAccountUsers(context.Background(), tt.accountID)

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

func TestCreatePlatformAccountUser(t *testing.T) {
	tests := []struct {
		name         string
		accountID    int
		req          CreatePlatformAccountUserRequest
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *PlatformAccountUser)
	}{
		{
			name:      "successful create",
			accountID: 1,
			req: CreatePlatformAccountUserRequest{
				UserID: 10,
				Role:   "agent",
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "account_id": 1, "user_id": 10, "role": "agent"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, user *PlatformAccountUser) {
				if user.UserID != 10 {
					t.Errorf("Expected user_id 10, got %d", user.UserID)
				}
				if user.Role != "agent" {
					t.Errorf("Expected role 'agent', got %s", user.Role)
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
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Platform().CreateAccountUser(context.Background(), tt.accountID, tt.req)

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

func TestDeletePlatformAccountUser(t *testing.T) {
	tests := []struct {
		name        string
		accountID   int
		userID      int
		statusCode  int
		expectError bool
		validateURL func(*testing.T, string)
	}{
		{
			name:        "successful delete with user_id",
			accountID:   1,
			userID:      10,
			statusCode:  http.StatusOK,
			expectError: false,
			validateURL: func(t *testing.T, url string) {
				if !strings.Contains(url, "user_id=10") {
					t.Errorf("Expected user_id query param, got %s", url)
				}
			},
		},
		{
			name:        "delete without user_id",
			accountID:   1,
			userID:      0,
			statusCode:  http.StatusOK,
			expectError: false,
			validateURL: func(t *testing.T, url string) {
				if strings.Contains(url, "user_id") {
					t.Errorf("Expected no user_id query param, got %s", url)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE, got %s", r.Method)
				}
				capturedURL = r.URL.String()
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.Platform().DeleteAccountUser(context.Background(), tt.accountID, tt.userID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateURL != nil {
				tt.validateURL(t, capturedURL)
			}
		})
	}
}
