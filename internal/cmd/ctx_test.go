package cmd

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestCtxCommand_Agent_WithURL(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"contact_id": 0,
			"status": "open",
			"inbox_id": 1,
			"created_at": 1700000000
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Hello", "message_type": 0, "private": false, "created_at": 1700000001}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"ctx", "https://app.chatwoot.com/app/accounts/1/conversations/123", "-o", "agent"})
		if err != nil {
			t.Fatalf("ctx failed: %v", err)
		}
	})

	var payload struct {
		Kind string         `json:"kind"`
		Item map[string]any `json:"item"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	if payload.Kind != "ctx" {
		t.Fatalf("expected kind ctx, got %q", payload.Kind)
	}
	if _, ok := payload.Item["messages"]; !ok {
		t.Fatalf("expected messages in payload item, got %#v", payload.Item)
	}
}

func TestCtxCommand_LightAlias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"contact_id": 456,
			"status": "open",
			"inbox_id": 48
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Hello", "message_type": 0, "private": false},
				{"id": 2, "content": "Internal note", "message_type": 1, "private": true},
				{"id": 3, "content": "Status changed", "message_type": 2, "private": false}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {
				"id": 456,
				"name": "TuTu",
				"phone_number": "+15550001111"
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"ctx", "123", "--li"})
		if err != nil {
			t.Fatalf("ctx --li failed: %v", err)
		}
	})
	if strings.Contains(output, "\n  ") {
		t.Fatalf("expected --li output to be compact by default, got pretty JSON:\n%s", output)
	}

	var payload struct {
		ID      int    `json:"id"`
		St      string `json:"st"`
		Inbox   int    `json:"ib"`
		Contact struct {
			ID   *int    `json:"id"`
			Name *string `json:"nm"`
		} `json:"ct"`
		Msgs []string `json:"msgs"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	if payload.ID != 123 || payload.St != "o" || payload.Inbox != 48 {
		t.Fatalf("unexpected light payload header: %#v", payload)
	}
	if payload.Contact.ID == nil || *payload.Contact.ID != 456 {
		t.Fatalf("expected ct.id=456, got %#v", payload.Contact.ID)
	}
	if payload.Contact.Name == nil || *payload.Contact.Name != "TuTu" {
		t.Fatalf("expected ct.nm=TuTu, got %#v", payload.Contact.Name)
	}

	if len(payload.Msgs) != 2 {
		t.Fatalf("expected 2 non-activity messages, got %d (%#v)", len(payload.Msgs), payload.Msgs)
	}
}

func TestCtxCommand_LightAlias_MetaSenderFallback(t *testing.T) {
	// Simulates real Chatwoot API behavior: contact_id is null/0 but
	// meta.sender contains the contact data.
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/789", jsonResponse(200, `{
			"id": 789,
			"contact_id": 0,
			"status": "pending",
			"inbox_id": 48,
			"meta": {
				"sender": {
					"id": 32649,
					"name": "秋萍🤔",
					"email": "t660537@gmail.com",
					"phone_number": "+8860981927778"
				}
			}
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/789/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Hello", "message_type": 0, "private": false}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"ctx", "789", "--li", "--cj"})
		if err != nil {
			t.Fatalf("ctx --li meta sender fallback failed: %v", err)
		}
	})

	var payload struct {
		ID      int    `json:"id"`
		St      string `json:"st"`
		Inbox   int    `json:"ib"`
		Contact struct {
			ID   *int    `json:"id"`
			Name *string `json:"nm"`
		} `json:"ct"`
		Msgs []string `json:"msgs"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	if payload.ID != 789 || payload.St != "p" || payload.Inbox != 48 {
		t.Fatalf("unexpected light payload header: %#v", payload)
	}
	if payload.Contact.ID == nil || *payload.Contact.ID != 32649 {
		t.Fatalf("expected ct.id=32649 from meta.sender, got %v", payload.Contact.ID)
	}
	if payload.Contact.Name == nil || *payload.Contact.Name != "秋萍🤔" {
		t.Fatalf("expected ct.nm from meta.sender, got %v", payload.Contact.Name)
	}
	if len(payload.Msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(payload.Msgs))
	}
}

func TestCtxCommand_LightAlias_WithQueryAliases(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"contact_id": 456,
			"status": "open",
			"inbox_id": 48
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "First", "message_type": 0, "private": false},
				{"id": 2, "content": "Second", "message_type": 1, "private": false},
				{"id": 3, "content": "Status changed", "message_type": 2, "private": false}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {
				"id": 456,
				"name": "TuTu"
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"ctx", "123", "--li", "--cj",
			"--jq", `{id: .id, st: .st, ib: .ib, ct: .ct, ls: .msgs[-1]}`,
		})
		if err != nil {
			t.Fatalf("ctx --li with query aliases failed: %v", err)
		}
	})

	var payload struct {
		ID      int `json:"id"`
		St      any `json:"st"`
		Inbox   int `json:"ib"`
		Contact struct {
			ID *int `json:"id"`
		} `json:"ct"`
		LS string `json:"ls"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	if payload.ID != 123 {
		t.Fatalf("expected id=123, got %d", payload.ID)
	}
	if payload.Inbox != 48 {
		t.Fatalf("expected ib=48, got %d", payload.Inbox)
	}
	if payload.Contact.ID == nil || *payload.Contact.ID != 456 {
		t.Fatalf("expected ct.id=456, got %#v", payload.Contact.ID)
	}
	if payload.LS != "> Second" {
		t.Fatalf("expected ls=Second, got %q", payload.LS)
	}
}

func TestCtxCommand_LightAlias_NoAliasExpansion(t *testing.T) {
	// Verify that jq queries on light output use literal keys,
	// not query alias expansion (st should NOT become status).
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/100", jsonResponse(200, `{
			"id": 100,
			"contact_id": 0,
			"status": "pending",
			"inbox_id": 48
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/100/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Test", "message_type": 0, "private": false}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"ctx", "100", "--li", "--cj", "--jq", ".st"})
		if err != nil {
			t.Fatalf("ctx --li --jq .st failed: %v", err)
		}
	})

	output = strings.TrimSpace(output)
	if output != `"p"` {
		t.Fatalf("expected jq .st to return \"p\" (short status), got %q — alias expansion may have changed .st to .status", output)
	}
}
