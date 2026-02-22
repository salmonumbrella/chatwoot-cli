package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
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

func TestContactsGetCommand_EmitID_SkipsAPICall(t *testing.T) {
	called := false
	setupTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	})

	out := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"contacts", "get", "123", "--emit", "id"}); err != nil {
			t.Fatalf("contacts get --emit id failed: %v", err)
		}
	})

	if called {
		t.Fatalf("expected no API call for --emit id")
	}
	if strings.TrimSpace(out) != "contact:123" {
		t.Fatalf("unexpected output: %q", out)
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

func TestContactsUpdateCommand_ByEmail(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/search", jsonResponse(200, `{
			"payload": [
				{"id": 321, "name": "John Doe", "email": "john@example.com"}
			],
			"meta": {"count": 1, "current_page": 1, "total_pages": 1}
		}`)).
		On("PATCH", "/api/v1/accounts/1/contacts/321", jsonResponse(200, `{
			"payload": {"id": 321, "name": "Updated Name", "email": "john@example.com"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "update", "john@example.com", "--name", "Updated Name"})
		if err != nil {
			t.Errorf("contacts update by email failed: %v", err)
		}
	})

	if !strings.Contains(output, "Updated Name") {
		t.Errorf("output missing 'Updated Name': %s", output)
	}
}

func TestContactsUpdateCommand_WithCompanyAndCountry(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/contacts/123", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"payload": {"id": 123, "name": "Test", "created_at": 1700000000}}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "update", "123", "--company", "Acme Corp", "--country", "Canada"})
		if err != nil {
			t.Errorf("contacts update with company/country failed: %v", err)
		}
	})

	if !strings.Contains(output, "123") {
		t.Errorf("output missing contact ID: %s", output)
	}

	additional, ok := receivedBody["additional_attributes"].(map[string]any)
	if !ok {
		t.Fatal("expected additional_attributes in request body")
	}
	if additional["company_name"] != "Acme Corp" {
		t.Errorf("expected company_name 'Acme Corp', got %v", additional["company_name"])
	}
	if additional["country"] != "Canada" {
		t.Errorf("expected country 'Canada', got %v", additional["country"])
	}
}

func TestContactsUpdateCommand_WithCustomAttr(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/contacts/123", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"payload": {"id": 123, "name": "Test", "created_at": 1700000000}}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "update", "123", "-A", "plan=enterprise", "-A", "region=APAC"})
		if err != nil {
			t.Errorf("contacts update with custom-attr failed: %v", err)
		}
	})

	if !strings.Contains(output, "123") {
		t.Errorf("output missing contact ID: %s", output)
	}

	customAttrs, ok := receivedBody["custom_attributes"].(map[string]any)
	if !ok {
		t.Fatal("expected custom_attributes in request body")
	}
	if customAttrs["plan"] != "enterprise" {
		t.Errorf("expected plan 'enterprise', got %v", customAttrs["plan"])
	}
	if customAttrs["region"] != "APAC" {
		t.Errorf("expected region 'APAC', got %v", customAttrs["region"])
	}
}

