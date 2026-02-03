package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/99designs/keyring"
	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/config"
)

func TestNewDashboardCmd(t *testing.T) {
	cmd := newDashboardCmd()

	if !strings.HasPrefix(cmd.Use, "dashboard ") {
		t.Errorf("Use = %q, want prefix 'dashboard '", cmd.Use)
	}

	contactFlag := cmd.Flag("contact")
	if contactFlag == nil {
		t.Error("Expected --contact flag")
	}

	conversationFlag := cmd.Flag("conversation")
	if conversationFlag == nil {
		t.Error("Expected --conversation flag")
	}

	noResolveFlag := cmd.Flag("no-resolve")
	if noResolveFlag == nil {
		t.Error("Expected --no-resolve flag")
	}

	noResolveWarningFlag := cmd.Flag("no-resolve-warning")
	if noResolveWarningFlag == nil {
		t.Error("Expected --no-resolve-warning flag")
	}

	pageFlag := cmd.Flag("page")
	if pageFlag == nil {
		t.Error("Expected --page flag")
	}

	perPageFlag := cmd.Flag("per-page")
	if perPageFlag == nil {
		t.Error("Expected --per-page flag")
	}

	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("Expected error for no args")
	}
	if err := cmd.Args(cmd, []string{"orders"}); err != nil {
		t.Errorf("Expected no error for single arg: %v", err)
	}
}

// setupDashboardTestEnv sets up a test environment with a mock keyring containing
// dashboard config and a mock HTTP server. Returns cleanup function.
func setupDashboardTestEnv(t *testing.T, chatwootHandler, dashboardHandler http.Handler, dashboardName string) (chatwootURL, dashboardURL string) {
	t.Helper()

	// Create mock servers
	chatwootServer := httptest.NewServer(chatwootHandler)
	t.Cleanup(chatwootServer.Close)

	dashboardServer := httptest.NewServer(dashboardHandler)
	t.Cleanup(dashboardServer.Close)

	// Create mock keyring with account containing dashboard config
	ring := keyring.NewArrayKeyring(nil)

	account := config.Account{
		BaseURL:   chatwootServer.URL,
		APIToken:  "test-token",
		AccountID: 1,
		Extensions: &config.Extensions{
			Dashboards: map[string]*config.DashboardConfig{
				dashboardName: {
					Name:      "Test Dashboard",
					Endpoint:  dashboardServer.URL,
					AuthToken: "test@example.com",
				},
			},
		},
	}

	data, _ := json.Marshal(account)
	_ = ring.Set(keyring.Item{Key: "default", Data: data})
	_ = ring.Set(keyring.Item{Key: "current_profile", Data: []byte("default")})

	// Install mock keyring
	cleanup := config.SetOpenKeyring(func(cfg keyring.Config) (keyring.Keyring, error) {
		return ring, nil
	})
	t.Cleanup(cleanup)

	// Set CHATWOOT_TESTING to skip URL validation
	t.Setenv("CHATWOOT_TESTING", "1")
	t.Setenv("CHATWOOT_OUTPUT", "text") // Ensure tests use text output by default

	return chatwootServer.URL, dashboardServer.URL
}

