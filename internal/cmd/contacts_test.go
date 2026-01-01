package cmd

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestContactsListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com", "phone_number": "+1234567890"},
				{"id": 2, "name": "Jane Doe", "email": "jane@example.com", "phone_number": "+0987654321"}
			],
			"meta": {"count": 2}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "list"})
		if err != nil {
			t.Errorf("contacts list failed: %v", err)
		}
	})

	if !strings.Contains(output, "John Doe") {
		t.Errorf("output missing 'John Doe': %s", output)
	}
	if !strings.Contains(output, "Jane Doe") {
		t.Errorf("output missing 'Jane Doe': %s", output)
	}
	if !strings.Contains(output, "john@example.com") {
		t.Errorf("output missing email: %s", output)
	}
}

func TestContactsListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com"}
			],
			"meta": {"count": 1}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "list", "--output", "json"})
		if err != nil {
			t.Errorf("contacts list --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing 'id' field: %s", output)
	}
	if !strings.Contains(output, `"name"`) {
		t.Errorf("JSON output missing 'name' field: %s", output)
	}
}

func TestContactsGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123", jsonResponse(200, `{
			"payload": {"id": 123, "name": "John Doe", "email": "john@example.com", "phone_number": "+1234567890"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "get", "123"})
		if err != nil {
			t.Errorf("contacts get failed: %v", err)
		}
	})

	if !strings.Contains(output, "John Doe") {
		t.Errorf("output missing 'John Doe': %s", output)
	}
}

func TestContactsShowCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {"id": 456, "name": "Jane Doe", "email": "jane@example.com"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "show", "456"})
		if err != nil {
			t.Errorf("contacts show failed: %v", err)
		}
	})

	if !strings.Contains(output, "Jane Doe") {
		t.Errorf("output missing 'Jane Doe': %s", output)
	}
}

func TestContactsSearchCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Smith", "email": "john.smith@example.com"}
			],
			"meta": {"count": 1}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "search", "--query", "John"})
		if err != nil {
			t.Errorf("contacts search failed: %v", err)
		}
	})

	if !strings.Contains(output, "John Smith") {
		t.Errorf("output missing 'John Smith': %s", output)
	}
}

func TestContactsSearchCommand_MissingQuery(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"contacts", "search"})
	if err == nil {
		t.Error("expected error for missing --query flag")
	}
}

func TestContactsCreateCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts", jsonResponse(200, `{
			"payload": {
				"contact": {"id": 789, "name": "New Contact", "email": "new@example.com", "phone_number": "+1111111111"}
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "create", "--name", "New Contact", "--email", "new@example.com", "--phone", "+1111111111"})
		if err != nil {
			t.Errorf("contacts create failed: %v", err)
		}
	})

	if !strings.Contains(output, "New Contact") {
		t.Errorf("output missing 'New Contact': %s", output)
	}
}

func TestContactsCreateCommand_MissingName(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"contacts", "create", "--email", "test@example.com"})
	if err == nil {
		t.Error("expected error for missing --name flag")
	}
}

func TestContactsUpdateCommand(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/contacts/123", jsonResponse(200, `{
			"payload": {"id": 123, "name": "Updated Name", "email": "updated@example.com"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "update", "123", "--name", "Updated Name"})
		if err != nil {
			t.Errorf("contacts update failed: %v", err)
		}
	})

	if !strings.Contains(output, "Updated Name") {
		t.Errorf("output missing 'Updated Name': %s", output)
	}
}

func TestContactsUpdateCommand_NoFlags(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"contacts", "update", "123"})
	if err == nil {
		t.Error("expected error when no update flags provided")
	}
}

func TestContactsDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/contacts/123", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "delete", "123"})
		if err != nil {
			t.Errorf("contacts delete failed: %v", err)
		}
	})

	if !strings.Contains(output, "deleted successfully") {
		t.Errorf("output missing success message: %s", output)
	}
}

func TestContactsLabelsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123/labels", jsonResponse(200, `{
			"labels": ["vip", "premium", "active"]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "labels", "123"})
		if err != nil {
			t.Errorf("contacts labels failed: %v", err)
		}
	})

	if !strings.Contains(output, "vip") {
		t.Errorf("output missing 'vip' label: %s", output)
	}
	if !strings.Contains(output, "premium") {
		t.Errorf("output missing 'premium' label: %s", output)
	}
}

func TestContactsLabelsAddCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts/123/labels", jsonResponse(200, `{
			"labels": ["vip", "new-label"]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "labels-add", "123", "--labels", "new-label"})
		if err != nil {
			t.Errorf("contacts labels-add failed: %v", err)
		}
	})

	if !strings.Contains(output, "Labels added successfully") {
		t.Errorf("output missing success message: %s", output)
	}
}

func TestContactsConversationsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123/conversations", jsonResponse(200, `{
			"payload": [
				{"id": 1, "status": "open", "inbox_id": 1, "unread_count": 5},
				{"id": 2, "status": "resolved", "inbox_id": 2, "unread_count": 0}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "conversations", "123"})
		if err != nil {
			t.Errorf("contacts conversations failed: %v", err)
		}
	})

	if !strings.Contains(output, "open") {
		t.Errorf("output missing 'open' status: %s", output)
	}
}

func TestContactsBulkAddLabel(t *testing.T) {
	callCount := 0
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts/1/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount++
			jsonResponse(200, `{"labels": ["vip"]}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/contacts/2/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount++
			jsonResponse(200, `{"labels": ["vip"]}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "bulk", "add-label", "--ids", "1,2", "--labels", "vip"})
		if err != nil {
			t.Errorf("bulk add-label failed: %v", err)
		}
	})

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}

	if !strings.Contains(output, "Added labels to 2 contacts") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestContactsBulkRemoveLabel(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/1/labels", jsonResponse(200, `{"labels": ["vip", "spam", "active"]}`)).
		On("POST", "/api/v1/accounts/1/contacts/1/labels", jsonResponse(200, `{"labels": ["vip", "active"]}`)).
		On("GET", "/api/v1/accounts/1/contacts/2/labels", jsonResponse(200, `{"labels": ["spam", "inactive"]}`)).
		On("POST", "/api/v1/accounts/1/contacts/2/labels", jsonResponse(200, `{"labels": ["inactive"]}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "bulk", "remove-label", "--ids", "1,2", "--labels", "spam"})
		if err != nil {
			t.Errorf("bulk remove-label failed: %v", err)
		}
	})

	if !strings.Contains(output, "Removed labels from 2 contacts") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestContactsBulkAddLabel_MissingFlags(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	tests := []struct {
		name string
		args []string
	}{
		{"missing ids", []string{"contacts", "bulk", "add-label", "--labels", "vip"}},
		{"missing labels", []string{"contacts", "bulk", "add-label", "--ids", "1,2"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Execute(context.Background(), tt.args)
			if err == nil {
				t.Error("expected error for missing required flags")
			}
		})
	}
}

func TestContactsNotesCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123/notes", jsonResponse(200, `[
			{"id": 1, "content": "VIP customer", "created_at": "2024-01-15T10:00:00Z", "user": {"email": "agent@example.com"}},
			{"id": 2, "content": "Follow up needed", "created_at": "2024-01-16T11:00:00Z"}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "notes", "123"})
		if err != nil {
			t.Errorf("contacts notes failed: %v", err)
		}
	})

	if !strings.Contains(output, "VIP customer") {
		t.Errorf("output missing note content: %s", output)
	}
}

func TestContactsNotesAddCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts/123/notes", jsonResponse(200, `{
			"id": 5, "content": "New note content", "created_at": "2024-01-20T10:00:00Z"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "notes-add", "123", "--content", "New note content"})
		if err != nil {
			t.Errorf("contacts notes-add failed: %v", err)
		}
	})

	if !strings.Contains(output, "Added note #5") {
		t.Errorf("output missing success message: %s", output)
	}
}

func TestContactsNotesDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/contacts/123/notes/5", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "notes-delete", "123", "5"})
		if err != nil {
			t.Errorf("contacts notes-delete failed: %v", err)
		}
	})

	if !strings.Contains(output, "Deleted note #5") {
		t.Errorf("output missing success message: %s", output)
	}
}

func TestContactsFilterCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts/filter", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Test User", "email": "test@example.com"}
			],
			"meta": {"count": 1}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "filter", "--payload", `[{"attribute_key":"name","filter_operator":"contains","values":["test"]}]`})
		if err != nil {
			t.Errorf("contacts filter failed: %v", err)
		}
	})

	if !strings.Contains(output, "Test User") {
		t.Errorf("output missing 'Test User': %s", output)
	}
}

func TestContactsFilterCommand_MissingPayload(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"contacts", "filter"})
	if err == nil {
		t.Error("expected error for missing --payload flag")
	}
}

func TestContactsFilterCommand_InvalidJSON(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"contacts", "filter", "--payload", "not-valid-json"})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