func TestContactsUpdateCommand_WithSocial(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/contacts/123", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"payload": {"id": 123, "name": "Test", "created_at": 1700000000}}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "update", "123", "-S", "twitter=https://twitter.com/acme", "-S", "linkedin=https://linkedin.com/company/acme"})
		if err != nil {
			t.Errorf("contacts update with social failed: %v", err)
		}
	})

	if !strings.Contains(output, "123") {
		t.Errorf("output missing contact ID: %s", output)
	}

	additional, ok := receivedBody["additional_attributes"].(map[string]any)
	if !ok {
		t.Fatal("expected additional_attributes in request body")
	}
	socialProfiles, ok := additional["social_profiles"].(map[string]any)
	if !ok {
		t.Fatal("expected social_profiles in additional_attributes")
	}
	if socialProfiles["twitter"] != "https://twitter.com/acme" {
		t.Errorf("expected twitter URL, got %v", socialProfiles["twitter"])
	}
	if socialProfiles["linkedin"] != "https://linkedin.com/company/acme" {
		t.Errorf("expected linkedin URL, got %v", socialProfiles["linkedin"])
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

	if !strings.Contains(output, "Deleted contact 123") {
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

	if !strings.Contains(output, "Added labels to contact 123") {
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

func TestContactsConversationsCommand_AgentResolveNames(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123/conversations", jsonResponse(200, `{
			"payload": [
				{"id": 10, "status": "open", "inbox_id": 7, "contact_id": 123, "unread_count": 1, "created_at": 1700000000}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 7, "name": "Support"}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/123", jsonResponse(200, `{
			"payload": {"id": 123, "name": "Jane Doe", "email": "jane@example.com"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "conversations", "123", "--output", "agent", "--resolve-names"})
		if err != nil {
			t.Errorf("contacts conversations --output agent --resolve-names failed: %v", err)
		}
	})

	var payload struct {
		Items []struct {
			Path []struct {
				Type  string `json:"type"`
				ID    int    `json:"id"`
				Label string `json:"label"`
			} `json:"path"`
			Contact *struct {
				Name string `json:"name"`
			} `json:"contact"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(payload.Items))
	}
	if payload.Items[0].Contact == nil || payload.Items[0].Contact.Name != "Jane Doe" {
		t.Fatalf("expected resolved contact name, got %#v", payload.Items[0].Contact)
	}
	foundInbox := false
	for _, entry := range payload.Items[0].Path {
		if entry.Type == "inbox" && entry.ID == 7 && entry.Label == "Support" {
			foundInbox = true
			break
		}
	}
	if !foundInbox {
		t.Fatalf("expected inbox label Support in path, got %#v", payload.Items[0].Path)
	}
}

func TestContactsConversationsCommand_Light(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123/conversations", jsonResponse(200, `{
			"payload": [
				{
					"id": 1,
					"status": "open",
					"inbox_id": 48,
					"unread_count": 2,
					"last_activity_at": 1700001000,
					"meta": {"sender": {"id": 123, "name": "Welgrow"}},
					"last_non_activity_message": {"content": "Order update?"},
					"custom_attributes": {"debug": true}
				}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "conversations", "123", "--light"})
		if err != nil {
			t.Fatalf("contacts conversations --light failed: %v", err)
		}
	})

	var payload struct {
		Items []struct {
			ID          int    `json:"id"`
			Status      string `json:"st"`
			InboxID     int    `json:"ib"`
			LastMessage string `json:"lm"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse light output: %v\noutput: %s", err, output)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected 1 conversation, got %d", len(payload.Items))
	}
	if payload.Items[0].Status != "o" || payload.Items[0].InboxID != 48 {
		t.Fatalf("unexpected conversation payload: %#v", payload.Items[0])
	}
	if payload.Items[0].LastMessage != "Order update?" {
		t.Fatalf("expected last message, got %q", payload.Items[0].LastMessage)
	}
	if strings.Contains(output, `"custom_attributes"`) {
		t.Fatal("light output should not include custom_attributes")
	}
}

func TestContactsBulkAddLabel(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts/1/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"labels": ["vip"]}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/contacts/2/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"labels": ["vip"]}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "bulk", "add-label", "--ids", "1,2", "--labels", "vip"})
		if err != nil {
			t.Errorf("bulk add-label failed: %v", err)
		}
	})

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}

	if !strings.Contains(output, "Added labels to 2 contacts") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestContactsBulkAddLabel_IdsFromStdin(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts/1/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"labels": ["vip"]}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/contacts/2/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"labels": ["vip"]}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.Write([]byte("1\n2\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "bulk", "add-label", "--ids", "@-", "--labels", "vip"})
		if err != nil {
			t.Errorf("bulk add-label failed: %v", err)
		}
	})

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
	}
	if !strings.Contains(output, "Added labels to 2 contacts") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestContactsBulkAddLabel_LabelsFromStdin(t *testing.T) {
	var callCount atomic.Int32
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts/1/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"labels": ["vip"]}`)(w, r)
		}).
		On("POST", "/api/v1/accounts/1/contacts/2/labels", func(w http.ResponseWriter, r *http.Request) {
			callCount.Add(1)
			jsonResponse(200, `{"labels": ["vip"]}`)(w, r)
		})

	setupTestEnvWithHandler(t, handler)

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.Write([]byte("vip\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "bulk", "add-label", "--ids", "1,2", "--labels", "@-"})
		if err != nil {
			t.Errorf("bulk add-label failed: %v", err)
		}
	})

	if callCount.Load() != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount.Load())
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

	if !strings.Contains(output, "Deleted contact note 5") {
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

func TestContactsContactableInboxesCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123/contactable_inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Website", "channel_type": "Channel::WebWidget"},
				{"id": 2, "name": "Email", "channel_type": "Channel::Email"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "contactable-inboxes", "123"})
		if err != nil {
			t.Errorf("contacts contactable-inboxes failed: %v", err)
		}
	})

	if !strings.Contains(output, "Website") {
		t.Errorf("output missing 'Website': %s", output)
	}
	if !strings.Contains(output, "Email") {
		t.Errorf("output missing 'Email': %s", output)
	}
}

func TestContactsContactableInboxesCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123/contactable_inboxes", jsonResponse(200, `{"payload": []}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "contactable-inboxes", "123"})
		if err != nil {
			t.Errorf("contacts contactable-inboxes failed: %v", err)
		}
	})

	if !strings.Contains(output, "No contactable inboxes found") {
		t.Errorf("expected empty message, got: %s", output)
	}
}

func TestContactsContactableInboxesCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123/contactable_inboxes", jsonResponse(200, `{
			"payload": [{"source_id": "src-123", "inbox": {"id": 1, "name": "Website", "channel_type": "Channel::WebWidget"}}]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "contactable-inboxes", "123", "-o", "json"})
		if err != nil {
			t.Errorf("contacts contactable-inboxes JSON failed: %v", err)
		}
	})

	if !strings.Contains(output, `"inbox"`) {
		t.Errorf("JSON output missing 'inbox' field: %s", output)
	}
	if !strings.Contains(output, `"source_id"`) {
		t.Errorf("JSON output missing 'source_id' field: %s", output)
	}
	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing 'id' field: %s", output)
	}
}

func TestContactsContactableInboxesCommand_JSONRootArrayQueryFallback(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123/contactable_inboxes", jsonResponse(200, `{
			"payload": [{"source_id": "src-123", "inbox": {"id": 1, "name": "Website", "channel_type": "Channel::WebWidget"}}]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "contactable-inboxes", "123", "-o", "json", "--jq", ".[].inbox.id"})
		if err != nil {
			t.Errorf("contacts contactable-inboxes JSON query failed: %v", err)
		}
	})

	if !strings.Contains(output, "1") {
		t.Errorf("expected filtered output to contain inbox id, got: %s", output)
	}
}

func TestContactsContactableInboxesCommand_InvalidID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"contacts", "contactable-inboxes", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestContactsCreateInboxCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts/123/contact_inboxes", jsonResponse(200, `{
			"source_id": "src-123",
			"inbox": {"id": 1, "name": "Website", "channel_type": "Channel::WebWidget"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "create-inbox", "123", "--inbox-id", "1"})
		if err != nil {
			t.Errorf("contacts create-inbox failed: %v", err)
		}
	})

	if !strings.Contains(output, "associated with inbox") {
		t.Errorf("output missing success message: %s", output)
	}
}

func TestContactsCreateInboxCommand_WithSourceID(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts/123/contact_inboxes", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"source_id": "+15551234567", "inbox": {"id": 1, "name": "SMS"}}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "create-inbox", "123", "--inbox-id", "1", "--source-id", "+15551234567"})
		if err != nil {
			t.Errorf("contacts create-inbox failed: %v", err)
		}
	})

	if !strings.Contains(output, "Source ID") {
		t.Errorf("output missing source ID: %s", output)
	}
	if receivedBody["source_id"] != "+15551234567" {
		t.Errorf("expected source_id '+15551234567', got %v", receivedBody["source_id"])
	}
}

func TestContactsCreateInboxCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts/123/contact_inboxes", jsonResponse(200, `{
			"source_id": "src-123",
			"inbox": {"id": 1, "name": "Website"}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "create-inbox", "123", "--inbox-id", "1", "-o", "json"})
		if err != nil {
			t.Errorf("contacts create-inbox JSON failed: %v", err)
		}
	})

	if !strings.Contains(output, `"source_id"`) {
		t.Errorf("JSON output missing 'source_id' field: %s", output)
	}
}

