package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAgents(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
		expectedLen  int
	}{
		{
			name:       "successful list",
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "name": "Agent One", "email": "agent1@example.com", "role": "agent"},
				{"id": 2, "name": "Agent Two", "email": "agent2@example.com", "role": "administrator"}
			]`,
			expectError: false,
			expectedLen: 2,
		},
		{
			name:         "empty list",
			statusCode:   http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			expectedLen:  0,
		},
		{
			name:         "server error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "internal server error"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}

				// Verify the path
				expectedPath := "/api/v1/accounts/1/agents"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				// Verify authentication header
				if r.Header.Get("api_access_token") != "test-token" {
					t.Error("Missing or incorrect api_access_token header")
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Agents().List(context.Background())

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && len(result) != tt.expectedLen {
				t.Errorf("Expected %d agents, got %d", tt.expectedLen, len(result))
			}
			if !tt.expectError && tt.expectedLen > 0 {
				if result[0].Role != "agent" {
					t.Errorf("Expected first agent role 'agent', got %s", result[0].Role)
				}
				if result[0].Name != "Agent One" {
					t.Errorf("Expected first agent name 'Agent One', got %s", result[0].Name)
				}
			}
		})
	}
}

func TestGetAgent(t *testing.T) {
	tests := []struct {
		name         string
		agentID      int
		statusCode   int
		responseBody string
		expectError  bool
		expectedName string
	}{
		{
			name:    "successful get - existing agent",
			agentID: 1,
			responseBody: `[
				{"id": 1, "name": "Agent One", "email": "agent1@example.com", "role": "agent"},
				{"id": 2, "name": "Agent Two", "email": "agent2@example.com", "role": "administrator"}
			]`,
			statusCode:   http.StatusOK,
			expectError:  false,
			expectedName: "Agent One",
		},
		{
			name:    "agent not found in list",
			agentID: 999,
			responseBody: `[
				{"id": 1, "name": "Agent One", "email": "agent1@example.com", "role": "agent"},
				{"id": 2, "name": "Agent Two", "email": "agent2@example.com", "role": "administrator"}
			]`,
			statusCode:  http.StatusOK,
			expectError: true,
		},
		{
			name:         "server error",
			agentID:      1,
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "internal server error"}`,
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
			result, err := client.Agents().Get(context.Background(), tt.agentID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError {
				if result == nil {
					t.Fatal("Expected result, got nil")
				}
				if result.ID != tt.agentID {
					t.Errorf("Expected ID %d, got %d", tt.agentID, result.ID)
				}
				if result.Name != tt.expectedName {
					t.Errorf("Expected name %s, got %s", tt.expectedName, result.Name)
				}
			}
		})
	}
}

func TestCreateAgent(t *testing.T) {
	tests := []struct {
		name         string
		agentName    string
		email        string
		role         string
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name:       "successful creation",
			agentName:  "New Agent",
			email:      "newagent@example.com",
			role:       "agent",
			statusCode: http.StatusCreated,
			responseBody: `{
				"id": 3,
				"name": "New Agent",
				"email": "newagent@example.com",
				"role": "agent"
			}`,
			expectError: false,
		},
		{
			name:         "validation error",
			agentName:    "",
			email:        "invalid",
			role:         "agent",
			statusCode:   http.StatusUnprocessableEntity,
			responseBody: `{"error": "validation failed"}`,
			expectError:  true,
		},
		{
			name:         "duplicate email",
			agentName:    "Duplicate",
			email:        "existing@example.com",
			role:         "agent",
			statusCode:   http.StatusConflict,
			responseBody: `{"error": "email already exists"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}

				expectedPath := "/api/v1/accounts/1/agents"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Agents().Create(context.Background(), tt.agentName, tt.email, tt.role)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError {
				if result == nil {
					t.Fatal("Expected result, got nil")
				}
				if result.Name != tt.agentName {
					t.Errorf("Expected name %s, got %s", tt.agentName, result.Name)
				}
				if result.Email != tt.email {
					t.Errorf("Expected email %s, got %s", tt.email, result.Email)
				}
				if result.Role != tt.role {
					t.Errorf("Expected role %s, got %s", tt.role, result.Role)
				}
			}
		})
	}
}

func TestUpdateAgent(t *testing.T) {
	tests := []struct {
		name         string
		agentID      int
		agentName    string
		role         string
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name:       "successful update - name and role",
			agentID:    1,
			agentName:  "Updated Name",
			role:       "administrator",
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"name": "Updated Name",
				"email": "agent@example.com",
				"role": "administrator"
			}`,
			expectError: false,
		},
		{
			name:       "successful update - name only",
			agentID:    1,
			agentName:  "Updated Name",
			role:       "",
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"name": "Updated Name",
				"email": "agent@example.com",
				"role": "agent"
			}`,
			expectError: false,
		},
		{
			name:       "successful update - role only",
			agentID:    1,
			agentName:  "",
			role:       "administrator",
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"name": "Agent One",
				"email": "agent@example.com",
				"role": "administrator"
			}`,
			expectError: false,
		},
		{
			name:         "agent not found",
			agentID:      999,
			agentName:    "Updated Name",
			role:         "agent",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Errorf("Expected PATCH, got %s", r.Method)
				}

				expectedPath := fmt.Sprintf("/api/v1/accounts/1/agents/%d", tt.agentID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Agents().Update(context.Background(), tt.agentID, tt.agentName, tt.role)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError {
				if result == nil {
					t.Fatal("Expected result, got nil")
				}
				if result.ID != tt.agentID {
					t.Errorf("Expected ID %d, got %d", tt.agentID, result.ID)
				}
				// Only check name if we set it
				if tt.agentName != "" && result.Name != tt.agentName {
					t.Errorf("Expected name %s, got %s", tt.agentName, result.Name)
				}
				// Only check role if we set it
				if tt.role != "" && result.Role != tt.role {
					t.Errorf("Expected role %s, got %s", tt.role, result.Role)
				}
			}
		})
	}
}