func TestDashboardCommand_AutoResolveHeuristic(t *testing.T) {
	// When --contact is provided and it matches a conversation ID,
	// the command should resolve it to the contact ID from that conversation

	chatwootHandler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/12345", jsonResponse(200, `{
			"id": 12345,
			"contact_id": 99999,
			"status": "open"
		}`))

	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request contains the resolved contact ID
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)

		contactID := int(req["contact_id"].(float64))
		if contactID != 99999 {
			t.Errorf("Expected contact_id 99999, got %d", contactID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items": [], "pagination": {"page": 1, "total_pages": 1}}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	stderr := captureStderr(t, func() {
		err := Execute(context.Background(), []string{"dashboard", "orders", "--contact", "12345"})
		if err != nil {
			t.Errorf("dashboard command failed: %v", err)
		}
	})

	// Should show resolve warning
	if !strings.Contains(stderr, "Note: --contact 12345 matched a conversation") {
		t.Errorf("Expected auto-resolve warning, got stderr: %s", stderr)
	}
}

func TestDashboardCommand_ContactWithNoResolve(t *testing.T) {
	// When --contact is provided with --no-resolve, it should NOT attempt
	// to resolve the contact as a conversation ID

	chatwootHandler := newRouteHandler()
	// No conversation endpoint needed - should not be called

	var receivedContactID int
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		receivedContactID = int(req["contact_id"].(float64))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items": [], "pagination": {"page": 1, "total_pages": 1}}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	err := Execute(context.Background(), []string{"dashboard", "orders", "--contact", "12345", "--no-resolve"})
	if err != nil {
		t.Errorf("dashboard command failed: %v", err)
	}

	// Contact ID should be passed directly without resolution
	if receivedContactID != 12345 {
		t.Errorf("Expected contact_id 12345 (no resolution), got %d", receivedContactID)
	}
}

func TestDashboardCommand_ConversationResolvesToContactID(t *testing.T) {
	// When --conversation is provided, it should always resolve to contact ID

	chatwootHandler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/24445", jsonResponse(200, `{
			"id": 24445,
			"contact_id": 180712,
			"status": "open"
		}`))

	var receivedContactID int
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		receivedContactID = int(req["contact_id"].(float64))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items": [], "pagination": {"page": 1, "total_pages": 1}}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	err := Execute(context.Background(), []string{"dashboard", "orders", "--conversation", "24445"})
	if err != nil {
		t.Errorf("dashboard command failed: %v", err)
	}

	if receivedContactID != 180712 {
		t.Errorf("Expected contact_id 180712 from conversation, got %d", receivedContactID)
	}
}

func TestDashboardCommand_ConversationResolvesFromMeta(t *testing.T) {
	// Test that contact ID can be resolved from meta.sender.id

	chatwootHandler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/24445", jsonResponse(200, `{
			"id": 24445,
			"contact_id": 0,
			"meta": {
				"sender": {
					"id": 180712,
					"name": "John Doe"
				}
			},
			"status": "open"
		}`))

	var receivedContactID int
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		receivedContactID = int(req["contact_id"].(float64))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items": [], "pagination": {"page": 1, "total_pages": 1}}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	err := Execute(context.Background(), []string{"dashboard", "orders", "--conversation", "24445"})
	if err != nil {
		t.Errorf("dashboard command failed: %v", err)
	}

	if receivedContactID != 180712 {
		t.Errorf("Expected contact_id 180712 from meta.sender, got %d", receivedContactID)
	}
}

func TestDashboardCommand_ErrorBothContactAndConversation(t *testing.T) {
	// When both --contact and --conversation are provided, it should error

	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("Dashboard API should not be called")
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	err := Execute(context.Background(), []string{"dashboard", "orders", "--contact", "123", "--conversation", "456"})
	if err == nil {
		t.Error("Expected error when both --contact and --conversation provided")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Errorf("Expected 'cannot be used together' error, got: %v", err)
	}
}

func TestDashboardCommand_ErrorNeitherContactNorConversation(t *testing.T) {
	// When neither --contact nor --conversation is provided, it should error

	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("Dashboard API should not be called")
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	err := Execute(context.Background(), []string{"dashboard", "orders"})
	if err == nil {
		t.Error("Expected error when neither --contact nor --conversation provided")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("Expected 'required' error, got: %v", err)
	}
}

func TestDashboardCommand_WarningsInJSONOutput(t *testing.T) {
	// When auto-resolve happens and JSON output is used, _warnings should appear

	chatwootHandler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/12345", jsonResponse(200, `{
			"id": 12345,
			"contact_id": 99999,
			"status": "open"
		}`))

	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items": [{"id": 1}], "pagination": {"page": 1, "total_pages": 1}}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"dashboard", "orders", "--contact", "12345", "-o", "json"})
		if err != nil {
			t.Errorf("dashboard command failed: %v", err)
		}
	})

	if !strings.Contains(output, `"_warnings"`) {
		t.Errorf("Expected _warnings in JSON output, got: %s", output)
	}
	if !strings.Contains(output, "matched a conversation") {
		t.Errorf("Expected warning message in _warnings, got: %s", output)
	}
}

func TestDashboardCommand_NoResolveWarningFlag(t *testing.T) {
	// When --no-resolve-warning is used, the warning should be suppressed in text output
	// but _warnings should NOT appear in JSON output

	chatwootHandler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/12345", jsonResponse(200, `{
			"id": 12345,
			"contact_id": 99999,
			"status": "open"
		}`))

	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items": [{"id": 1}], "pagination": {"page": 1, "total_pages": 1}}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	// Test text output - warning should be suppressed
	stderr := captureStderr(t, func() {
		err := Execute(context.Background(), []string{"dashboard", "orders", "--contact", "12345", "--no-resolve-warning"})
		if err != nil {
			t.Errorf("dashboard command failed: %v", err)
		}
	})

	if strings.Contains(stderr, "Note:") {
		t.Errorf("Warning should be suppressed with --no-resolve-warning, got stderr: %s", stderr)
	}

	// Test JSON output - _warnings should NOT appear
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"dashboard", "orders", "--contact", "12345", "--no-resolve-warning", "-o", "json"})
		if err != nil {
			t.Errorf("dashboard command failed: %v", err)
		}
	})

	if strings.Contains(output, `"_warnings"`) {
		t.Errorf("_warnings should not appear with --no-resolve-warning, got: %s", output)
	}
}

func TestDashboardCommand_DashboardNotFound(t *testing.T) {
	// When the dashboard config is not found, it should error

	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("Dashboard API should not be called")
	})

	// Set up with "orders" dashboard
	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	// Try to use "nonexistent" dashboard
	err := Execute(context.Background(), []string{"dashboard", "nonexistent", "--contact", "123", "--no-resolve"})
	if err == nil {
		t.Error("Expected error for non-existent dashboard")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestDashboardCommand_DashboardNotFoundWithSuggestions(t *testing.T) {
	// When the dashboard config is not found but others exist, it should suggest available ones

	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("Dashboard API should not be called")
	})

	// Set up with "orders" dashboard
	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	// Try to use "invalid" dashboard
	err := Execute(context.Background(), []string{"dashboard", "invalid", "--contact", "123", "--no-resolve"})
	if err == nil {
		t.Error("Expected error for non-existent dashboard")
	}
	if !strings.Contains(err.Error(), "orders") {
		t.Errorf("Expected suggestion of available dashboards, got: %v", err)
	}
}

func TestDashboardCommand_ConversationNotFound(t *testing.T) {
	// When --conversation references a non-existent conversation, it should error

	chatwootHandler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/99999", jsonResponse(404, `{"error": "Conversation not found"}`))

	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("Dashboard API should not be called")
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	err := Execute(context.Background(), []string{"dashboard", "orders", "--conversation", "99999"})
	if err == nil {
		t.Error("Expected error for non-existent conversation")
	}
	if !strings.Contains(err.Error(), "failed to resolve conversation") {
		t.Errorf("Expected 'failed to resolve conversation' error, got: %v", err)
	}
}

func TestDashboardCommand_ConversationNoContactID(t *testing.T) {
	// When a conversation has no contact_id, it should error

	chatwootHandler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/12345", jsonResponse(200, `{
			"id": 12345,
			"contact_id": 0,
			"status": "open"
		}`))

	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("Dashboard API should not be called")
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	err := Execute(context.Background(), []string{"dashboard", "orders", "--conversation", "12345"})
	if err == nil {
		t.Error("Expected error when conversation has no contact ID")
	}
	if !strings.Contains(err.Error(), "does not include a contact id") {
		t.Errorf("Expected 'does not include a contact id' error, got: %v", err)
	}
}

func TestDashboardCommand_DashboardAPIError(t *testing.T) {
	// When the dashboard API returns an error, it should be reported

	chatwootHandler := newRouteHandler()

	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"error": "Internal server error"}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	err := Execute(context.Background(), []string{"dashboard", "orders", "--contact", "123", "--no-resolve"})
	if err == nil {
		t.Error("Expected error for dashboard API failure")
	}
	if !strings.Contains(err.Error(), "dashboard query failed") {
		t.Errorf("Expected 'dashboard query failed' error, got: %v", err)
	}
}

func TestDashboardCommand_AutoResolveDoesNotMatchConversation(t *testing.T) {
	// When --contact ID does not match any conversation, use it directly

	chatwootHandler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/180712", jsonResponse(404, `{"error": "Not found"}`))

	var receivedContactID int
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		receivedContactID = int(req["contact_id"].(float64))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items": [], "pagination": {"page": 1, "total_pages": 1}}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	stderr := captureStderr(t, func() {
		err := Execute(context.Background(), []string{"dashboard", "orders", "--contact", "180712"})
		if err != nil {
			t.Errorf("dashboard command failed: %v", err)
		}
	})

	// Should NOT show resolve warning since no resolution happened
	if strings.Contains(stderr, "Note:") {
		t.Errorf("Should not show warning when no resolution happened, got stderr: %s", stderr)
	}

	// Contact ID should be used directly
	if receivedContactID != 180712 {
		t.Errorf("Expected contact_id 180712 (direct use), got %d", receivedContactID)
	}
}

func TestDashboardCommand_PaginationParams(t *testing.T) {
	// Test that --page and --per-page are passed correctly

	chatwootHandler := newRouteHandler()

	var receivedPage, receivedPerPage int
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		receivedPage = int(req["page"].(float64))
		receivedPerPage = int(req["per_page"].(float64))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items": [], "pagination": {"page": 2, "total_pages": 5}}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	err := Execute(context.Background(), []string{"dashboard", "orders", "--contact", "123", "--no-resolve", "--page", "2", "--per-page", "25"})
	if err != nil {
		t.Errorf("dashboard command failed: %v", err)
	}

	if receivedPage != 2 {
		t.Errorf("Expected page 2, got %d", receivedPage)
	}
	if receivedPerPage != 25 {
		t.Errorf("Expected per_page 25, got %d", receivedPerPage)
	}
}

func TestAddDashboardWarning(t *testing.T) {
	tests := []struct {
		name     string
		result   map[string]any
		warning  string
		expected []string
	}{
		{
			name:     "add first warning",
			result:   map[string]any{},
			warning:  "First warning",
			expected: []string{"First warning"},
		},
		{
			name:     "add to existing warnings",
			result:   map[string]any{"_warnings": []string{"Existing"}},
			warning:  "New warning",
			expected: []string{"Existing", "New warning"},
		},
		{
			name:     "empty warning ignored",
			result:   map[string]any{},
			warning:  "",
			expected: nil, // No _warnings key should be added
		},
		{
			name:     "nil result ignored",
			result:   nil,
			warning:  "Warning",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addDashboardWarning(tt.result, tt.warning)

			if tt.expected == nil {
				if tt.result != nil {
					if _, ok := tt.result["_warnings"]; ok {
						t.Errorf("Expected no _warnings key")
					}
				}
				return
			}

			warnings, ok := tt.result["_warnings"].([]string)
			if !ok {
				t.Fatalf("Expected _warnings to be []string")
			}
			if len(warnings) != len(tt.expected) {
				t.Fatalf("Expected %d warnings, got %d", len(tt.expected), len(warnings))
			}
			for i, w := range warnings {
				if w != tt.expected[i] {
					t.Errorf("Warning[%d] = %q, want %q", i, w, tt.expected[i])
				}
			}
		})
	}
}

func TestExtractContactIDFromConversation(t *testing.T) {
	tests := []struct {
		name     string
		convJSON string
		expected int
		ok       bool
	}{
		{
			name:     "contact_id present",
			convJSON: `{"id": 1, "contact_id": 123}`,
			expected: 123,
			ok:       true,
		},
		{
			name:     "contact_id zero, meta.sender.id present",
			convJSON: `{"id": 1, "contact_id": 0, "meta": {"sender": {"id": 456}}}`,
			expected: 456,
			ok:       true,
		},
		{
			name:     "contact_id zero, no meta",
			convJSON: `{"id": 1, "contact_id": 0}`,
			expected: 0,
			ok:       false,
		},
		{
			name:     "contact_id zero, meta.sender.id zero",
			convJSON: `{"id": 1, "contact_id": 0, "meta": {"sender": {"id": 0}}}`,
			expected: 0,
			ok:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var conv api.Conversation
			_ = json.Unmarshal([]byte(tt.convJSON), &conv)

			got, ok := extractContactIDFromConversation(&conv)
			if ok != tt.ok {
				t.Errorf("extractContactIDFromConversation() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.expected {
				t.Errorf("extractContactIDFromConversation() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestParseAnyInt(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
		ok       bool
	}{
		{name: "float64", input: float64(123), expected: 123, ok: true},
		{name: "int", input: int(456), expected: 456, ok: true},
		{name: "int64", input: int64(789), expected: 789, ok: true},
		{name: "string numeric", input: "123", expected: 123, ok: true},
		{name: "json.Number", input: json.Number("456"), expected: 456, ok: true},
		{name: "string non-numeric", input: "abc", expected: 0, ok: false},
		{name: "nil", input: nil, expected: 0, ok: false},
		{name: "bool", input: true, expected: 0, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseAnyInt(tt.input)
			if ok != tt.ok {
				t.Errorf("parseAnyInt(%v) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if got != tt.expected {
				t.Errorf("parseAnyInt(%v) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		// Basic type tests
		{name: "empty string", input: "", expected: ""},
		{name: "nil", input: nil, expected: "-"},
		{name: "bool true", input: true, expected: "yes"},
		{name: "bool false", input: false, expected: "no"},
		{name: "integer float", input: float64(42), expected: "42"},
		{name: "decimal float", input: float64(3.14159), expected: "3.14"},
		{name: "other type", input: 123, expected: "123"},

		// String truncation - ASCII
		{name: "short ASCII", input: "hello", expected: "hello"},
		{name: "exactly 30 chars", input: "123456789012345678901234567890", expected: "123456789012345678901234567890"},
		{name: "31 chars truncated", input: "1234567890123456789012345678901", expected: "123456789012345678901234567..."},

		// String truncation - Unicode (emoji are multi-byte but single runes)
		{
			name:     "emoji within limit",
			input:    "Hello 🎉🎊🎁", // 10 characters (7 ASCII + 3 emoji)
			expected: "Hello 🎉🎊🎁",
		},
		{
			name:     "emoji string exactly 30 chars",
			input:    "🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉", // 30 emoji
			expected: "🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉",
		},
		{
			name:     "emoji string over 30 chars truncated",
			input:    "🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉", // 31 emoji
			expected: "🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉...",  // 27 emoji + ...
		},

		// String truncation - CJK characters (multi-byte UTF-8)
		{
			name:     "CJK within limit",
			input:    "你好世界", // 4 characters
			expected: "你好世界",
		},
		{
			name:     "CJK exactly 30 chars",
			input:    "一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十", // 30 CJK characters
			expected: "一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十",
		},
		{
			name:     "CJK over 30 chars truncated",
			input:    "一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十一", // 31 CJK characters
			expected: "一二三四五六七八九十一二三四五六七八九十一二三四五六七...",  // 27 CJK + ...
		},

		// Mixed content
		{
			name:     "mixed ASCII and emoji over limit",
			input:    "Product: 🎁 Gift Box Special Edition 2024!", // 41 chars
			expected: "Product: 🎁 Gift Box Special...",
		},

		// Slice and map types
		{name: "empty slice", input: []any{}, expected: "[]"},
		{name: "slice with items", input: []any{1, 2, 3}, expected: "[3 items]"},
		{name: "empty map", input: map[string]any{}, expected: "{}"},
		{name: "map with keys", input: map[string]any{"a": 1, "b": 2}, expected: "{2 keys}"},

		// Default type handling
		{name: "int type", input: int64(1234567890), expected: "1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.input)
			if got != tt.expected {
				t.Errorf("formatValue(%v) = %q, want %q", tt.input, got, tt.expected)
			}

			// Verify truncated strings end with "..." and have correct rune count
			if s, ok := tt.input.(string); ok && len([]rune(s)) > 30 {
				if !strings.HasSuffix(got, "...") {
					t.Errorf("formatValue(%v) should end with '...', got %q", tt.input, got)
				}
				// 27 runes + 3 for "..." = 30 total runes
				gotRunes := []rune(got)
				if len(gotRunes) != 30 {
					t.Errorf("formatValue(%v) should have 30 runes, got %d", tt.input, len(gotRunes))
				}
			}
		})
	}
}
