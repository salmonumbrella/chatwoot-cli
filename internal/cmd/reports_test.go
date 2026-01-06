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