func TestDeleteAgent(t *testing.T) {
	tests := []struct {
		name         string
		agentID      int
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful deletion",
			agentID:      1,
			statusCode:   http.StatusOK,
			responseBody: `{}`,
			expectError:  false,
		},
		{
			name:         "agent not found",
			agentID:      999,
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
			expectError:  true,
		},
		{
			name:         "forbidden",
			agentID:      1,
			statusCode:   http.StatusForbidden,
			responseBody: `{"error": "forbidden"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE, got %s", r.Method)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.Agents().Delete(context.Background(), tt.agentID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestBulkCreateAgents(t *testing.T) {
	tests := []struct {
		name         string
		emails       []string
		statusCode   int
		responseBody string
		expectError  bool
		expectedLen  int
	}{
		{
			name:       "successful bulk create",
			emails:     []string{"agent1@example.com", "agent2@example.com"},
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "name": "Agent One", "email": "agent1@example.com", "role": "agent"},
				{"id": 2, "name": "Agent Two", "email": "agent2@example.com", "role": "agent"}
			]`,
			expectError: false,
			expectedLen: 2,
		},
		{
			name:       "single email",
			emails:     []string{"single@example.com"},
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "name": "Single Agent", "email": "single@example.com", "role": "agent"}
			]`,
			expectError: false,
			expectedLen: 1,
		},
		{
			name:         "validation error",
			emails:       []string{"invalid-email"},
			statusCode:   http.StatusUnprocessableEntity,
			responseBody: `{"error": "invalid email format"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}

				expectedPath := "/api/v1/accounts/1/agents/bulk_create"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Agents().BulkCreate(context.Background(), tt.emails)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && len(result) != tt.expectedLen {
				t.Errorf("Expected %d agents, got %d", tt.expectedLen, len(result))
			}
		})
	}
}

func TestFindAgentByNameOrEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		_, _ = w.Write([]byte(`[
			{"id": 1, "name": "Alice Johnson", "email": "alice@example.com", "role": "agent"},
			{"id": 2, "name": "Alice Cooper", "email": "alice.cooper@example.com", "role": "agent"},
			{"id": 3, "name": "Bob Smith", "email": "bob@example.com", "role": "administrator"}
		]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)

	t.Run("exact email match", func(t *testing.T) {
		agent, err := client.Agents().Find(context.Background(), "alice@example.com")
		if err != nil {
			t.Fatalf("Find exact email error: %v", err)
		}
		if agent.ID != 1 {
			t.Fatalf("Find exact email ID = %d, want 1", agent.ID)
		}
	})

	t.Run("single prefix match", func(t *testing.T) {
		agent, err := client.Agents().Find(context.Background(), "bob")
		if err != nil {
			t.Fatalf("Find prefix error: %v", err)
		}
		if agent.ID != 3 {
			t.Fatalf("Find prefix ID = %d, want 3", agent.ID)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		_, err := client.Agents().Find(context.Background(), "nobody")
		if err == nil {
			t.Fatal("expected not-found error, got nil")
		}
		if apiErr, ok := err.(*APIError); !ok || apiErr.StatusCode != 404 {
			t.Fatalf("expected APIError 404, got %T (%v)", err, err)
		}
	})

	t.Run("ambiguous matches", func(t *testing.T) {
		_, err := client.Agents().Find(context.Background(), "alice")
		if err == nil {
			t.Fatal("expected ambiguous error, got nil")
		}
		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("expected APIError, got %T (%v)", err, err)
		}
		if apiErr.StatusCode != 400 {
			t.Fatalf("ambiguous status code = %d, want 400", apiErr.StatusCode)
		}
		if apiErr.Body == "" {
			t.Fatal("expected ambiguous error body to include suggestions")
		}
	})
}
