package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListInboxes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"payload": [
				{"id": 1, "name": "Email Inbox", "channel_type": "Channel::Email"},
				{"id": 2, "name": "Web Chat", "channel_type": "Channel::WebWidget"}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.ListInboxes(context.Background())

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 inboxes, got %d", len(result))
	}
	if result[0].ID != 1 {
		t.Errorf("Expected ID 1, got %d", result[0].ID)
	}
	if result[0].Name != "Email Inbox" {
		t.Errorf("Expected name 'Email Inbox', got %s", result[0].Name)
	}
	if result[0].ChannelType != "Channel::Email" {
		t.Errorf("Expected channel type 'Channel::Email', got %s", result[0].ChannelType)
	}
	if result[1].ChannelType != "Channel::WebWidget" {
		t.Errorf("Expected channel type 'Channel::WebWidget', got %s", result[1].ChannelType)
	}
}

func TestGetInbox(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes/1" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes/1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 1,
			"name": "Email Inbox",
			"channel_type": "Channel::Email",
			"greeting_enabled": true,
			"greeting_message": "Welcome!",
			"enable_auto_assignment": true
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.GetInbox(context.Background(), 1)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}
	if result.Name != "Email Inbox" {
		t.Errorf("Expected name 'Email Inbox', got %s", result.Name)
	}
	if result.ChannelType != "Channel::Email" {
		t.Errorf("Expected channel type 'Channel::Email', got %s", result.ChannelType)
	}
	if !result.GreetingEnabled {
		t.Error("Expected greeting_enabled to be true")
	}
	if result.GreetingMessage != "Welcome!" {
		t.Errorf("Expected greeting message 'Welcome!', got %s", result.GreetingMessage)
	}
}

func TestCreateInbox(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 3,
			"name": "New Inbox",
			"channel_type": "Channel::Api"
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.CreateInbox(context.Background(), CreateInboxRequest{
		Name:        "New Inbox",
		ChannelType: "api",
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 3 {
		t.Errorf("Expected ID 3, got %d", result.ID)
	}
	if result.Name != "New Inbox" {
		t.Errorf("Expected name 'New Inbox', got %s", result.Name)
	}
	if result.ChannelType != "Channel::Api" {
		t.Errorf("Expected channel type 'Channel::Api', got %s", result.ChannelType)
	}
}

func TestUpdateInbox(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes/1" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes/1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 1,
			"name": "Updated Inbox",
			"channel_type": "Channel::Email"
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.UpdateInbox(context.Background(), 1, UpdateInboxRequest{
		Name: "Updated Inbox",
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}
	if result.Name != "Updated Inbox" {
		t.Errorf("Expected name 'Updated Inbox', got %s", result.Name)
	}
}

func TestDeleteInbox(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes/1" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes/1, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.DeleteInbox(context.Background(), 1)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetInboxAgentBot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes/1/agent_bot" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes/1/agent_bot, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 10,
			"name": "Support Bot",
			"description": "Handles common queries",
			"outgoing_url": "https://example.com/bot",
			"account_id": 1
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.GetInboxAgentBot(context.Background(), 1)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 10 {
		t.Errorf("Expected ID 10, got %d", result.ID)
	}
	if result.Name != "Support Bot" {
		t.Errorf("Expected name 'Support Bot', got %s", result.Name)
	}
	if result.Description != "Handles common queries" {
		t.Errorf("Expected description 'Handles common queries', got %s", result.Description)
	}
	if result.OutgoingURL != "https://example.com/bot" {
		t.Errorf("Expected outgoing URL 'https://example.com/bot', got %s", result.OutgoingURL)
	}
}

func TestSetInboxAgentBot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes/1/set_agent_bot" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes/1/set_agent_bot, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.SetInboxAgentBot(context.Background(), 1, 10)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestListInboxMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inbox_members/1" {
			t.Errorf("Expected path /api/v1/accounts/1/inbox_members/1, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"payload": [
				{"id": 1, "name": "Agent One", "email": "agent1@example.com", "role": "agent"},
				{"id": 2, "name": "Agent Two", "email": "agent2@example.com", "role": "agent"}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.ListInboxMembers(context.Background(), 1)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(result))
	}
	if result[0].ID != 1 {
		t.Errorf("Expected ID 1, got %d", result[0].ID)
	}
	if result[0].Name != "Agent One" {
		t.Errorf("Expected name 'Agent One', got %s", result[0].Name)
	}
	if result[0].Email != "agent1@example.com" {
		t.Errorf("Expected email 'agent1@example.com', got %s", result[0].Email)
	}
	if result[1].Name != "Agent Two" {
		t.Errorf("Expected name 'Agent Two', got %s", result[1].Name)
	}
}

func TestAddInboxMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inbox_members" {
			t.Errorf("Expected path /api/v1/accounts/1/inbox_members, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.AddInboxMembers(context.Background(), 1, []int{10, 20})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRemoveInboxMembers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inbox_members" {
			t.Errorf("Expected path /api/v1/accounts/1/inbox_members, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.RemoveInboxMembers(context.Background(), 1, []int{10, 20})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
