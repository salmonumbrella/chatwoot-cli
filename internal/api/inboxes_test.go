package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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
	result, err := client.Inboxes().List(context.Background())
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
	result, err := client.Inboxes().Get(context.Background(), 1)
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
	result, err := client.Inboxes().Create(context.Background(), CreateInboxRequest{
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
	result, err := client.Inboxes().Update(context.Background(), 1, UpdateInboxRequest{
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
	err := client.Inboxes().Delete(context.Background(), 1)
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
	result, err := client.Inboxes().GetAgentBot(context.Background(), 1)
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
	err := client.Inboxes().SetAgentBot(context.Background(), 1, 10)
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
	result, err := client.Inboxes().ListMembers(context.Background(), 1)
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
	err := client.Inboxes().AddMembers(context.Background(), 1, []int{10, 20})
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
	err := client.Inboxes().RemoveMembers(context.Background(), 1, []int{10, 20})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetInboxTriage(t *testing.T) {
	// Track request counts for concurrent requests
	var contactRequests, messageRequests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// Get inbox
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/accounts/1/inboxes/1":
			_, _ = w.Write([]byte(`{
				"id": 1,
				"name": "Support Inbox",
				"channel_type": "Channel::Email"
			}`))

		// List messages for conversation 100 (must be before generic conversations match)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/accounts/1/conversations/100/messages":
			messageRequests.Add(1)
			_, _ = w.Write([]byte(`{
				"payload": [
					{
						"id": 1000,
						"conversation_id": 100,
						"content": "Hi, I need help with my billing issue",
						"message_type": 0,
						"created_at": 1700000500
					}
				]
			}`))

		// List messages for conversation 101
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/accounts/1/conversations/101/messages":
			messageRequests.Add(1)
			_, _ = w.Write([]byte(`{
				"payload": [
					{
						"id": 1001,
						"conversation_id": 101,
						"content": "We have resolved your issue",
						"message_type": 1,
						"created_at": 1700001500
					}
				]
			}`))

		// List conversations (after specific message paths)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/accounts/1/conversations":
			// Verify query parameters
			query := r.URL.Query()
			if query.Get("inbox_id") != "1" {
				t.Errorf("Expected inbox_id=1, got %s", query.Get("inbox_id"))
			}
			if query.Get("status") != "open" {
				t.Errorf("Expected status=open, got %s", query.Get("status"))
			}

			_, _ = w.Write([]byte(`{
				"data": {
					"meta": {"current_page": 1, "per_page": 25, "total_pages": 1, "total_count": 2},
					"payload": [
						{
							"id": 100,
							"display_id": 1001,
							"account_id": 1,
							"inbox_id": 1,
							"status": "open",
							"priority": "high",
							"contact_id": 200,
							"unread_count": 3,
							"created_at": 1700000000,
							"labels": ["urgent", "billing"]
						},
						{
							"id": 101,
							"display_id": 1002,
							"account_id": 1,
							"inbox_id": 1,
							"status": "pending",
							"contact_id": 201,
							"unread_count": 0,
							"created_at": 1700001000,
							"labels": []
						}
					]
				}
			}`))

		// Get contact 200
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/accounts/1/contacts/200":
			contactRequests.Add(1)
			_, _ = w.Write([]byte(`{
				"payload": {
					"id": 200,
					"name": "John Doe",
					"email": "john@example.com"
				}
			}`))

		// Get contact 201
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/accounts/1/contacts/201":
			contactRequests.Add(1)
			_, _ = w.Write([]byte(`{
				"payload": {
					"id": 201,
					"name": "Jane Smith",
					"email": "jane@example.com"
				}
			}`))

		default:
			t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Inboxes().Triage(context.Background(), 1, "", 25)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify inbox info
	if result.InboxID != 1 {
		t.Errorf("Expected InboxID 1, got %d", result.InboxID)
	}
	if result.InboxName != "Support Inbox" {
		t.Errorf("Expected InboxName 'Support Inbox', got %s", result.InboxName)
	}

	// Verify summary
	if result.Summary.Open != 1 {
		t.Errorf("Expected 1 open, got %d", result.Summary.Open)
	}
	if result.Summary.Pending != 1 {
		t.Errorf("Expected 1 pending, got %d", result.Summary.Pending)
	}
	if result.Summary.Unread != 1 {
		t.Errorf("Expected 1 unread, got %d", result.Summary.Unread)
	}

	// Verify conversations
	if len(result.Conversations) != 2 {
		t.Fatalf("Expected 2 conversations, got %d", len(result.Conversations))
	}

	// First conversation
	conv1 := result.Conversations[0]
	if conv1.ID != 100 {
		t.Errorf("Expected conv ID 100, got %d", conv1.ID)
	}
	if conv1.Contact.Name != "John Doe" {
		t.Errorf("Expected contact name 'John Doe', got %s", conv1.Contact.Name)
	}
	if conv1.Contact.Email != "john@example.com" {
		t.Errorf("Expected contact email 'john@example.com', got %s", conv1.Contact.Email)
	}
	if conv1.LastMessage == nil {
		t.Error("Expected last message, got nil")
	} else {
		if conv1.LastMessage.Content != "Hi, I need help with my billing issue" {
			t.Errorf("Expected message content, got %s", conv1.LastMessage.Content)
		}
		if conv1.LastMessage.Type != "incoming" {
			t.Errorf("Expected message type 'incoming', got %s", conv1.LastMessage.Type)
		}
	}
	if len(conv1.Labels) != 2 {
		t.Errorf("Expected 2 labels, got %d", len(conv1.Labels))
	}

	// Second conversation
	conv2 := result.Conversations[1]
	if conv2.ID != 101 {
		t.Errorf("Expected conv ID 101, got %d", conv2.ID)
	}
	if conv2.Contact.Name != "Jane Smith" {
		t.Errorf("Expected contact name 'Jane Smith', got %s", conv2.Contact.Name)
	}
	if conv2.LastMessage == nil {
		t.Error("Expected last message, got nil")
	} else if conv2.LastMessage.Type != "outgoing" {
		t.Errorf("Expected message type 'outgoing', got %s", conv2.LastMessage.Type)
	}

	// Verify parallel requests were made
	if contactRequests.Load() != 2 {
		t.Errorf("Expected 2 contact requests, got %d", contactRequests.Load())
	}
	if messageRequests.Load() != 2 {
		t.Errorf("Expected 2 message requests, got %d", messageRequests.Load())
	}
}

func TestGetInboxTriageWithLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/accounts/1/inboxes/1":
			_, _ = w.Write([]byte(`{"id": 1, "name": "Support"}`))

		case strings.HasPrefix(r.URL.Path, "/api/v1/accounts/1/conversations") && !strings.Contains(r.URL.Path, "/messages"):
			_, _ = w.Write([]byte(`{
				"data": {
					"meta": {"current_page": 1, "total_count": 5},
					"payload": [
						{"id": 1, "status": "open", "contact_id": 100, "unread_count": 0, "created_at": 1700000000},
						{"id": 2, "status": "open", "contact_id": 101, "unread_count": 0, "created_at": 1700000000},
						{"id": 3, "status": "open", "contact_id": 102, "unread_count": 0, "created_at": 1700000000},
						{"id": 4, "status": "open", "contact_id": 103, "unread_count": 0, "created_at": 1700000000},
						{"id": 5, "status": "open", "contact_id": 104, "unread_count": 0, "created_at": 1700000000}
					]
				}
			}`))

		case strings.HasPrefix(r.URL.Path, "/api/v1/accounts/1/contacts/"):
			_, _ = w.Write([]byte(`{"payload": {"id": 100, "name": "Test User"}}`))

		case strings.Contains(r.URL.Path, "/messages"):
			_, _ = w.Write([]byte(`{"payload": []}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Inboxes().Triage(context.Background(), 1, "open", 2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify limit was applied
	if len(result.Conversations) != 2 {
		t.Errorf("Expected 2 conversations (limit), got %d", len(result.Conversations))
	}
}

func TestGetInboxTriageEmptyInbox(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/accounts/1/inboxes/1":
			_, _ = w.Write([]byte(`{"id": 1, "name": "Empty Inbox"}`))

		case strings.HasPrefix(r.URL.Path, "/api/v1/accounts/1/conversations"):
			_, _ = w.Write([]byte(`{
				"data": {
					"meta": {"current_page": 1, "total_count": 0},
					"payload": []
				}
			}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Inboxes().Triage(context.Background(), 1, "open", 25)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result.Conversations) != 0 {
		t.Errorf("Expected 0 conversations, got %d", len(result.Conversations))
	}
	if result.Summary.Open != 0 {
		t.Errorf("Expected 0 open, got %d", result.Summary.Open)
	}
}

func TestGetInboxTriageInboxNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "Inbox not found"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	_, err := client.Inboxes().Triage(context.Background(), 999, "", 25)

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get inbox") {
		t.Errorf("Expected 'failed to get inbox' error, got %v", err)
	}
}

func TestGetInboxTriageContactFetchFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/accounts/1/inboxes/1":
			_, _ = w.Write([]byte(`{"id": 1, "name": "Support"}`))

		case strings.HasPrefix(r.URL.Path, "/api/v1/accounts/1/conversations") && !strings.Contains(r.URL.Path, "/messages"):
			_, _ = w.Write([]byte(`{
				"data": {
					"meta": {"current_page": 1},
					"payload": [
						{"id": 100, "status": "open", "contact_id": 200, "unread_count": 1, "created_at": 1700000000}
					]
				}
			}`))

		case strings.HasPrefix(r.URL.Path, "/api/v1/accounts/1/contacts/"):
			// Contact fetch fails
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "Server error"}`))

		case strings.Contains(r.URL.Path, "/messages"):
			_, _ = w.Write([]byte(`{"payload": [{"id": 1, "content": "Hello", "message_type": 0, "created_at": 1700000000}]}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Inboxes().Triage(context.Background(), 1, "open", 25)
	// Should not error - contact fetch failure is handled gracefully
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Conversation should still be present with contact ID fallback
	if len(result.Conversations) != 1 {
		t.Fatalf("Expected 1 conversation, got %d", len(result.Conversations))
	}

	// Contact should have ID but no name (fallback behavior)
	if result.Conversations[0].Contact.ID != 200 {
		t.Errorf("Expected contact ID 200, got %d", result.Conversations[0].Contact.ID)
	}
	if result.Conversations[0].Contact.Name != "" {
		t.Errorf("Expected empty contact name (fallback), got %s", result.Conversations[0].Contact.Name)
	}

	// Last message should still be present
	if result.Conversations[0].LastMessage == nil {
		t.Error("Expected last message even when contact fetch fails")
	}
}

