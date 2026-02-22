package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestAgentsListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 1, "name": "Agent Smith", "email": "smith@example.com", "role": "agent", "availability_status": "online"},
			{"id": 2, "name": "Agent Jones", "email": "jones@example.com", "role": "admin", "availability_status": "offline"}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"agents", "list"})
		if err != nil {
			t.Errorf("agents list failed: %v", err)
		}
	})

	if !strings.Contains(output, "Agent Smith") {
		t.Errorf("output missing 'Agent Smith': %s", output)
	}
	if !strings.Contains(output, "Agent Jones") {
		t.Errorf("output missing 'Agent Jones': %s", output)
	}
	if !strings.Contains(output, "smith@example.com") {
		t.Errorf("output missing email: %s", output)
	}
	if !strings.Contains(output, "admin") {
		t.Errorf("output missing role: %s", output)
	}
}

func TestAgentsListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 1, "name": "Agent Smith", "email": "smith@example.com", "role": "agent"}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"agents", "list", "--output", "json"})
		if err != nil {
			t.Errorf("agents list --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"id"`) {
		t.Errorf("JSON output missing 'id' field: %s", output)
	}
	if !strings.Contains(output, `"name"`) {
		t.Errorf("JSON output missing 'name' field: %s", output)
	}
	if !strings.Contains(output, `"email"`) {
		t.Errorf("JSON output missing 'email' field: %s", output)
	}
}

func TestAgentsListCommand_EmptyList(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStderr(t, func() {
		err := Execute(context.Background(), []string{"agents", "list"})
		if err != nil {
			t.Errorf("agents list failed: %v", err)
		}
	})

	if !strings.Contains(output, "No agents found") {
		t.Errorf("expected 'No agents found' message, got: %s", output)
	}
}

func TestAgentsGetCommand(t *testing.T) {
	// GetAgent uses ListAgents internally and filters by ID
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 1, "name": "Agent Smith", "email": "smith@example.com", "role": "agent", "availability_status": "online"},
			{"id": 2, "name": "Agent Jones", "email": "jones@example.com", "role": "admin", "availability_status": "offline"}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"agents", "get", "2"})
		if err != nil {
			t.Errorf("agents get failed: %v", err)
		}
	})

	if !strings.Contains(output, "Agent Jones") {
		t.Errorf("output missing 'Agent Jones': %s", output)
	}
	if !strings.Contains(output, "jones@example.com") {
		t.Errorf("output missing email: %s", output)
	}
}

func TestAgentsGetCommand_NotFound(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(200, `[
			{"id": 1, "name": "Agent Smith", "email": "smith@example.com", "role": "agent"}
		]`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"agents", "get", "999"})
	if err == nil {
		t.Error("expected error for non-existent agent ID")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error message should mention 'not found': %v", err)
	}
}

func TestAgentsGetCommand_InvalidID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `[]`))

	err := Execute(context.Background(), []string{"agents", "get", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestAgentsGetCommand_MissingID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `[]`))

	err := Execute(context.Background(), []string{"agents", "get"})
	if err == nil {
		t.Error("expected error for missing ID argument")
	}
}

func TestAgentsCreateCommand(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/agents", jsonResponse(200, `{
			"id": 3,
			"name": "New Agent",
			"email": "new@example.com",
			"role": "agent",
			"availability_status": "offline"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"agents", "create", "--name", "New Agent", "--email", "new@example.com", "--role", "agent"})
		if err != nil {
			t.Errorf("agents create failed: %v", err)
		}
	})

	if !strings.Contains(output, "New Agent") {
		t.Errorf("output missing 'New Agent': %s", output)
	}
	if !strings.Contains(output, "new@example.com") {
		t.Errorf("output missing email: %s", output)
	}
}

func TestAgentsCreateCommand_AdminRole(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/agents", jsonResponse(200, `{
			"id": 4,
			"name": "Admin User",
			"email": "admin@example.com",
			"role": "admin",
			"availability_status": "offline"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"agents", "create", "--name", "Admin User", "--email", "admin@example.com", "--role", "admin"})
		if err != nil {
			t.Errorf("agents create failed: %v", err)
		}
	})

	if !strings.Contains(output, "Admin User") {
		t.Errorf("output missing 'Admin User': %s", output)
	}
	if !strings.Contains(output, "admin") {
		t.Errorf("output missing role: %s", output)
	}
}

func TestAgentsCreateCommand_MissingName(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"agents", "create", "--email", "test@example.com", "--role", "agent"})
	if err == nil {
		t.Error("expected error for missing --name flag")
	}
}

func TestAgentsCreateCommand_MissingEmail(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"agents", "create", "--name", "Test", "--role", "agent"})
	if err == nil {
		t.Error("expected error for missing --email flag")
	}
}

