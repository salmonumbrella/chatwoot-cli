package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestInboxesListCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 48, "name": "Store LINE", "channel_type": "Channel::Api", "avatar_url": "https://long-url", "greeting_enabled": false, "enable_auto_assignment": true}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "list", "--li"})
		if err != nil {
			t.Fatalf("inboxes list --li failed: %v", err)
		}
	})

	var payload struct {
		Items []struct {
			ID          int    `json:"id"`
			Name        string `json:"nm"`
			ChannelType string `json:"ch"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light JSON: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	item := payload.Items[0]
	if item.ID != 48 || item.Name != "Store LINE" || item.ChannelType != "Channel::Api" {
		t.Fatalf("unexpected light item: %+v", item)
	}
	if strings.Contains(output, "avatar_url") {
		t.Fatal("light output should not contain avatar_url")
	}
	if strings.Contains(output, "greeting_enabled") {
		t.Fatal("light output should not contain greeting_enabled")
	}
}

func TestAgentsListCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 1, "name": "Agent Smith", "email": "smith@example.com", "role": "agent", "availability_status": "online", "thumbnail": "https://thumb.jpg"}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"agents", "list", "--li"})
		if err != nil {
			t.Fatalf("agents list --li failed: %v", err)
		}
	})

	var payload struct {
		Items []struct {
			ID    int    `json:"id"`
			Name  string `json:"nm"`
			Avail string `json:"av"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light JSON: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	item := payload.Items[0]
	if item.ID != 1 || item.Name != "Agent Smith" || item.Avail != "online" {
		t.Fatalf("unexpected light item: %+v", item)
	}
	if strings.Contains(output, `"email"`) {
		t.Fatal("light output should not contain email")
	}
	if strings.Contains(output, "thumbnail") {
		t.Fatal("light output should not contain thumbnail")
	}
}

func TestTeamsListCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams", jsonResponse(200, `[
			{"id": 5, "name": "Support", "description": "Support team", "allow_auto_assign": true, "account_id": 1}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"teams", "list", "--li"})
		if err != nil {
			t.Fatalf("teams list --li failed: %v", err)
		}
	})

	var payload struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"nm"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light JSON: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	item := payload.Items[0]
	if item.ID != 5 || item.Name != "Support" {
		t.Fatalf("unexpected light item: %+v", item)
	}
	if strings.Contains(output, "description") {
		t.Fatal("light output should not contain description")
	}
	if strings.Contains(output, "account_id") {
		t.Fatal("light output should not contain account_id")
	}
}

func TestLabelsListCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{
			"payload": [
				{"id": 10, "title": "urgent", "color": "#FF0000", "description": "High priority", "show_on_sidebar": true}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"labels", "list", "--li"})
		if err != nil {
			t.Fatalf("labels list --li failed: %v", err)
		}
	})

	var payload struct {
		Items []struct {
			ID    int    `json:"id"`
			Title string `json:"t"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light JSON: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	item := payload.Items[0]
	if item.ID != 10 || item.Title != "urgent" {
		t.Fatalf("unexpected light item: %+v", item)
	}
	if strings.Contains(output, "color") {
		t.Fatal("light output should not contain color")
	}
	if strings.Contains(output, "show_on_sidebar") {
		t.Fatal("light output should not contain show_on_sidebar")
	}
}

func TestCannedResponsesListCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 2, "short_code": "greeting", "content": "Hello, how can I help?", "account_id": 1}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"canned-responses", "list", "--li"})
		if err != nil {
			t.Fatalf("canned-responses list --li failed: %v", err)
		}
	})

	var payload struct {
		Items []struct {
			ID   int    `json:"id"`
			Code string `json:"code"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light JSON: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	item := payload.Items[0]
	if item.ID != 2 || item.Code != "greeting" {
		t.Fatalf("unexpected light item: %+v", item)
	}
	if strings.Contains(output, `"content"`) {
		t.Fatal("light output should not contain content")
	}
	if strings.Contains(output, "account_id") {
		t.Fatal("light output should not contain account_id")
	}
}

func TestAutomationRulesListCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/automation_rules", jsonResponse(200, `{
			"payload": [
				{"id": 3, "name": "Welcome Message", "event_name": "message_created", "active": true, "description": "Auto welcome", "conditions": [], "actions": [], "account_id": 1}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"automation-rules", "list", "--li"})
		if err != nil {
			t.Fatalf("automation-rules list --li failed: %v", err)
		}
	})

	var payload struct {
		Items []struct {
			ID     int    `json:"id"`
			Name   string `json:"nm"`
			Event  string `json:"ev"`
			Active bool   `json:"on"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light JSON: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	item := payload.Items[0]
	if item.ID != 3 || item.Name != "Welcome Message" || item.Event != "message_created" || !item.Active {
		t.Fatalf("unexpected light item: %+v", item)
	}
	if strings.Contains(output, `"description"`) {
		t.Fatal("light output should not contain description")
	}
	if strings.Contains(output, "conditions") {
		t.Fatal("light output should not contain conditions")
	}
}

func TestIntegrationsAppsCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/integrations/apps", jsonResponse(200, `{
			"payload": [
				{"id": "slack", "name": "Slack", "description": "Slack integration", "enabled": true, "hooks": []}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"integrations", "apps", "--li"})
		if err != nil {
			t.Fatalf("integrations apps --li failed: %v", err)
		}
	})

	var payload struct {
		Items []struct {
			ID      string `json:"id"`
			Name    string `json:"nm"`
			Enabled bool   `json:"on"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light JSON: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	item := payload.Items[0]
	if item.ID != "slack" || item.Name != "Slack" || !item.Enabled {
		t.Fatalf("unexpected light item: %+v", item)
	}
	if strings.Contains(output, `"description"`) {
		t.Fatal("light output should not contain description")
	}
	if strings.Contains(output, "hooks") {
		t.Fatal("light output should not contain hooks")
	}
}

func TestCustomFiltersListCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_filters", jsonResponse(200, `[
			{"id": 7, "name": "Open Conversations", "filter_type": "conversation", "query": {"status": "open"}}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"custom-filters", "list", "--li"})
		if err != nil {
			t.Fatalf("custom-filters list --li failed: %v", err)
		}
	})

	var payload struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"nm"`
			Type string `json:"type"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light JSON: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(payload.Items))
	}
	item := payload.Items[0]
	if item.ID != 7 || item.Name != "Open Conversations" || item.Type != "conversation" {
		t.Fatalf("unexpected light item: %+v", item)
	}
	if strings.Contains(output, "query") {
		t.Fatal("light output should not contain query")
	}
}

func TestConversationsGetCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 48,
			"contact_id": 456,
			"status": "open",
			"unread_count": 3,
			"last_activity_at": 1700005000,
			"muted": false,
			"custom_attributes": {"tier": "gold"},
			"meta": {
				"sender": {"id": 456, "name": "Jane Doe", "email": "jane@example.com"},
				"assignee": {"id": 7, "name": "Agent Smith"}
			},
			"last_non_activity_message": {"content": "Need help with order"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "--li"})
		if err != nil {
			t.Fatalf("conversations get --li failed: %v", err)
		}
	})

	var item struct {
		ID          int    `json:"id"`
		Status      string `json:"st"`
		InboxID     int    `json:"ib"`
		UnreadCount int    `json:"ur"`
		Contact     *struct {
			ID   *int    `json:"id"`
			Name *string `json:"nm"`
		} `json:"ct"`
		Assignee *struct {
			ID   *int    `json:"id"`
			Name *string `json:"nm"`
		} `json:"ag"`
		LastMessage *string `json:"lm"`
	}
	if err := json.Unmarshal([]byte(output), &item); err != nil {
		t.Fatalf("failed to parse light JSON: %v\noutput: %s", err, output)
	}
	if item.ID != 123 || item.Status != "o" || item.InboxID != 48 || item.UnreadCount != 3 {
		t.Fatalf("unexpected light item: %+v", item)
	}
	if item.Contact == nil || item.Contact.ID == nil || *item.Contact.ID != 456 {
		t.Fatalf("expected contact id 456, got %+v", item.Contact)
	}
	if item.Assignee == nil || item.Assignee.Name == nil || *item.Assignee.Name != "Agent Smith" {
		t.Fatalf("expected assignee name Agent Smith, got %+v", item.Assignee)
	}
	if item.LastMessage == nil || *item.LastMessage != "Need help with order" {
		t.Fatalf("expected last_msg, got %v", item.LastMessage)
	}
	if strings.Contains(output, "custom_attributes") {
		t.Fatal("light output should not contain custom_attributes")
	}
	if strings.Contains(output, "muted") {
		t.Fatal("light output should not contain muted")
	}
}
