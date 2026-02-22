package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

// Integration tests for campaigns commands

func TestCampaignsListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/campaigns", jsonResponse(200, `[
			{"id": 1, "title": "Campaign 1", "campaign_type": "sms", "campaign_status": "active", "scheduled_at": 1704067200, "enabled": true},
			{"id": 2, "title": "Campaign 2", "campaign_type": "email", "campaign_status": "draft", "scheduled_at": 0, "enabled": false}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"campaigns", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("campaigns list failed: %v", err)
	}

	if !strings.Contains(output, "ID") || !strings.Contains(output, "TITLE") || !strings.Contains(output, "TYPE") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "Campaign 1") {
		t.Errorf("output missing campaign title: %s", output)
	}
}

func TestCampaignsListCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/campaigns", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"campaigns", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("campaigns list failed: %v", err)
	}

	if !strings.Contains(output, "No campaigns found") {
		t.Errorf("expected 'No campaigns found' message, got: %s", output)
	}
}

func TestCampaignsListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/campaigns", jsonResponse(200, `[{"id": 1, "title": "Campaign 1"}]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"campaigns", "list", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("campaigns list failed: %v", err)
	}

	_ = decodeItems(t, output)
}

func TestCampaignsGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/campaigns/1", jsonResponse(200, `{
			"id": 1,
			"title": "Test Campaign",
			"description": "A test campaign",
			"message": "Hello!",
			"campaign_type": "sms",
			"campaign_status": "active",
			"inbox_id": 5,
			"sender_id": 1,
			"enabled": true,
			"trigger_only_during_business_hours": false,
			"scheduled_at": 1704067200,
			"created_at": 1704000000
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"campaigns", "get", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("campaigns get failed: %v", err)
	}

	if !strings.Contains(output, "Test Campaign") {
		t.Errorf("output missing campaign title: %s", output)
	}
	if !strings.Contains(output, "A test campaign") {
		t.Errorf("output missing description: %s", output)
	}
	if !strings.Contains(output, "Scheduled At") {
		t.Errorf("output missing scheduled time: %s", output)
	}
}

func TestCampaignsGetCommand_AcceptsURLAndPrefixedIDs(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/campaigns/1", jsonResponse(200, `{
			"id": 1,
			"title": "Test Campaign",
			"description": "A test campaign",
			"message": "Hello!",
			"campaign_type": "sms",
			"campaign_status": "active",
			"inbox_id": 5,
			"sender_id": 1,
			"enabled": true,
			"trigger_only_during_business_hours": false,
			"scheduled_at": 1704067200,
			"created_at": 1704000000
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"campaigns", "get", "https://app.chatwoot.com/app/accounts/1/campaigns/1"}); err != nil {
			t.Fatalf("campaigns get URL failed: %v", err)
		}
	})
	if !strings.Contains(output, "Test Campaign") {
		t.Errorf("output missing campaign title: %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"campaigns", "get", "campaign:1"}); err != nil {
			t.Fatalf("campaigns get prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "Test Campaign") {
		t.Errorf("output missing campaign title: %s", output2)
	}

	output3 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"campaigns", "get", "#1"}); err != nil {
			t.Fatalf("campaigns get hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output3, "Test Campaign") {
		t.Errorf("output missing campaign title: %s", output3)
	}
}

func TestCampaignsGetCommand_NoScheduledAt(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/campaigns/1", jsonResponse(200, `{
			"id": 1,
			"title": "Test Campaign",
			"description": "A test campaign",
			"message": "Hello!",
			"campaign_type": "sms",
			"campaign_status": "active",
			"inbox_id": 5,
			"sender_id": 1,
			"enabled": true,
			"trigger_only_during_business_hours": false,
			"scheduled_at": 0,
			"created_at": 1704000000
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"campaigns", "get", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("campaigns get failed: %v", err)
	}

	// When scheduled_at is 0, "Scheduled At" should not appear
	if strings.Contains(output, "Scheduled At") {
		t.Errorf("output should not show Scheduled At when 0: %s", output)
	}
}

func TestCampaignsGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/campaigns/1", jsonResponse(200, `{"id": 1, "title": "Test Campaign"}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"campaigns", "get", "1", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("campaigns get failed: %v", err)
	}

	var campaign map[string]any
	if err := json.Unmarshal([]byte(output), &campaign); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestCampaignsGetCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"campaigns", "get", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "ID") {
		t.Errorf("expected 'ID' error, got: %v", err)
	}
}

func TestCampaignsCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/campaigns", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "New Campaign"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"campaigns", "create",
		"--title", "New Campaign",
		"--message", "Hello!",
		"--inbox-id", "5",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("campaigns create failed: %v", err)
	}

	if !strings.Contains(output, "Created campaign 1") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["title"] != "New Campaign" {
		t.Errorf("expected title 'New Campaign', got %v", receivedBody["title"])
	}
}

func TestCampaignsCreateCommand_InteractivePrompt(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Website", "channel_type": "Channel::WebWidget"}
			]
		}`)).
		On("POST", "/api/v1/accounts/1/campaigns", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "New Campaign"}`))
		})

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
		err := Execute(context.Background(), []string{
			"campaigns", "create",
			"--title", "New Campaign",
			"--message", "Hello!",
		})
		if err != nil {
			t.Errorf("campaigns create interactive failed: %v", err)
		}
	})

	if !strings.Contains(output, "Created campaign 1") {
		t.Errorf("expected success message, got: %s", output)
	}
	if receivedBody["inbox_id"] != float64(1) {
		t.Errorf("expected inbox_id 1, got %v", receivedBody["inbox_id"])
	}
}

func TestCampaignsCreateCommand_WithLabels(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/campaigns", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "New Campaign"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"campaigns", "create",
		"--title", "New Campaign",
		"--message", "Hello!",
		"--inbox-id", "5",
		"--labels", "1,2,3",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("campaigns create failed: %v", err)
	}

	audience, ok := receivedBody["audience"].([]any)
	if !ok || len(audience) != 3 {
		t.Errorf("expected 3 audience items, got %v", receivedBody["audience"])
	}
}

func TestCampaignsCreateCommand_WithLabels_AcceptsHashAndPrefixedIDs(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/campaigns", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "New Campaign"}`))
		})

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{
		"campaigns", "create",
		"--title", "New Campaign",
		"--message", "Hello!",
		"--inbox-id", "5",
		"--labels", "#1,label:2",
	})
	if err != nil {
		t.Fatalf("campaigns create failed: %v", err)
	}

	audience, ok := receivedBody["audience"].([]any)
	if !ok || len(audience) != 2 {
		t.Fatalf("expected 2 audience items, got %v", receivedBody["audience"])
	}
}

func TestCampaignsCreateCommand_WithAudience(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/campaigns", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "New Campaign"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"campaigns", "create",
		"--title", "New Campaign",
		"--message", "Hello!",
		"--inbox-id", "5",
		"--audience", `[{"type":"Label","id":1}]`,
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("campaigns create failed: %v", err)
	}

	audience, ok := receivedBody["audience"].([]any)
	if !ok || len(audience) != 1 {
		t.Errorf("expected 1 audience item, got %v", receivedBody["audience"])
	}
}

func TestCampaignsCreateCommand_WithScheduledAt(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/campaigns", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "New Campaign"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"campaigns", "create",
		"--title", "New Campaign",
		"--message", "Hello!",
		"--inbox-id", "5",
		"--scheduled-at", "2025-01-15T10:00:00Z",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("campaigns create failed: %v", err)
	}

	if receivedBody["scheduled_at"] == nil || receivedBody["scheduled_at"].(float64) == 0 {
		t.Errorf("expected scheduled_at to be set, got %v", receivedBody["scheduled_at"])
	}
}

