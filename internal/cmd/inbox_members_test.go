package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestInboxMembersListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inbox_members/1", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com", "role": "agent", "availability_status": "online"},
				{"id": 2, "name": "Jane Smith", "email": "jane@example.com", "role": "agent", "availability_status": "offline"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"inbox-members", "list", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("inbox-members list failed: %v", err)
	}

	if !strings.Contains(output, "John Doe") {
		t.Errorf("output missing member name: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "EMAIL") || !strings.Contains(output, "ROLE") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "online") {
		t.Errorf("output missing availability status: %s", output)
	}
}

func TestInboxMembersListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inbox_members/1", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com", "role": "agent"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"inbox-members", "list", "1", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("inbox-members list failed: %v", err)
	}

	members := decodeItems(t, output)
	if len(members) != 1 {
		t.Errorf("expected 1 member, got %d", len(members))
	}
}

func TestInboxMembersListCommand_NoStatus(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inbox_members/1", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com", "role": "agent"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"inbox-members", "list", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("inbox-members list failed: %v", err)
	}

	// When no status, should show "-"
	if !strings.Contains(output, "-") {
		t.Errorf("output should show '-' for missing status: %s", output)
	}
}

func TestInboxMembersListCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"inbox-members", "list", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "inbox ID") {
		t.Errorf("expected 'inbox ID' error, got: %v", err)
	}
}

func TestInboxMembersAddCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/inbox_members", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"inbox-members", "add", "1",
		"--user-ids", "2,3,4",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("inbox-members add failed: %v", err)
	}

	if !strings.Contains(output, "Added 3 member(s) to inbox 1") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["inbox_id"] != float64(1) {
		t.Errorf("expected inbox_id 1, got %v", receivedBody["inbox_id"])
	}

	userIDs := receivedBody["user_ids"].([]any)
	if len(userIDs) != 3 {
		t.Errorf("expected 3 user_ids, got %d", len(userIDs))
	}
}

func TestInboxMembersAddCommand_UserIDsFromStdin(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/inbox_members", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
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
		_, _ = w.Write([]byte("2\n3\n4\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"inbox-members", "add", "1",
			"--user-ids", "@-",
		})
		if err != nil {
			t.Errorf("inbox-members add failed: %v", err)
		}
	})

	if !strings.Contains(output, "Added 3 member(s) to inbox 1") {
		t.Errorf("expected success message, got: %s", output)
	}

	userIDs := receivedBody["user_ids"].([]any)
	if len(userIDs) != 3 {
		t.Errorf("expected 3 user_ids, got %d", len(userIDs))
	}
}

func TestInboxMembersAddCommand_InvalidInboxID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"inbox-members", "add", "invalid",
		"--user-ids", "1,2",
	})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "inbox ID") {
		t.Errorf("expected 'inbox ID' error, got: %v", err)
	}
}

func TestInboxMembersAddCommand_InvalidUserIDs(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"inbox-members", "add", "1",
		"--user-ids", "abc,def",
	})
	if err == nil {
		t.Error("expected error for invalid user IDs")
	}
	if !strings.Contains(err.Error(), "user ID") {
		t.Errorf("expected 'user ID' error, got: %v", err)
	}
}

func TestInboxMembersAddCommand_EmptyUserIDs(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"inbox-members", "add", "1",
		"--user-ids", "  ,  ,  ",
	})
	if err == nil {
		t.Error("expected error for empty user IDs")
	}
	if !strings.Contains(err.Error(), "no valid user IDs") {
		t.Errorf("expected 'no valid user IDs' error, got: %v", err)
	}
}

func TestInboxMembersRemoveCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/inbox_members", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"inbox-members", "remove", "1",
		"--user-ids", "2,3",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("inbox-members remove failed: %v", err)
	}

	if !strings.Contains(output, "Removed 2 member(s) from inbox 1") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["inbox_id"] != float64(1) {
		t.Errorf("expected inbox_id 1, got %v", receivedBody["inbox_id"])
	}
}

func TestInboxMembersRemoveCommand_InvalidInboxID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"inbox-members", "remove", "invalid",
		"--user-ids", "1,2",
	})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "inbox ID") {
		t.Errorf("expected 'inbox ID' error, got: %v", err)
	}
}

func TestInboxMembersUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/inbox_members", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"inbox-members", "update", "1",
		"--user-ids", "5,6,7",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("inbox-members update failed: %v", err)
	}

	if !strings.Contains(output, "Updated inbox members 1") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["inbox_id"] != float64(1) {
		t.Errorf("expected inbox_id 1, got %v", receivedBody["inbox_id"])
	}

	userIDs := receivedBody["user_ids"].([]any)
	if len(userIDs) != 3 {
		t.Errorf("expected 3 user_ids, got %d", len(userIDs))
	}
}

func TestInboxMembersUpdateCommand_InvalidInboxID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"inbox-members", "update", "invalid",
		"--user-ids", "1,2",
	})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "inbox ID") {
		t.Errorf("expected 'inbox ID' error, got: %v", err)
	}
}

func TestInboxMembersListCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inbox_members/1", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"inbox-members", "list", "1"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

// Test the "inbox_members" alias (underscore version)
func TestInboxMembersListCommand_UnderscoreAlias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inbox_members/1", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "John Doe", "email": "john@example.com", "role": "agent"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"inbox_members", "list", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("inbox_members list failed: %v", err)
	}

	if !strings.Contains(output, "John Doe") {
		t.Errorf("output missing member name: %s", output)
	}
}

func TestParseUserIDs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
	}{
		{"single ID", "1", []int{1}, false},
		{"multiple IDs", "1,2,3", []int{1, 2, 3}, false},
		{"with spaces", "1, 2, 3", []int{1, 2, 3}, false},
		{"with empty parts", "1,,2", []int{1, 2}, false},
		{"invalid ID", "1,abc", nil, true},
		{"all empty", "  ,  ", nil, true},
		{"negative ID", "-1", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseUserIDs(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseUserIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("parseUserIDs() got %d IDs, want %d", len(got), len(tt.want))
					return
				}
				for i, v := range got {
					if v != tt.want[i] {
						t.Errorf("parseUserIDs()[%d] = %d, want %d", i, v, tt.want[i])
					}
				}
			}
		})
	}
}
