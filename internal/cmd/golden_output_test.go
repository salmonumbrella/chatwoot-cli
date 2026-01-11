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

func TestGoldenInboxesGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/11", jsonResponse(200, `{
			"id": 11,
			"name": "Sales",
			"channel_type": "email",
			"greeting_enabled": true,
			"greeting_message": "Hi there",
			"enable_auto_assignment": false
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"inboxes", "get", "11", "-o", "json"}); err != nil {
			t.Fatalf("inboxes get failed: %v", err)
		}
	})

	assertGolden(t, "inboxes_get.json", output)
}

func TestGoldenConversationsGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/101", jsonResponse(200, `{
			"id": 101,
			"account_id": 1,
			"inbox_id": 10,
			"status": "open",
			"priority": "high",
			"assignee_id": 7,
			"team_id": 3,
			"contact_id": 501,
			"muted": false,
			"unread_count": 2,
			"created_at": 1700000000,
			"last_activity_at": 1700000500
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"conversations", "get", "101", "-o", "json"}); err != nil {
			t.Fatalf("conversations get failed: %v", err)
		}
	})

	assertGolden(t, "conversations_get.json", output)
}

func TestGoldenContactsGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/2001", jsonResponse(200, `{
			"payload": {
				"id": 2001,
				"name": "Ada Lovelace",
				"email": "ada@example.com",
				"phone_number": "+15555550100",
				"identifier": "ada-001",
				"created_at": 1700001000
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"contacts", "get", "2001", "-o", "json"}); err != nil {
			t.Fatalf("contacts get failed: %v", err)
		}
	})

	assertGolden(t, "contacts_get.json", output)
}

func TestGoldenAgentsListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 1, "name": "Ava", "email": "ava@example.com", "role": "admin"},
			{"id": 2, "name": "Ben", "email": "ben@example.com", "role": "agent", "availability_status": "online"}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"agents", "list", "-o", "json"}); err != nil {
			t.Fatalf("agents list failed: %v", err)
		}
	})

	assertGolden(t, "agents_list.json", output)
}

func TestGoldenTeamsListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams", jsonResponse(200, `[
			{"id": 11, "name": "Support", "description": "Tier 1", "allow_auto_assign": true, "account_id": 1},
			{"id": 12, "name": "Escalations", "allow_auto_assign": false, "account_id": 1}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"teams", "list", "-o", "json"}); err != nil {
			t.Fatalf("teams list failed: %v", err)
		}
	})

	assertGolden(t, "teams_list.json", output)
}

func TestGoldenWebhooksListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{
			"payload": {
				"webhooks": [
					{"id": 21, "url": "https://example.com/a", "subscriptions": ["conversation_created"], "account_id": 1},
					{"id": 22, "url": "https://example.com/b", "subscriptions": ["message_created", "message_updated"], "account_id": 1}
				]
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"webhooks", "list", "-o", "json"}); err != nil {
			t.Fatalf("webhooks list failed: %v", err)
		}
	})

	assertGolden(t, "webhooks_list.json", output)
}

func TestGoldenCampaignsListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/campaigns", jsonResponse(200, `[
			{
				"id": 31,
				"title": "Promo",
				"description": "One-off blast",
				"message": "Hello!",
				"enabled": true,
				"campaign_type": "one_off",
				"campaign_status": "active",
				"inbox_id": 5,
				"trigger_only_during_business_hours": false,
				"created_at": 1700006000,
				"account_id": 1
			}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"campaigns", "list", "-o", "json"}); err != nil {
			t.Fatalf("campaigns list failed: %v", err)
		}
	})

	assertGolden(t, "campaigns_list.json", output)
}

func TestGoldenAgentsGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 1, "name": "Ava", "email": "ava@example.com", "role": "admin"},
			{"id": 2, "name": "Ben", "email": "ben@example.com", "role": "agent", "availability_status": "online"}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"agents", "get", "2", "-o", "json"}); err != nil {
			t.Fatalf("agents get failed: %v", err)
		}
	})

	assertGolden(t, "agents_get.json", output)
}

func TestGoldenTeamsGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams/12", jsonResponse(200, `{
			"id": 12,
			"name": "Escalations",
			"description": "Tier 2",
			"allow_auto_assign": false,
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"teams", "get", "12", "-o", "json"}); err != nil {
			t.Fatalf("teams get failed: %v", err)
		}
	})

	assertGolden(t, "teams_get.json", output)
}

func TestGoldenWebhooksGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{
			"payload": {
				"webhooks": [
					{"id": 21, "url": "https://example.com/a", "subscriptions": ["conversation_created"], "account_id": 1},
					{"id": 22, "url": "https://example.com/b", "subscriptions": ["message_created", "message_updated"], "account_id": 1}
				]
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"webhooks", "get", "22", "-o", "json"}); err != nil {
			t.Fatalf("webhooks get failed: %v", err)
		}
	})

	assertGolden(t, "webhooks_get.json", output)
}

func TestGoldenCampaignsGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/campaigns/31", jsonResponse(200, `{
			"id": 31,
			"title": "Promo",
			"description": "One-off blast",
			"message": "Hello!",
			"enabled": true,
			"campaign_type": "one_off",
			"campaign_status": "active",
			"inbox_id": 5,
			"trigger_only_during_business_hours": false,
			"created_at": 1700006000,
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"campaigns", "get", "31", "-o", "json"}); err != nil {
			t.Fatalf("campaigns get failed: %v", err)
		}
	})

	assertGolden(t, "campaigns_get.json", output)
}