func TestUpdateInboxMembers(t *testing.T) {
	tests := []struct {
		name        string
		inboxID     int
		userIDs     []int
		statusCode  int
		expectError bool
		validateReq func(*testing.T, *http.Request, []byte)
	}{
		{
			name:        "successful update",
			inboxID:     1,
			userIDs:     []int{10, 20, 30},
			statusCode:  http.StatusOK,
			expectError: false,
			validateReq: func(t *testing.T, r *http.Request, body []byte) {
				if r.Method != http.MethodPatch {
					t.Errorf("Expected PATCH, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/accounts/1/inbox_members" {
					t.Errorf("Expected path /api/v1/accounts/1/inbox_members, got %s", r.URL.Path)
				}
				// Verify body contains inbox_id and user_ids
				bodyStr := string(body)
				if !strings.Contains(bodyStr, `"inbox_id":1`) && !strings.Contains(bodyStr, `"inbox_id": 1`) {
					t.Errorf("Expected inbox_id in body, got %s", bodyStr)
				}
			},
		},
		{
			name:        "empty user list",
			inboxID:     2,
			userIDs:     []int{},
			statusCode:  http.StatusOK,
			expectError: false,
			validateReq: func(t *testing.T, r *http.Request, body []byte) {
				bodyStr := string(body)
				if !strings.Contains(bodyStr, `"user_ids":[]`) && !strings.Contains(bodyStr, `"user_ids": []`) {
					t.Errorf("Expected empty user_ids array in body, got %s", bodyStr)
				}
			},
		},
		{
			name:        "server error",
			inboxID:     3,
			userIDs:     []int{1},
			statusCode:  http.StatusInternalServerError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedBody, _ = io.ReadAll(r.Body)
				if tt.validateReq != nil {
					tt.validateReq(t, r, capturedBody)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.Inboxes().UpdateMembers(context.Background(), tt.inboxID, tt.userIDs)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestGetInboxCampaigns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes/1/campaigns" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes/1/campaigns, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 1, "title": "Welcome Campaign", "enabled": true},
			{"id": 2, "title": "Exit Intent", "enabled": false}
		]`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Inboxes().Campaigns(context.Background(), 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("Expected 2 campaigns, got %d", len(result))
	}
}

func TestSyncInboxTemplates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes/1/sync_templates" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes/1/sync_templates, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.Inboxes().SyncTemplates(context.Background(), 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetInboxHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes/1/health" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes/1/health, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "healthy", "last_checked": "2025-01-01T12:00:00Z"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Inboxes().Health(context.Background(), 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", result["status"])
	}
}

func TestDeleteInboxAvatar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes/1/avatar" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes/1/avatar, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.Inboxes().DeleteAvatar(context.Background(), 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetInboxCSATTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes/1/csat_template" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes/1/csat_template, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1, "question": "How was your experience?", "message": "Thank you for your feedback!"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Inboxes().CSATTemplate(context.Background(), 1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}
	if result.Question != "How was your experience?" {
		t.Errorf("Expected question 'How was your experience?', got %s", result.Question)
	}
	if result.Message != "Thank you for your feedback!" {
		t.Errorf("Expected message 'Thank you for your feedback!', got %s", result.Message)
	}
}

func TestCreateInboxCSATTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/accounts/1/inboxes/1/csat_template" {
			t.Errorf("Expected path /api/v1/accounts/1/inboxes/1/csat_template, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1, "question": "Rate us!", "message": "Thanks!"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Inboxes().CreateCSATTemplate(context.Background(), 1, "Rate us!", "Thanks!")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result.Question != "Rate us!" {
		t.Errorf("Expected question 'Rate us!', got %s", result.Question)
	}
	if result.Message != "Thanks!" {
		t.Errorf("Expected message 'Thanks!', got %s", result.Message)
	}
}
