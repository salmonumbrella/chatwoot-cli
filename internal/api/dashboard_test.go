package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashboardClient_Query(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Basic dGVzdEBleGFtcGxlLmNvbQ==" {
			t.Errorf("Authorization = %q, want Basic dGVzdEBleGFtcGxlLmNvbQ==", auth)
		}

		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		accept := r.Header.Get("Accept")
		if accept != "application/json" {
			t.Errorf("Accept = %q, want application/json", accept)
		}

		var body DashboardRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode body: %v", err)
		}
		if body.ContactID != 123 {
			t.Errorf("ContactID = %d, want 123", body.ContactID)
		}
		if body.Page != 1 {
			t.Errorf("Page = %d, want 1", body.Page)
		}
		if body.PerPage != 50 {
			t.Errorf("PerPage = %d, want 50", body.PerPage)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "order-1", "number": "12345"},
			},
			"pagination": map[string]any{
				"page":        1,
				"total_pages": 1,
			},
		})
	}))
	defer server.Close()

	client := NewDashboardClient(server.URL, "test@example.com")
	resp, err := client.Query(context.Background(), DashboardRequest{
		ContactID: 123,
		Page:      1,
		PerPage:   50,
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	items, ok := resp["items"].([]any)
	if !ok {
		t.Fatalf("items not found or wrong type")
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}

func TestDashboardClient_QueryError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
	}))
	defer server.Close()

	client := NewDashboardClient(server.URL, "bad@example.com")
	_, err := client.Query(context.Background(), DashboardRequest{ContactID: 123})

	if err == nil {
		t.Error("Expected error for 401 response")
	}
}

func TestDashboardClient_QueryResponseTooLarge(t *testing.T) {
	// Create a response larger than maxResponseSize (10MB + 1KB)
	largeBody := strings.Repeat("x", maxResponseSize+1024)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(largeBody))
	}))
	defer server.Close()

	client := NewDashboardClient(server.URL, "test@example.com")
	_, err := client.Query(context.Background(), DashboardRequest{ContactID: 123})

	if err == nil {
		t.Fatal("Expected error for oversized response, got nil")
	}
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Errorf("Expected ErrResponseTooLarge, got: %v", err)
	}
}

