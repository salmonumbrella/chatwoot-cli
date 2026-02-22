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

func TestTeamsListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams", jsonResponse(200, `[
			{"id": 1, "name": "Support", "description": "Support team", "allow_auto_assign": true, "account_id": 1},
			{"id": 2, "name": "Sales", "description": "Sales team", "allow_auto_assign": false, "account_id": 1}
		]`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"teams", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams list failed: %v", err)
	}

	// Verify output contains expected teams
	if !strings.Contains(output, "Support") {
		t.Errorf("output missing 'Support': %s", output)
	}
	if !strings.Contains(output, "Sales") {
		t.Errorf("output missing 'Sales': %s", output)
	}
	// Check auto-assign column
	if !strings.Contains(output, "yes") {
		t.Errorf("output missing 'yes' for auto-assign: %s", output)
	}
	if !strings.Contains(output, "no") {
		t.Errorf("output missing 'no' for auto-assign: %s", output)
	}
	// Check headers
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestTeamsListCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"teams", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("teams list failed: %v", err)
	}
	// Empty list should still have headers
}

func TestTeamsListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams", jsonResponse(200, `[
			{"id": 1, "name": "Support", "description": "Support team"}
		]`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"teams", "list", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams list failed: %v", err)
	}

	teams := decodeItems(t, output)
	if len(teams) != 1 {
		t.Errorf("expected 1 team, got %d", len(teams))
	}
}

func TestTeamsGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams/123", jsonResponse(200, `{
			"id": 123,
			"name": "Engineering",
			"description": "Engineering team",
			"allow_auto_assign": true,
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"teams", "get", "123"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams get failed: %v", err)
	}

	// Verify output contains team details
	if !strings.Contains(output, "Engineering") {
		t.Errorf("output missing 'Engineering': %s", output)
	}
	if !strings.Contains(output, "123") {
		t.Errorf("output missing ID: %s", output)
	}
}

func TestTeamsGetCommand_AcceptsURLAndPrefixedIDs(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams/123", jsonResponse(200, `{
			"id": 123,
			"name": "Engineering",
			"description": "Engineering team",
			"allow_auto_assign": true,
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"teams", "get", "https://app.chatwoot.com/app/accounts/1/teams/123"}); err != nil {
			t.Fatalf("teams get URL failed: %v", err)
		}
	})
	if !strings.Contains(output, "Engineering") {
		t.Errorf("output missing 'Engineering': %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"teams", "get", "team:123"}); err != nil {
			t.Fatalf("teams get prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "Engineering") {
		t.Errorf("output missing 'Engineering': %s", output2)
	}

	output3 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"teams", "get", "#123"}); err != nil {
			t.Fatalf("teams get hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output3, "Engineering") {
		t.Errorf("output missing 'Engineering': %s", output3)
	}
}

func TestTeamsGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams/123", jsonResponse(200, `{
			"id": 123,
			"name": "Engineering",
			"description": "Engineering team",
			"allow_auto_assign": true,
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"teams", "get", "123", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams get failed: %v", err)
	}

	// Verify it's valid JSON
	var team map[string]any
	if err := json.Unmarshal([]byte(output), &team); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if team["name"] != "Engineering" {
		t.Errorf("expected name 'Engineering', got %v", team["name"])
	}
}

func TestTeamsGetCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "get", "abc"})
	if err == nil {
		t.Error("expected error for invalid team ID")
	}

	err = Execute(context.Background(), []string{"teams", "get", "-1"})
	if err == nil {
		t.Error("expected error for negative team ID")
	}
}

func TestTeamsGetCommand_MissingID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "get"})
	if err == nil {
		t.Error("expected error when team ID is missing")
	}
}

func TestTeamsCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/teams", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 3, "name": "DevOps", "description": "DevOps team"}`))
		})

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"teams", "create",
		"--name", "DevOps",
		"--description", "DevOps team",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams create failed: %v", err)
	}

	// Verify output
	if !strings.Contains(output, "Created team") {
		t.Errorf("expected success message, got: %s", output)
	}

	// Verify request body
	if receivedBody["name"] != "DevOps" {
		t.Errorf("expected name 'DevOps', got %v", receivedBody["name"])
	}
	if receivedBody["description"] != "DevOps team" {
		t.Errorf("expected description 'DevOps team', got %v", receivedBody["description"])
	}
}

func TestTeamsCreateCommand_MinimalOptions(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/teams", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 4, "name": "Minimal"}`))
		})

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"teams", "create", "--name", "Minimal"})
	if err != nil {
		t.Errorf("teams create failed: %v", err)
	}

	// Verify required field is sent
	if receivedBody["name"] != "Minimal" {
		t.Errorf("expected name 'Minimal', got %v", receivedBody["name"])
	}
}

