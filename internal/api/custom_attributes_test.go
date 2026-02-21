package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTranslateModelToAPIValue(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected int
	}{
		{
			name:     "contact",
			model:    "contact",
			expected: 1,
		},
		{
			name:     "contact_attribute",
			model:    "contact_attribute",
			expected: 1,
		},
		{
			name:     "conversation",
			model:    "conversation",
			expected: 0,
		},
		{
			name:     "conversation_attribute",
			model:    "conversation_attribute",
			expected: 0,
		},
		{
			name:     "invalid",
			model:    "invalid",
			expected: -1,
		},
		{
			name:     "empty string",
			model:    "",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translateModelToAPIValue(tt.model)
			if result != tt.expected {
				t.Errorf("translateModelToAPIValue(%q) = %d, want %d", tt.model, result, tt.expected)
			}
		})
	}
}

func TestTranslateModelToQueryParam(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{
			name:     "contact",
			model:    "contact",
			expected: "1",
		},
		{
			name:     "contact_attribute",
			model:    "contact_attribute",
			expected: "1",
		},
		{
			name:     "conversation",
			model:    "conversation",
			expected: "0",
		},
		{
			name:     "conversation_attribute",
			model:    "conversation_attribute",
			expected: "0",
		},
		{
			name:     "invalid passes through",
			model:    "invalid",
			expected: "invalid",
		},
		{
			name:     "empty string passes through",
			model:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translateModelToQueryParam(tt.model)
			if result != tt.expected {
				t.Errorf("translateModelToQueryParam(%q) = %q, want %q", tt.model, result, tt.expected)
			}
		})
	}
}

func TestListCustomAttributes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/custom_attribute_definitions" {
			t.Errorf("Expected path /api/v1/accounts/1/custom_attribute_definitions, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 1, "attribute_display_name": "Company Size", "attribute_key": "company_size", "attribute_model": "contact_attribute", "attribute_display_type": "text"},
			{"id": 2, "attribute_display_name": "Priority", "attribute_key": "priority", "attribute_model": "conversation_attribute", "attribute_display_type": "list"}
		]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CustomAttributes().List(context.Background(), "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 custom attributes, got %d", len(result))
	}
	if result[0].AttributeDisplayName != "Company Size" {
		t.Errorf("Expected display name 'Company Size', got %s", result[0].AttributeDisplayName)
	}
	if result[0].AttributeKey != "company_size" {
		t.Errorf("Expected key 'company_size', got %s", result[0].AttributeKey)
	}
	if result[1].AttributeDisplayName != "Priority" {
		t.Errorf("Expected display name 'Priority', got %s", result[1].AttributeDisplayName)
	}
}

func TestListCustomAttributes_FilterByContact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		expectedPath := "/api/v1/accounts/1/custom_attribute_definitions"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		// Check query param
		if r.URL.Query().Get("attribute_model") != "1" {
			t.Errorf("Expected attribute_model=1, got %s", r.URL.Query().Get("attribute_model"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 1, "attribute_display_name": "Company Size", "attribute_key": "company_size", "attribute_model": "contact_attribute", "attribute_display_type": "text"}
		]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CustomAttributes().List(context.Background(), "contact")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 custom attribute, got %d", len(result))
	}
	if result[0].AttributeModel != "contact_attribute" {
		t.Errorf("Expected model 'contact_attribute', got %s", result[0].AttributeModel)
	}
}

func TestListCustomAttributes_FilterByConversation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		// Check query param
		if r.URL.Query().Get("attribute_model") != "0" {
			t.Errorf("Expected attribute_model=0, got %s", r.URL.Query().Get("attribute_model"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 2, "attribute_display_name": "Priority", "attribute_key": "priority", "attribute_model": "conversation_attribute", "attribute_display_type": "list"}
		]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CustomAttributes().List(context.Background(), "conversation")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 custom attribute, got %d", len(result))
	}
	if result[0].AttributeModel != "conversation_attribute" {
		t.Errorf("Expected model 'conversation_attribute', got %s", result[0].AttributeModel)
	}
}

func TestListCustomAttributes_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CustomAttributes().List(context.Background(), "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected 0 custom attributes, got %d", len(result))
	}
}

func TestListCustomAttributes_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "Unauthorized"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "invalid-token", 1)
	result, err := client.CustomAttributes().List(context.Background(), "")

	if err == nil {
		t.Error("Expected error for unauthorized request, got nil")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %+v", result)
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected *APIError, got %T", err)
	} else if apiErr.StatusCode != 401 {
		t.Errorf("Expected status code 401, got %d", apiErr.StatusCode)
	}
}

func TestGetCustomAttribute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/custom_attribute_definitions/1" {
			t.Errorf("Expected path /api/v1/accounts/1/custom_attribute_definitions/1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1, "attribute_display_name": "Company Size", "attribute_key": "company_size", "attribute_model": "contact_attribute", "attribute_display_type": "text"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CustomAttributes().Get(context.Background(), 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}
	if result.AttributeDisplayName != "Company Size" {
		t.Errorf("Expected display name 'Company Size', got %s", result.AttributeDisplayName)
	}
	if result.AttributeKey != "company_size" {
		t.Errorf("Expected key 'company_size', got %s", result.AttributeKey)
	}
	if result.AttributeModel != "contact_attribute" {
		t.Errorf("Expected model 'contact_attribute', got %s", result.AttributeModel)
	}
	if result.AttributeDisplayType != "text" {
		t.Errorf("Expected display type 'text', got %s", result.AttributeDisplayType)
	}
}