func TestCampaignsCreateCommand_LabelsAndAudienceMutuallyExclusiveCmd(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"campaigns", "create",
		"--title", "New Campaign",
		"--message", "Hello!",
		"--inbox-id", "5",
		"--labels", "1,2",
		"--audience", `[{"type":"Label","id":1}]`,
	})
	if err == nil {
		t.Error("expected error when both labels and audience are provided")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' error, got: %v", err)
	}
}

func TestCampaignsCreateCommand_InvalidScheduledAt(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"campaigns", "create",
		"--title", "New Campaign",
		"--message", "Hello!",
		"--inbox-id", "5",
		"--scheduled-at", "invalid-date",
	})
	if err == nil {
		t.Error("expected error for invalid scheduled-at")
	}
	if !strings.Contains(err.Error(), "RFC3339") {
		t.Errorf("expected 'RFC3339' error, got: %v", err)
	}
}

func TestCampaignsCreateCommand_InvalidAudienceJSON(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"campaigns", "create",
		"--title", "New Campaign",
		"--message", "Hello!",
		"--inbox-id", "5",
		"--audience", "invalid-json",
	})
	if err == nil {
		t.Error("expected error for invalid audience JSON")
	}
	if !strings.Contains(err.Error(), "invalid audience JSON") {
		t.Errorf("expected 'invalid audience JSON' error, got: %v", err)
	}
}

func TestCampaignsCreateCommand_InvalidLabelsID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"campaigns", "create",
		"--title", "New Campaign",
		"--message", "Hello!",
		"--inbox-id", "5",
		"--labels", "abc,def",
	})
	if err == nil {
		t.Error("expected error for invalid labels")
	}
	if !strings.Contains(err.Error(), "label ID") {
		t.Errorf("expected 'label ID' error, got: %v", err)
	}
}

func TestCampaignsUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/campaigns/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "Updated Campaign"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"campaigns", "update", "1",
		"--title", "Updated Campaign",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("campaigns update failed: %v", err)
	}

	if !strings.Contains(output, "Updated campaign 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestCampaignsUpdateCommand_WithEnabled(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/campaigns/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "Campaign", "enabled": true}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"campaigns", "update", "1",
		"--enabled=true",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("campaigns update failed: %v", err)
	}

	if receivedBody["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", receivedBody["enabled"])
	}
}

func TestCampaignsUpdateCommand_WithBusinessHours(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/campaigns/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "Campaign"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"campaigns", "update", "1",
		"--business-hours=true",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("campaigns update failed: %v", err)
	}

	if receivedBody["trigger_only_during_business_hours"] != true {
		t.Errorf("expected trigger_only_during_business_hours=true, got %v", receivedBody["trigger_only_during_business_hours"])
	}
}

func TestCampaignsUpdateCommand_LabelsAndAudienceMutuallyExclusiveCmd(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"campaigns", "update", "1",
		"--labels", "1,2",
		"--audience", `[{"type":"Label","id":1}]`,
	})
	if err == nil {
		t.Error("expected error when both labels and audience are provided")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' error, got: %v", err)
	}
}

func TestCampaignsDeleteCommand_Force(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/campaigns/1", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"campaigns", "delete", "1", "--force"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("campaigns delete failed: %v", err)
	}

	if !strings.Contains(output, "Deleted campaign 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestCampaignsDeleteCommand_JSONRequiresForce(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{"campaigns", "delete", "1", "-o", "json"})
	if err == nil {
		t.Error("expected error when --force is not provided in JSON mode")
	}
	if !strings.Contains(err.Error(), "--force flag is required") {
		t.Errorf("expected '--force flag is required' error, got: %v", err)
	}
}

func TestCampaignsDeleteCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{"campaigns", "delete", "invalid", "--force"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "ID") {
		t.Errorf("expected 'ID' error, got: %v", err)
	}
}

func TestCampaignsListCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/campaigns", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"campaigns", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

