package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListPortals(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []Portal)
	}{
		{
			name:       "successful list",
			statusCode: http.StatusOK,
			responseBody: `{
				"payload": [
					{"id": 1, "name": "Support", "slug": "support", "account_id": 1},
					{"id": 2, "name": "Docs", "slug": "docs", "account_id": 1}
				],
				"meta": {"current_page": 1, "portals_count": 2}
			}`,
			expectError: false,
			validateFunc: func(t *testing.T, portals []Portal) {
				if len(portals) != 2 {
					t.Errorf("Expected 2 portals, got %d", len(portals))
				}
				if portals[0].Name != "Support" {
					t.Errorf("Expected name 'Support', got %s", portals[0].Name)
				}
				if portals[0].Slug != "support" {
					t.Errorf("Expected slug 'support', got %s", portals[0].Slug)
				}
			},
		},
		{
			name:         "empty list",
			statusCode:   http.StatusOK,
			responseBody: `{"payload": [], "meta": {"current_page": 1, "portals_count": 0}}`,
			expectError:  false,
			validateFunc: func(t *testing.T, portals []Portal) {
				if len(portals) != 0 {
					t.Errorf("Expected 0 portals, got %d", len(portals))
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
			result, err := client.ListPortals(context.Background())

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

func TestGetPortal(t *testing.T) {
	tests := []struct {
		name         string
		portalSlug   string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Portal)
	}{
		{
			name:         "successful get",
			portalSlug:   "support",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Support", "slug": "support", "custom_domain": "help.example.com", "color": "#0052CC"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, portal *Portal) {
				if portal.ID != 1 {
					t.Errorf("Expected ID 1, got %d", portal.ID)
				}
				if portal.CustomDomain != "help.example.com" {
					t.Errorf("Expected custom_domain 'help.example.com', got %s", portal.CustomDomain)
				}
			},
		},
		{
			name:         "not found",
			portalSlug:   "nonexistent",
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
			result, err := client.GetPortal(context.Background(), tt.portalSlug)

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

func TestCreatePortal(t *testing.T) {
	tests := []struct {
		name         string
		portalName   string
		portalSlug   string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Portal, map[string]any)
	}{
		{
			name:         "successful create",
			portalName:   "New Portal",
			portalSlug:   "new-portal",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "New Portal", "slug": "new-portal"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, portal *Portal, body map[string]any) {
				if portal.Name != "New Portal" {
					t.Errorf("Expected name 'New Portal', got %s", portal.Name)
				}
				// Check that the body has portal wrapper
				portalData, ok := body["portal"].(map[string]any)
				if !ok {
					t.Error("Expected portal wrapper in body")
				}
				if portalData["name"] != "New Portal" {
					t.Errorf("Expected name in body, got %v", portalData["name"])
				}
				if portalData["slug"] != "new-portal" {
					t.Errorf("Expected slug in body, got %v", portalData["slug"])
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
			result, err := client.CreatePortal(context.Background(), tt.portalName, tt.portalSlug)

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

func TestUpdatePortal(t *testing.T) {
	tests := []struct {
		name         string
		portalSlug   string
		newName      string
		newSlug      string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Portal, map[string]any)
	}{
		{
			name:         "update name and slug",
			portalSlug:   "support",
			newName:      "Updated Portal",
			newSlug:      "updated-portal",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Updated Portal", "slug": "updated-portal"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, portal *Portal, body map[string]any) {
				if portal.Name != "Updated Portal" {
					t.Errorf("Expected name 'Updated Portal', got %s", portal.Name)
				}
			},
		},
		{
			name:         "partial update - name only",
			portalSlug:   "support",
			newName:      "Updated Name",
			newSlug:      "",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "Updated Name", "slug": "support"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, portal *Portal, body map[string]any) {
				portalData := body["portal"].(map[string]any)
				if _, ok := portalData["slug"]; ok {
					t.Error("Expected no slug in body when empty")
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
			result, err := client.UpdatePortal(context.Background(), tt.portalSlug, tt.newName, tt.newSlug)

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

func TestDeletePortal(t *testing.T) {
	tests := []struct {
		name        string
		portalSlug  string
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			portalSlug:  "support",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			portalSlug:  "nonexistent",
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
			err := client.DeletePortal(context.Background(), tt.portalSlug)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestListPortalArticles(t *testing.T) {
	tests := []struct {
		name         string
		portalSlug   string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []Article)
	}{
		{
			name:       "successful list",
			portalSlug: "support",
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "portal_id": 1, "category_id": 1, "title": "Getting Started", "slug": "getting-started", "status": "published"},
				{"id": 2, "portal_id": 1, "category_id": 1, "title": "FAQ", "slug": "faq", "status": "draft"}
			]`,
			expectError: false,
			validateFunc: func(t *testing.T, articles []Article) {
				if len(articles) != 2 {
					t.Errorf("Expected 2 articles, got %d", len(articles))
				}
				if articles[0].Title != "Getting Started" {
					t.Errorf("Expected title 'Getting Started', got %s", articles[0].Title)
				}
			},
		},
		{
			name:         "empty list",
			portalSlug:   "support",
			statusCode:   http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			validateFunc: func(t *testing.T, articles []Article) {
				if len(articles) != 0 {
					t.Errorf("Expected 0 articles, got %d", len(articles))
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
				if !strings.Contains(r.URL.Path, "/articles") {
					t.Errorf("Expected articles path, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.ListPortalArticles(context.Background(), tt.portalSlug)

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

func TestListPortalCategories(t *testing.T) {
	tests := []struct {
		name         string
		portalSlug   string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, []Category)
	}{
		{
			name:       "successful list",
			portalSlug: "support",
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "portal_id": 1, "name": "General", "slug": "general", "position": 1},
				{"id": 2, "portal_id": 1, "name": "Billing", "slug": "billing", "position": 2}
			]`,
			expectError: false,
			validateFunc: func(t *testing.T, categories []Category) {
				if len(categories) != 2 {
					t.Errorf("Expected 2 categories, got %d", len(categories))
				}
				if categories[0].Name != "General" {
					t.Errorf("Expected name 'General', got %s", categories[0].Name)
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
				if !strings.Contains(r.URL.Path, "/categories") {
					t.Errorf("Expected categories path, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.ListPortalCategories(context.Background(), tt.portalSlug)

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

func TestGetArticle(t *testing.T) {
	tests := []struct {
		name         string
		portalSlug   string
		articleID    int
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Article)
	}{
		{
			name:         "successful get",
			portalSlug:   "support",
			articleID:    1,
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "portal_id": 1, "title": "Getting Started", "content": "Welcome!", "views": 100}`,
			expectError:  false,
			validateFunc: func(t *testing.T, article *Article) {
				if article.ID != 1 {
					t.Errorf("Expected ID 1, got %d", article.ID)
				}
				if article.Views != 100 {
					t.Errorf("Expected views 100, got %d", article.Views)
				}
			},
		},
		{
			name:         "not found",
			portalSlug:   "support",
			articleID:    999,
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
			result, err := client.GetArticle(context.Background(), tt.portalSlug, tt.articleID)

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

func TestCreateArticle(t *testing.T) {
	tests := []struct {
		name         string
		portalSlug   string
		params       map[string]any
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Article)
	}{
		{
			name:       "successful create",
			portalSlug: "support",
			params: map[string]any{
				"title":       "New Article",
				"content":     "Article content",
				"category_id": 1,
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "portal_id": 1, "title": "New Article", "content": "Article content"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, article *Article) {
				if article.Title != "New Article" {
					t.Errorf("Expected title 'New Article', got %s", article.Title)
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
			result, err := client.CreateArticle(context.Background(), tt.portalSlug, tt.params)

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

func TestUpdateArticle(t *testing.T) {
	tests := []struct {
		name         string
		portalSlug   string
		articleID    int
		params       map[string]any
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Article)
	}{
		{
			name:       "successful update",
			portalSlug: "support",
			articleID:  1,
			params: map[string]any{
				"title": "Updated Title",
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "portal_id": 1, "title": "Updated Title"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, article *Article) {
				if article.Title != "Updated Title" {
					t.Errorf("Expected title 'Updated Title', got %s", article.Title)
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
			result, err := client.UpdateArticle(context.Background(), tt.portalSlug, tt.articleID, tt.params)

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

func TestDeleteArticle(t *testing.T) {
	tests := []struct {
		name        string
		portalSlug  string
		articleID   int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			portalSlug:  "support",
			articleID:   1,
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			portalSlug:  "support",
			articleID:   999,
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
			err := client.DeleteArticle(context.Background(), tt.portalSlug, tt.articleID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGetCategory(t *testing.T) {
	tests := []struct {
		name         string
		portalSlug   string
		categorySlug string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Category)
	}{
		{
			name:         "successful get",
			portalSlug:   "support",
			categorySlug: "general",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "portal_id": 1, "name": "General", "slug": "general", "description": "General topics", "position": 1}`,
			expectError:  false,
			validateFunc: func(t *testing.T, category *Category) {
				if category.ID != 1 {
					t.Errorf("Expected ID 1, got %d", category.ID)
				}
				if category.Description != "General topics" {
					t.Errorf("Expected description 'General topics', got %s", category.Description)
				}
			},
		},
		{
			name:         "not found",
			portalSlug:   "support",
			categorySlug: "nonexistent",
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
			result, err := client.GetCategory(context.Background(), tt.portalSlug, tt.categorySlug)

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

func TestCreateCategory(t *testing.T) {
	tests := []struct {
		name         string
		portalSlug   string
		params       map[string]any
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Category)
	}{
		{
			name:       "successful create",
			portalSlug: "support",
			params: map[string]any{
				"name":        "New Category",
				"slug":        "new-category",
				"description": "A new category",
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "portal_id": 1, "name": "New Category", "slug": "new-category"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, category *Category) {
				if category.Name != "New Category" {
					t.Errorf("Expected name 'New Category', got %s", category.Name)
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
			result, err := client.CreateCategory(context.Background(), tt.portalSlug, tt.params)

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

func TestUpdateCategory(t *testing.T) {
	tests := []struct {
		name         string
		portalSlug   string
		categorySlug string
		params       map[string]any
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *Category)
	}{
		{
			name:         "successful update",
			portalSlug:   "support",
			categorySlug: "general",
			params: map[string]any{
				"name": "Updated Category",
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "portal_id": 1, "name": "Updated Category", "slug": "general"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, category *Category) {
				if category.Name != "Updated Category" {
					t.Errorf("Expected name 'Updated Category', got %s", category.Name)
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
			result, err := client.UpdateCategory(context.Background(), tt.portalSlug, tt.categorySlug, tt.params)

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

func TestDeleteCategory(t *testing.T) {
	tests := []struct {
		name         string
		portalSlug   string
		categorySlug string
		statusCode   int
		expectError  bool
	}{
		{
			name:         "successful delete",
			portalSlug:   "support",
			categorySlug: "general",
			statusCode:   http.StatusOK,
			expectError:  false,
		},
		{
			name:         "not found",
			portalSlug:   "support",
			categorySlug: "nonexistent",
			statusCode:   http.StatusNotFound,
			expectError:  true,
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
			err := client.DeleteCategory(context.Background(), tt.portalSlug, tt.categorySlug)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestArchivePortal(t *testing.T) {
	tests := []struct {
		name        string
		portalSlug  string
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful archive",
			portalSlug:  "support",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			portalSlug:  "nonexistent",
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch {
					t.Errorf("Expected PATCH, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/archive") {
					t.Errorf("Expected archive path, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.ArchivePortal(context.Background(), tt.portalSlug)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDeletePortalLogo(t *testing.T) {
	tests := []struct {
		name        string
		portalSlug  string
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete logo",
			portalSlug:  "support",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			portalSlug:  "nonexistent",
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
				if !strings.Contains(r.URL.Path, "/logo") {
					t.Errorf("Expected logo path, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.DeletePortalLogo(context.Background(), tt.portalSlug)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestSendPortalInstructions(t *testing.T) {
	tests := []struct {
		name        string
		portalSlug  string
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful send instructions",
			portalSlug:  "support",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			portalSlug:  "nonexistent",
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/send_instructions") {
					t.Errorf("Expected send_instructions path, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.SendPortalInstructions(context.Background(), tt.portalSlug)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGetPortalSSLStatus(t *testing.T) {
	tests := []struct {
		name         string
		portalSlug   string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, map[string]any)
	}{
		{
			name:         "successful get ssl status",
			portalSlug:   "support",
			statusCode:   http.StatusOK,
			responseBody: `{"ssl_enabled": true, "certificate_expiry": "2025-12-31"}`,
			expectError:  false,
			validateFunc: func(t *testing.T, result map[string]any) {
				if result["ssl_enabled"] != true {
					t.Errorf("Expected ssl_enabled true, got %v", result["ssl_enabled"])
				}
			},
		},
		{
			name:         "not found",
			portalSlug:   "nonexistent",
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
				if !strings.Contains(r.URL.Path, "/ssl_status") {
					t.Errorf("Expected ssl_status path, got %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.GetPortalSSLStatus(context.Background(), tt.portalSlug)

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

func TestReorderArticles(t *testing.T) {
	tests := []struct {
		name        string
		portalSlug  string
		articleIDs  []int
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful reorder",
			portalSlug:  "support",
			articleIDs:  []int{3, 1, 2},
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			portalSlug:  "nonexistent",
			articleIDs:  []int{1, 2},
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/articles/reorder") {
					t.Errorf("Expected reorder path, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.ReorderArticles(context.Background(), tt.portalSlug, tt.articleIDs)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
