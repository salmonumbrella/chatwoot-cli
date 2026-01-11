package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func assertGolden(t *testing.T, name string, got string) {
	t.Helper()

	path := filepath.Join("testdata", "golden", name)
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("failed to create golden directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}
	if string(want) != got {
		t.Fatalf("golden output mismatch for %s (set UPDATE_GOLDEN=1 to update)", name)
	}
}

func TestGoldenLabelsListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{
			"payload": [
				{"id": 1, "title": "Bug", "description": "Bug reports", "color": "#ff0000", "show_on_sidebar": true},
				{"id": 2, "title": "Feature", "color": "#00ff00", "show_on_sidebar": false}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"labels", "list", "-o", "json"}); err != nil {
			t.Fatalf("labels list failed: %v", err)
		}
	})

	assertGolden(t, "labels_list.json", output)
}

func TestGoldenLabelsGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels/123", jsonResponse(200, `{
			"id": 123,
			"title": "Important",
			"description": "Important issues",
			"color": "#0000ff",
			"show_on_sidebar": true
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"labels", "get", "123", "-o", "json"}); err != nil {
			t.Fatalf("labels get failed: %v", err)
		}
	})

	assertGolden(t, "labels_get.json", output)
}

func TestGoldenInboxesListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{
					"id": 10,
					"name": "Support",
					"channel_type": "website",
					"greeting_enabled": false,
					"enable_auto_assignment": true
				},
				{
					"id": 11,
					"name": "Sales",
					"channel_type": "email",
					"greeting_enabled": true,
					"greeting_message": "Hi there",
					"enable_auto_assignment": false
				}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"inboxes", "list", "-o", "json"}); err != nil {
			t.Fatalf("inboxes list failed: %v", err)
		}
	})

	assertGolden(t, "inboxes_list.json", output)
}

func TestGoldenConversationsListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, `{
			"data": {
				"meta": {
					"current_page": 1,
					"total_pages": 1
				},
				"payload": [
					{
						"id": 101,
						"account_id": 1,
						"inbox_id": 10,
						"status": "open",
						"contact_id": 501,
						"muted": false,
						"unread_count": 2,
						"created_at": 1700000000,
						"last_activity_at": 1700000500
					}
				]
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"conversations", "list", "--status", "open", "-o", "json"}); err != nil {
			t.Fatalf("conversations list failed: %v", err)
		}
	})

	assertGolden(t, "conversations_list.json", output)
}

func TestGoldenContactsListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts", jsonResponse(200, `{
			"payload": [
				{
					"id": 2001,
					"name": "Ada Lovelace",
					"email": "ada@example.com",
					"phone_number": "+15555550100",
					"created_at": 1700001000
				},
				{
					"id": 2002,
					"name": "Grace Hopper",
					"created_at": 1700002000
				}
			],
			"meta": {
				"current_page": 1,
				"total_pages": 1
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"contacts", "list", "-o", "json"}); err != nil {
			t.Fatalf("contacts list failed: %v", err)
		}
	})

	assertGolden(t, "contacts_list.json", output)
}

func TestGoldenMessagesListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{
					"id": 9001,
					"conversation_id": 123,
					"content": "Hello there",
					"content_type": "text",
					"message_type": 1,
					"private": false,
					"created_at": 1700003000
				},
				{
					"id": 9002,
					"conversation_id": 123,
					"content": "Internal note",
					"content_type": "text",
					"message_type": 2,
					"private": true,
					"created_at": 1700004000
				}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"messages", "list", "123", "-o", "json"}); err != nil {
			t.Fatalf("messages list failed: %v", err)
		}
	})

	assertGolden(t, "messages_list.json", output)
}
