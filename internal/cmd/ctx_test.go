package cmd

import (
	"context"
	"encoding/json"
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
		err := Execute(context.Background(), []string{"ctx", "123", "--li", "--cj"})
		if err != nil {
			t.Fatalf("ctx --li failed: %v", err)
		}
	})

	var payload struct {
		ID      int    `json:"id"`
		St      string `json:"st"`
		Inbox   int    `json:"inbox"`
		Contact struct {
			ID    *int    `json:"id"`
			Name  *string `json:"name"`
			Email *string `json:"email"`
			Phone *string `json:"phone"`
		} `json:"contact"`
		Msgs []string `json:"msgs"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	if payload.ID != 123 || payload.St != "open" || payload.Inbox != 48 {
		t.Fatalf("unexpected light payload header: %#v", payload)
	}
	if payload.Contact.ID == nil || *payload.Contact.ID != 456 {
		t.Fatalf("expected contact.id=456, got %#v", payload.Contact.ID)
	}
	if payload.Contact.Name == nil || *payload.Contact.Name != "TuTu" {
		t.Fatalf("expected contact.name=TuTu, got %#v", payload.Contact.Name)
	}
	if payload.Contact.Phone == nil || *payload.Contact.Phone != "+15550001111" {
		t.Fatalf("expected contact.phone set, got %#v", payload.Contact.Phone)
	}

	if len(payload.Msgs) != 2 {
		t.Fatalf("expected 2 non-activity messages, got %d (%#v)", len(payload.Msgs), payload.Msgs)
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
			"--jq", "{i, cst, ib, ctc, ls: .mg[-1]}",
		})
		if err != nil {
			t.Fatalf("ctx --li with query aliases failed: %v", err)
		}
	})

	var payload struct {
		ID      int `json:"id"`
		St      any `json:"st"`
		Inbox   int `json:"inbox"`
		Contact struct {
			ID *int `json:"id"`
		} `json:"contact"`
		LS string `json:"ls"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	if payload.ID != 123 {
		t.Fatalf("expected id=123, got %d", payload.ID)
	}
	if payload.Inbox != 48 {
		t.Fatalf("expected inbox=48, got %d", payload.Inbox)
	}
	if payload.Contact.ID == nil || *payload.Contact.ID != 456 {
		t.Fatalf("expected contact.id=456, got %#v", payload.Contact.ID)
	}
	if payload.LS != "Second" {
		t.Fatalf("expected ls=Second, got %q", payload.LS)
	}
}