func TestContactsCreateInboxCommand_NoInboxDetails(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts/123/contact_inboxes", jsonResponse(200, `{
			"source_id": "src-123"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "create-inbox", "123", "--inbox-id", "1"})
		if err != nil {
			t.Errorf("contacts create-inbox failed: %v", err)
		}
	})

	if !strings.Contains(output, "no details returned") {
		t.Errorf("expected 'no details returned' message, got: %s", output)
	}
}

func TestContactsCreateInboxCommand_InteractivePrompt(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Website", "channel_type": "Channel::WebWidget"}
			]
		}`)).
		On("POST", "/api/v1/accounts/1/contacts/123/contact_inboxes", jsonResponse(200, `{
			"source_id": "src-123",
			"inbox": {"id": 1, "name": "Website", "channel_type": "Channel::WebWidget"}
		}`))

	setupTestEnvWithHandler(t, handler)
	t.Setenv("CHATWOOT_FORCE_INTERACTIVE", "true")

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.Write([]byte("1\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "create-inbox", "123"})
		if err != nil {
			t.Errorf("contacts create-inbox interactive failed: %v", err)
		}
	})

	if !strings.Contains(output, "Contact 123 associated with inbox 1") {
		t.Errorf("expected interactive association output, got: %s", output)
	}
}

func TestContactsCreateInboxCommand_MissingInboxID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"contacts", "create-inbox", "123", "--no-input"})
	if err == nil {
		t.Error("expected error for missing --inbox-id")
	}
}

func TestContactsMergeCmd_RequiresArgs(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{"contacts", "merge"}},
		{"one arg", []string{"contacts", "merge", "123"}},
		{"one arg with force", []string{"contacts", "merge", "123", "--force"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Execute(context.Background(), tt.args)
			if err == nil {
				t.Error("expected error for missing arguments")
			}
		})
	}
}

func TestContactsMergeCmd_ValidatesIDs(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	tests := []struct {
		name   string
		args   []string
		errMsg string
	}{
		{"invalid keep-id", []string{"contacts", "merge", "invalid", "456", "--force"}, "keep-id"},
		{"invalid delete-id", []string{"contacts", "merge", "123", "invalid", "--force"}, "delete-id"},
		{"both invalid", []string{"contacts", "merge", "abc", "xyz", "--force"}, "keep-id"},
		{"negative keep-id", []string{"contacts", "merge", "-1", "456", "--force"}, "keep-id"},
		{"negative delete-id", []string{"contacts", "merge", "123", "-1", "--force"}, "delete-id"},
		{"zero keep-id", []string{"contacts", "merge", "0", "456", "--force"}, "keep-id"},
		{"zero delete-id", []string{"contacts", "merge", "123", "0", "--force"}, "delete-id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Execute(context.Background(), tt.args)
			if err == nil {
				t.Error("expected error for invalid ID")
			}
		})
	}
}

func TestContactsMergeCmd_SameID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"contacts", "merge", "123", "123", "--force"})
	if err == nil {
		t.Error("expected error when merging contact with itself")
	}
	if !strings.Contains(err.Error(), "cannot merge contact with itself") {
		t.Errorf("expected 'cannot merge contact with itself' error, got: %v", err)
	}
}

func TestContactsMergeCmd_ForceRequired(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"contacts", "merge", "123", "456", "-o", "json"})
	if err == nil {
		t.Error("expected error when --force not provided with JSON output")
	}
	if !strings.Contains(err.Error(), "--force flag is required") {
		t.Errorf("expected '--force flag is required' error, got: %v", err)
	}
}

func TestContactsMergeCmd_Success(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/actions/contact_merge", jsonResponse(200, `{
			"id": 123,
			"name": "Merged Contact",
			"email": "merged@example.com"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "merge", "123", "456", "--force"})
		if err != nil {
			t.Errorf("contacts merge failed: %v", err)
		}
	})

	if !strings.Contains(output, "Successfully merged contact") {
		t.Errorf("output missing success message: %s", output)
	}
	if !strings.Contains(output, "#456 into #123") {
		t.Errorf("output missing merge details: %s", output)
	}
}

func TestContactsMergeCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/actions/contact_merge", jsonResponse(200, `{
			"id": 123,
			"name": "Merged Contact",
			"email": "merged@example.com"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "merge", "123", "456", "--force", "-o", "json"})
		if err != nil {
			t.Errorf("contacts merge JSON failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing 'id' field: %s", output)
	}
}

func TestContactsMergeCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/actions/contact_merge", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"contacts", "merge", "123", "456", "--force"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

// TestContactsMergeCmd_InteractiveFlow verifies that when --force is not provided,
// the command fetches both contacts before displaying the confirmation prompt.
func TestContactsMergeCmd_InteractiveFlow(t *testing.T) {
	var getContact123Called, getContact456Called bool

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123", func(w http.ResponseWriter, r *http.Request) {
			getContact123Called = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"payload": {"id": 123, "name": "Keep Contact", "email": "keep@example.com"}}`))
		}).
		On("GET", "/api/v1/accounts/1/contacts/456", func(w http.ResponseWriter, r *http.Request) {
			getContact456Called = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"payload": {"id": 456, "name": "Delete Contact", "email": "delete@example.com", "phone_number": "+1234567890"}}`))
		})

	setupTestEnvWithHandler(t, handler)

	// Run without --force - since there's no stdin, it will prompt and fail to read
	// but the important thing is that it fetches both contacts first
	output := captureStdout(t, func() {
		_ = Execute(context.Background(), []string{"contacts", "merge", "123", "456"})
	})

	// Verify both contacts were fetched
	if !getContact123Called {
		t.Error("expected GET request for contact 123 (keep contact)")
	}
	if !getContact456Called {
		t.Error("expected GET request for contact 456 (delete contact)")
	}

	// Verify the confirmation prompt is displayed with correct information
	if !strings.Contains(output, "MERGE CONTACTS") {
		t.Errorf("output missing 'MERGE CONTACTS' header: %s", output)
	}
	if !strings.Contains(output, "KEEP (base)") {
		t.Errorf("output missing 'KEEP (base)' label: %s", output)
	}
	if !strings.Contains(output, "DELETE (mergee)") {
		t.Errorf("output missing 'DELETE (mergee)' label: %s", output)
	}
	if !strings.Contains(output, "Keep Contact") {
		t.Errorf("output missing keep contact name: %s", output)
	}
	if !strings.Contains(output, "Delete Contact") {
		t.Errorf("output missing delete contact name: %s", output)
	}
	if !strings.Contains(output, "PERMANENTLY DELETED") {
		t.Errorf("output missing deletion warning: %s", output)
	}
}

