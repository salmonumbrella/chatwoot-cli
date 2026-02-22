package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListCustomFilters(t *testing.T) {
	tests := []struct {
		name         string
		filterType   string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []CustomFilter)
		validateURL  func(*testing.T, string)
	}{
		{
			name:       "successful list all",
			filterType: "",
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "name": "Open Conversations", "filter_type": "conversation", "query": {"status": "open"}},
				{"id": 2, "name": "VIP Contacts", "filter_type": "contact", "query": {"labels": ["vip"]}}
			]`,
			expectError: false,
			validateFunc: func(t *testing.T, filters []CustomFilter) {
				if len(filters) != 2 {
					t.Errorf("Expected 2 filters, got %d", len(filters))
				}
				if filters[0].Name != "Open Conversations" {
					t.Errorf("Expected name 'Open Conversations', got %s", filters[0].Name)
				}
				if filters[0].FilterType != "conversation" {
					t.Errorf("Expected filter_type 'conversation', got %s", filters[0].FilterType)
				}
			},
			validateURL: func(t *testing.T, url string) {
				if strings.Contains(url, "filter_type=") {
					t.Error("Expected no filter_type query param when empty")
				}
			},
		},
		{
			name:       "list by filter type",
			filterType: "conversation",
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "name": "Open Conversations", "filter_type": "conversation", "query": {"status": "open"}}
			]`,
			expectError: false,
			validateFunc: func(t *testing.T, filters []CustomFilter) {
				if len(filters) != 1 {
					t.Errorf("Expected 1 filter, got %d", len(filters))
				}
			},
			validateURL: func(t *testing.T, url string) {
				if !strings.Contains(url, "filter_type=conversation") {
					t.Errorf("Expected filter_type query param, got %s", url)
				}
			},
		},
		{
			name:         "empty list",
			filterType:   "",
			statusCode:   http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			validateFunc: func(t *testing.T, filters []CustomFilter) {
				if len(filters) != 0 {
					t.Errorf("Expected 0 filters, got %d", len(filters))
				}
			},
		},
		{
			name:         "server error",
			filterType:   "",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "internal error"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				capturedURL = r.URL.String()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.CustomFilters().List(context.Background(), tt.filterType)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.validateFunc != nil && result != nil {
				tt.validateFunc(t, result)
			}
			if tt.validateURL != nil {
				tt.validateURL(t, capturedURL)
			}
		})
	}
}

func TestGetCustomFilter(t *testing.T) {
	tests := []struct {
		name         string
		filterID     int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *CustomFilter)
	}{
		{
			name:         "successful get",
			filterID:     1,
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Open Conversations", "filter_type": "conversation", "query": {"status": "open", "labels": ["support"]}}`,
			expectError:  false,
			validateFunc: func(t *testing.T, filter *CustomFilter) {
				if filter.ID != 1 {
					t.Errorf("Expected ID 1, got %d", filter.ID)
				}
				if filter.Name != "Open Conversations" {
					t.Errorf("Expected name 'Open Conversations', got %s", filter.Name)
				}
				if filter.Query == nil {
					t.Error("Expected query to be non-nil")
				}
				if filter.Query["status"] != "open" {
					t.Errorf("Expected query.status 'open', got %v", filter.Query["status"])
				}
			},
		},
		{
			name:         "not found",
			filterID:     999,
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
			result, err := client.CustomFilters().Get(context.Background(), tt.filterID)

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

func TestCreateCustomFilter(t *testing.T) {
	tests := []struct {
		name         string
		filterName   string
		filterType   string
		query        map[string]any
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *CustomFilter, map[string]any)
	}{
		{
			name:       "successful create",
			filterName: "New Filter",
			filterType: "conversation",
			query:      map[string]any{"status": "pending"},
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"name": "New Filter",
				"filter_type": "conversation",
				"query": {"status": "pending"}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, filter *CustomFilter, body map[string]any) {
				if filter.Name != "New Filter" {
					t.Errorf("Expected name 'New Filter', got %s", filter.Name)
				}
				if body["name"] != "New Filter" {
					t.Errorf("Expected name in body, got %v", body["name"])
				}
				if body["filter_type"] != "conversation" {
					t.Errorf("Expected filter_type in body, got %v", body["filter_type"])
				}
				query := body["query"].(map[string]any)
				if query["status"] != "pending" {
					t.Errorf("Expected query.status in body, got %v", query["status"])
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
			result, err := client.CustomFilters().Create(context.Background(), tt.filterName, tt.filterType, tt.query)

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

func TestUpdateCustomFilter(t *testing.T) {
	tests := []struct {
		name         string
		filterID     int
		filterName   string
		query        map[string]any
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *CustomFilter, map[string]any)
	}{
		{
			name:       "update all fields",
			filterID:   1,
			filterName: "Updated Filter",
			query:      map[string]any{"status": "resolved"},
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"name": "Updated Filter",
				"filter_type": "conversation",
				"query": {"status": "resolved"}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, filter *CustomFilter, body map[string]any) {
				if filter.Name != "Updated Filter" {
					t.Errorf("Expected name 'Updated Filter', got %s", filter.Name)
				}
				if body["name"] != "Updated Filter" {
					t.Errorf("Expected name in body, got %v", body["name"])
				}
			},
		},
		{
			name:         "partial update - name only",
			filterID:     1,
			filterName:   "Only Name",
			query:        nil,
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Only Name", "filter_type": "conversation", "query": {"status": "open"}}`,
			expectError:  false,
			validateFunc: func(t *testing.T, filter *CustomFilter, body map[string]any) {
				if _, ok := body["query"]; ok {
					t.Error("Expected no query in body when nil")
				}
			},
		},
		{
			name:         "partial update - query only",
			filterID:     1,
			filterName:   "",
			query:        map[string]any{"priority": "high"},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Existing", "filter_type": "conversation", "query": {"priority": "high"}}`,
			expectError:  false,
			validateFunc: func(t *testing.T, filter *CustomFilter, body map[string]any) {
				if _, ok := body["name"]; ok {
					t.Error("Expected no name in body when empty")
				}
			},
		},
		{
			name:         "not found",
			filterID:     999,
			filterName:   "Test",
			query:        nil,
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
			result, err := client.CustomFilters().Update(context.Background(), tt.filterID, tt.filterName, tt.query)

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

func TestDeleteCustomFilter(t *testing.T) {
	tests := []struct {
		name        string
		filterID    int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			filterID:    1,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			filterID:    999,
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
			err := client.CustomFilters().Delete(context.Background(), tt.filterID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
