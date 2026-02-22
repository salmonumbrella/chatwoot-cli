package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestReportsSummaryCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/summary", jsonResponse(200, `{
			"conversations_count": 100,
			"resolutions_count": 80,
			"incoming_messages_count": 500,
			"outgoing_messages_count": 450,
			"avg_first_response_time": "2h 30m",
			"avg_resolution_time": "5h 15m"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"reports", "summary",
		"--type", "account",
		"--from", "2024-01-01",
		"--to", "2024-01-31",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports summary failed: %v", err)
	}

	if !strings.Contains(output, "Conversations: 100") {
		t.Errorf("output missing conversations count: %s", output)
	}
	if !strings.Contains(output, "Resolutions: 80") {
		t.Errorf("output missing resolutions count: %s", output)
	}
}

func TestReportsSummaryCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/summary", jsonResponse(200, `{
			"conversations_count": 100,
			"resolutions_count": 80,
			"incoming_messages_count": 500,
			"outgoing_messages_count": 450
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"reports", "summary",
		"--type", "account",
		"--from", "2024-01-01",
		"--to", "2024-01-31",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports summary failed: %v", err)
	}

	var report map[string]any
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if report["conversations_count"] != float64(100) {
		t.Errorf("expected conversations_count 100, got %v", report["conversations_count"])
	}
}

func TestReportsSummaryCommand_MissingType(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"reports", "summary",
		"--from", "2024-01-01",
		"--to", "2024-01-31",
	})
	if err == nil {
		t.Error("expected error when --type is missing")
	}
	if !strings.Contains(err.Error(), "--type is required") {
		t.Errorf("expected '--type is required' error, got: %v", err)
	}
}

func TestReportsSummaryCommand_MissingFrom(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"reports", "summary",
		"--type", "account",
		"--to", "2024-01-31",
	})
	if err == nil {
		t.Error("expected error when --from is missing")
	}
	if !strings.Contains(err.Error(), "--from is required") {
		t.Errorf("expected '--from is required' error, got: %v", err)
	}
}

func TestReportsSummaryCommand_MissingTo(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"reports", "summary",
		"--type", "account",
		"--from", "2024-01-01",
	})
	if err == nil {
		t.Error("expected error when --to is missing")
	}
	if !strings.Contains(err.Error(), "--to is required") {
		t.Errorf("expected '--to is required' error, got: %v", err)
	}
}

func TestReportsSummaryCommand_InvalidDateFormat(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"reports", "summary",
		"--type", "account",
		"--from", "01-01-2024",
		"--to", "2024-01-31",
	})
	if err == nil {
		t.Error("expected error for invalid date format")
	}
	if !strings.Contains(err.Error(), "invalid date format") {
		t.Errorf("expected 'invalid date format' error, got: %v", err)
	}
}

func TestReportsDataCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports", jsonResponse(200, `[
			{"timestamp": 1704067200, "value": "10"},
			{"timestamp": 1704153600, "value": "15"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"reports", "data",
		"--metric", "conversations_count",
		"--type", "account",
		"--from", "2024-01-01",
		"--to", "2024-01-31",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports data failed: %v", err)
	}

	if !strings.Contains(output, "TIMESTAMP") || !strings.Contains(output, "VALUE") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "10") || !strings.Contains(output, "15") {
		t.Errorf("output missing expected values: %s", output)
	}
}

func TestReportsDataCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports", jsonResponse(200, `[
			{"timestamp": 1704067200, "value": "10"},
			{"timestamp": 1704153600, "value": "15"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"reports", "data",
		"--metric", "conversations_count",
		"--type", "account",
		"--from", "2024-01-01",
		"--to", "2024-01-31",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports data failed: %v", err)
	}

	report := decodeItems(t, output)
	if len(report) != 2 {
		t.Errorf("expected 2 data points, got %d", len(report))
	}
}

func TestReportsDataCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"reports", "data",
		"--metric", "conversations_count",
		"--type", "account",
		"--from", "2024-01-01",
		"--to", "2024-01-31",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports data failed: %v", err)
	}

	if !strings.Contains(output, "No data points found") {
		t.Errorf("expected 'No data points found' message, got: %s", output)
	}
}

func TestReportsDataCommand_MissingMetric(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"reports", "data",
		"--type", "account",
		"--from", "2024-01-01",
		"--to", "2024-01-31",
	})
	if err == nil {
		t.Error("expected error when --metric is missing")
	}
	if !strings.Contains(err.Error(), "--metric is required") {
		t.Errorf("expected '--metric is required' error, got: %v", err)
	}
}

func TestReportsLiveCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/conversations", jsonResponse(200, `{
			"open": 10,
			"unattended": 5,
			"unassigned": 3
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "live"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports live failed: %v", err)
	}

	if !strings.Contains(output, "Live Conversation Metrics") {
		t.Errorf("output missing header: %s", output)
	}
	if !strings.Contains(output, "Open:") {
		t.Errorf("output missing 'Open:': %s", output)
	}
}

func TestReportsLiveCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/conversations", jsonResponse(200, `{
			"open": 10,
			"unattended": 5,
			"unassigned": 3
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "live", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports live failed: %v", err)
	}

	var metrics map[string]any
	if err := json.Unmarshal([]byte(output), &metrics); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestReportsAgentsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/conversations", jsonResponse(200, `[
			{"id": 1, "name": "Agent One", "email": "agent1@test.com", "availability": "online", "metric": {"open": 5, "unattended": 2}},
			{"id": 2, "name": "Agent Two", "email": "agent2@test.com", "availability": "offline", "metric": {"open": 3, "unattended": 1}}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "agents"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports agents failed: %v", err)
	}

	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "EMAIL") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "Agent One") {
		t.Errorf("output missing agent name: %s", output)
	}
}

func TestReportsAgentsCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/conversations", jsonResponse(200, `[
			{"id": 1, "name": "Agent One", "email": "agent1@test.com", "availability": "online", "metric": {"open": 5, "unattended": 2}}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "agents", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports agents failed: %v", err)
	}

	_ = decodeItems(t, output)
}

func TestReportsAgentsCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/conversations", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "agents"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports agents failed: %v", err)
	}

	if !strings.Contains(output, "No agent data found") {
		t.Errorf("expected 'No agent data found' message, got: %s", output)
	}
}

func TestReportsAgentSummaryCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/summary_reports/agent", jsonResponse(200, `[
			{"id": 1, "conversations_count": 12, "resolved_conversations_count": 10, "avg_resolution_time": 3600, "avg_first_response_time": 120, "avg_reply_time": 240}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "agent-summary"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports agent-summary failed: %v", err)
	}

	if !strings.Contains(output, "AVG_FIRST_RESPONSE") || !strings.Contains(output, "AVG_REPLY") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "3600") {
		t.Errorf("output missing expected data row: %s", output)
	}
}

func TestReportsAgentSummaryCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/summary_reports/agent", jsonResponse(200, `[
			{"id": 1, "conversations_count": 12, "resolved_conversations_count": 10, "avg_resolution_time": 3600, "avg_first_response_time": 120, "avg_reply_time": 240}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "agent-summary", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports agent-summary failed: %v", err)
	}

	entries := decodeItems(t, output)
	if len(entries) == 0 {
		t.Errorf("expected summary entries, got: %v", entries)
	}
}

func TestReportsAgentSummaryCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/summary_reports/agent", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "agent-summary"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports agent-summary failed: %v", err)
	}

	if !strings.Contains(output, "No agent summary data found") {
		t.Errorf("expected 'No agent summary data found' message, got: %s", output)
	}
}

func TestReportsInboxesCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/summary_reports/inbox", jsonResponse(200, `[
			{"id": 2, "conversations_count": 8, "resolved_conversations_count": 6, "avg_resolution_time": 1800, "avg_first_response_time": 90, "avg_reply_time": 180}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "inboxes"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports inboxes failed: %v", err)
	}

	if !strings.Contains(output, "AVG_FIRST_RESPONSE") || !strings.Contains(output, "AVG_REPLY") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "1800") {
		t.Errorf("output missing expected data row: %s", output)
	}
}

func TestReportsInboxesCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/summary_reports/inbox", jsonResponse(200, `[
			{"id": 2, "conversations_count": 8, "resolved_conversations_count": 6, "avg_resolution_time": 1800, "avg_first_response_time": 90, "avg_reply_time": 180}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "inboxes", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports inboxes failed: %v", err)
	}

	entries := decodeItems(t, output)
	if len(entries) == 0 {
		t.Errorf("expected summary entries, got: %v", entries)
	}
}

func TestReportsInboxesCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/summary_reports/inbox", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "inboxes"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports inboxes failed: %v", err)
	}

	if !strings.Contains(output, "No inbox summary data found") {
		t.Errorf("expected 'No inbox summary data found' message, got: %s", output)
	}
}

func TestReportsTeamsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/summary_reports/team", jsonResponse(200, `[
			{"id": 3, "conversations_count": 20, "resolved_conversations_count": 15, "avg_resolution_time": 2400, "avg_first_response_time": 150, "avg_reply_time": 300}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "teams"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports teams failed: %v", err)
	}

	if !strings.Contains(output, "AVG_FIRST_RESPONSE") || !strings.Contains(output, "AVG_REPLY") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "2400") {
		t.Errorf("output missing expected data row: %s", output)
	}
}

func TestReportsTeamsCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/summary_reports/team", jsonResponse(200, `[
			{"id": 3, "conversations_count": 20, "resolved_conversations_count": 15, "avg_resolution_time": 2400, "avg_first_response_time": 150, "avg_reply_time": 300}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "teams", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports teams failed: %v", err)
	}

	entries := decodeItems(t, output)
	if len(entries) == 0 {
		t.Errorf("expected summary entries, got: %v", entries)
	}
}

func TestReportsTeamsCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/summary_reports/team", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "teams"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports teams failed: %v", err)
	}

	if !strings.Contains(output, "No team summary data found") {
		t.Errorf("expected 'No team summary data found' message, got: %s", output)
	}
}

func TestReportsChannelsCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/summary_reports/channel", jsonResponse(200, `{
			"Channel::WebWidget": {"open": 5, "resolved": 10, "pending": 2, "snoozed": 1, "total": 18},
			"Channel::Email": {"open": 3, "resolved": 4, "pending": 1, "snoozed": 0, "total": 8}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "channels"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports channels failed: %v", err)
	}

	if !strings.Contains(output, "CHANNEL") || !strings.Contains(output, "TOTAL") {
		t.Errorf("output missing expected headers: %s", output)
	}
	if !strings.Contains(output, "Channel::WebWidget") || !strings.Contains(output, "Channel::Email") {
		t.Errorf("output missing channel names: %s", output)
	}
}

func TestReportsChannelsCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/summary_reports/channel", jsonResponse(200, `{
			"Channel::WebWidget": {"open": 5, "resolved": 10, "pending": 2, "snoozed": 1, "total": 18}
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "channels", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports channels failed: %v", err)
	}

	var summary map[string]map[string]any
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
	if _, ok := summary["Channel::WebWidget"]; !ok {
		t.Errorf("expected Channel::WebWidget key, got: %v", summary)
	}
}

func TestReportsChannelsCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/summary_reports/channel", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "channels"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports channels failed: %v", err)
	}

	if !strings.Contains(output, "No channel summary data found") {
		t.Errorf("expected 'No channel summary data found' message, got: %s", output)
	}
}

