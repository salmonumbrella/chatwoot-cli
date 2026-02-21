package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListCannedResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/canned_responses" {
			t.Errorf("Expected path /api/v1/accounts/1/canned_responses, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 1, "short_code": "greeting", "content": "Hello! How can I help?", "account_id": 1},
			{"id": 2, "short_code": "thanks", "content": "Thank you for contacting us!", "account_id": 1}
		]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CannedResponses().List(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 canned responses, got %d", len(result))
	}
	if result[0].ShortCode != "greeting" {
		t.Errorf("Expected short code 'greeting', got %s", result[0].ShortCode)
	}
	if result[0].Content != "Hello! How can I help?" {
		t.Errorf("Expected content 'Hello! How can I help?', got %s", result[0].Content)
	}
	if result[1].ShortCode != "thanks" {
		t.Errorf("Expected short code 'thanks', got %s", result[1].ShortCode)
	}
}

func TestListCannedResponses_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CannedResponses().List(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected 0 canned responses, got %d", len(result))
	}
}

func TestGetCannedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 1, "short_code": "greeting", "content": "Hello! How can I help?", "account_id": 1},
			{"id": 2, "short_code": "thanks", "content": "Thank you for contacting us!", "account_id": 1}
		]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CannedResponses().Get(context.Background(), 2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 2 {
		t.Errorf("Expected ID 2, got %d", result.ID)
	}
	if result.ShortCode != "thanks" {
		t.Errorf("Expected short code 'thanks', got %s", result.ShortCode)
	}
}

func TestGetCannedResponse_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 1, "short_code": "greeting", "content": "Hello! How can I help?", "account_id": 1}
		]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CannedResponses().Get(context.Background(), 999)

	if err == nil {
		t.Error("Expected error for non-existent canned response, got nil")
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

func TestCreateCannedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/canned_responses" {
			t.Errorf("Expected path /api/v1/accounts/1/canned_responses, got %s", r.URL.Path)
		}

		var body map[string]map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body["canned_response"]["short_code"] != "test" {
			t.Errorf("Expected short_code 'test', got %s", body["canned_response"]["short_code"])
		}
		if body["canned_response"]["content"] != "Test content" {
			t.Errorf("Expected content 'Test content', got %s", body["canned_response"]["content"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1, "short_code": "test", "content": "Test content", "account_id": 1}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CannedResponses().Create(context.Background(), "test", "Test content")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}
	if result.ShortCode != "test" {
		t.Errorf("Expected short code 'test', got %s", result.ShortCode)
	}
	if result.Content != "Test content" {
		t.Errorf("Expected content 'Test content', got %s", result.Content)
	}
}

func TestUpdateCannedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/canned_responses/1" {
			t.Errorf("Expected path /api/v1/accounts/1/canned_responses/1, got %s", r.URL.Path)
		}

		var body map[string]map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body["canned_response"]["short_code"] != "updated" {
			t.Errorf("Expected short_code 'updated', got %s", body["canned_response"]["short_code"])
		}
		if body["canned_response"]["content"] != "Updated content" {
			t.Errorf("Expected content 'Updated content', got %s", body["canned_response"]["content"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1, "short_code": "updated", "content": "Updated content", "account_id": 1}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CannedResponses().Update(context.Background(), 1, "updated", "Updated content")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}
	if result.ShortCode != "updated" {
		t.Errorf("Expected short code 'updated', got %s", result.ShortCode)
	}
	if result.Content != "Updated content" {
		t.Errorf("Expected content 'Updated content', got %s", result.Content)
	}
}

func TestDeleteCannedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/canned_responses/1" {
			t.Errorf("Expected path /api/v1/accounts/1/canned_responses/1, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.CannedResponses().Delete(context.Background(), 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDeleteCannedResponse_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Canned response not found"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.CannedResponses().Delete(context.Background(), 999)

	if err == nil {
		t.Error("Expected error for non-existent canned response, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected *APIError, got %T", err)
	} else if apiErr.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", apiErr.StatusCode)
	}
}