// TestContactsMergeCmd_VerifiesIDMapping verifies that keep-id maps to base_contact_id
// and delete-id maps to mergee_contact_id in the API request.
func TestContactsMergeCmd_VerifiesIDMapping(t *testing.T) {
	var capturedBody map[string]int

	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/actions/contact_merge", func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
				t.Errorf("failed to decode request body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id": 100, "name": "Merged", "email": "merged@example.com"}`))
		})

	setupTestEnvWithHandler(t, handler)

	_ = Execute(context.Background(), []string{"contacts", "merge", "100", "200", "--force"})

	// Verify the ID mapping: keep-id (100) -> base_contact_id, delete-id (200) -> mergee_contact_id
	if capturedBody["base_contact_id"] != 100 {
		t.Errorf("expected base_contact_id=100 (keep-id), got %d", capturedBody["base_contact_id"])
	}
	if capturedBody["mergee_contact_id"] != 200 {
		t.Errorf("expected mergee_contact_id=200 (delete-id), got %d", capturedBody["mergee_contact_id"])
	}
}

// TestContactsMergeCmd_CancellationFlow verifies that when user provides input
// that is NOT "merge", the merge is cancelled and the API is not called.
func TestContactsGetWithOpenConversations(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123", jsonResponse(200, `{
			"payload": {"id": 123, "name": "John Doe", "email": "john@example.com"}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/123/conversations", jsonResponse(200, `{
			"payload": [
				{"id": 1, "status": "open", "inbox_id": 1, "unread_count": 5, "created_at": 1700000000},
				{"id": 2, "status": "pending", "inbox_id": 2, "unread_count": 2, "created_at": 1700001000},
				{"id": 3, "status": "resolved", "inbox_id": 1, "unread_count": 0, "created_at": 1699000000}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "get", "123", "--output", "agent", "--with-open-conversations"})
		if err != nil {
			t.Errorf("contacts get with open conversations failed: %v", err)
		}
	})

	var payload struct {
		Kind string `json:"kind"`
		Item struct {
			ID           int `json:"id"`
			Relationship *struct {
				TotalConversations int `json:"total_conversations"`
				OpenConversations  int `json:"open_conversations"`
			} `json:"relationship"`
			OpenConversations []struct {
				ID     int    `json:"id"`
				Status string `json:"status"`
			} `json:"open_conversations"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	// Verify kind
	if payload.Kind != "contacts.get" {
		t.Errorf("expected kind 'contacts.get', got %s", payload.Kind)
	}

	// Verify contact ID
	if payload.Item.ID != 123 {
		t.Errorf("expected contact ID 123, got %d", payload.Item.ID)
	}

	// Verify relationship stats
	if payload.Item.Relationship == nil {
		t.Fatal("expected relationship data to be present")
	}
	if payload.Item.Relationship.TotalConversations != 3 {
		t.Errorf("expected 3 total conversations, got %d", payload.Item.Relationship.TotalConversations)
	}
	if payload.Item.Relationship.OpenConversations != 2 {
		t.Errorf("expected 2 open conversations in relationship, got %d", payload.Item.Relationship.OpenConversations)
	}

	// Verify open_conversations array contains only open/pending
	if len(payload.Item.OpenConversations) != 2 {
		t.Errorf("expected 2 open conversations in array, got %d", len(payload.Item.OpenConversations))
	}
	for _, conv := range payload.Item.OpenConversations {
		if conv.Status != "open" && conv.Status != "pending" {
			t.Errorf("expected only open/pending conversations, got status %s for ID %d", conv.Status, conv.ID)
		}
	}
}

func TestContactsShowWithOpenConversations(t *testing.T) {
	// Test that the 'show' alias also supports --with-open-conversations
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123", jsonResponse(200, `{
			"payload": {"id": 123, "name": "Jane Doe", "email": "jane@example.com"}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/123/conversations", jsonResponse(200, `{
			"payload": [
				{"id": 10, "status": "pending", "inbox_id": 1, "unread_count": 3, "created_at": 1700000000}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "show", "123", "--output", "agent", "--with-open-conversations"})
		if err != nil {
			t.Errorf("contacts show with open conversations failed: %v", err)
		}
	})

	var payload struct {
		Kind string `json:"kind"`
		Item struct {
			OpenConversations []struct {
				ID     int    `json:"id"`
				Status string `json:"status"`
			} `json:"open_conversations"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	// Verify kind is contacts.show (not contacts.get)
	if payload.Kind != "contacts.show" {
		t.Errorf("expected kind 'contacts.show', got %s", payload.Kind)
	}

	// Verify open_conversations is populated
	if len(payload.Item.OpenConversations) != 1 {
		t.Errorf("expected 1 open conversation, got %d", len(payload.Item.OpenConversations))
	}
	if payload.Item.OpenConversations[0].Status != "pending" {
		t.Errorf("expected pending status, got %s", payload.Item.OpenConversations[0].Status)
	}
}

func TestContactsGetWithOpenConversations_WithoutFlag(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123", jsonResponse(200, `{
			"payload": {"id": 123, "name": "John Doe", "email": "john@example.com"}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/123/conversations", jsonResponse(200, `{
			"payload": [
				{"id": 1, "status": "open", "inbox_id": 1, "unread_count": 5, "created_at": 1700000000}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "get", "123", "--output", "agent"})
		if err != nil {
			t.Errorf("contacts get without open conversations flag failed: %v", err)
		}
	})

	// Parse the JSON output to verify structure
	var payload struct {
		Kind string `json:"kind"`
		Item struct {
			ID                int   `json:"id"`
			OpenConversations []any `json:"open_conversations"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	// Verify open_conversations array is NOT present when flag is not set
	// (nil means the field was not in JSON, empty slice means it was present but empty)
	if payload.Item.OpenConversations != nil {
		t.Errorf("expected no open_conversations array when flag not set, got: %v", payload.Item.OpenConversations)
	}

	// Relationship should still be present (check via raw string since we didn't parse it)
	if !strings.Contains(output, `"relationship"`) {
		t.Errorf("expected relationship field to be present, got: %s", output)
	}
}

func TestContactsMergeCmd_CancellationFlow(t *testing.T) {
	var mergeAPICalled bool

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123", jsonResponse(200, `{
			"payload": {"id": 123, "name": "Keep Contact", "email": "keep@example.com"}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {"id": 456, "name": "Delete Contact", "email": "delete@example.com"}
		}`)).
		On("POST", "/api/v1/accounts/1/actions/contact_merge", func(w http.ResponseWriter, r *http.Request) {
			mergeAPICalled = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id": 123, "name": "Merged Contact"}`))
		})

	setupTestEnvWithHandler(t, handler)

	// Create a pipe to mock stdin with "no" input (not "merge")
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Write test input to the pipe and close the write end
	go func() {
		_, _ = w.Write([]byte("no\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "merge", "123", "456"})
		if err != nil {
			t.Errorf("expected no error on cancellation, got: %v", err)
		}
	})

	os.Stdin = oldStdin

	// Verify the merge API was NOT called
	if mergeAPICalled {
		t.Error("merge API should NOT have been called when user cancels")
	}

	// Verify the output contains the cancellation message
	if !strings.Contains(output, "Merge cancelled.") {
		t.Errorf("output missing 'Merge cancelled.' message: %s", output)
	}
}