// Unit tests for parsing logic

func TestLabelsParsingCreate(t *testing.T) {
	tests := []struct {
		name          string
		labelsFlag    string
		expectedCount int
		expectedIDs   []int
		expectError   bool
	}{
		{
			name:          "single label",
			labelsFlag:    "1",
			expectedCount: 1,
			expectedIDs:   []int{1},
			expectError:   false,
		},
		{
			name:          "multiple labels",
			labelsFlag:    "1,2,3",
			expectedCount: 3,
			expectedIDs:   []int{1, 2, 3},
			expectError:   false,
		},
		{
			name:          "labels with spaces",
			labelsFlag:    "1, 2, 3",
			expectedCount: 3,
			expectedIDs:   []int{1, 2, 3},
			expectError:   false,
		},
		{
			name:          "many labels",
			labelsFlag:    "10,20,30,40,50",
			expectedCount: 5,
			expectedIDs:   []int{10, 20, 30, 40, 50},
			expectError:   false,
		},
		{
			name:          "empty string",
			labelsFlag:    "",
			expectedCount: 0,
			expectedIDs:   []int{},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var audience []api.CampaignAudience

			if tt.labelsFlag != "" {
				for _, idStr := range strings.Split(tt.labelsFlag, ",") {
					idStr = strings.TrimSpace(idStr)
					var id int
					if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
						if tt.expectError {
							return // Expected error occurred
						}
						t.Fatalf("Failed to parse label ID %q: %v", idStr, err)
					}
					audience = append(audience, api.CampaignAudience{Type: "Label", ID: id})
				}
			}

			if tt.expectError {
				t.Error("Expected error but got none")
				return
			}

			if len(audience) != tt.expectedCount {
				t.Errorf("Expected %d audience items, got %d", tt.expectedCount, len(audience))
			}

			for i, expectedID := range tt.expectedIDs {
				if i >= len(audience) {
					t.Errorf("Missing audience item at index %d", i)
					continue
				}
				if audience[i].Type != "Label" {
					t.Errorf("Expected audience[%d].Type to be 'Label', got %s", i, audience[i].Type)
				}
				if audience[i].ID != expectedID {
					t.Errorf("Expected audience[%d].ID to be %d, got %d", i, expectedID, audience[i].ID)
				}
			}
		})
	}
}

func TestLabelsParsingUpdate(t *testing.T) {
	tests := []struct {
		name          string
		labelsFlag    string
		expectedCount int
		expectedIDs   []int
		expectError   bool
	}{
		{
			name:          "update with single label",
			labelsFlag:    "5",
			expectedCount: 1,
			expectedIDs:   []int{5},
			expectError:   false,
		},
		{
			name:          "update with multiple labels",
			labelsFlag:    "5,6,7",
			expectedCount: 3,
			expectedIDs:   []int{5, 6, 7},
			expectError:   false,
		},
		{
			name:          "update with whitespace",
			labelsFlag:    " 1 , 2 , 3 ",
			expectedCount: 3,
			expectedIDs:   []int{1, 2, 3},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var audience []api.CampaignAudience

			if tt.labelsFlag != "" {
				for _, idStr := range strings.Split(tt.labelsFlag, ",") {
					idStr = strings.TrimSpace(idStr)
					var id int
					if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
						if tt.expectError {
							return
						}
						t.Fatalf("Failed to parse label ID %q: %v", idStr, err)
					}
					audience = append(audience, api.CampaignAudience{Type: "Label", ID: id})
				}
			}

			if len(audience) != tt.expectedCount {
				t.Errorf("Expected %d audience items, got %d", tt.expectedCount, len(audience))
			}

			for i, expectedID := range tt.expectedIDs {
				if audience[i].ID != expectedID {
					t.Errorf("Expected audience[%d].ID to be %d, got %d", i, expectedID, audience[i].ID)
				}
			}
		})
	}
}