func TestTeamsCreateCommand_EmitID(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/teams", jsonResponse(200, `{"id": 4, "name": "Minimal"}`))

	setupTestEnvWithHandler(t, handler)

	out := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"teams", "create", "--name", "Minimal", "--emit", "id"}); err != nil {
			t.Fatalf("teams create --emit id failed: %v", err)
		}
	})
	if strings.TrimSpace(out) != "team:4" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestTeamsCreateCommand_MissingName(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "create"})
	if err == nil {
		t.Error("expected error when name is missing")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected '--name is required' error, got: %v", err)
	}
}

func TestTeamsCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/teams", jsonResponse(200, `{
			"id": 5,
			"name": "json-team",
			"description": "JSON test team"
		}`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"teams", "create",
		"--name", "json-team",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams create failed: %v", err)
	}

	// Verify it's valid JSON
	var team map[string]any
	if err := json.Unmarshal([]byte(output), &team); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if team["name"] != "json-team" {
		t.Errorf("expected name 'json-team', got %v", team["name"])
	}
}

func TestTeamsUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/teams/10", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 10, "name": "Updated Team", "description": "Updated desc"}`))
		})

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"teams", "update", "10",
		"--name", "Updated Team",
		"--description", "Updated desc",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams update failed: %v", err)
	}

	// Verify output
	if !strings.Contains(output, "Updated team") {
		t.Errorf("expected success message, got: %s", output)
	}

	// Verify request body
	if receivedBody["name"] != "Updated Team" {
		t.Errorf("expected name 'Updated Team', got %v", receivedBody["name"])
	}
	if receivedBody["description"] != "Updated desc" {
		t.Errorf("expected description 'Updated desc', got %v", receivedBody["description"])
	}
}

func TestTeamsUpdateCommand_PartialUpdate(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/teams/20", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 20, "name": "Partial Update"}`))
		})

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"teams", "update", "20", "--name", "Partial Update"})
	if err != nil {
		t.Errorf("teams update failed: %v", err)
	}

	// Verify only name is in body
	if receivedBody["name"] != "Partial Update" {
		t.Errorf("expected name 'Partial Update', got %v", receivedBody["name"])
	}
	if _, ok := receivedBody["description"]; ok && receivedBody["description"] != "" {
		t.Errorf("description should be empty or not in body when not specified")
	}
}

func TestTeamsUpdateCommand_NoChanges(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "update", "10"})
	if err == nil {
		t.Error("expected error when no changes provided")
	}
	if !strings.Contains(err.Error(), "at least one of --name or --description") {
		t.Errorf("expected appropriate error message, got: %v", err)
	}
}

func TestTeamsUpdateCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "update", "abc", "--name", "test"})
	if err == nil {
		t.Error("expected error for invalid team ID")
	}
}

func TestTeamsUpdateCommand_MissingID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "update", "--name", "test"})
	if err == nil {
		t.Error("expected error when team ID is missing")
	}
}

func TestTeamsDeleteCommand(t *testing.T) {
	deleteRequested := false
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/teams/50", func(w http.ResponseWriter, r *http.Request) {
			deleteRequested = true
			w.WriteHeader(http.StatusOK)
		})

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"teams", "delete", "50"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams delete failed: %v", err)
	}

	if !deleteRequested {
		t.Error("expected DELETE request to be made")
	}

	if !strings.Contains(output, "Deleted team 50") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestTeamsDeleteCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/teams/50", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"teams", "delete", "50", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams delete failed: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if result["deleted"] != true {
		t.Errorf("expected deleted: true, got %v", result["deleted"])
	}
}

func TestTeamsDeleteCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "delete", "abc"})
	if err == nil {
		t.Error("expected error for invalid team ID")
	}
}

func TestTeamsDeleteCommand_MissingID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "delete"})
	if err == nil {
		t.Error("expected error when team ID is missing")
	}
}

func TestTeamsMembersCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams/5/team_members", jsonResponse(200, `[
			{"id": 1, "name": "John Doe", "email": "john@example.com", "role": "agent", "availability_status": "online"},
			{"id": 2, "name": "Jane Smith", "email": "jane@example.com", "role": "administrator", "availability_status": "offline"}
		]`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"teams", "members", "5"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams members failed: %v", err)
	}

	// Verify output contains expected members
	if !strings.Contains(output, "John Doe") {
		t.Errorf("output missing 'John Doe': %s", output)
	}
	if !strings.Contains(output, "Jane Smith") {
		t.Errorf("output missing 'Jane Smith': %s", output)
	}
	if !strings.Contains(output, "john@example.com") {
		t.Errorf("output missing email: %s", output)
	}
	// Check headers
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestTeamsMembersCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams/5/team_members", jsonResponse(200, `[
			{"id": 1, "name": "John Doe", "email": "john@example.com"}
		]`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"teams", "members", "5", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams members failed: %v", err)
	}

	members := decodeItems(t, output)
	if len(members) != 1 {
		t.Errorf("expected 1 member, got %d", len(members))
	}
}

func TestTeamsMembersCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "members", "abc"})
	if err == nil {
		t.Error("expected error for invalid team ID")
	}
}

func TestTeamsMembersCommand_MissingID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "members"})
	if err == nil {
		t.Error("expected error when team ID is missing")
	}
}

func TestTeamsMembersAddCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/teams/5/team_members", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
		})

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"teams", "members-add", "5",
		"--user-ids", "1,2,3",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams members-add failed: %v", err)
	}

	// Verify output
	if !strings.Contains(output, "Added 3 member(s)") {
		t.Errorf("expected success message, got: %s", output)
	}

	// Verify request body contains user IDs
	userIDs, ok := receivedBody["user_ids"].([]any)
	if !ok {
		t.Errorf("expected user_ids array in body, got %v", receivedBody)
	}
	if len(userIDs) != 3 {
		t.Errorf("expected 3 user IDs, got %d", len(userIDs))
	}
}

func TestTeamsMembersAddCommand_UserIDsFromStdin(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/teams/5/team_members", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
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
		_, _ = w.Write([]byte("1\n2\n3\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"teams", "members-add", "5",
			"--user-ids", "@-",
		})
		if err != nil {
			t.Errorf("teams members-add failed: %v", err)
		}
	})

	if !strings.Contains(output, "Added 3 member(s)") {
		t.Errorf("expected success message, got: %s", output)
	}

	userIDs, ok := receivedBody["user_ids"].([]any)
	if !ok {
		t.Errorf("expected user_ids array in body, got %v", receivedBody)
	}
	if len(userIDs) != 3 {
		t.Errorf("expected 3 user IDs, got %d", len(userIDs))
	}
}

func TestTeamsMembersAddCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/teams/5/team_members", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"teams", "members-add", "5",
		"--user-ids", "1,2",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams members-add failed: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if result["added_count"] != float64(2) {
		t.Errorf("expected added_count: 2, got %v", result["added_count"])
	}
}

func TestTeamsMembersAddCommand_MissingUserIds(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "members-add", "5"})
	if err == nil {
		t.Error("expected error when user-ids is missing")
	}
	if !strings.Contains(err.Error(), "--user-ids is required") {
		t.Errorf("expected '--user-ids is required' error, got: %v", err)
	}
}

func TestTeamsMembersAddCommand_InvalidUserIds(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "members-add", "5", "--user-ids", "abc,def"})
	if err == nil {
		t.Error("expected error for invalid user IDs")
	}
}

func TestTeamsMembersAddCommand_InvalidTeamID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "members-add", "abc", "--user-ids", "1,2"})
	if err == nil {
		t.Error("expected error for invalid team ID")
	}
}

func TestTeamsMembersRemoveCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/teams/5/team_members", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.WriteHeader(http.StatusOK)
		})

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"teams", "members-remove", "5",
		"--user-ids", "1,2",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams members-remove failed: %v", err)
	}

	// Verify output
	if !strings.Contains(output, "Removed 2 member(s)") {
		t.Errorf("expected success message, got: %s", output)
	}

	// Verify request body contains user IDs
	userIDs, ok := receivedBody["user_ids"].([]any)
	if !ok {
		t.Errorf("expected user_ids array in body, got %v", receivedBody)
	}
	if len(userIDs) != 2 {
		t.Errorf("expected 2 user IDs, got %d", len(userIDs))
	}
}

func TestTeamsMembersRemoveCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/teams/5/team_members", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"teams", "members-remove", "5",
		"--user-ids", "1",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("teams members-remove failed: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if result["removed_count"] != float64(1) {
		t.Errorf("expected removed_count: 1, got %v", result["removed_count"])
	}
}

func TestTeamsMembersRemoveCommand_MissingUserIds(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "members-remove", "5"})
	if err == nil {
		t.Error("expected error when user-ids is missing")
	}
	if !strings.Contains(err.Error(), "--user-ids is required") {
		t.Errorf("expected '--user-ids is required' error, got: %v", err)
	}
}

func TestTeamsMembersRemoveCommand_InvalidTeamID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"teams", "members-remove", "abc", "--user-ids", "1,2"})
	if err == nil {
		t.Error("expected error for invalid team ID")
	}
}

func TestTeamsCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"teams", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestTeamsGetCommand_NotFound(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/teams/999", jsonResponse(404, `{"error": "Team not found"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"teams", "get", "999"})
	if err == nil {
		t.Error("expected error for not found team")
	}
}