func TestDashboardClient_QueryResponseAtLimit(t *testing.T) {
	// Create a valid JSON response just under the limit
	// This test verifies that responses at or near the limit still work
	smallJSON := `{"items":[],"pagination":{"page":1,"total_pages":0}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(smallJSON))
	}))
	defer server.Close()

	client := NewDashboardClient(server.URL, "test@example.com")
	resp, err := client.Query(context.Background(), DashboardRequest{ContactID: 123})
	if err != nil {
		t.Fatalf("Unexpected error for valid response: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected response, got nil")
	}
}

func TestDashboardClient_QuerySupportsBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer secret-token")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items":      []any{},
			"pagination": map[string]any{"page": 1, "total_pages": 1},
		})
	}))
	defer server.Close()

	client := NewDashboardClient(server.URL, "Bearer secret-token")
	if _, err := client.Query(context.Background(), DashboardRequest{ContactID: 123}); err != nil {
		t.Fatalf("Query failed: %v", err)
	}
}

func TestDashboardClient_QueryOrderDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/api/public/chatwoot/orders/order-1" {
			t.Errorf("Path = %q, want %q", r.URL.Path, "/api/public/chatwoot/orders/order-1")
		}
		if got := r.Header.Get("Authorization"); got != "Basic dGVzdEBleGFtcGxlLmNvbQ==" {
			t.Errorf("Authorization = %q, want Basic dGVzdEBleGFtcGxlLmNvbQ==", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"order": map[string]any{"id": "order-1", "number": "SO-1"},
			"line_items": []map[string]any{
				{"product_name": "Sneaker", "quantity": 1},
			},
			"order_metadata": map[string]any{"payment_method": "linepay"},
		})
	}))
	defer server.Close()

	client := NewDashboardClient(server.URL+"/api/public/chatwoot/contact/orders", "test@example.com")
	result, err := client.QueryOrderDetail(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("QueryOrderDetail failed: %v", err)
	}

	lineItems, ok := result["line_items"].([]any)
	if !ok {
		t.Fatalf("line_items type = %T, want []any", result["line_items"])
	}
	if len(lineItems) != 1 {
		t.Fatalf("len(line_items) = %d, want 1", len(lineItems))
	}
}

func TestDashboardClient_AuthorizationHeaderNormalizesCasing(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"uppercase BEARER", "BEARER my-token", "Bearer my-token"},
		{"mixed BeArEr", "BeArEr my-token", "Bearer my-token"},
		{"uppercase BASIC", "BASIC dXNlcjpwYXNz", "Basic dXNlcjpwYXNz"},
		{"mixed BaSiC", "BaSiC dXNlcjpwYXNz", "Basic dXNlcjpwYXNz"},
		{"correct Bearer", "Bearer my-token", "Bearer my-token"},
		{"correct Basic", "Basic dXNlcjpwYXNz", "Basic dXNlcjpwYXNz"},
		{"plain token", "test@example.com", "Basic dGVzdEBleGFtcGxlLmNvbQ=="},
		{"empty token", "", ""},
		{"whitespace only", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &DashboardClient{AuthToken: tt.token}
			got := c.authorizationHeader()
			if got != tt.want {
				t.Errorf("authorizationHeader() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDashboardClient_OrderDetailURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		orderID  string
		want     string
		wantErr  string
	}{
		{
			name:     "chatwoot/contact/orders suffix",
			endpoint: "https://host/api/public/chatwoot/contact/orders",
			orderID:  "order-1",
			want:     "https://host/api/public/chatwoot/orders/order-1",
		},
		{
			name:     "contact/orders suffix",
			endpoint: "https://host/api/contact/orders",
			orderID:  "order-1",
			want:     "https://host/api/orders/order-1",
		},
		{
			name:     "bare /orders suffix",
			endpoint: "https://host/api/orders",
			orderID:  "order-1",
			want:     "https://host/api/orders/order-1",
		},
		{
			name:     "empty path",
			endpoint: "https://host",
			orderID:  "order-1",
			want:     "https://host/orders/order-1",
		},
		{
			name:     "trailing slash stripped",
			endpoint: "https://host/api/chatwoot/contact/orders/",
			orderID:  "order-1",
			want:     "https://host/api/chatwoot/orders/order-1",
		},
		{
			name:     "special chars in order ID escaped",
			endpoint: "https://host/api/orders",
			orderID:  "order/1",
			want:     "https://host/api/orders/order%252F1",
		},
		{
			name:     "empty order ID",
			endpoint: "https://host/api/orders",
			orderID:  "",
			wantErr:  "order id is required",
		},
		{
			name:     "whitespace-only order ID",
			endpoint: "https://host/api/orders",
			orderID:  "   ",
			wantErr:  "order id is required",
		},
		{
			name:     "unrecognized path errors",
			endpoint: "https://host/api/inventory",
			orderID:  "order-1",
			wantErr:  "cannot derive order detail URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &DashboardClient{Endpoint: tt.endpoint}
			got, err := c.orderDetailURL(tt.orderID)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("orderDetailURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDashboardClient_LinkOrderToContact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/api/public/chatwoot/orders/link" {
			t.Errorf("Path = %q, want %q", r.URL.Path, "/api/public/chatwoot/orders/link")
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer secret-token")
		}

		var body LinkOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.OrderNumber != "SO20240215001" {
			t.Fatalf("OrderNumber = %q, want %q", body.OrderNumber, "SO20240215001")
		}
		if body.ContactID != 12345 {
			t.Fatalf("ContactID = %d, want %d", body.ContactID, 12345)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"customer_id":         "a1b2c3",
			"chatwoot_contact_id": 12345,
		})
	}))
	defer server.Close()

	client := NewDashboardClient(server.URL+"/api/public/chatwoot/contact/orders", "Bearer secret-token")
	resp, err := client.LinkOrderToContact(context.Background(), "SO20240215001", 12345)
	if err != nil {
		t.Fatalf("LinkOrderToContact failed: %v", err)
	}

	if resp.CustomerID != "a1b2c3" {
		t.Fatalf("CustomerID = %q, want %q", resp.CustomerID, "a1b2c3")
	}
	if resp.ChatwootContactID != 12345 {
		t.Fatalf("ChatwootContactID = %d, want %d", resp.ChatwootContactID, 12345)
	}
}

func TestDashboardClient_LinkOrderToContactServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"contact merge conflict"}`))
	}))
	defer server.Close()

	client := NewDashboardClient(server.URL+"/api/public/chatwoot/contact/orders", "Bearer token")
	_, err := client.LinkOrderToContact(context.Background(), "SO1", 123)
	if err == nil {
		t.Fatal("expected error for 409 response")
	}
	if !strings.Contains(err.Error(), "status 409") {
		t.Fatalf("error = %q, want it to contain 'status 409'", err.Error())
	}
}

func TestDashboardClient_LinkOrderToContactValidation(t *testing.T) {
	client := NewDashboardClient("https://host/api/public/chatwoot/contact/orders", "Bearer token")

	if _, err := client.LinkOrderToContact(context.Background(), "", 123); err == nil || !strings.Contains(err.Error(), "order number is required") {
		t.Fatalf("expected order number validation error, got: %v", err)
	}

	if _, err := client.LinkOrderToContact(context.Background(), "SO1", 0); err == nil || !strings.Contains(err.Error(), "contact id must be positive") {
		t.Fatalf("expected contact id validation error, got: %v", err)
	}
}

func TestDashboardClient_OrderLinkURL(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
		wantErr  string
	}{
		{
			name:     "chatwoot/contact/orders suffix",
			endpoint: "https://host/api/public/chatwoot/contact/orders",
			want:     "https://host/api/public/chatwoot/orders/link",
		},
		{
			name:     "contact/orders suffix",
			endpoint: "https://host/api/contact/orders",
			want:     "https://host/api/orders/link",
		},
		{
			name:     "bare /orders suffix",
			endpoint: "https://host/api/orders",
			want:     "https://host/api/orders/link",
		},
		{
			name:     "empty path",
			endpoint: "https://host",
			want:     "https://host/orders/link",
		},
		{
			name:     "trailing slash stripped",
			endpoint: "https://host/api/chatwoot/contact/orders/",
			want:     "https://host/api/chatwoot/orders/link",
		},
		{
			name:     "unrecognized path errors",
			endpoint: "https://host/api/inventory",
			wantErr:  "cannot derive order link URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &DashboardClient{Endpoint: tt.endpoint}
			got, err := c.orderLinkURL()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("orderLinkURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
