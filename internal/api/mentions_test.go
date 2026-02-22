package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFindMentions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data": {
					"meta": {"total_pages": 1, "current_page": 1},
					"payload": [{"id": 100, "status": "open", "last_activity_at": 1704153600}]
				}
			}`))
		case "/api/v1/accounts/1/conversations/100/messages":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"payload": [
					{
						"id": 1,
						"content": "Hey mention://user/42/TestAgent check this",
						"message_type": 2,
						"private": true,
						"created_at": 1704067200,
						"sender": {"id": 10, "name": "Other Agent"}
					},
					{
						"id": 2,
						"content": "Not a private note",
						"message_type": 1,
						"private": false,
						"created_at": 1704067100
					}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "not found"}`))
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Mentions().Find(context.Background(), FindMentionsParams{
		UserID: 42,
		Limit:  50,
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("Expected 1 mention, got %d", len(result))
	}
	if result[0].ConversationID != 100 {
		t.Errorf("Expected conversation ID 100, got %d", result[0].ConversationID)
	}
	if result[0].MessageID != 1 {
		t.Errorf("Expected message ID 1, got %d", result[0].MessageID)
	}
	if result[0].SenderName != "Other Agent" {
		t.Errorf("Expected sender 'Other Agent', got %s", result[0].SenderName)
	}
}

func TestFindMentions_RequiresUserID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	_, err := client.Mentions().Find(context.Background(), FindMentionsParams{
		UserID: 0, // Zero user ID should fail
	})

	if err == nil {
		t.Error("Expected error for zero user ID")
	}
}

func TestFindMentions_WithConversationID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Should only hit the messages endpoint for conversation 123
		if r.URL.Path == "/api/v1/accounts/1/conversations/123/messages" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"payload": [
					{
						"id": 1,
						"content": "mention://user/42/TestAgent hello",
						"message_type": 2,
						"private": true,
						"created_at": 1704067200,
						"sender": {"id": 10, "name": "Sender"}
					}
				]
			}`))
			return
		}

		// Should NOT hit the conversations list endpoint
		if r.URL.Path == "/api/v1/accounts/1/conversations" {
			t.Error("Should not list all conversations when conversation ID is provided")
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Mentions().Find(context.Background(), FindMentionsParams{
		UserID:         42,
		ConversationID: 123,
		Limit:          50,
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 mention, got %d", len(result))
	}
}

func TestFindMentions_WithSinceFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data": {
					"meta": {"total_pages": 1},
					"payload": [{"id": 100, "status": "open", "last_activity_at": 1704153600}]
				}
			}`))
		case "/api/v1/accounts/1/conversations/100/messages":
			w.WriteHeader(http.StatusOK)
			// Two mentions: one old, one new
			_, _ = w.Write([]byte(`{
				"payload": [
					{
						"id": 1,
						"content": "mention://user/42/Test old mention",
						"message_type": 2,
						"private": true,
						"created_at": 1704067200,
						"sender": {"id": 10, "name": "Old"}
					},
					{
						"id": 2,
						"content": "mention://user/42/Test new mention",
						"message_type": 2,
						"private": true,
						"created_at": 1704240000,
						"sender": {"id": 11, "name": "New"}
					}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)

	// Set since to filter out the old mention
	sinceTime := time.Unix(1704150000, 0) // Between old and new
	result, err := client.Mentions().Find(context.Background(), FindMentionsParams{
		UserID: 42,
		Since:  &sinceTime,
		Limit:  50,
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should only get the new mention
	if len(result) != 1 {
		t.Errorf("Expected 1 mention (filtered), got %d", len(result))
	}
	if len(result) > 0 && result[0].MessageID != 2 {
		t.Errorf("Expected message ID 2 (new mention), got %d", result[0].MessageID)
	}
}

func TestFindMentions_Limit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data": {
					"meta": {"total_pages": 1},
					"payload": [{"id": 100, "status": "open", "last_activity_at": 1704153600}]
				}
			}`))
		case "/api/v1/accounts/1/conversations/100/messages":
			w.WriteHeader(http.StatusOK)
			// Return multiple mentions
			_, _ = w.Write([]byte(`{
				"payload": [
					{"id": 1, "content": "mention://user/42/Test m1", "message_type": 2, "private": true, "created_at": 1704067200, "sender": {"name": "A"}},
					{"id": 2, "content": "mention://user/42/Test m2", "message_type": 2, "private": true, "created_at": 1704067300, "sender": {"name": "B"}},
					{"id": 3, "content": "mention://user/42/Test m3", "message_type": 2, "private": true, "created_at": 1704067400, "sender": {"name": "C"}}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Mentions().Find(context.Background(), FindMentionsParams{
		UserID: 42,
		Limit:  2, // Only want 2 mentions
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 mentions (limited), got %d", len(result))
	}
}

func TestFindMentions_DefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data": {
					"meta": {"total_pages": 0},
					"payload": []
				}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	_, err := client.Mentions().Find(context.Background(), FindMentionsParams{
		UserID: 42,
		Limit:  0, // Zero limit should default to 50
	})
	if err != nil {
		t.Errorf("Unexpected error with zero limit: %v", err)
	}
}

func TestFindMentions_IgnoresNonPrivateMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data": {
					"meta": {"total_pages": 1},
					"payload": [{"id": 100, "status": "open", "last_activity_at": 1704153600}]
				}
			}`))
		case "/api/v1/accounts/1/conversations/100/messages":
			w.WriteHeader(http.StatusOK)
			// Message with mention but not private
			_, _ = w.Write([]byte(`{
				"payload": [
					{
						"id": 1,
						"content": "mention://user/42/Test public mention",
						"message_type": 1,
						"private": false,
						"created_at": 1704067200
					}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Mentions().Find(context.Background(), FindMentionsParams{
		UserID: 42,
		Limit:  50,
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should get no mentions because the message is not private
	if len(result) != 0 {
		t.Errorf("Expected 0 mentions (non-private ignored), got %d", len(result))
	}
}

func TestFindMentions_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	_, err := client.Mentions().Find(context.Background(), FindMentionsParams{
		UserID: 42,
		Limit:  50,
	})

	if err == nil {
		t.Error("Expected error for API failure")
	}
}

func TestFindMentions_UnknownSender(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/accounts/1/conversations":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"data": {
					"meta": {"total_pages": 1},
					"payload": [{"id": 100, "status": "open", "last_activity_at": 1704153600}]
				}
			}`))
		case "/api/v1/accounts/1/conversations/100/messages":
			w.WriteHeader(http.StatusOK)
			// Message without sender info
			_, _ = w.Write([]byte(`{
				"payload": [
					{
						"id": 1,
						"content": "mention://user/42/Test mention",
						"message_type": 2,
						"private": true,
						"created_at": 1704067200
					}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	result, err := client.Mentions().Find(context.Background(), FindMentionsParams{
		UserID: 42,
		Limit:  50,
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 mention, got %d", len(result))
	}

	// Should default to "Unknown" when sender is not present
	if result[0].SenderName != "Unknown" {
		t.Errorf("Expected sender 'Unknown', got %s", result[0].SenderName)
	}
}
