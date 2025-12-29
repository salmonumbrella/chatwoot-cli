package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListContacts(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *ContactList)
	}{
		{
			name: "successful list",
			page: 1,
			responseBody: `{
				"payload": [
					{"id": 1, "name": "John Doe", "email": "john@example.com", "created_at": 1700000000},
					{"id": 2, "name": "Jane Doe", "email": "jane@example.com", "created_at": 1700001000}
				],
				"meta": {"current_page": 1, "total_pages": 1, "total_count": 2}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *ContactList) {
				if len(result.Payload) != 2 {
					t.Errorf("Expected 2 contacts, got %d", len(result.Payload))
				}
				if result.Payload[0].Name != "John Doe" {
					t.Errorf("Expected name 'John Doe', got %s", result.Payload[0].Name)
				}
			},
		},
		{
			name:         "empty list",
			page:         1,
			responseBody: `{"payload": [], "meta": {"current_page": 1, "total_pages": 0}}`,
			expectError:  false,
			validateFunc: func(t *testing.T, result *ContactList) {
				if len(result.Payload) != 0 {
					t.Errorf("Expected 0 contacts, got %d", len(result.Payload))
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
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.ListContacts(context.Background(), ListContactsParams{Page: tt.page})

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

func TestGetContact(t *testing.T) {
	tests := []struct {
		name         string
		contactID    int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Contact)
	}{
		{
			name:       "successful get",
			contactID:  123,
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": {
					"id": 123,
					"name": "John Doe",
					"email": "john@example.com",
					"phone_number": "+1234567890",
					"created_at": 1700000000
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, contact *Contact) {
				if contact.ID != 123 {
					t.Errorf("Expected ID 123, got %d", contact.ID)
				}
				if contact.Email != "john@example.com" {
					t.Errorf("Expected email john@example.com, got %s", contact.Email)
				}
			},
		},
		{
			name:         "not found",
			contactID:    999,
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
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
			result, err := client.GetContact(context.Background(), tt.contactID)

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

func TestCreateContact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"payload": {
				"contact": {
					"id": 1,
					"name": "New Contact",
					"email": "new@example.com",
					"created_at": 1700000000
				},
				"contact_inbox": {}
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CreateContact(context.Background(), "New Contact", "new@example.com", "")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Name != "New Contact" {
		t.Errorf("Expected name 'New Contact', got %s", result.Name)
	}
}

func TestSearchContacts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("q") == "" {
			t.Error("Expected q query parameter")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"payload": [{"id": 1, "name": "John", "created_at": 1700000000}],
			"meta": {"current_page": 1}
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.SearchContacts(context.Background(), "john")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result.Payload) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(result.Payload))
	}
}