func TestGetCustomAttribute_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Custom attribute not found"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CustomAttributes().Get(context.Background(), 999)

	if err == nil {
		t.Error("Expected error for non-existent custom attribute, got nil")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %+v", result)
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected *APIError, got %T", err)
	} else if apiErr.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", apiErr.StatusCode)
	}
}

func TestCreateCustomAttribute_Contact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/custom_attribute_definitions" {
			t.Errorf("Expected path /api/v1/accounts/1/custom_attribute_definitions, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body["attribute_display_name"] != "Company Size" {
			t.Errorf("Expected attribute_display_name 'Company Size', got %v", body["attribute_display_name"])
		}
		if body["attribute_key"] != "company_size" {
			t.Errorf("Expected attribute_key 'company_size', got %v", body["attribute_key"])
		}
		// Verify model is translated to integer 1 for contact
		if body["attribute_model"] != float64(1) {
			t.Errorf("Expected attribute_model 1 (contact), got %v", body["attribute_model"])
		}
		if body["attribute_display_type"] != "text" {
			t.Errorf("Expected attribute_display_type 'text', got %v", body["attribute_display_type"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1, "attribute_display_name": "Company Size", "attribute_key": "company_size", "attribute_model": "contact_attribute", "attribute_display_type": "text"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CustomAttributes().Create(context.Background(), "Company Size", "company_size", "contact", "text")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}
	if result.AttributeDisplayName != "Company Size" {
		t.Errorf("Expected display name 'Company Size', got %s", result.AttributeDisplayName)
	}
	if result.AttributeKey != "company_size" {
		t.Errorf("Expected key 'company_size', got %s", result.AttributeKey)
	}
}

func TestCreateCustomAttribute_Conversation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body["attribute_display_name"] != "Priority" {
			t.Errorf("Expected attribute_display_name 'Priority', got %v", body["attribute_display_name"])
		}
		if body["attribute_key"] != "priority" {
			t.Errorf("Expected attribute_key 'priority', got %v", body["attribute_key"])
		}
		// Verify model is translated to integer 0 for conversation
		if body["attribute_model"] != float64(0) {
			t.Errorf("Expected attribute_model 0 (conversation), got %v", body["attribute_model"])
		}
		if body["attribute_display_type"] != "list" {
			t.Errorf("Expected attribute_display_type 'list', got %v", body["attribute_display_type"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 2, "attribute_display_name": "Priority", "attribute_key": "priority", "attribute_model": "conversation_attribute", "attribute_display_type": "list"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CustomAttributes().Create(context.Background(), "Priority", "priority", "conversation", "list")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 2 {
		t.Errorf("Expected ID 2, got %d", result.ID)
	}
	if result.AttributeDisplayName != "Priority" {
		t.Errorf("Expected display name 'Priority', got %s", result.AttributeDisplayName)
	}
	if result.AttributeKey != "priority" {
		t.Errorf("Expected key 'priority', got %s", result.AttributeKey)
	}
	if result.AttributeDisplayType != "list" {
		t.Errorf("Expected display type 'list', got %s", result.AttributeDisplayType)
	}
}

func TestUpdateCustomAttribute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/custom_attribute_definitions/1" {
			t.Errorf("Expected path /api/v1/accounts/1/custom_attribute_definitions/1, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body["attribute_display_name"] != "Updated Company Size" {
			t.Errorf("Expected attribute_display_name 'Updated Company Size', got %v", body["attribute_display_name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1, "attribute_display_name": "Updated Company Size", "attribute_key": "company_size", "attribute_model": "contact_attribute", "attribute_display_type": "text"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CustomAttributes().Update(context.Background(), 1, "Updated Company Size")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}
	if result.AttributeDisplayName != "Updated Company Size" {
		t.Errorf("Expected display name 'Updated Company Size', got %s", result.AttributeDisplayName)
	}
}

func TestUpdateCustomAttribute_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Custom attribute not found"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CustomAttributes().Update(context.Background(), 999, "Updated Name")

	if err == nil {
		t.Error("Expected error for non-existent custom attribute, got nil")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %+v", result)
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected *APIError, got %T", err)
	} else if apiErr.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", apiErr.StatusCode)
	}
}

func TestDeleteCustomAttribute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/custom_attribute_definitions/1" {
			t.Errorf("Expected path /api/v1/accounts/1/custom_attribute_definitions/1, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.CustomAttributes().Delete(context.Background(), 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDeleteCustomAttribute_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Custom attribute not found"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.CustomAttributes().Delete(context.Background(), 999)

	if err == nil {
		t.Error("Expected error for non-existent custom attribute, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected *APIError, got %T", err)
	} else if apiErr.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", apiErr.StatusCode)
	}
}
