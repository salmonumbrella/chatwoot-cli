package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestAuditLogsListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/audit_logs", jsonResponse(200, `{
			"payload": [
				{"id": 1, "action": "create", "auditable_type": "User", "user_id": 1, "username": "admin", "created_at": "2024-01-01T00:00:00Z"},
				{"id": 2, "action": "update", "auditable_type": "Account", "user_id": 2, "username": "", "created_at": "2024-01-02T00:00:00Z"}
			],
			"meta": {
				"current_page": 1,
				"total_pages": 2,
				"total_count": 20
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"audit-logs", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("audit-logs list failed: %v", err)
	}

	if !strings.Contains(output, "create") {
		t.Errorf("output missing action: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "ACTION") || !strings.Contains(output, "TYPE") || !strings.Contains(output, "USER") {
		t.Errorf("output missing expected headers: %s", output)
	}
	// Username should appear for first entry
	if !strings.Contains(output, "admin") {
		t.Errorf("output missing username: %s", output)
	}
	// Pagination info should appear
	if !strings.Contains(output, "Page 1 of 2") {
		t.Errorf("output missing pagination info: %s", output)
	}
}

func TestAuditLogsListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/audit_logs", jsonResponse(200, `{
			"payload": [
				{"id": 1, "action": "create", "auditable_type": "User", "user_id": 1}
			],
			"meta": {"current_page": 1, "total_pages": 1, "total_count": 1}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"audit-logs", "list", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("audit-logs list failed: %v", err)
	}

	logs := decodeItems(t, output)
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
}

func TestAuditLogsListCommand_NoUsername(t *testing.T) {
	// When username is empty, it should show user_id instead
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/audit_logs", jsonResponse(200, `{
			"payload": [
				{"id": 1, "action": "create", "auditable_type": "User", "user_id": 42, "username": "", "created_at": "2024-01-01T00:00:00Z"}
			],
			"meta": {"current_page": 1, "total_pages": 1, "total_count": 1}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"audit-logs", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("audit-logs list failed: %v", err)
	}

	// Should show user ID "42" when username is empty
	if !strings.Contains(output, "42") {
		t.Errorf("output should show user ID when username is empty: %s", output)
	}
}

func TestAuditLogsListCommand_NoPagination(t *testing.T) {
	// When total_pages is 0, pagination info should not be shown
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/audit_logs", jsonResponse(200, `{
			"payload": [
				{"id": 1, "action": "create", "auditable_type": "User", "user_id": 1, "username": "admin", "created_at": "2024-01-01T00:00:00Z"}
			],
			"meta": {"current_page": 0, "total_pages": 0, "total_count": 0}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"audit-logs", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("audit-logs list failed: %v", err)
	}

	// Should NOT show pagination info when total_pages is 0
	if strings.Contains(output, "Page") && strings.Contains(output, "of") {
		t.Errorf("output should not show pagination when total_pages is 0: %s", output)
	}
}

func TestAuditLogsListCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/audit_logs", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"audit-logs", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

// Test the "audit" alias
func TestAuditLogsListCommand_AuditAlias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/audit_logs", jsonResponse(200, `{
			"payload": [{"id": 1, "action": "create", "auditable_type": "User", "user_id": 1, "username": "admin", "created_at": "2024-01-01T00:00:00Z"}],
			"meta": {"current_page": 1, "total_pages": 1, "total_count": 1}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"audit", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("audit list failed: %v", err)
	}

	if !strings.Contains(output, "create") {
		t.Errorf("output missing action: %s", output)
	}
}
