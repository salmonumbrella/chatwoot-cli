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

func TestAutomationRulesListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/automation_rules", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Welcome Message", "event_name": "message_created", "active": true},
				{"id": 2, "name": "Auto Assign", "event_name": "conversation_opened", "active": false}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"automation-rules", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("automation-rules list failed: %v", err)
	}

	if !strings.Contains(output, "Welcome Message") {
		t.Errorf("output missing rule name: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "EVENT") || !strings.Contains(output, "ACTIVE") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "yes") || !strings.Contains(output, "no") {
		t.Errorf("output missing active status: %s", output)
	}
}

func TestAutomationRulesListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/automation_rules", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Welcome Message", "event_name": "message_created", "active": true}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"automation-rules", "list", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("automation-rules list failed: %v", err)
	}

	rules := decodeItems(t, output)
	if len(rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(rules))
	}
}

func TestAutomationRulesListCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/automation_rules", jsonResponse(200, `{"payload": []}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"automation-rules", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("automation-rules list failed: %v", err)
	}

	if !strings.Contains(output, "No automation rules found") {
		t.Errorf("expected empty message, got: %s", output)
	}
}

func TestAutomationRulesGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/automation_rules/1", jsonResponse(200, `{
			"payload": {
				"id": 1,
				"name": "Welcome Message",
				"event_name": "message_created",
				"active": true,
				"description": "Sends welcome message to new visitors",
				"conditions": [{"field": "status", "operator": "equals", "value": "open"}],
				"actions": [{"type": "send_message", "params": {"message": "Welcome!"}}]
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"automation-rules", "get", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("automation-rules get failed: %v", err)
	}

	if !strings.Contains(output, "Welcome Message") {
		t.Errorf("output missing rule name: %s", output)
	}
	if !strings.Contains(output, "message_created") {
		t.Errorf("output missing event name: %s", output)
	}
	if !strings.Contains(output, "Sends welcome message") {
		t.Errorf("output missing description: %s", output)
	}
}

func TestAutomationRulesGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/automation_rules/1", jsonResponse(200, `{
			"payload": {
				"id": 1,
				"name": "Welcome Message",
				"event_name": "message_created",
				"active": true
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"automation-rules", "get", "1", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("automation-rules get failed: %v", err)
	}

	var rule map[string]any
	if err := json.Unmarshal([]byte(output), &rule); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if rule["name"] != "Welcome Message" {
		t.Errorf("expected name 'Welcome Message', got %v", rule["name"])
	}
}

func TestAutomationRulesGetCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"automation-rules", "get", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "invalid rule ID") {
		t.Errorf("expected 'invalid rule ID' error, got: %v", err)
	}
}

func TestAutomationRulesGetCommand_AcceptsHashAndPrefixedIDs(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/automation_rules/1", jsonResponse(200, `{
			"payload": {
				"id": 1,
				"name": "Welcome Message",
				"event_name": "message_created",
				"active": true
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"automation-rules", "get", "#1"}); err != nil {
			t.Fatalf("automation-rules get hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output, "Welcome Message") {
		t.Errorf("output missing name: %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"automation-rules", "get", "rule:1"}); err != nil {
			t.Fatalf("automation-rules get prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "Welcome Message") {
		t.Errorf("output missing name: %s", output2)
	}
}

func TestAutomationRulesCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/automation_rules", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "New Rule", "event_name": "message_created", "active": true}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"automation-rules", "create",
		"--name", "New Rule",
		"--event-name", "message_created",
		"--conditions", `[{"field":"status","operator":"equals","value":"open"}]`,
		"--actions", `[{"type":"send_message"}]`,
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("automation-rules create failed: %v", err)
	}

	if !strings.Contains(output, "Created automation rule 1: New Rule") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["name"] != "New Rule" {
		t.Errorf("expected name 'New Rule', got %v", receivedBody["name"])
	}
	if receivedBody["event_name"] != "message_created" {
		t.Errorf("expected event_name 'message_created', got %v", receivedBody["event_name"])
	}
}

func TestAutomationRulesCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/automation_rules", jsonResponse(200, `{
			"id": 1,
			"name": "New Rule",
			"event_name": "message_created",
			"active": true
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"automation-rules", "create",
		"--name", "New Rule",
		"--event-name", "message_created",
		"--conditions", `[{"field":"status"}]`,
		"--actions", `[{"type":"send"}]`,
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("automation-rules create failed: %v", err)
	}

	var rule map[string]any
	if err := json.Unmarshal([]byte(output), &rule); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestAutomationRulesCreateCommand_InvalidConditions(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"automation-rules", "create",
		"--name", "Test",
		"--event-name", "message_created",
		"--conditions", "invalid-json",
		"--actions", `[{"type":"send"}]`,
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid conditions JSON") {
		t.Errorf("expected 'invalid conditions JSON' error, got: %v", err)
	}
}

func TestAutomationRulesCreateCommand_InvalidActions(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"automation-rules", "create",
		"--name", "Test",
		"--event-name", "message_created",
		"--conditions", `[{"field":"status"}]`,
		"--actions", "invalid-json",
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid actions JSON") {
		t.Errorf("expected 'invalid actions JSON' error, got: %v", err)
	}
}

func TestAutomationRulesUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/automation_rules/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"payload": {"id": 1, "name": "Updated Rule", "event_name": "message_created"}}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"automation-rules", "update", "1",
		"--name", "Updated Rule",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("automation-rules update failed: %v", err)
	}

	if !strings.Contains(output, "Updated automation rule 1: Updated Rule") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["name"] != "Updated Rule" {
		t.Errorf("expected name 'Updated Rule', got %v", receivedBody["name"])
	}
}

func TestAutomationRulesUpdateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/automation_rules/1", jsonResponse(200, `{
			"payload": {
				"id": 1,
				"name": "Updated Rule",
				"event_name": "message_created"
			}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"automation-rules", "update", "1",
		"--name", "Updated Rule",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("automation-rules update failed: %v", err)
	}

	var rule map[string]any
	if err := json.Unmarshal([]byte(output), &rule); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestAutomationRulesUpdateCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"automation-rules", "update", "invalid", "--name", "Test"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "invalid rule ID") {
		t.Errorf("expected 'invalid rule ID' error, got: %v", err)
	}
}

func TestAutomationRulesUpdateCommand_InvalidConditions(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"automation-rules", "update", "1",
		"--conditions", "invalid-json",
	})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid conditions JSON") {
		t.Errorf("expected 'invalid conditions JSON' error, got: %v", err)
	}
}

func TestAutomationRulesDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/automation_rules/1", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"automation-rules", "delete", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("automation-rules delete failed: %v", err)
	}

	if !strings.Contains(output, "Deleted automation rule 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestAutomationRulesDeleteCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/automation_rules/1", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"automation-rules", "delete", "1", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("automation-rules delete failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if result["deleted"] != true {
		t.Errorf("expected deleted to be true, got %v", result["deleted"])
	}
}

func TestAutomationRulesDeleteCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"automation-rules", "delete", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "invalid rule ID") {
		t.Errorf("expected 'invalid rule ID' error, got: %v", err)
	}
}

func TestAutomationRulesListCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/automation_rules", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"automation-rules", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

// Test the "automation" alias
func TestAutomationRulesListCommand_AutomationAlias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/automation_rules", jsonResponse(200, `{
			"payload": [{"id": 1, "name": "Rule One", "event_name": "message_created", "active": true}]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"automation", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("automation list failed: %v", err)
	}

	if !strings.Contains(output, "Rule One") {
		t.Errorf("output missing rule name: %s", output)
	}
}

// Test the "rules" alias
func TestAutomationRulesListCommand_RulesAlias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/automation_rules", jsonResponse(200, `{
			"payload": [{"id": 1, "name": "Rule One", "event_name": "message_created", "active": true}]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"rules", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("rules list failed: %v", err)
	}

	if !strings.Contains(output, "Rule One") {
		t.Errorf("output missing rule name: %s", output)
	}
}