func TestAgentsCreateCommand_MissingRole(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"agents", "create", "--name", "Test", "--email", "test@example.com"})
	if err == nil {
		t.Error("expected error for missing --role flag")
	}
}

func TestAgentsCreateCommand_InvalidRole(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"agents", "create", "--name", "Test", "--email", "test@example.com", "--role", "superadmin"})
	if err == nil {
		t.Error("expected error for invalid role")
	}
	if !strings.Contains(err.Error(), "agent") || !strings.Contains(err.Error(), "admin") {
		t.Errorf("error should mention valid roles: %v", err)
	}
}

func TestAgentsUpdateCommand(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/agents/1", jsonResponse(200, `{
			"id": 1,
			"name": "Updated Name",
			"email": "smith@example.com",
			"role": "agent",
			"availability_status": "online"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"agents", "update", "1", "--name", "Updated Name"})
		if err != nil {
			t.Errorf("agents update failed: %v", err)
		}
	})

	if !strings.Contains(output, "Updated Name") {
		t.Errorf("output missing 'Updated Name': %s", output)
	}
}

func TestAgentsBulkCreateCommand_EmailsFromStdin(t *testing.T) {
	var received struct {
		Emails []string `json:"emails"`
	}

	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/agents/bulk_create", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&received)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"id": 1, "name": "A", "email": "a@example.com", "role": "agent"},
				{"id": 2, "name": "B", "email": "b@example.com", "role": "agent"}
			]`))
		})

	setupTestEnvWithHandler(t, handler)

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	_, _ = w.WriteString("a@example.com\nb@example.com\n")
	_ = w.Close()

	err := Execute(context.Background(), []string{"agents", "bulk-create", "--emails", "@-"})
	if err != nil {
		t.Fatalf("agents bulk-create failed: %v", err)
	}
	if len(received.Emails) != 2 {
		t.Fatalf("expected 2 emails, got %d (%v)", len(received.Emails), received.Emails)
	}
	if received.Emails[0] != "a@example.com" || received.Emails[1] != "b@example.com" {
		t.Fatalf("unexpected emails: %v", received.Emails)
	}
}

func TestAgentsUpdateCommand_ChangeRole(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/agents/2", jsonResponse(200, `{
			"id": 2,
			"name": "Agent Jones",
			"email": "jones@example.com",
			"role": "admin",
			"availability_status": "offline"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"agents", "update", "2", "--role", "admin"})
		if err != nil {
			t.Errorf("agents update failed: %v", err)
		}
	})

	if !strings.Contains(output, "admin") {
		t.Errorf("output missing updated role: %s", output)
	}
}

func TestAgentsUpdateCommand_BothNameAndRole(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/agents/1", jsonResponse(200, `{
			"id": 1,
			"name": "New Name",
			"email": "smith@example.com",
			"role": "admin",
			"availability_status": "online"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"agents", "update", "1", "--name", "New Name", "--role", "admin"})
		if err != nil {
			t.Errorf("agents update failed: %v", err)
		}
	})

	if !strings.Contains(output, "New Name") {
		t.Errorf("output missing 'New Name': %s", output)
	}
	if !strings.Contains(output, "admin") {
		t.Errorf("output missing role: %s", output)
	}
}

func TestAgentsUpdateCommand_NoFlags(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"agents", "update", "1"})
	if err == nil {
		t.Error("expected error when no update flags provided")
	}
}

func TestAgentsUpdateCommand_InvalidRole(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"agents", "update", "1", "--role", "superadmin"})
	if err == nil {
		t.Error("expected error for invalid role")
	}
}

func TestAgentsUpdateCommand_InvalidID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"agents", "update", "invalid", "--name", "Test"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestAgentsDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/agents/1", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"agents", "delete", "1"})
		if err != nil {
			t.Errorf("agents delete failed: %v", err)
		}
	})

	if !strings.Contains(output, "Deleted agent 1") {
		t.Errorf("output missing success message: %s", output)
	}
}

func TestAgentsDeleteCommand_InvalidID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"agents", "delete", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestAgentsDeleteCommand_MissingID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	err := Execute(context.Background(), []string{"agents", "delete"})
	if err == nil {
		t.Error("expected error for missing ID argument")
	}
}

func TestAgentsDeleteCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/agents/1", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"agents", "delete", "1", "--output", "json"})
		if err != nil {
			t.Errorf("agents delete --json failed: %v", err)
		}
	})

	// In JSON mode, no output is printed for delete
	if strings.Contains(output, "Deleted agent") {
		t.Errorf("JSON mode should not print success message: %s", output)
	}
}

func TestAgentsCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agents", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"agents", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}
