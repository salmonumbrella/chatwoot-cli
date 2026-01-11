package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListLabels(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []Label)
	}{
		{
			name:       "successful list",
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [
					{"id": 1, "title": "Bug", "description": "Bug reports", "color": "#FF0000", "show_on_sidebar": true},
					{"id": 2, "title": "Feature", "description": "Feature requests", "color": "#00FF00", "show_on_sidebar": false}
				]
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, labels []Label) {
				if len(labels) != 2 {
					t.Errorf("Expected 2 labels, got %d", len(labels))
				}
				if labels[0].Title != "Bug" {
					t.Errorf("Expected title 'Bug', got %s", labels[0].Title)
				}
				if labels[0].Color != "#FF0000" {
					t.Errorf("Expected color '#FF0000', got %s", labels[0].Color)
				}
				if !labels[0].ShowOnSidebar {
					t.Error("Expected ShowOnSidebar to be true")
				}
			},
		},
		{
			name:         "empty list",
			statusCode:   http.StatusOK,
			responseBody: `{"payload": []}`,
			expectError:  false,
			validateFunc: func(t *testing.T, labels []Label) {
				if len(labels) != 0 {
					t.Errorf("Expected 0 labels, got %d", len(labels))
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
			result, err := client.Labels().List(context.Background())

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

func TestGetLabel(t *testing.T) {
	tests := []struct {
		name         string
		labelID      int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Label)
	}{
		{
			name:       "successful get",
			labelID:    1,
			statusCode: http.StatusOK,
			responseBody: `{
				"id": 1,
				"title": "Bug",
				"description": "Bug reports",
				"color": "#FF0000",
				"show_on_sidebar": true
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, label *Label) {
				if label.ID != 1 {
					t.Errorf("Expected ID 1, got %d", label.ID)
				}
				if label.Title != "Bug" {
					t.Errorf("Expected title 'Bug', got %s", label.Title)
				}
			},
		},
		{
			name:         "not found",
			labelID:      999,
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
			result, err := client.Labels().Get(context.Background(), tt.labelID)

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

func TestCreateLabel(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		description   string
		color         string
		showOnSidebar bool
		statusCode    int
		responseBody  string
		expectError   bool
		validateFunc  func(*testing.T, *Label, map[string]any)
	}{
		{
			name:          "create with all fields",
			title:         "New Label",
			description:   "A new label",
			color:         "#00FF00",
			showOnSidebar: true,
			statusCode:    http.StatusOK,
			responseBody: `{
				"id": 3,
				"title": "New Label",
				"description": "A new label",
				"color": "#00FF00",
				"show_on_sidebar": true
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, label *Label, body map[string]any) {
				if label.ID != 3 {
					t.Errorf("Expected ID 3, got %d", label.ID)
				}
				if body["title"] != "New Label" {
					t.Errorf("Expected title 'New Label' in body, got %v", body["title"])
				}
				if body["description"] != "A new label" {
					t.Errorf("Expected description in body, got %v", body["description"])
				}
				if body["color"] != "#00FF00" {
					t.Errorf("Expected color in body, got %v", body["color"])
				}
			},
		},
		{
			name:          "create with minimal fields",
			title:         "Minimal",
			description:   "",
			color:         "",
			showOnSidebar: false,
			statusCode:    http.StatusOK,
			responseBody:  `{"id": 4, "title": "Minimal", "show_on_sidebar": false}`,
			expectError:   false,
			validateFunc: func(t *testing.T, label *Label, body map[string]any) {
				if _, ok := body["description"]; ok {
					t.Error("Expected no description in body when empty")
				}
				if _, ok := body["color"]; ok {
					t.Error("Expected no color in body when empty")
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
			result, err := client.Labels().Create(context.Background(), tt.title, tt.description, tt.color, tt.showOnSidebar)

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

func TestUpdateLabel(t *testing.T) {
	tests := []struct {
		name          string
		labelID       int
		title         string
		description   string
		color         string
		showOnSidebar *bool
		statusCode    int
		responseBody  string
		expectError   bool
		validateFunc  func(*testing.T, *Label, map[string]any)
	}{
		{
			name:          "update all fields",
			labelID:       1,
			title:         "Updated",
			description:   "Updated desc",
			color:         "#0000FF",
			showOnSidebar: boolPtr(true),
			statusCode:    http.StatusOK,
			responseBody: `{
				"id": 1,
				"title": "Updated",
				"description": "Updated desc",
				"color": "#0000FF",
				"show_on_sidebar": true
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, label *Label, body map[string]any) {
				if label.Title != "Updated" {
					t.Errorf("Expected title 'Updated', got %s", label.Title)
				}
				if body["title"] != "Updated" {
					t.Errorf("Expected title in body, got %v", body["title"])
				}
				if body["show_on_sidebar"] != true {
					t.Errorf("Expected show_on_sidebar in body, got %v", body["show_on_sidebar"])
				}
			},
		},
		{
			name:          "partial update",
			labelID:       1,
			title:         "Only Title",
			description:   "",
			color:         "",
			showOnSidebar: nil,
			statusCode:    http.StatusOK,
			responseBody:  `{"id": 1, "title": "Only Title"}`,
			expectError:   false,
			validateFunc: func(t *testing.T, label *Label, body map[string]any) {
				if _, ok := body["description"]; ok {
					t.Error("Expected no description in body")
				}
				if _, ok := body["show_on_sidebar"]; ok {
					t.Error("Expected no show_on_sidebar in body")
				}
			},
		},
		{
			name:         "not found",
			labelID:      999,
			title:        "Test",
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
			result, err := client.Labels().Update(context.Background(), tt.labelID, tt.title, tt.description, tt.color, tt.showOnSidebar)

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

func TestDeleteLabel(t *testing.T) {
	tests := []struct {
		name        string
		labelID     int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			labelID:     1,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			labelID:     999,
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
			err := client.Labels().Delete(context.Background(), tt.labelID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// boolPtr is a helper to create a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}
