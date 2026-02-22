package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShopifyAuth(t *testing.T) {
	tests := []struct {
		name         string
		shopDomain   string
		code         string
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful auth",
			shopDomain:   "mystore.myshopify.com",
			code:         "auth_code_123",
			statusCode:   http.StatusOK,
			responseBody: `{"success": true}`,
			expectError:  false,
		},
		{
			name:         "invalid code",
			shopDomain:   "mystore.myshopify.com",
			code:         "invalid",
			statusCode:   http.StatusUnauthorized,
			responseBody: `{"error": "invalid code"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/accounts/1/integrations/shopify/auth" {
					t.Errorf("Unexpected path: %s", r.URL.Path)
				}
				_ = json.NewDecoder(r.Body).Decode(&capturedBody)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.Shopify().Auth(context.Background(), tt.shopDomain, tt.code)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if capturedBody["shop"] != tt.shopDomain {
				t.Errorf("Expected shop %s, got %v", tt.shopDomain, capturedBody["shop"])
			}
		})
	}
}

func TestListShopifyOrders(t *testing.T) {
	tests := []struct {
		name         string
		contactID    int
		statusCode   int
		responseBody string
		expectError  bool
		expectCount  int
	}{
		{
			name:       "successful list",
			contactID:  123,
			statusCode: http.StatusOK,
			responseBody: `[
				{"id": 1, "name": "#1001", "email": "customer@example.com", "total_price": "99.99", "currency": "USD", "financial_status": "paid"},
				{"id": 2, "name": "#1002", "email": "another@example.com", "total_price": "149.99", "currency": "USD", "financial_status": "pending"}
			]`,
			expectError: false,
			expectCount: 2,
		},
		{
			name:         "empty list",
			contactID:    456,
			statusCode:   http.StatusOK,
			responseBody: `[]`,
			expectError:  false,
			expectCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/v1/accounts/1/integrations/shopify/orders?contact_id=%d", tt.contactID)
				actualPath := r.URL.Path + "?" + r.URL.RawQuery
				if actualPath != expectedPath {
					t.Errorf("Unexpected path: %s, expected: %s", actualPath, expectedPath)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			orders, err := client.Shopify().ListOrders(context.Background(), tt.contactID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if len(orders) != tt.expectCount {
				t.Errorf("Expected %d orders, got %d", tt.expectCount, len(orders))
			}
		})
	}
}

func TestDeleteShopifyIntegration(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful delete",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
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
			err := client.Shopify().Delete(context.Background())

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