func TestContactsMergeDryRun(t *testing.T) {
	var mergeAPICalled bool

	handler := newRouteHandler().
		// Target contact (will be kept)
		On("GET", "/api/v1/accounts/1/contacts/123", jsonResponse(200, `{
			"payload": {
				"id": 123,
				"name": "Alice Smith",
				"email": "alice@example.com",
				"phone_number": "+1111111111"
			}
		}`)).
		// Source contact (will be deleted)
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {
				"id": 456,
				"name": "Bob Jones",
				"email": "bob@example.com",
				"phone_number": "+2222222222"
			}
		}`)).
		// Conversations for target contact
		On("GET", "/api/v1/accounts/1/contacts/123/conversations", jsonResponse(200, `{
			"payload": [
				{"id": 1, "status": "open"},
				{"id": 2, "status": "resolved"},
				{"id": 3, "status": "resolved"}
			]
		}`)).
		// Conversations for source contact
		On("GET", "/api/v1/accounts/1/contacts/456/conversations", jsonResponse(200, `{
			"payload": [
				{"id": 4, "status": "open"},
				{"id": 5, "status": "open"}
			]
		}`)).
		// Merge API - should NOT be called
		On("POST", "/api/v1/accounts/1/actions/contact_merge", func(w http.ResponseWriter, r *http.Request) {
			mergeAPICalled = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id": 123, "name": "Merged Contact"}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "merge", "123", "456", "--dry-run"})
		if err != nil {
			t.Errorf("contacts merge --dry-run failed: %v", err)
		}
	})

	// Verify merge API was NOT called
	if mergeAPICalled {
		t.Error("merge API should NOT have been called with --dry-run")
	}

	// Verify DRY RUN header is present
	if !strings.Contains(output, "DRY RUN") {
		t.Errorf("output missing 'DRY RUN' header: %s", output)
	}

	// Verify both contact names appear
	if !strings.Contains(output, "Alice Smith") {
		t.Errorf("output missing target contact name 'Alice Smith': %s", output)
	}
	if !strings.Contains(output, "Bob Jones") {
		t.Errorf("output missing source contact name 'Bob Jones': %s", output)
	}

	// Verify SOURCE and TARGET sections
	if !strings.Contains(output, "SOURCE") {
		t.Errorf("output missing 'SOURCE' section: %s", output)
	}
	if !strings.Contains(output, "TARGET") {
		t.Errorf("output missing 'TARGET' section: %s", output)
	}

	// Verify "without --dry-run" instruction
	if !strings.Contains(output, "without --dry-run") {
		t.Errorf("output missing instructions to run without --dry-run: %s", output)
	}
}

func TestContactsMergeDryRun_ShowsConflicts(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123", jsonResponse(200, `{
			"payload": {
				"id": 123,
				"name": "Alice",
				"email": "alice@example.com",
				"phone_number": "+1111111111"
			}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {
				"id": 456,
				"name": "Bob",
				"email": "bob@different.com",
				"phone_number": "+2222222222"
			}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/123/conversations", jsonResponse(200, `{"payload": []}`)).
		On("GET", "/api/v1/accounts/1/contacts/456/conversations", jsonResponse(200, `{"payload": []}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "merge", "123", "456", "--dry-run"})
		if err != nil {
			t.Errorf("contacts merge --dry-run failed: %v", err)
		}
	})

	// Should show conflicts when email differs
	if !strings.Contains(output, "CONFLICTS") {
		t.Errorf("output missing 'CONFLICTS' section when emails differ: %s", output)
	}
}

func TestContactsMergeDryRun_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123", jsonResponse(200, `{
			"payload": {
				"id": 123,
				"name": "Alice",
				"email": "alice@example.com"
			}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"payload": {
				"id": 456,
				"name": "Bob",
				"email": ""
			}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/123/conversations", jsonResponse(200, `{"payload": []}`)).
		On("GET", "/api/v1/accounts/1/contacts/456/conversations", jsonResponse(200, `{"payload": []}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "merge", "123", "456", "--dry-run", "-o", "json"})
		if err != nil {
			t.Errorf("contacts merge --dry-run JSON failed: %v", err)
		}
	})

	// Should be valid JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("expected valid JSON output, got error: %v\noutput: %s", err, output)
	}

	// Should have dry_run field
	if _, ok := result["dry_run"]; !ok {
		t.Errorf("JSON output missing 'dry_run' field: %s", output)
	}

	// Should have source and target
	if _, ok := result["source"]; !ok {
		t.Errorf("JSON output missing 'source' field: %s", output)
	}
	if _, ok := result["target"]; !ok {
		t.Errorf("JSON output missing 'target' field: %s", output)
	}
}

func TestContactsGetLight(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/136014", jsonResponse(200, `{
			"payload": {
				"id": 136014,
				"name": "Jane Doe",
				"email": "jane@example.com",
				"phone_number": "+886912345678",
				"custom_attributes": {"tier": "gold"},
				"created_at": 1700000000
			}
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/136014/conversations", jsonResponse(200, `{
			"payload": [
				{
					"id": 8821,
					"status": "open",
					"inbox_id": 3,
					"last_non_activity_message": {"content": "When will my order arrive?"}
				},
				{
					"id": 9999,
					"status": "resolved",
					"inbox_id": 3,
					"last_non_activity_message": {"content": "Old resolved message"}
				}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "get", "136014", "--light"})
		if err != nil {
			t.Fatalf("contacts get --light failed: %v", err)
		}
	})

	// Should contain light fields with short keys
	if !strings.Contains(output, `"nm"`) {
		t.Error("expected short key 'nm' in output")
	}
	// Should have conversation with last message
	if !strings.Contains(output, `"When will my order arrive?"`) {
		t.Error("expected last message in output")
	}
	// Should NOT contain resolved conversation
	if strings.Contains(output, `"Old resolved message"`) {
		t.Error("should not include resolved conversations")
	}
	// Should NOT contain custom_attributes (stripped in light mode)
	if strings.Contains(output, `"custom_attributes"`) {
		t.Error("should not contain custom_attributes in light mode")
	}
	if strings.Contains(output, `"tier"`) {
		t.Error("should not contain custom attribute values in light mode")
	}
}

func TestContactsListLight(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts", jsonResponse(200, `{
			"payload": [
				{
					"id": 136014,
					"name": "Jane Doe",
					"email": "jane@example.com",
					"phone_number": "+886912345678",
					"custom_attributes": {"tier": "gold"},
					"created_at": 1700000000
				},
				{
					"id": 42,
					"name": "Bob",
					"email": "",
					"phone_number": "+886998765432",
					"custom_attributes": {},
					"created_at": 1700000001
				}
			],
			"meta": {"count": 2}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "list", "--light"})
		if err != nil {
			t.Fatalf("contacts list --light failed: %v", err)
		}
	})

	// Should be a JSON array
	if !strings.HasPrefix(strings.TrimSpace(output), "[") {
		t.Errorf("expected JSON array output, got: %s", output)
	}
	if !strings.Contains(output, `"nm"`) {
		t.Error("expected short key 'nm'")
	}
	// Should NOT have conversations (too expensive for list)
	if strings.Contains(output, `"cvs"`) {
		t.Error("should not include cvs in list mode")
	}
	// Should NOT have full field names or custom attributes
	if strings.Contains(output, `"created_at"`) {
		t.Error("should not contain created_at in light mode")
	}
	if strings.Contains(output, `"custom_attributes"`) {
		t.Error("should not contain custom_attributes in light mode")
	}
}
