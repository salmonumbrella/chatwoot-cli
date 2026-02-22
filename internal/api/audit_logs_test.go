package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAuditLogs(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *AuditLogList)
		validatePath func(*testing.T, string)
	}{
		{
			name:       "successful list with results",
			page:       1,
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [
					{
						"id": 1,
						"action": "create",
						"auditable_type": "Conversation",
						"auditable_id": 100,
						"user_id": 5,
						"username": "admin@example.com",
						"audited_changes": {"status": ["open", "resolved"]},
						"created_at": "2024-01-15T10:30:00Z"
					},
					{
						"id": 2,
						"action": "update",
						"auditable_type": "Contact",
						"auditable_id": 50,
						"user_id": 5,
						"username": "admin@example.com",
						"audited_changes": {"name": ["Old Name", "New Name"]},
						"created_at": "2024-01-15T11:00:00Z"
					}
				],
				"meta": {"current_page": 1, "total_pages": 3, "total_count": 25}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *AuditLogList) {
				if len(result.Payload) != 2 {
					t.Errorf("Expected 2 audit logs, got %d", len(result.Payload))
				}
				if result.Payload[0].Action != "create" {
					t.Errorf("Expected action 'create', got %s", result.Payload[0].Action)
				}
				if result.Payload[0].AuditableType != "Conversation" {
					t.Errorf("Expected auditable_type 'Conversation', got %s", result.Payload[0].AuditableType)
				}
				if result.Payload[0].UserID != 5 {
					t.Errorf("Expected user_id 5, got %d", result.Payload[0].UserID)
				}
				if result.Payload[0].Username != "admin@example.com" {
					t.Errorf("Expected username 'admin@example.com', got %s", result.Payload[0].Username)
				}
				if int(result.Meta.TotalPages) != 3 {
					t.Errorf("Expected 3 total pages, got %d", result.Meta.TotalPages)
				}
			},
			validatePath: func(t *testing.T, path string) {
				expected := "/api/v1/accounts/1/audit_logs?page=1"
				if path != expected {
					t.Errorf("Expected path %s, got %s", expected, path)
				}
			},
		},
		{
			name:       "empty list",
			page:       1,
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [],
				"meta": {"current_page": 1, "total_pages": 0, "total_count": 0}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *AuditLogList) {
				if len(result.Payload) != 0 {
					t.Errorf("Expected 0 audit logs, got %d", len(result.Payload))
				}
			},
		},
		{
			name:       "page zero omits query param",
			page:       0,
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [{"id": 1, "action": "create", "auditable_type": "Conversation", "auditable_id": 1, "user_id": 1, "created_at": "2024-01-15T10:30:00Z"}],
				"meta": {"current_page": 1}
			}`,
			expectError: false,
			validatePath: func(t *testing.T, path string) {
				expected := "/api/v1/accounts/1/audit_logs"
				if path != expected {
					t.Errorf("Expected path %s (no page param), got %s", expected, path)
				}
			},
		},
		{
			name:         "unauthorized",
			page:         1,
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error": "Unauthorized"}`,
			expectError:  true,
		},
		{
			name:         "server error",
			page:         1,
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "Internal server error"}`,
			expectError:  true,
		},
		{
			name:         "forbidden - no access to audit logs",
			page:         1,
			statusCode:   http.StatusForbidden,
			responseBody: `{"error": "You are not authorized to access this resource"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}

				// Verify headers
				if r.Header.Get("api_access_token") == "" {
					t.Error("Missing api_access_token header")
				}

				// Validate path if needed
				if tt.validatePath != nil {
					tt.validatePath(t, r.URL.String())
				}

				// Send response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.AuditLogs().List(context.Background(), tt.page)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// For error cases, verify error type
			if tt.expectError && err != nil {
				apiErr, ok := err.(*APIError)
				if !ok {
					t.Errorf("Expected APIError, got %T", err)
				} else if apiErr.StatusCode != tt.statusCode {
					t.Errorf("Expected status code %d, got %d", tt.statusCode, apiErr.StatusCode)
				}
			}

			// Run custom validation
			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}
