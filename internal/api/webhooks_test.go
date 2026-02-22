package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListWebhooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/webhooks" {
			t.Errorf("Expected path /api/v1/accounts/1/webhooks, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"payload": {
				"webhooks": [
					{"id": 1, "url": "https://example.com/hook1", "subscriptions": ["message_created"], "account_id": 1},
					{"id": 2, "url": "https://example.com/hook2", "subscriptions": ["conversation_created"], "account_id": 1}
				]
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Webhooks().List(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 webhooks, got %d", len(result))
	}
	if result[0].ID != 1 {
		t.Errorf("Expected first webhook ID 1, got %d", result[0].ID)
	}
	if result[0].URL != "https://example.com/hook1" {
		t.Errorf("Expected first webhook URL https://example.com/hook1, got %s", result[0].URL)
	}
	if len(result[0].Subscriptions) != 1 || result[0].Subscriptions[0] != "message_created" {
		t.Errorf("Expected first webhook subscriptions [message_created], got %v", result[0].Subscriptions)
	}
}

func TestListWebhooksEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"payload": {
				"webhooks": []
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Webhooks().List(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("Expected 0 webhooks, got %d", len(result))
	}
}

func TestGetWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"payload": {
				"webhooks": [
					{"id": 1, "url": "https://example.com/hook1", "subscriptions": ["message_created"], "account_id": 1},
					{"id": 2, "url": "https://example.com/hook2", "subscriptions": ["conversation_created"], "account_id": 1}
				]
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Webhooks().Get(context.Background(), 2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected webhook, got nil")
	}
	if result.ID != 2 {
		t.Errorf("Expected webhook ID 2, got %d", result.ID)
	}
	if result.URL != "https://example.com/hook2" {
		t.Errorf("Expected URL https://example.com/hook2, got %s", result.URL)
	}
}

func TestGetWebhookNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"payload": {
				"webhooks": [
					{"id": 1, "url": "https://example.com/hook1", "subscriptions": ["message_created"], "account_id": 1}
				]
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Webhooks().Get(context.Background(), 999)

	if err == nil {
		t.Error("Expected error for non-existent webhook, got nil")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected APIError, got %T", err)
	} else if apiErr.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", apiErr.StatusCode)
	}
}

func TestCreateWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/webhooks" {
			t.Errorf("Expected path /api/v1/accounts/1/webhooks, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body["url"] != "https://example.com/webhook" {
			t.Errorf("Expected URL https://example.com/webhook, got %v", body["url"])
		}
		subscriptions, ok := body["subscriptions"].([]any)
		if !ok || len(subscriptions) != 2 {
			t.Errorf("Expected 2 subscriptions, got %v", body["subscriptions"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"payload": {
				"webhook": {"id": 1, "url": "https://example.com/webhook", "subscriptions": ["message_created", "conversation_created"], "account_id": 1}
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Webhooks().Create(context.Background(), "https://example.com/webhook", []string{"message_created", "conversation_created"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected webhook, got nil")
	}
	if result.URL != "https://example.com/webhook" {
		t.Errorf("Expected URL https://example.com/webhook, got %s", result.URL)
	}
	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}
	if len(result.Subscriptions) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(result.Subscriptions))
	}
}

func TestUpdateWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/webhooks/1" {
			t.Errorf("Expected path /api/v1/accounts/1/webhooks/1, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if body["url"] != "https://example.com/updated" {
			t.Errorf("Expected URL https://example.com/updated, got %v", body["url"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"payload": {
				"webhook": {"id": 1, "url": "https://example.com/updated", "subscriptions": ["message_created"], "account_id": 1}
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Webhooks().Update(context.Background(), 1, "https://example.com/updated", []string{"message_created"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected webhook, got nil")
	}
	if result.URL != "https://example.com/updated" {
		t.Errorf("Expected URL https://example.com/updated, got %s", result.URL)
	}
}

func TestUpdateWebhookPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Verify only subscriptions are in the body (URL should be empty string and not sent)
		if _, hasURL := body["url"]; hasURL {
			t.Error("Expected URL to not be in request body when empty string passed")
		}
		if _, hasSubs := body["subscriptions"]; !hasSubs {
			t.Error("Expected subscriptions in request body")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"payload": {
				"webhook": {"id": 1, "url": "https://example.com/original", "subscriptions": ["conversation_created"], "account_id": 1}
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Webhooks().Update(context.Background(), 1, "", []string{"conversation_created"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected webhook, got nil")
	}
	if len(result.Subscriptions) != 1 || result.Subscriptions[0] != "conversation_created" {
		t.Errorf("Expected subscriptions [conversation_created], got %v", result.Subscriptions)
	}
}

func TestDeleteWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/webhooks/1" {
			t.Errorf("Expected path /api/v1/accounts/1/webhooks/1, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.Webhooks().Delete(context.Background(), 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDeleteWebhookNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Webhook not found"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.Webhooks().Delete(context.Background(), 999)

	if err == nil {
		t.Error("Expected error for non-existent webhook, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Errorf("Expected APIError, got %T", err)
	} else if apiErr.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", apiErr.StatusCode)
	}
}
