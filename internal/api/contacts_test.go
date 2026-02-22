package api

import (
	"context"
	"encoding/json"
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
			result, err := client.Contacts().List(context.Background(), ListContactsParams{Page: tt.page})

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
			result, err := client.Contacts().Get(context.Background(), tt.contactID)

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
	result, err := client.Contacts().Create(context.Background(), "New Contact", "new@example.com", "")
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
	result, err := client.Contacts().Search(context.Background(), "john", 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result.Payload) != 1 {
		t.Errorf("Expected 1 contact, got %d", len(result.Payload))
	}
}

func TestUpdateContact(t *testing.T) {
	tests := []struct {
		name         string
		contactID    int
		updateName   string
		updateEmail  string
		updatePhone  string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Contact)
	}{
		{
			name:        "successful update",
			contactID:   123,
			updateName:  "Updated Name",
			updateEmail: "updated@example.com",
			updatePhone: "+1987654321",
			statusCode:  http.StatusOK,
			responseBody: `{
				"payload": {
					"id": 123,
					"name": "Updated Name",
					"email": "updated@example.com",
					"phone_number": "+1987654321",
					"created_at": 1700000000
				}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, contact *Contact) {
				if contact.Name != "Updated Name" {
					t.Errorf("Expected name 'Updated Name', got %s", contact.Name)
				}
				if contact.Email != "updated@example.com" {
					t.Errorf("Expected email 'updated@example.com', got %s", contact.Email)
				}
			},
		},
		{
			name:         "partial update - name only",
			contactID:    123,
			updateName:   "Just Name",
			updateEmail:  "",
			updatePhone:  "",
			statusCode:   http.StatusOK,
			responseBody: `{"payload": {"id": 123, "name": "Just Name", "created_at": 1700000000}}`,
			expectError:  false,
			validateFunc: func(t *testing.T, contact *Contact) {
				if contact.Name != "Just Name" {
					t.Errorf("Expected name 'Just Name', got %s", contact.Name)
				}
			},
		},
		{
			name:         "contact not found",
			contactID:    999,
			updateName:   "Name",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "Contact not found"}`,
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
			result, err := client.Contacts().Update(context.Background(), tt.contactID, tt.updateName, tt.updateEmail, tt.updatePhone)

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

func TestUpdateContactWithOpts(t *testing.T) {
	tests := []struct {
		name         string
		opts         UpdateContactOpts
		validateBody func(*testing.T, map[string]any)
	}{
		{
			name: "company and country",
			opts: UpdateContactOpts{
				Company: "Acme Corp",
				Country: "Canada",
			},
			validateBody: func(t *testing.T, body map[string]any) {
				additional, ok := body["additional_attributes"].(map[string]any)
				if !ok {
					t.Fatal("expected additional_attributes in request body")
				}
				if additional["company_name"] != "Acme Corp" {
					t.Errorf("expected company_name 'Acme Corp', got %v", additional["company_name"])
				}
				if additional["country"] != "Canada" {
					t.Errorf("expected country 'Canada', got %v", additional["country"])
				}
			},
		},
		{
			name: "custom attributes",
			opts: UpdateContactOpts{
				CustomAttributes: map[string]any{
					"plan":   "enterprise",
					"region": "APAC",
				},
			},
			validateBody: func(t *testing.T, body map[string]any) {
				customAttrs, ok := body["custom_attributes"].(map[string]any)
				if !ok {
					t.Fatal("expected custom_attributes in request body")
				}
				if customAttrs["plan"] != "enterprise" {
					t.Errorf("expected plan 'enterprise', got %v", customAttrs["plan"])
				}
				if customAttrs["region"] != "APAC" {
					t.Errorf("expected region 'APAC', got %v", customAttrs["region"])
				}
			},
		},
		{
			name: "social profiles",
			opts: UpdateContactOpts{
				SocialProfiles: map[string]string{
					"twitter":  "https://twitter.com/acme",
					"linkedin": "https://linkedin.com/company/acme",
				},
			},
			validateBody: func(t *testing.T, body map[string]any) {
				additional, ok := body["additional_attributes"].(map[string]any)
				if !ok {
					t.Fatal("expected additional_attributes in request body")
				}
				socialProfiles, ok := additional["social_profiles"].(map[string]any)
				if !ok {
					t.Fatal("expected social_profiles in additional_attributes")
				}
				if socialProfiles["twitter"] != "https://twitter.com/acme" {
					t.Errorf("expected twitter URL, got %v", socialProfiles["twitter"])
				}
				if socialProfiles["linkedin"] != "https://linkedin.com/company/acme" {
					t.Errorf("expected linkedin URL, got %v", socialProfiles["linkedin"])
				}
			},
		},
		{
			name: "mixed name company and custom attr",
			opts: UpdateContactOpts{
				Name:    "Alice Smith",
				Company: "Acme Corp",
				CustomAttributes: map[string]any{
					"tier": "gold",
				},
			},
			validateBody: func(t *testing.T, body map[string]any) {
				if body["name"] != "Alice Smith" {
					t.Errorf("expected name 'Alice Smith', got %v", body["name"])
				}
				additional, ok := body["additional_attributes"].(map[string]any)
				if !ok {
					t.Fatal("expected additional_attributes in request body")
				}
				if additional["company_name"] != "Acme Corp" {
					t.Errorf("expected company_name 'Acme Corp', got %v", additional["company_name"])
				}
				customAttrs, ok := body["custom_attributes"].(map[string]any)
				if !ok {
					t.Fatal("expected custom_attributes in request body")
				}
				if customAttrs["tier"] != "gold" {
					t.Errorf("expected tier 'gold', got %v", customAttrs["tier"])
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
				if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"payload": {"id": 123, "name": "Test", "created_at": 1700000000}}`))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			_, err := client.Contacts().UpdateWithOpts(context.Background(), 123, tt.opts)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			tt.validateBody(t, capturedBody)
		})
	}
}

func TestDeleteContact(t *testing.T) {
	tests := []struct {
		name        string
		contactID   int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			contactID:   123,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "contact not found",
			contactID:   999,
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
		{
			name:        "unauthorized",
			contactID:   123,
			statusCode:  http.StatusUnauthorized,
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
			err := client.Contacts().Delete(context.Background(), tt.contactID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestFilterContacts(t *testing.T) {
	tests := []struct {
		name         string
		payload      map[string]any
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *ContactList)
	}{
		{
			name: "successful filter",
			payload: map[string]any{
				"payload": []map[string]any{
					{"attribute_key": "email", "filter_operator": "contains", "values": []string{"@example.com"}},
				},
			},
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [
					{"id": 1, "name": "John", "email": "john@example.com", "created_at": 1700000000},
					{"id": 2, "name": "Jane", "email": "jane@example.com", "created_at": 1700001000}
				],
				"meta": {"current_page": 1, "total_count": 2}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *ContactList) {
				if len(result.Payload) != 2 {
					t.Errorf("Expected 2 contacts, got %d", len(result.Payload))
				}
			},
		},
		{
			name:         "empty filter results",
			payload:      map[string]any{},
			statusCode:   http.StatusOK,
			responseBody: `{"payload": [], "meta": {"current_page": 1, "total_count": 0}}`,
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
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Contacts().Filter(context.Background(), tt.payload)

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

func TestGetContactConversations(t *testing.T) {
	tests := []struct {
		name         string
		contactID    int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []Conversation)
	}{
		{
			name:       "successful get conversations",
			contactID:  123,
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [
					{"id": 1, "status": "open", "inbox_id": 5, "created_at": 1700000000},
					{"id": 2, "status": "resolved", "inbox_id": 5, "created_at": 1700001000}
				]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result []Conversation) {
				if len(result) != 2 {
					t.Errorf("Expected 2 conversations, got %d", len(result))
				}
				if result[0].Status != "open" {
					t.Errorf("Expected status 'open', got %s", result[0].Status)
				}
			},
		},
		{
			name:         "no conversations",
			contactID:    123,
			statusCode:   http.StatusOK,
			responseBody: `{"payload": []}`,
			expectError:  false,
			validateFunc: func(t *testing.T, result []Conversation) {
				if len(result) != 0 {
					t.Errorf("Expected 0 conversations, got %d", len(result))
				}
			},
		},
		{
			name:         "contact not found",
			contactID:    999,
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "Contact not found"}`,
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
			result, err := client.Contacts().Conversations(context.Background(), tt.contactID)

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

func TestGetContactLabels(t *testing.T) {
	tests := []struct {
		name         string
		contactID    int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []string)
	}{
		{
			name:         "successful get labels",
			contactID:    123,
			statusCode:   http.StatusOK,
			responseBody: `{"labels": ["vip", "priority", "enterprise"]}`,
			expectError:  false,
			validateFunc: func(t *testing.T, result []string) {
				if len(result) != 3 {
					t.Errorf("Expected 3 labels, got %d", len(result))
				}
				if result[0] != "vip" {
					t.Errorf("Expected first label 'vip', got %s", result[0])
				}
			},
		},
		{
			name:         "no labels",
			contactID:    123,
			statusCode:   http.StatusOK,
			responseBody: `{"labels": []}`,
			expectError:  false,
			validateFunc: func(t *testing.T, result []string) {
				if len(result) != 0 {
					t.Errorf("Expected 0 labels, got %d", len(result))
				}
			},
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
			result, err := client.Contacts().Labels(context.Background(), tt.contactID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestAddContactLabels(t *testing.T) {
	tests := []struct {
		name         string
		contactID    int
		labels       []string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []string)
	}{
		{
			name:         "successful add labels",
			contactID:    123,
			labels:       []string{"new-label", "another-label"},
			statusCode:   http.StatusOK,
			responseBody: `{"labels": ["existing", "new-label", "another-label"]}`,
			expectError:  false,
			validateFunc: func(t *testing.T, result []string) {
				if len(result) != 3 {
					t.Errorf("Expected 3 labels, got %d", len(result))
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
			result, err := client.Contacts().AddLabels(context.Background(), tt.contactID, tt.labels)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestGetContactableInboxes(t *testing.T) {
	tests := []struct {
		name         string
		contactID    int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []ContactInbox)
	}{
		{
			name:       "successful get inboxes (contact inbox shape)",
			contactID:  123,
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [
					{"source_id": "src-1", "inbox": {"id": 1, "name": "Email Inbox", "channel_type": "Channel::Email"}},
					{"source_id": "src-2", "inbox": {"id": 2, "name": "Website", "channel_type": "Channel::WebWidget"}}
				]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result []ContactInbox) {
				if len(result) != 2 {
					t.Errorf("Expected 2 inboxes, got %d", len(result))
				}
				if result[0].SourceID != "src-1" {
					t.Errorf("Expected source_id 'src-1', got %s", result[0].SourceID)
				}
				if result[0].Inbox.Name != "Email Inbox" {
					t.Errorf("Expected name 'Email Inbox', got %s", result[0].Inbox.Name)
				}
			},
		},
		{
			name:       "successful get inboxes (legacy plain inbox shape)",
			contactID:  123,
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [
					{"id": 1, "name": "Email Inbox", "channel_type": "Channel::Email"}
				]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result []ContactInbox) {
				if len(result) != 1 {
					t.Errorf("Expected 1 inbox, got %d", len(result))
				}
				if result[0].Inbox.ID != 1 {
					t.Errorf("Expected inbox id 1, got %d", result[0].Inbox.ID)
				}
			},
		},
		{
			name:         "no contactable inboxes",
			contactID:    123,
			statusCode:   http.StatusOK,
			responseBody: `{"payload": []}`,
			expectError:  false,
			validateFunc: func(t *testing.T, result []ContactInbox) {
				if len(result) != 0 {
					t.Errorf("Expected 0 inboxes, got %d", len(result))
				}
			},
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
			result, err := client.Contacts().ContactableInboxes(context.Background(), tt.contactID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestCreateContactInbox(t *testing.T) {
	tests := []struct {
		name         string
		contactID    int
		inboxID      int
		sourceID     string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *ContactInbox)
	}{
		{
			name:       "successful create with source_id",
			contactID:  123,
			inboxID:    5,
			sourceID:   "custom-source-123",
			statusCode: http.StatusOK,
			responseBody: `{
				"source_id": "custom-source-123",
				"inbox": {"id": 5, "name": "Website", "channel_type": "Channel::WebWidget"}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *ContactInbox) {
				if result.SourceID != "custom-source-123" {
					t.Errorf("Expected source_id 'custom-source-123', got %s", result.SourceID)
				}
				if result.Inbox.ID != 5 {
					t.Errorf("Expected inbox ID 5, got %d", result.Inbox.ID)
				}
			},
		},
		{
			name:       "successful create without source_id",
			contactID:  123,
			inboxID:    5,
			sourceID:   "",
			statusCode: http.StatusOK,
			responseBody: `{
				"source_id": "auto-generated-id",
				"inbox": {"id": 5, "name": "Website", "channel_type": "Channel::WebWidget"}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *ContactInbox) {
				if result.SourceID != "auto-generated-id" {
					t.Errorf("Expected source_id 'auto-generated-id', got %s", result.SourceID)
				}
			},
		},
		{
			name:         "error - inbox not found",
			contactID:    123,
			inboxID:      999,
			sourceID:     "",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "Inbox not found"}`,
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
			result, err := client.Contacts().CreateInbox(context.Background(), tt.contactID, tt.inboxID, tt.sourceID)

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

func TestGetContactNotes(t *testing.T) {
	tests := []struct {
		name         string
		contactID    int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []ContactNote)
	}{
		{
			name:       "successful get notes",
			contactID:  123,
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "content": "First note", "contact_id": 123, "user_id": 5, "created_at": "2024-01-15T10:00:00Z"},
				{"id": 2, "content": "Second note", "contact_id": 123, "user_id": 5, "created_at": "2024-01-15T11:00:00Z"}
			]`,
			expectError: false,
			validateFunc: func(t *testing.T, result []ContactNote) {
				if len(result) != 2 {
					t.Errorf("Expected 2 notes, got %d", len(result))
				}
				if result[0].Content != "First note" {
					t.Errorf("Expected content 'First note', got %s", result[0].Content)
				}
			},
		},
		{
			name:         "no notes",
			contactID:    123,
			statusCode:   http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			validateFunc: func(t *testing.T, result []ContactNote) {
				if len(result) != 0 {
					t.Errorf("Expected 0 notes, got %d", len(result))
				}
			},
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
			result, err := client.Contacts().Notes(context.Background(), tt.contactID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestCreateContactNote(t *testing.T) {
	tests := []struct {
		name         string
		contactID    int
		content      string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *ContactNote)
	}{
		{
			name:       "successful create note",
			contactID:  123,
			content:    "This is a new note",
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"content": "This is a new note",
				"contact_id": 123,
				"user_id": 5,
				"created_at": "2024-01-15T10:00:00Z"
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, result *ContactNote) {
				if result.ID != 1 {
					t.Errorf("Expected ID 1, got %d", result.ID)
				}
				if result.Content != "This is a new note" {
					t.Errorf("Expected content 'This is a new note', got %s", result.Content)
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
			result, err := client.Contacts().CreateNote(context.Background(), tt.contactID, tt.content)

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

func TestDeleteContactNote(t *testing.T) {
	tests := []struct {
		name        string
		contactID   int
		noteID      int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			contactID:   123,
			noteID:      1,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "note not found",
			contactID:   123,
			noteID:      999,
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
			err := client.Contacts().DeleteNote(context.Background(), tt.contactID, tt.noteID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestMergeContacts(t *testing.T) {
	tests := []struct {
		name            string
		baseContactID   int
		mergeeContactID int
		statusCode      int
		responseBody    string
		expectError     bool
		validateFunc    func(*testing.T, *Contact)
	}{
		{
			name:            "successful merge",
			baseContactID:   1,
			mergeeContactID: 2,
			statusCode:      http.StatusOK,
			responseBody: `{
				"id": 1,
				"name": "John Doe",
				"email": "john@example.com",
				"phone_number": "+1234567890",
				"created_at": 1700000000
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, contact *Contact) {
				if contact.ID != 1 {
					t.Errorf("Expected ID 1, got %d", contact.ID)
				}
				if contact.Name != "John Doe" {
					t.Errorf("Expected name 'John Doe', got %s", contact.Name)
				}
				if contact.Email != "john@example.com" {
					t.Errorf("Expected email 'john@example.com', got %s", contact.Email)
				}
			},
		},
		{
			name:            "contact not found",
			baseContactID:   999,
			mergeeContactID: 1000,
			statusCode:      http.StatusNotFound,
			responseBody:    `{"error": "Contact not found"}`,
			expectError:     true,
		},
		{
			name:            "same contact merge error",
			baseContactID:   123,
			mergeeContactID: 123,
			statusCode:      http.StatusUnprocessableEntity,
			responseBody:    `{"error": "Cannot merge contact with itself"}`,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify HTTP method
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}

				// Verify endpoint path
				expectedPath := "/api/v1/accounts/1/actions/contact_merge"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				// Verify request body structure
				var body map[string]int
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}

				if body["base_contact_id"] != tt.baseContactID {
					t.Errorf("Expected base_contact_id %d, got %d", tt.baseContactID, body["base_contact_id"])
				}
				if body["mergee_contact_id"] != tt.mergeeContactID {
					t.Errorf("Expected mergee_contact_id %d, got %d", tt.mergeeContactID, body["mergee_contact_id"])
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.Contacts().Merge(context.Background(), tt.baseContactID, tt.mergeeContactID)

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
