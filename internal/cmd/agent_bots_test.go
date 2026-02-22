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

func TestAgentBotsListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agent_bots", jsonResponse(200, `[
			{"id": 1, "name": "Bot One", "outgoing_url": "https://bot1.example.com"},
			{"id": 2, "name": "Bot Two", "outgoing_url": "https://bot2.example.com"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"agent-bots", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("agent-bots list failed: %v", err)
	}

	if !strings.Contains(output, "Bot One") {
		t.Errorf("output missing bot name: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "OUTGOING_URL") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestAgentBotsListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agent_bots", jsonResponse(200, `[
			{"id": 1, "name": "Bot One", "outgoing_url": "https://bot1.example.com"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"agent-bots", "list", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("agent-bots list failed: %v", err)
	}

	bots := decodeItems(t, output)
	if len(bots) != 1 {
		t.Errorf("expected 1 bot, got %d", len(bots))
	}
}

func TestAgentBotsListCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agent_bots", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"agent-bots", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("agent-bots list failed: %v", err)
	}

	if !strings.Contains(output, "No agent bots found") {
		t.Errorf("expected empty message, got: %s", output)
	}
}

func TestAgentBotsGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agent_bots/1", jsonResponse(200, `{
			"id": 1,
			"name": "Bot One",
			"description": "Test bot description",
			"outgoing_url": "https://bot1.example.com"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"agent-bots", "get", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("agent-bots get failed: %v", err)
	}

	if !strings.Contains(output, "Bot One") {
		t.Errorf("output missing bot name: %s", output)
	}
	if !strings.Contains(output, "Test bot description") {
		t.Errorf("output missing description: %s", output)
	}
	if !strings.Contains(output, "https://bot1.example.com") {
		t.Errorf("output missing outgoing URL: %s", output)
	}
}

func TestAgentBotsGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agent_bots/1", jsonResponse(200, `{
			"id": 1,
			"name": "Bot One",
			"outgoing_url": "https://bot1.example.com"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"agent-bots", "get", "1", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("agent-bots get failed: %v", err)
	}

	var bot map[string]any
	if err := json.Unmarshal([]byte(output), &bot); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if bot["name"] != "Bot One" {
		t.Errorf("expected name 'Bot One', got %v", bot["name"])
	}
}

func TestAgentBotsGetCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"agent-bots", "get", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "invalid bot ID") {
		t.Errorf("expected 'invalid bot ID' error, got: %v", err)
	}
}

func TestAgentBotsGetCommand_AcceptsHashAndPrefixedIDs(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agent_bots/1", jsonResponse(200, `{
			"id": 1,
			"name": "Bot One",
			"description": "Test bot description",
			"outgoing_url": "https://bot1.example.com"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"agent-bots", "get", "#1"}); err != nil {
			t.Fatalf("agent-bots get hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output, "Bot One") {
		t.Errorf("output missing bot name: %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"agent-bots", "get", "bot:1"}); err != nil {
			t.Fatalf("agent-bots get prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "Bot One") {
		t.Errorf("output missing bot name: %s", output2)
	}
}

func TestAgentBotsCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/agent_bots", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "New Bot", "outgoing_url": "https://bot.example.com"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"agent-bots", "create",
		"--name", "New Bot",
		"--outgoing-url", "https://bot.example.com",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("agent-bots create failed: %v", err)
	}

	if !strings.Contains(output, "Created agent bot 1: New Bot") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["name"] != "New Bot" {
		t.Errorf("expected name 'New Bot', got %v", receivedBody["name"])
	}
	if receivedBody["outgoing_url"] != "https://bot.example.com" {
		t.Errorf("expected outgoing_url 'https://bot.example.com', got %v", receivedBody["outgoing_url"])
	}
}

func TestAgentBotsCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/agent_bots", jsonResponse(200, `{
			"id": 1,
			"name": "New Bot",
			"outgoing_url": "https://bot.example.com"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"agent-bots", "create",
		"--name", "New Bot",
		"--outgoing-url", "https://bot.example.com",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("agent-bots create failed: %v", err)
	}

	var bot map[string]any
	if err := json.Unmarshal([]byte(output), &bot); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestAgentBotsCreateCommand_InvalidURL(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"agent-bots", "create",
		"--name", "Test Bot",
		"--outgoing-url", "ftp://invalid-scheme.com",
	})
	if err == nil {
		t.Error("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "invalid outgoing URL") {
		t.Errorf("expected 'invalid outgoing URL' error, got: %v", err)
	}
}

func TestAgentBotsUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/agent_bots/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "Updated Bot", "outgoing_url": "https://updated.example.com"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"agent-bots", "update", "1",
		"--name", "Updated Bot",
		"--outgoing-url", "https://updated.example.com",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("agent-bots update failed: %v", err)
	}

	if !strings.Contains(output, "Updated agent bot 1: Updated Bot") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["name"] != "Updated Bot" {
		t.Errorf("expected name 'Updated Bot', got %v", receivedBody["name"])
	}
}

func TestAgentBotsUpdateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/agent_bots/1", jsonResponse(200, `{
			"id": 1,
			"name": "Updated Bot",
			"outgoing_url": "https://updated.example.com"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"agent-bots", "update", "1",
		"--name", "Updated Bot",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("agent-bots update failed: %v", err)
	}

	var bot map[string]any
	if err := json.Unmarshal([]byte(output), &bot); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestAgentBotsUpdateCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"agent-bots", "update", "invalid", "--name", "Test"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "invalid bot ID") {
		t.Errorf("expected 'invalid bot ID' error, got: %v", err)
	}
}

func TestAgentBotsDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/agent_bots/1", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"agent-bots", "delete", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("agent-bots delete failed: %v", err)
	}

	if !strings.Contains(output, "Deleted agent bot 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestAgentBotsDeleteCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/agent_bots/1", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"agent-bots", "delete", "1", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("agent-bots delete failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if result["deleted"] != true {
		t.Errorf("expected deleted to be true, got %v", result["deleted"])
	}
}

func TestAgentBotsDeleteCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"agent-bots", "delete", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "invalid bot ID") {
		t.Errorf("expected 'invalid bot ID' error, got: %v", err)
	}
}

func TestAgentBotsListCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agent_bots", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"agent-bots", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

// Test the "bots" alias
func TestAgentBotsListCommand_BotsAlias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/agent_bots", jsonResponse(200, `[
			{"id": 1, "name": "Bot One", "outgoing_url": "https://bot1.example.com"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"bots", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("bots list failed: %v", err)
	}

	if !strings.Contains(output, "Bot One") {
		t.Errorf("output missing bot name: %s", output)
	}
}