func TestReportsChannelsCommand_WithFilters(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/accounts/1/summary_reports/channel" {
			query := r.URL.Query()
			if query.Get("since") == "" {
				t.Error("expected since query param to be set")
			}
			if query.Get("until") == "" {
				t.Error("expected until query param to be set")
			}
			if query.Get("business_hours") != "true" {
				t.Errorf("expected business_hours=true, got %s", query.Get("business_hours"))
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{
				"Channel::WebWidget": {"open": 5, "resolved": 10, "pending": 2, "snoozed": 1, "total": 18}
			}`))
			return
		}
		http.NotFound(w, r)
	})

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{
		"reports", "channels",
		"--from", "2024-01-01",
		"--to", "2024-01-31",
		"--business-hours",
	})
	if err != nil {
		t.Errorf("reports channels with filters failed: %v", err)
	}
}

func TestReportsChannelsCommand_InvalidDateFormat(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"reports", "channels",
		"--from", "01-01-2024",
	})
	if err == nil {
		t.Error("expected error for invalid date format")
	}
	if !strings.Contains(err.Error(), "invalid date format") {
		t.Errorf("expected 'invalid date format' error, got: %v", err)
	}
}

func TestReportsEventsListCommand(t *testing.T) {
	// Use a handler that matches path containing /reporting_events
	// because the API has a known path duplication issue
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/reporting_events") && !strings.Contains(r.URL.Path, "/conversations/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`[
				{"id": 1, "name": "conversation_opened", "value": 10, "created_at": "2024-01-15T10:00:00Z"},
				{"id": 2, "name": "conversation_closed", "value": 5, "created_at": "2024-01-15T11:00:00Z"}
			]`))
			return
		}
		http.NotFound(w, r)
	})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "events", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports events list failed: %v", err)
	}

	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "VALUE") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestReportsEventsListCommand_WithTimeFilters(t *testing.T) {
	expectedSince := time.Date(2026, 1, 27, 0, 0, 0, 0, time.UTC).Unix()
	expectedUntil := time.Date(2026, 1, 28, 1, 0, 0, 0, time.UTC).Unix()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/reporting_events") && !strings.Contains(r.URL.Path, "/conversations/") {
			query := r.URL.Query()
			if query.Get("since") != strconv.FormatInt(expectedSince, 10) {
				t.Errorf("expected since=%d, got %s", expectedSince, query.Get("since"))
			}
			if query.Get("until") != strconv.FormatInt(expectedUntil, 10) {
				t.Errorf("expected until=%d, got %s", expectedUntil, query.Get("until"))
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`[]`))
			return
		}
		http.NotFound(w, r)
	})

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{
		"reports", "events", "list",
		"--since", "2026-01-27",
		"--until", "2026-01-28T01:00:00Z",
	})
	if err != nil {
		t.Errorf("reports events list with time filters failed: %v", err)
	}
}

func TestReportsEventsConversationCommand(t *testing.T) {
	// Use a handler that matches path containing /conversations/123/reporting_events
	// because the API has a known path duplication issue
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/conversations/123/reporting_events") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`[
				{"id": 1, "name": "message_sent", "value": 1, "created_at": "2024-01-15T10:00:00Z"}
			]`))
			return
		}
		http.NotFound(w, r)
	})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"reports", "events", "conversation", "123"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("reports events conversation failed: %v", err)
	}

	if !strings.Contains(output, "ID") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestReportsEventsConversationCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"reports", "events", "conversation", "abc"})
	if err == nil {
		t.Error("expected error for invalid conversation ID")
	}
}

func TestReportsCommand_Alias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/conversations", jsonResponse(200, `{
			"open": 10,
			"unattended": 5,
			"unassigned": 3
		}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"report", "live"})
	if err != nil {
		t.Errorf("report alias should work: %v", err)
	}
}

func TestReportsCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/conversations", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"reports", "live"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestReportsInboxLabelMatrixCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/inbox_label_matrix", jsonResponse(200, `[
			{"inbox_id": 1, "label_id": 2, "count": 15},
			{"inbox_id": 1, "label_id": 3, "count": 8},
			{"inbox_id": 2, "label_id": 2, "count": 22}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"reports", "inbox-label-matrix",
			"--from", "2024-01-01",
			"--to", "2024-01-31",
		})
		if err != nil {
			t.Errorf("command failed: %v", err)
		}
	})

	if !strings.Contains(output, "15") {
		t.Errorf("output missing count 15: %s", output)
	}
	if !strings.Contains(output, "22") {
		t.Errorf("output missing count 22: %s", output)
	}
}

func TestReportsInboxLabelMatrixCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/inbox_label_matrix", jsonResponse(200, `[
			{"inbox_id": 1, "label_id": 2, "count": 15}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"reports", "inbox-label-matrix",
			"--from", "2024-01-01",
			"--to", "2024-01-31",
			"-o", "json",
		})
		if err != nil {
			t.Errorf("command failed: %v", err)
		}
	})

	entries := decodeItems(t, output)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestReportsInboxLabelMatrixCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/inbox_label_matrix", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"reports", "inbox-label-matrix",
			"--from", "2024-01-01",
			"--to", "2024-01-31",
		})
		if err != nil {
			t.Errorf("command failed: %v", err)
		}
	})

	if !strings.Contains(output, "No inbox-label matrix data found") {
		t.Errorf("expected empty message, got: %s", output)
	}
}

func TestReportsResponseTimeDistributionCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/first_response_time_distribution", jsonResponse(200, `{
			"Channel::WebWidget": {"0-1h": 10, "1-4h": 5, "4-8h": 2, "8-24h": 1, "24h+": 0},
			"Channel::Email": {"0-1h": 3, "1-4h": 8, "4-8h": 4, "8-24h": 6, "24h+": 2}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"reports", "response-time",
			"--from", "2024-01-01",
			"--to", "2024-01-31",
		})
		if err != nil {
			t.Errorf("command failed: %v", err)
		}
	})

	if !strings.Contains(output, "Channel::WebWidget") {
		t.Errorf("output missing channel type: %s", output)
	}
	if !strings.Contains(output, "0-1h") {
		t.Errorf("output missing time bucket: %s", output)
	}
}

func TestReportsResponseTimeDistributionCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/first_response_time_distribution", jsonResponse(200, `{
			"Channel::WebWidget": {"0-1h": 10}
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"reports", "response-time",
			"--from", "2024-01-01",
			"--to", "2024-01-31",
			"-o", "json",
		})
		if err != nil {
			t.Errorf("command failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestReportsResponseTimeDistributionCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/first_response_time_distribution", jsonResponse(200, `{}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"reports", "response-time",
			"--from", "2024-01-01",
			"--to", "2024-01-31",
		})
		if err != nil {
			t.Errorf("command failed: %v", err)
		}
	})

	if !strings.Contains(output, "No response time distribution data found") {
		t.Errorf("expected empty message, got: %s", output)
	}
}

func TestReportsOutgoingMessagesCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/outgoing_messages_count", jsonResponse(200, `[
			{"id": 1, "count": 42},
			{"id": 2, "count": 35}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"reports", "outgoing-messages",
			"--from", "2024-01-01",
			"--to", "2024-01-31",
			"--group-by", "agent",
		})
		if err != nil {
			t.Errorf("command failed: %v", err)
		}
	})

	if !strings.Contains(output, "42") {
		t.Errorf("output missing count 42: %s", output)
	}
}

func TestReportsOutgoingMessagesCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/outgoing_messages_count", jsonResponse(200, `[
			{"id": 1, "count": 42}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"reports", "outgoing-messages",
			"--from", "2024-01-01",
			"--to", "2024-01-31",
			"-o", "json",
		})
		if err != nil {
			t.Errorf("command failed: %v", err)
		}
	})

	items := decodeItems(t, output)
	if len(items) != 1 {
		t.Errorf("expected 1 entry, got %d", len(items))
	}
}

func TestReportsOutgoingMessagesCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v2/accounts/1/reports/outgoing_messages_count", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"reports", "outgoing-messages",
			"--from", "2024-01-01",
			"--to", "2024-01-31",
		})
		if err != nil {
			t.Errorf("command failed: %v", err)
		}
	})

	if !strings.Contains(output, "No outgoing messages data found") {
		t.Errorf("expected empty message, got: %s", output)
	}
}
