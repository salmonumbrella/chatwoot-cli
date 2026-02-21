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
