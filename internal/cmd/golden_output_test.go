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

func TestGoldenLabelsCreateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/labels", jsonResponse(200, `{
			"id": 301,
			"title": "Urgent",
			"description": "High priority issues",
			"color": "#ff0000",
			"show_on_sidebar": true
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"labels", "create", "--title", "Urgent", "--color", "#ff0000", "--description", "High priority issues", "--show-on-sidebar", "-o", "json"}); err != nil {
			t.Fatalf("labels create failed: %v", err)
		}
	})

	assertGolden(t, "labels_create.json", output)
}

func TestGoldenLabelsUpdateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/labels/301", jsonResponse(200, `{
			"id": 301,
			"title": "Urgent",
			"description": "High priority issues",
			"color": "#00ff00",
			"show_on_sidebar": true
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"labels", "update", "301", "--color", "#00ff00", "-o", "json"}); err != nil {
			t.Fatalf("labels update failed: %v", err)
		}
	})

	assertGolden(t, "labels_update.json", output)
}

func TestGoldenLabelsDeleteJSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/labels/301", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"labels", "delete", "301", "-o", "json"}); err != nil {
			t.Fatalf("labels delete failed: %v", err)
		}
	})

	assertGolden(t, "labels_delete.json", output)
}

func TestGoldenContactsCreateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts", jsonResponse(200, `{
			"payload": {
				"contact": {
					"id": 4001,
					"name": "Jane Doe",
					"email": "jane@example.com",
					"phone_number": "+15555550123",
					"created_at": 1700007000
				},
				"contact_inbox": {}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"contacts", "create", "--name", "Jane Doe", "--email", "jane@example.com", "--phone", "+15555550123", "-o", "json"}); err != nil {
			t.Fatalf("contacts create failed: %v", err)
		}
	})

	assertGolden(t, "contacts_create.json", output)
}

func TestGoldenContactsUpdateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/contacts/4001", jsonResponse(200, `{
			"payload": {
				"id": 4001,
				"name": "Jane Doe",
				"email": "jane@example.com",
				"phone_number": "+15555550123",
				"created_at": 1700007000
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"contacts", "update", "4001", "--email", "jane@example.com", "-o", "json"}); err != nil {
			t.Fatalf("contacts update failed: %v", err)
		}
	})

	assertGolden(t, "contacts_update.json", output)
}

func TestGoldenContactsDeleteJSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/contacts/4001", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"contacts", "delete", "4001", "-o", "json"}); err != nil {
			t.Fatalf("contacts delete failed: %v", err)
		}
	})

	assertGolden(t, "contacts_delete.json", output)
}

func TestGoldenTeamsCreateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/teams", jsonResponse(200, `{
			"id": 501,
			"name": "Support",
			"description": "Tier 1",
			"allow_auto_assign": true,
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"teams", "create", "--name", "Support", "--description", "Tier 1", "-o", "json"}); err != nil {
			t.Fatalf("teams create failed: %v", err)
		}
	})

	assertGolden(t, "teams_create.json", output)
}

func TestGoldenTeamsUpdateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/teams/501", jsonResponse(200, `{
			"id": 501,
			"name": "Support",
			"description": "Tier 1 - updated",
			"allow_auto_assign": true,
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"teams", "update", "501", "--description", "Tier 1 - updated", "-o", "json"}); err != nil {
			t.Fatalf("teams update failed: %v", err)
		}
	})

	assertGolden(t, "teams_update.json", output)
}

func TestGoldenTeamsDeleteJSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/teams/501", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"teams", "delete", "501", "-o", "json"}); err != nil {
			t.Fatalf("teams delete failed: %v", err)
		}
	})

	assertGolden(t, "teams_delete.json", output)
}

func TestGoldenWebhooksCreateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/webhooks", jsonResponse(200, `{
			"payload": {
				"webhook": {
					"id": 601,
					"url": "https://example.com/webhook",
					"subscriptions": ["conversation_created", "message_created"],
					"account_id": 1
				}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"webhooks", "create", "--url", "https://example.com/webhook", "--subscriptions", "conversation_created,message_created", "-o", "json"}); err != nil {
			t.Fatalf("webhooks create failed: %v", err)
		}
	})

	assertGolden(t, "webhooks_create.json", output)
}

func TestGoldenWebhooksUpdateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/webhooks/601", jsonResponse(200, `{
			"payload": {
				"webhook": {
					"id": 601,
					"url": "https://example.com/webhook-updated",
					"subscriptions": ["message_created"],
					"account_id": 1
				}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"webhooks", "update", "601", "--url", "https://example.com/webhook-updated", "-o", "json"}); err != nil {
			t.Fatalf("webhooks update failed: %v", err)
		}
	})

	assertGolden(t, "webhooks_update.json", output)
}

func TestGoldenWebhooksDeleteJSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/webhooks/601", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"webhooks", "delete", "601", "-o", "json"}); err != nil {
			t.Fatalf("webhooks delete failed: %v", err)
		}
	})

	assertGolden(t, "webhooks_delete.json", output)
}

func TestGoldenCampaignsCreateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/campaigns", jsonResponse(200, `{
			"id": 701,
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
		if err := Execute(context.Background(), []string{"campaigns", "create", "--title", "Promo", "--message", "Hello!", "--inbox-id", "5", "-o", "json"}); err != nil {
			t.Fatalf("campaigns create failed: %v", err)
		}
	})

	assertGolden(t, "campaigns_create.json", output)
}

func TestGoldenCampaignsUpdateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/campaigns/701", jsonResponse(200, `{
			"id": 701,
			"title": "Promo Updated",
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
		if err := Execute(context.Background(), []string{"campaigns", "update", "701", "--title", "Promo Updated", "-o", "json"}); err != nil {
			t.Fatalf("campaigns update failed: %v", err)
		}
	})

	assertGolden(t, "campaigns_update.json", output)
}

func TestGoldenCampaignsDeleteJSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/campaigns/701", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"campaigns", "delete", "701", "--force", "-o", "json"}); err != nil {
			t.Fatalf("campaigns delete failed: %v", err)
		}
	})

	assertGolden(t, "campaigns_delete.json", output)
}

func TestGoldenAgentBotsListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agent_bots", jsonResponse(200, `[
			{"id": 801, "name": "Helper", "outgoing_url": "https://example.com/bot", "account_id": 1},
			{"id": 802, "name": "Triage", "outgoing_url": "https://example.com/triage", "account_id": 1}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"agent-bots", "list", "-o", "json"}); err != nil {
			t.Fatalf("agent-bots list failed: %v", err)
		}
	})

	assertGolden(t, "agent_bots_list.json", output)
}

func TestGoldenAgentBotsGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agent_bots/802", jsonResponse(200, `{
			"id": 802,
			"name": "Triage",
			"description": "Routes conversations",
			"outgoing_url": "https://example.com/triage",
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"agent-bots", "get", "802", "-o", "json"}); err != nil {
			t.Fatalf("agent-bots get failed: %v", err)
		}
	})

	assertGolden(t, "agent_bots_get.json", output)
}

func TestGoldenAgentBotsCreateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/agent_bots", jsonResponse(200, `{
			"id": 803,
			"name": "AutoResponder",
			"description": "Handles FAQs",
			"outgoing_url": "https://example.com/auto",
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"agent-bots", "create", "--name", "AutoResponder", "--outgoing-url", "https://example.com/auto", "-o", "json"}); err != nil {
			t.Fatalf("agent-bots create failed: %v", err)
		}
	})

	assertGolden(t, "agent_bots_create.json", output)
}

func TestGoldenAgentBotsUpdateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/agent_bots/803", jsonResponse(200, `{
			"id": 803,
			"name": "AutoResponder",
			"description": "Handles FAQs",
			"outgoing_url": "https://example.com/auto-updated",
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"agent-bots", "update", "803", "--outgoing-url", "https://example.com/auto-updated", "-o", "json"}); err != nil {
			t.Fatalf("agent-bots update failed: %v", err)
		}
	})

	assertGolden(t, "agent_bots_update.json", output)
}

func TestGoldenAgentBotsDeleteJSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/agent_bots/803", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"agent-bots", "delete", "803", "-o", "json"}); err != nil {
			t.Fatalf("agent-bots delete failed: %v", err)
		}
	})

	assertGolden(t, "agent_bots_delete.json", output)
}

func TestGoldenAutomationRulesListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/automation_rules", jsonResponse(200, `{
			"payload": [
				{"id": 901, "name": "Auto assign", "event_name": "conversation_created", "active": true, "conditions": [], "actions": [], "account_id": 1}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"automation-rules", "list", "-o", "json"}); err != nil {
			t.Fatalf("automation-rules list failed: %v", err)
		}
	})

	assertGolden(t, "automation_rules_list.json", output)
}

func TestGoldenAutomationRulesGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/automation_rules/901", jsonResponse(200, `{
			"payload": {
				"id": 901,
				"name": "Auto assign",
				"event_name": "conversation_created",
				"conditions": [{"attribute_key":"status","filter_operator":"equals","values":["open"]}],
				"actions": [{"action_name":"assign_team","action_params":{"team_id":3}}],
				"active": true,
				"account_id": 1
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"automation-rules", "get", "901", "-o", "json"}); err != nil {
			t.Fatalf("automation-rules get failed: %v", err)
		}
	})

	assertGolden(t, "automation_rules_get.json", output)
}

func TestGoldenAutomationRulesCreateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/automation_rules", jsonResponse(200, `{
			"id": 902,
			"name": "Auto tag",
			"event_name": "conversation_created",
			"conditions": [{"attribute_key":"priority","filter_operator":"equals","values":["high"]}],
			"actions": [{"action_name":"add_label","action_params":{"labels":["vip"]}}],
			"active": true,
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"automation-rules", "create", "--name", "Auto tag", "--event-name", "conversation_created", "--conditions", "[{\"attribute_key\":\"priority\",\"filter_operator\":\"equals\",\"values\":[\"high\"]}]", "--actions", "[{\"action_name\":\"add_label\",\"action_params\":{\"labels\":[\"vip\"]}}]", "-o", "json"}); err != nil {
			t.Fatalf("automation-rules create failed: %v", err)
		}
	})

	assertGolden(t, "automation_rules_create.json", output)
}

func TestGoldenAutomationRulesUpdateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/automation_rules/902", jsonResponse(200, `{
			"payload": {
				"id": 902,
				"name": "Auto tag updated",
				"event_name": "conversation_created",
				"conditions": [{"attribute_key":"priority","filter_operator":"equals","values":["high"]}],
				"actions": [{"action_name":"add_label","action_params":{"labels":["vip"]}}],
				"active": true,
				"account_id": 1
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"automation-rules", "update", "902", "--name", "Auto tag updated", "--conditions", "[{\"attribute_key\":\"priority\",\"filter_operator\":\"equals\",\"values\":[\"high\"]}]", "--actions", "[{\"action_name\":\"add_label\",\"action_params\":{\"labels\":[\"vip\"]}}]", "-o", "json"}); err != nil {
			t.Fatalf("automation-rules update failed: %v", err)
		}
	})

	assertGolden(t, "automation_rules_update.json", output)
}

func TestGoldenAutomationRulesDeleteJSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/automation_rules/902", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"automation-rules", "delete", "902", "-o", "json"}); err != nil {
			t.Fatalf("automation-rules delete failed: %v", err)
		}
	})

	assertGolden(t, "automation_rules_delete.json", output)
}

func TestGoldenCannedResponsesListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 1001, "short_code": "hi", "content": "Hello!", "account_id": 1},
			{"id": 1002, "short_code": "bye", "content": "Goodbye!", "account_id": 1}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"canned-responses", "list", "-o", "json"}); err != nil {
			t.Fatalf("canned-responses list failed: %v", err)
		}
	})

	assertGolden(t, "canned_responses_list.json", output)
}

func TestGoldenCannedResponsesGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 1001, "short_code": "hi", "content": "Hello!", "account_id": 1},
			{"id": 1002, "short_code": "bye", "content": "Goodbye!", "account_id": 1}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"canned-responses", "get", "1002", "-o", "json"}); err != nil {
			t.Fatalf("canned-responses get failed: %v", err)
		}
	})

	assertGolden(t, "canned_responses_get.json", output)
}

func TestGoldenCannedResponsesCreateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `{
			"id": 1003,
			"short_code": "thanks",
			"content": "Thanks for reaching out!",
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"canned-responses", "create", "--short-code", "thanks", "--content", "Thanks for reaching out!", "-o", "json"}); err != nil {
			t.Fatalf("canned-responses create failed: %v", err)
		}
	})

	assertGolden(t, "canned_responses_create.json", output)
}

func TestGoldenCannedResponsesUpdateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 1003, "short_code": "thanks", "content": "Thanks for reaching out!", "account_id": 1}
		]`)).
		On("PATCH", "/api/v1/accounts/1/canned_responses/1003", jsonResponse(200, `{
			"id": 1003,
			"short_code": "thanks",
			"content": "Thanks!",
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"canned-responses", "update", "1003", "--content", "Thanks!", "-o", "json"}); err != nil {
			t.Fatalf("canned-responses update failed: %v", err)
		}
	})

	assertGolden(t, "canned_responses_update.json", output)
}

func TestGoldenCannedResponsesDeleteJSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/canned_responses/1003", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"canned-responses", "delete", "1003", "-o", "json"}); err != nil {
			t.Fatalf("canned-responses delete failed: %v", err)
		}
	})

	assertGolden(t, "canned_responses_delete.json", output)
}

func TestGoldenCustomAttributesListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_attribute_definitions", jsonResponse(200, `[
			{"id": 1101, "attribute_display_name": "Plan", "attribute_key": "plan", "attribute_model": "contact_attribute", "attribute_display_type": "text"},
			{"id": 1102, "attribute_display_name": "Tier", "attribute_key": "tier", "attribute_model": "contact_attribute", "attribute_display_type": "list", "attribute_values": ["gold", "silver"]}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-attributes", "list", "-o", "json"}); err != nil {
			t.Fatalf("custom-attributes list failed: %v", err)
		}
	})

	assertGolden(t, "custom_attributes_list.json", output)
}

func TestGoldenCustomAttributesGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_attribute_definitions/1102", jsonResponse(200, `{
			"id": 1102,
			"attribute_display_name": "Tier",
			"attribute_key": "tier",
			"attribute_model": "contact_attribute",
			"attribute_display_type": "list",
			"attribute_values": ["gold", "silver"]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-attributes", "get", "1102", "-o", "json"}); err != nil {
			t.Fatalf("custom-attributes get failed: %v", err)
		}
	})

	assertGolden(t, "custom_attributes_get.json", output)
}

func TestGoldenCustomAttributesCreateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/custom_attribute_definitions", jsonResponse(200, `{
			"id": 1103,
			"attribute_display_name": "Segment",
			"attribute_key": "segment",
			"attribute_model": "contact_attribute",
			"attribute_display_type": "text"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-attributes", "create", "--name", "Segment", "--model", "contact", "--type", "text", "--key", "segment", "-o", "json"}); err != nil {
			t.Fatalf("custom-attributes create failed: %v", err)
		}
	})

	assertGolden(t, "custom_attributes_create.json", output)
}

func TestGoldenCustomAttributesUpdateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/custom_attribute_definitions/1103", jsonResponse(200, `{
			"id": 1103,
			"attribute_display_name": "Segment Updated",
			"attribute_key": "segment",
			"attribute_model": "contact_attribute",
			"attribute_display_type": "text"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-attributes", "update", "1103", "--name", "Segment Updated", "-o", "json"}); err != nil {
			t.Fatalf("custom-attributes update failed: %v", err)
		}
	})

	assertGolden(t, "custom_attributes_update.json", output)
}

func TestGoldenCustomAttributesDeleteJSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/custom_attribute_definitions/1103", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-attributes", "delete", "1103", "-o", "json"}); err != nil {
			t.Fatalf("custom-attributes delete failed: %v", err)
		}
	})

	assertGolden(t, "custom_attributes_delete.json", output)
}

func TestGoldenCustomFiltersListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters", jsonResponse(200, `[
			{"id": 1201, "name": "Open", "filter_type": "conversation", "query": {"status":"open"}},
			{"id": 1202, "name": "VIP", "filter_type": "contact", "query": {"custom_attribute":"vip"}}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-filters", "list", "-o", "json"}); err != nil {
			t.Fatalf("custom-filters list failed: %v", err)
		}
	})

	assertGolden(t, "custom_filters_list.json", output)
}

func TestGoldenCustomFiltersGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters/1202", jsonResponse(200, `{
			"id": 1202,
			"name": "VIP",
			"filter_type": "contact",
			"query": {"custom_attribute":"vip"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-filters", "get", "1202", "-o", "json"}); err != nil {
			t.Fatalf("custom-filters get failed: %v", err)
		}
	})

	assertGolden(t, "custom_filters_get.json", output)
}

func TestGoldenCustomFiltersCreateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/custom_filters", jsonResponse(200, `{
			"id": 1203,
			"name": "Pending",
			"filter_type": "conversation",
			"query": {"status":"pending"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-filters", "create", "--name", "Pending", "--type", "conversation", "--query", "{\"status\":\"pending\"}", "-o", "json"}); err != nil {
			t.Fatalf("custom-filters create failed: %v", err)
		}
	})

	assertGolden(t, "custom_filters_create.json", output)
}

func TestGoldenCustomFiltersUpdateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/custom_filters/1203", jsonResponse(200, `{
			"id": 1203,
			"name": "Pending Updated",
			"filter_type": "conversation",
			"query": {"status":"pending"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-filters", "update", "1203", "--name", "Pending Updated", "-o", "json"}); err != nil {
			t.Fatalf("custom-filters update failed: %v", err)
		}
	})

	assertGolden(t, "custom_filters_update.json", output)
}

func TestGoldenCustomFiltersDeleteJSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/custom_filters/1203", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-filters", "delete", "1203", "-o", "json"}); err != nil {
			t.Fatalf("custom-filters delete failed: %v", err)
		}
	})

	assertGolden(t, "custom_filters_delete.json", output)
}

func TestGoldenPortalsListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals", jsonResponse(200, `{
			"payload": [
				{"id": 1301, "name": "Help Center", "slug": "help-center", "account_id": 1},
				{"id": 1302, "name": "Docs", "slug": "docs", "account_id": 1}
			],
			"meta": {
				"current_page": 1,
				"portals_count": 2
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"portals", "list", "-o", "json"}); err != nil {
			t.Fatalf("portals list failed: %v", err)
		}
	})

	assertGolden(t, "portals_list.json", output)
}

func TestGoldenPortalsGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals/help-center", jsonResponse(200, `{
			"id": 1301,
			"name": "Help Center",
			"slug": "help-center",
			"custom_domain": "help.example.com",
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"portals", "get", "help-center", "-o", "json"}); err != nil {
			t.Fatalf("portals get failed: %v", err)
		}
	})

	assertGolden(t, "portals_get.json", output)
}

func TestGoldenPortalsCreateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/portals", jsonResponse(200, `{
			"id": 1303,
			"name": "Support",
			"slug": "support",
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"portals", "create", "--name", "Support", "--slug", "support", "-o", "json"}); err != nil {
			t.Fatalf("portals create failed: %v", err)
		}
	})

	assertGolden(t, "portals_create.json", output)
}

func TestGoldenPortalsUpdateJSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/portals/help-center", jsonResponse(200, `{
			"id": 1301,
			"name": "Help Center Updated",
			"slug": "help-center",
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"portals", "update", "help-center", "--name", "Help Center Updated", "-o", "json"}); err != nil {
			t.Fatalf("portals update failed: %v", err)
		}
	})

	assertGolden(t, "portals_update.json", output)
}

func TestGoldenPortalsDeleteJSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/portals/help-center", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"portals", "delete", "help-center", "-o", "json"}); err != nil {
			t.Fatalf("portals delete failed: %v", err)
		}
	})

	assertGolden(t, "portals_delete.json", output)
}