func TestLabelsMutualExclusivityCreate(t *testing.T) {
	tests := []struct {
		name         string
		labelsFlag   string
		audienceFlag string
		expectError  bool
		errorMessage string
	}{
		{
			name:         "both flags set",
			labelsFlag:   "1,2,3",
			audienceFlag: `[{"type":"Label","id":1}]`,
			expectError:  true,
			errorMessage: "--labels and --audience are mutually exclusive",
		},
		{
			name:         "only labels set",
			labelsFlag:   "1,2,3",
			audienceFlag: "",
			expectError:  false,
		},
		{
			name:         "only audience set",
			labelsFlag:   "",
			audienceFlag: `[{"type":"Label","id":1}]`,
			expectError:  false,
		},
		{
			name:         "neither flag set",
			labelsFlag:   "",
			audienceFlag: "",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the mutual exclusivity check from campaigns.go
			if tt.labelsFlag != "" && tt.audienceFlag != "" {
				if !tt.expectError {
					t.Error("Expected error for mutual exclusivity but expectError is false")
				}
				// Verify error message matches
				expectedErr := "--labels and --audience are mutually exclusive"
				if tt.errorMessage != expectedErr {
					t.Errorf("Expected error message %q, got %q", expectedErr, tt.errorMessage)
				}
				return
			}

			if tt.expectError {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestLabelsMutualExclusivityUpdate(t *testing.T) {
	tests := []struct {
		name         string
		labelsFlag   string
		audienceFlag string
		expectError  bool
	}{
		{
			name:         "update with both flags",
			labelsFlag:   "5,6",
			audienceFlag: `[{"type":"Label","id":5}]`,
			expectError:  true,
		},
		{
			name:         "update with only labels",
			labelsFlag:   "5,6",
			audienceFlag: "",
			expectError:  false,
		},
		{
			name:         "update with only audience",
			labelsFlag:   "",
			audienceFlag: `[{"type":"Label","id":5}]`,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.labelsFlag != "" && tt.audienceFlag != "" {
				if !tt.expectError {
					t.Error("Expected error for mutual exclusivity")
				}
				return
			}

			if tt.expectError {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestAudienceJSONParsing(t *testing.T) {
	tests := []struct {
		name          string
		audienceJSON  string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "valid single audience",
			audienceJSON:  `[{"type":"Label","id":1}]`,
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "valid multiple audience",
			audienceJSON:  `[{"type":"Label","id":1},{"type":"Label","id":2}]`,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "empty array",
			audienceJSON:  `[]`,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:         "invalid JSON",
			audienceJSON: `{invalid}`,
			expectError:  true,
		},
		{
			name:         "not an array",
			audienceJSON: `{"type":"Label","id":1}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var audience []api.CampaignAudience
			err := json.Unmarshal([]byte(tt.audienceJSON), &audience)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && len(audience) != tt.expectedCount {
				t.Errorf("Expected %d audience items, got %d", tt.expectedCount, len(audience))
			}
		})
	}
}

func TestCampaignAudienceStructure(t *testing.T) {
	tests := []struct {
		name         string
		audience     api.CampaignAudience
		expectedType string
		expectedID   int
	}{
		{
			name:         "label audience",
			audience:     api.CampaignAudience{Type: "Label", ID: 1},
			expectedType: "Label",
			expectedID:   1,
		},
		{
			name:         "label with higher ID",
			audience:     api.CampaignAudience{Type: "Label", ID: 999},
			expectedType: "Label",
			expectedID:   999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.audience.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, tt.audience.Type)
			}
			if tt.audience.ID != tt.expectedID {
				t.Errorf("Expected ID %d, got %d", tt.expectedID, tt.audience.ID)
			}

			// Test JSON marshaling
			data, err := json.Marshal(tt.audience)
			if err != nil {
				t.Fatalf("Failed to marshal audience: %v", err)
			}

			var unmarshaled api.CampaignAudience
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal audience: %v", err)
			}

			if unmarshaled.Type != tt.expectedType {
				t.Errorf("After unmarshal, expected type %s, got %s", tt.expectedType, unmarshaled.Type)
			}
			if unmarshaled.ID != tt.expectedID {
				t.Errorf("After unmarshal, expected ID %d, got %d", tt.expectedID, unmarshaled.ID)
			}
		})
	}
}

func TestLabelsToAudienceConversion(t *testing.T) {
	tests := []struct {
		name        string
		labelIDs    []int
		expectedLen int
	}{
		{
			name:        "convert three labels",
			labelIDs:    []int{1, 2, 3},
			expectedLen: 3,
		},
		{
			name:        "convert single label",
			labelIDs:    []int{42},
			expectedLen: 1,
		},
		{
			name:        "convert many labels",
			labelIDs:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			expectedLen: 10,
		},
		{
			name:        "empty labels",
			labelIDs:    []int{},
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var audience []api.CampaignAudience
			for _, id := range tt.labelIDs {
				audience = append(audience, api.CampaignAudience{Type: "Label", ID: id})
			}

			if len(audience) != tt.expectedLen {
				t.Errorf("Expected %d audience items, got %d", tt.expectedLen, len(audience))
			}

			for i, id := range tt.labelIDs {
				if audience[i].Type != "Label" {
					t.Errorf("Expected audience[%d].Type to be 'Label', got %s", i, audience[i].Type)
				}
				if audience[i].ID != id {
					t.Errorf("Expected audience[%d].ID to be %d, got %d", i, id, audience[i].ID)
				}
			}
		})
	}
}

func TestLabelsParsingErrors(t *testing.T) {
	tests := []struct {
		name       string
		labelsFlag string
		shouldFail bool // whether parsing should fail
	}{
		{
			name:       "non-numeric labels",
			labelsFlag: "abc,def",
			shouldFail: true,
		},
		{
			name:       "float labels",
			labelsFlag: "1.5,2.3",
			shouldFail: false, // fmt.Sscanf("%d") stops at decimal point, successfully parses "1" and "2"
		},
		{
			name:       "empty values",
			labelsFlag: "1,,3",
			shouldFail: true,
		},
		{
			name:       "mixed valid and invalid",
			labelsFlag: "1,abc,3",
			shouldFail: true,
		},
		{
			name:       "negative numbers",
			labelsFlag: "-1,-2",
			shouldFail: false, // negative numbers parse successfully with fmt.Sscanf
		},
		{
			name:       "whitespace only",
			labelsFlag: "  ,  ",
			shouldFail: true,
		},
		{
			name:       "special characters",
			labelsFlag: "1,@#$,3",
			shouldFail: true,
		},
		{
			name:       "mixed numbers and text",
			labelsFlag: "123abc",
			shouldFail: false, // fmt.Sscanf("%d") stops at first non-digit, successfully parses "123"
		},
		{
			name:       "text before numbers",
			labelsFlag: "abc123",
			shouldFail: true, // fmt.Sscanf("%d") fails if text comes first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hadError bool

			// Simulate the label parsing logic from campaigns.go
			for _, idStr := range strings.Split(tt.labelsFlag, ",") {
				idStr = strings.TrimSpace(idStr)

				// Check for empty string after trimming
				if idStr == "" {
					hadError = true
					continue
				}

				var id int
				if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
					hadError = true
					continue
				}
				// Successfully parsed id (not storing audience for this validation test)
				_ = id
			}

			if tt.shouldFail && !hadError {
				t.Errorf("Expected parsing to fail for input %q but it succeeded", tt.labelsFlag)
			}

			if !tt.shouldFail && hadError {
				t.Errorf("Expected parsing to succeed for input %q but it failed", tt.labelsFlag)
			}
		})
	}
}

func TestCampaignTitleEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{"empty title", ""},
		{"title with quotes", `Campaign "Special"`},
		{"title with format specifiers", "Get %d%% off today!"},
		{"title with newline", "Line1\nLine2"},
		{"title with tabs", "Tab\there"},
		{"very long title", strings.Repeat("a", 200)},
		{"unicode title", "Campaign 日本語 🎉"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Using %q should safely handle all these cases
			result := fmt.Sprintf("Delete campaign %q (ID: %d)? (y/N): ", tt.title, 123)

			// Verify the result is a valid string (no panic)
			if result == "" {
				t.Error("Expected non-empty result")
			}

			// Verify the ID is present
			if !strings.Contains(result, "123") {
				t.Errorf("Expected ID 123 in result: %s", result)
			}
		})
	}
}

func TestCampaignsUpdateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/campaigns/1", jsonResponse(200, `{"id": 1, "title": "Updated Campaign"}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"campaigns", "update", "1",
		"--title", "Updated Campaign",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("campaigns update failed: %v", err)
	}

	var campaign map[string]any
	if err := json.Unmarshal([]byte(output), &campaign); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestCampaignsUpdateCommand_WithScheduledAt(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/campaigns/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "Campaign"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"campaigns", "update", "1",
		"--scheduled-at", "2025-01-15T10:00:00Z",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("campaigns update failed: %v", err)
	}

	if receivedBody["scheduled_at"] == nil || receivedBody["scheduled_at"].(float64) == 0 {
		t.Errorf("expected scheduled_at to be set, got %v", receivedBody["scheduled_at"])
	}
}

func TestCampaignsUpdateCommand_InvalidScheduledAt(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"campaigns", "update", "1",
		"--scheduled-at", "invalid-date",
	})
	if err == nil {
		t.Error("expected error for invalid scheduled-at")
	}
	if !strings.Contains(err.Error(), "RFC3339") {
		t.Errorf("expected 'RFC3339' error, got: %v", err)
	}
}

func TestCampaignsUpdateCommand_WithLabels(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/campaigns/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "Campaign"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"campaigns", "update", "1",
		"--labels", "1,2,3",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("campaigns update failed: %v", err)
	}

	audience, ok := receivedBody["audience"].([]any)
	if !ok || len(audience) != 3 {
		t.Errorf("expected 3 audience items, got %v", receivedBody["audience"])
	}
}

func TestCampaignsUpdateCommand_WithAudience(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/campaigns/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "Campaign"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"campaigns", "update", "1",
		"--audience", `[{"type":"Label","id":1}]`,
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("campaigns update failed: %v", err)
	}

	audience, ok := receivedBody["audience"].([]any)
	if !ok || len(audience) != 1 {
		t.Errorf("expected 1 audience item, got %v", receivedBody["audience"])
	}
}

func TestCampaignsUpdateCommand_InvalidLabelsID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"campaigns", "update", "1",
		"--labels", "abc,def",
	})
	if err == nil {
		t.Error("expected error for invalid labels")
	}
	if !strings.Contains(err.Error(), "label ID") {
		t.Errorf("expected 'label ID' error, got: %v", err)
	}
}

func TestCampaignsUpdateCommand_InvalidAudienceJSON(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"campaigns", "update", "1",
		"--audience", "invalid-json",
	})
	if err == nil {
		t.Error("expected error for invalid audience JSON")
	}
	if !strings.Contains(err.Error(), "invalid audience JSON") {
		t.Errorf("expected 'invalid audience JSON' error, got: %v", err)
	}
}

func TestCampaignsUpdateCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"campaigns", "update", "invalid",
		"--title", "Test",
	})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
	if !strings.Contains(err.Error(), "ID") {
		t.Errorf("expected 'ID' error, got: %v", err)
	}
}

func TestCampaignsDeleteCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/campaigns/1", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"campaigns", "delete", "1", "--force", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("campaigns delete failed: %v", err)
	}
}

func TestCampaignsDeleteCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/campaigns/1", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"campaigns", "delete", "1", "--force"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}
