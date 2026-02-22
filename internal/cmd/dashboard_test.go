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

// setupMultiDashboardTestEnv sets up a test environment with multiple dashboards configured.
func setupMultiDashboardTestEnv(t *testing.T, chatwootHandler, dashboardHandler http.Handler, dashboardNames []string) {
	t.Helper()

	chatwootServer := httptest.NewServer(chatwootHandler)
	t.Cleanup(chatwootServer.Close)

	dashboardServer := httptest.NewServer(dashboardHandler)
	t.Cleanup(dashboardServer.Close)

	ring := keyring.NewArrayKeyring(nil)

	dashboards := make(map[string]*config.DashboardConfig, len(dashboardNames))
	for _, name := range dashboardNames {
		dashboards[name] = &config.DashboardConfig{
			Name:      name,
			Endpoint:  dashboardServer.URL,
			AuthToken: "test@example.com",
		}
	}

	account := config.Account{
		BaseURL:   chatwootServer.URL,
		APIToken:  "test-token",
		AccountID: 1,
		Extensions: &config.Extensions{
			Dashboards: dashboards,
		},
	}

	data, _ := json.Marshal(account)
	_ = ring.Set(keyring.Item{Key: "default", Data: data})
	_ = ring.Set(keyring.Item{Key: "current_profile", Data: []byte("default")})

	cleanup := config.SetOpenKeyring(func(cfg keyring.Config) (keyring.Keyring, error) {
		return ring, nil
	})
	t.Cleanup(cleanup)

	t.Setenv("CHATWOOT_TESTING", "1")
	t.Setenv("CHATWOOT_OUTPUT", "text")
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

func TestIsSubsequence(t *testing.T) {
	tests := []struct {
		needle   string
		haystack string
		want     bool
	}{
		{needle: "ods", haystack: "orders", want: true},
		{needle: "od", haystack: "orders", want: true},
		{needle: "os", haystack: "orders", want: true},
		{needle: "orders", haystack: "orders", want: true},
		{needle: "ord", haystack: "orders", want: true},
		{needle: "", haystack: "orders", want: true},
		{needle: "xyz", haystack: "orders", want: false},
		{needle: "sdo", haystack: "orders", want: false},
		{needle: "ordersx", haystack: "orders", want: false},
		{needle: "ods", haystack: "", want: false},
		{needle: "tk", haystack: "tickets", want: true},
		{needle: "tks", haystack: "tickets", want: true},
		{needle: "tts", haystack: "tickets", want: true},
		{needle: "ov", haystack: "overview", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.needle+"_in_"+tt.haystack, func(t *testing.T) {
			got := isSubsequence(tt.needle, tt.haystack)
			if got != tt.want {
				t.Errorf("isSubsequence(%q, %q) = %v, want %v", tt.needle, tt.haystack, got, tt.want)
			}
		})
	}
}

func TestDashboardCommand_SubsequenceMatch(t *testing.T) {
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items": [], "pagination": {"page": 1, "total_pages": 1}}`))
	})

	setupMultiDashboardTestEnv(t, chatwootHandler, dashboardHandler, []string{"orders", "overview", "tickets"})

	// "ods" is a subsequence of "orders" (o-d-s in o-r-d-e-r-s) but NOT a prefix
	err := Execute(context.Background(), []string{"dashboard", "ods", "--contact", "123", "--no-resolve"})
	if err != nil {
		t.Errorf("subsequence 'ods' should match 'orders': %v", err)
	}

	// "tks" is a subsequence of "tickets" (t-k-s in t-i-c-k-e-t-s)
	err = Execute(context.Background(), []string{"dashboard", "tks", "--contact", "123", "--no-resolve"})
	if err != nil {
		t.Errorf("subsequence 'tks' should match 'tickets': %v", err)
	}
}

func TestDashboardCommand_SubsequenceMatchAmbiguous(t *testing.T) {
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("Dashboard API should not be called for ambiguous match")
	})

	setupMultiDashboardTestEnv(t, chatwootHandler, dashboardHandler, []string{"orders", "overview", "tickets"})

	// "oe" is a subsequence of both "orders" (o...e in o-r-d-e-r-s) and "overview" (o...e in o-v-e-r-v-i-e-w)
	err := Execute(context.Background(), []string{"dashboard", "oe", "--contact", "123", "--no-resolve"})
	if err == nil {
		t.Error("Expected error for ambiguous subsequence 'oe'")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("Expected 'ambiguous' error, got: %v", err)
	}
}

func TestDashboardCommand_PrefixPreferredOverSubsequence(t *testing.T) {
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items": [], "pagination": {"page": 1, "total_pages": 1}}`))
	})

	setupMultiDashboardTestEnv(t, chatwootHandler, dashboardHandler, []string{"orders", "overview", "tickets"})

	// "ord" is a prefix of "orders" — should match via prefix, not fall through to subsequence
	err := Execute(context.Background(), []string{"dashboard", "ord", "--contact", "123", "--no-resolve"})
	if err != nil {
		t.Errorf("prefix 'ord' should match 'orders': %v", err)
	}
}

func TestDashboardCommand_JQFilterWithAliases(t *testing.T) {
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"items": [
				{"id": 1, "order_total": 1500, "order_status": "completed"},
				{"id": 2, "order_total": 5000, "order_status": "pending"},
				{"id": 3, "order_total": 2000, "order_status": "completed"}
			],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	// Test that .it resolves to .items and .ot resolves to .order_total
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "json", "--jq", "[.it[] | select(.ot > 2000)]",
		})
		if err != nil {
			t.Fatalf("--jq with aliases failed: %v", err)
		}
	})

	// Should only contain the item with order_total 5000
	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse output JSON: %v\nOutput: %s", err, output)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 item with order_total > 2000, got %d. Output: %s", len(result), output)
	}

	if id, ok := result[0]["id"].(float64); !ok || int(id) != 2 {
		t.Errorf("Expected item with id 2, got: %v", result[0])
	}
}

func TestDashboardCommand_JQFilterBackslashBang(t *testing.T) {
	// Verify that \! in --jq expressions is normalized to ! (zsh history expansion workaround)
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"items": [
				{"id": 1, "order_total": 1500},
				{"id": 2, "order_total": null},
				{"id": 3, "order_total": 2000}
			],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	// Use \!= which zsh would produce — should be normalized to !=
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "json", "--jq", `[.items[] | select(.order_total \!= null)]`,
		})
		if err != nil {
			t.Fatalf("--jq with backslash-bang failed: %v", err)
		}
	})

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse output JSON: %v\nOutput: %s", err, output)
	}

	if len(result) != 2 {
		t.Fatalf("Expected 2 items with non-null order_total, got %d. Output: %s", len(result), output)
	}
}

func TestDashboardCommand_AgentModeJQAliases(t *testing.T) {
	// In agent mode, .it (alias for .items) must work because the dashboard
	// result should use a ListEnvelope with items at the top level, not a
	// DataEnvelope that nests items under .data.
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {"customer_name": "Test User"},
			"items": [
				{"number": "001", "order_total": 1500},
				{"number": "002", "order_total": 5000},
				{"number": "003", "order_total": 2000}
			],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	// Test .it alias works in agent mode
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "agent", "--jq", "[.it[] | {n: .name, ot}]",
		})
		if err != nil {
			t.Fatalf("--jq with .it alias in agent mode failed: %v", err)
		}
	})

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse output JSON: %v\nOutput: %s", err, output)
	}

	if len(result) != 3 {
		t.Fatalf("Expected 3 items, got %d. Output: %s", len(result), output)
	}
}

func TestDashboardCommand_AgentModeStructure(t *testing.T) {
	// Verify agent mode produces a ListEnvelope with items at the top level
	// and metadata in the meta field.
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {"customer_name": "Test User"},
			"items": [{"id": 1}],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "agent",
		})
		if err != nil {
			t.Fatalf("agent mode output failed: %v", err)
		}
	})

	var envelope map[string]any
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("Failed to parse output JSON: %v\nOutput: %s", err, output)
	}

	// Should have kind, items, and meta at the top level
	if _, ok := envelope["kind"]; !ok {
		t.Error("Expected 'kind' key in agent output")
	}
	if _, ok := envelope["items"]; !ok {
		t.Error("Expected 'items' key at top level in agent output")
	}
	if _, ok := envelope["meta"]; !ok {
		t.Error("Expected 'meta' key in agent output")
	}

	// Should NOT have 'data' key (old DataEnvelope structure)
	if _, ok := envelope["data"]; ok {
		t.Error("Agent output should not use DataEnvelope with 'data' key")
	}

	// Meta should contain customer_info and pagination
	meta, ok := envelope["meta"].(map[string]any)
	if !ok {
		t.Fatal("Expected meta to be a map")
	}
	if _, ok := meta["customer_info"]; !ok {
		t.Error("Expected customer_info in meta")
	}
	if _, ok := meta["pagination"]; !ok {
		t.Error("Expected pagination in meta")
	}
}

func TestDashboardCommand_AgentModeWarnings(t *testing.T) {
	// Verify _warnings are preserved in agent mode meta
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
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "12345",
			"-o", "agent",
		})
		if err != nil {
			t.Fatalf("agent mode output failed: %v", err)
		}
	})

	var envelope map[string]any
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("Failed to parse output JSON: %v\nOutput: %s", err, output)
	}

	meta, ok := envelope["meta"].(map[string]any)
	if !ok {
		t.Fatal("Expected meta to be a map")
	}

	if _, ok := meta["_warnings"]; !ok {
		t.Error("Expected _warnings in meta (auto-resolve warning should be present)")
	}
}

func TestCompactDashboardResult(t *testing.T) {
	input := map[string]any{
		"customer_info": map[string]any{
			"customer_name":        "Test User",
			"membership_tier_name": "Gold",
			"total_spend":          50000.0,
			"extra_field":          "ignored",
		},
		"items": []any{
			map[string]any{
				"number":              "ORD-001",
				"shopline_created_at": "2026-01-15T10:30:00Z",
				"order_total":         1500.0,
				"order_status":        "completed",
				"payment_status":      "paid",
				"delivery_status":     "delivered",
				"total_items_count":   3.0,
				"uuid":                "abc-123-def",
				"customer_email":      "test@example.com",
				"shipping_address":    map[string]any{"city": "Taipei"},
				"line_items":          []any{1, 2, 3},
			},
			map[string]any{
				"number":              "ORD-002",
				"shopline_created_at": "2026-02-01T14:00:00Z",
				"order_total":         3000.0,
				"order_status":        "pending",
				"payment_status":      "unpaid",
				"delivery_status":     "unfulfilled",
				"total_items_count":   1.0,
				"uuid":                "xyz-789",
				"notes":               "rush order",
			},
		},
		"pagination": map[string]any{
			"page":          1.0,
			"total_pages":   1.0,
			"total_records": 2.0,
		},
	}

	result := compactDashboardResult(input)

	// Top-level should have tier, items, and pagination
	tier, ok := result["tier"].(string)
	if !ok || tier != "Gold" {
		t.Errorf("Expected tier=Gold, got %v", result["tier"])
	}

	items, ok := result["items"].([]any)
	if !ok {
		t.Fatalf("Expected items to be []any, got %T", result["items"])
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	// First order should have exactly 7 fields
	order1, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected order to be map[string]any, got %T", items[0])
	}
	if len(order1) != 7 {
		t.Errorf("Expected 7 fields per order, got %d: %v", len(order1), order1)
	}

	// Verify essential fields
	if order1["num"] != "ORD-001" {
		t.Errorf("num = %v, want ORD-001", order1["num"])
	}
	if order1["dt"] != "2026-01-15" {
		t.Errorf("dt = %v, want 2026-01-15", order1["dt"])
	}
	if order1["tot"] != 1500.0 {
		t.Errorf("tot = %v, want 1500", order1["tot"])
	}
	if order1["st"] != "completed" {
		t.Errorf("st = %v, want completed", order1["st"])
	}
	if order1["pay"] != "paid" {
		t.Errorf("pay = %v, want paid", order1["pay"])
	}
	if order1["dlv"] != "delivered" {
		t.Errorf("dlv = %v, want delivered", order1["dlv"])
	}
	if order1["items"] != 3.0 {
		t.Errorf("items = %v, want 3", order1["items"])
	}

	// Verify stripped fields are gone
	if _, exists := order1["uuid"]; exists {
		t.Error("uuid should be stripped in compact mode")
	}
	if _, exists := order1["customer_email"]; exists {
		t.Error("customer_email should be stripped in compact mode")
	}

	// Pagination should be preserved
	if _, ok := result["pg"].(map[string]any); !ok {
		t.Error("Expected pg to be preserved")
	}
}

func TestCompactDashboardResult_DateTruncation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "ISO datetime", input: "2026-01-15T10:30:00Z", expected: "2026-01-15"},
		{name: "date only", input: "2026-01-15", expected: "2026-01-15"},
		{name: "short string", input: "2026", expected: "2026"},
		{name: "empty", input: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]any{
				"items": []any{
					map[string]any{
						"shopline_created_at": tt.input,
						"number":              "X",
					},
				},
			}

			result := compactDashboardResult(input)
			items := result["items"].([]any)
			order := items[0].(map[string]any)

			if order["dt"] != tt.expected {
				t.Errorf("dt = %q, want %q", order["dt"], tt.expected)
			}
		})
	}
}

func TestCompactDashboardResult_NoItems(t *testing.T) {
	input := map[string]any{
		"customer_info": map[string]any{
			"membership_tier_name": "Silver",
		},
		"pagination": map[string]any{"page": 1.0},
	}

	result := compactDashboardResult(input)

	if result["tier"] != "Silver" {
		t.Errorf("tier = %v, want Silver", result["tier"])
	}
	items, ok := result["items"].([]any)
	if !ok {
		t.Fatalf("Expected items to be []any, got %T", result["items"])
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items))
	}
}

func TestCompactDashboardResult_NoCustomerInfo(t *testing.T) {
	input := map[string]any{
		"items": []any{
			map[string]any{"number": "ORD-001", "order_total": 100.0},
		},
	}

	result := compactDashboardResult(input)

	// tier should be absent when no customer_info
	if _, exists := result["tier"]; exists {
		t.Error("tier should not be set when customer_info is missing")
	}

	items := result["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("Expected 1 order, got %d", len(items))
	}
}

func TestDashboardCommand_CompactJSONOutput(t *testing.T) {
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {
				"customer_name": "Test User",
				"membership_tier_name": "Gold",
				"total_spend": 50000
			},
			"items": [
				{
					"number": "ORD-001",
					"shopline_created_at": "2026-01-15T10:30:00Z",
					"order_total": 1500,
					"order_status": "completed",
					"payment_status": "paid",
					"delivery_status": "delivered",
					"total_items_count": 3,
					"uuid": "abc-123",
					"customer_email": "test@example.com"
				}
			],
			"pagination": {"page": 1, "total_pages": 1, "total_records": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "json", "--compact",
		})
		if err != nil {
			t.Fatalf("compact JSON output failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse compact JSON: %v\nOutput: %s", err, output)
	}

	// Should have compact structure
	if result["tier"] != "Gold" {
		t.Errorf("tier = %v, want Gold", result["tier"])
	}

	items, ok := result["items"].([]any)
	if !ok {
		t.Fatalf("Expected items array, got %T", result["items"])
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	order := items[0].(map[string]any)
	if order["dt"] != "2026-01-15" {
		t.Errorf("dt = %v, want 2026-01-15", order["dt"])
	}

	// Verbose fields should be absent
	if _, exists := order["uuid"]; exists {
		t.Error("uuid should not appear in compact output")
	}
	if _, exists := order["customer_email"]; exists {
		t.Error("customer_email should not appear in compact output")
	}
}

func TestDashboardCommand_CompactAgentOutput(t *testing.T) {
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {
				"customer_name": "Test User",
				"membership_tier_name": "Diamond"
			},
			"items": [
				{"number": "ORD-001", "order_total": 1500, "order_status": "completed", "payment_status": "paid", "delivery_status": "delivered", "total_items_count": 2, "shopline_created_at": "2026-01-15T10:30:00Z", "uuid": "hidden"}
			],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "agent", "--compact",
		})
		if err != nil {
			t.Fatalf("compact agent output failed: %v", err)
		}
	})

	var envelope map[string]any
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("Failed to parse compact agent JSON: %v\nOutput: %s", err, output)
	}

	// Verify the envelope has kind and items
	if _, ok := envelope["kind"]; !ok {
		t.Error("Expected kind in agent envelope")
	}

	// For agent mode, items should be the compact orders
	items, ok := envelope["items"].([]any)
	if !ok {
		t.Fatalf("Expected items array, got %T", envelope["items"])
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	item := items[0].(map[string]any)
	if _, exists := item["uuid"]; exists {
		t.Error("uuid should not appear in compact agent output")
	}
	if item["dt"] != "2026-01-15" {
		t.Errorf("dt = %v, want 2026-01-15", item["dt"])
	}

	// Meta should contain tier
	meta, ok := envelope["meta"].(map[string]any)
	if !ok {
		t.Fatalf("Expected meta map, got %T", envelope["meta"])
	}
	if meta["tier"] != "Diamond" {
		t.Errorf("meta.tier = %v, want Diamond", meta["tier"])
	}
}

func TestDashboardCommand_CompactWithBriefAlias(t *testing.T) {
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {"membership_tier_name": "Bronze"},
			"items": [{"number": "X", "order_total": 100}],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "json", "--brief",
		})
		if err != nil {
			t.Fatalf("--brief alias failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse brief JSON: %v\nOutput: %s", err, output)
	}

	if result["tier"] != "Bronze" {
		t.Errorf("tier = %v, want Bronze (via --brief alias)", result["tier"])
	}
}

func TestDashboardCommand_CompactWithSummaryAlias(t *testing.T) {
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {"membership_tier_name": "Silver"},
			"items": [{"number": "X", "order_total": 200}],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "json", "--summary",
		})
		if err != nil {
			t.Fatalf("--summary alias failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse summary JSON: %v\nOutput: %s", err, output)
	}

	if result["tier"] != "Silver" {
		t.Errorf("tier = %v, want Silver (via --summary alias)", result["tier"])
	}
}

func TestDashboardCommand_CompactWithJQ(t *testing.T) {
	// Compact output should compose with --jq
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {"membership_tier_name": "Gold"},
			"items": [
				{"number": "ORD-001", "order_total": 1500, "order_status": "completed", "payment_status": "paid", "delivery_status": "delivered", "total_items_count": 3, "shopline_created_at": "2026-01-15T10:30:00Z"},
				{"number": "ORD-002", "order_total": 5000, "order_status": "pending", "payment_status": "unpaid", "delivery_status": "unfulfilled", "total_items_count": 1, "shopline_created_at": "2026-02-01T14:00:00Z"}
			],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "json", "--compact", "--jq", "[.items[] | select(.tot > 2000)]",
		})
		if err != nil {
			t.Fatalf("compact + jq failed: %v", err)
		}
	})

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse compact+jq JSON: %v\nOutput: %s", err, output)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 order with total > 2000, got %d", len(result))
	}
	if result[0]["num"] != "ORD-002" {
		t.Errorf("Expected ORD-002, got %v", result[0]["num"])
	}
}

func TestDashboardCommand_WithoutCompactUnchanged(t *testing.T) {
	// Without --compact, output should be unchanged (full fields)
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {"customer_name": "Test"},
			"items": [{"number": "ORD-001", "uuid": "abc-123", "order_total": 100}],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "json",
		})
		if err != nil {
			t.Fatalf("non-compact JSON output failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, output)
	}

	// Full output should have customer_info (not just tier)
	if _, ok := result["customer_info"]; !ok {
		t.Error("Full output should have customer_info")
	}

	// Full output should have items with all fields
	items := result["items"].([]any)
	order := items[0].(map[string]any)
	if _, exists := order["uuid"]; !exists {
		t.Error("Full output should include uuid")
	}
}

func TestDashboardCommand_CompactShortFlag(t *testing.T) {
	// Test -c short flag works
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {"membership_tier_name": "Gold"},
			"items": [{"number": "X", "order_total": 100}],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-j", "-c",
		})
		if err != nil {
			t.Fatalf("-c short flag failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse compact JSON: %v\nOutput: %s", err, output)
	}

	if _, ok := result["tier"]; !ok {
		t.Error("-c flag should produce compact output with tier field")
	}
}

func TestDashboardCommand_CompactIgnoredForTextOutput(t *testing.T) {
	// --compact should be silently ignored for text output mode, rendering the
	// full text table unchanged.
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {
				"customer_name": "Test User",
				"membership_tier_name": "Gold",
				"total_spend": 50000
			},
			"items": [
				{
					"number": "ORD-001",
					"order_total": 1500,
					"order_status": "completed",
					"payment_status": "paid",
					"uuid": "abc-123"
				}
			],
			"pagination": {"page": 1, "total_pages": 1, "total_records": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	// Capture text output WITHOUT --compact
	outputWithout := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "text",
		})
		if err != nil {
			t.Fatalf("text output without --compact failed: %v", err)
		}
	})

	// Capture text output WITH --compact
	outputWith := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "text", "--compact",
		})
		if err != nil {
			t.Fatalf("text output with --compact failed: %v", err)
		}
	})

	if outputWithout != outputWith {
		t.Errorf("--compact should not change text output.\nWithout: %s\nWith:    %s", outputWithout, outputWith)
	}

	// Sanity: text output should contain table headers, not JSON structure
	if !strings.Contains(outputWith, "NUMBER") && !strings.Contains(outputWith, "ORD-001") {
		t.Errorf("Expected text table output, got: %s", outputWith)
	}
}

func TestDashboardCommand_PgAlias(t *testing.T) {
	// Verify the --pg alias for --page is registered and functional.
	chatwootHandler := newRouteHandler()

	var receivedPage int
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		receivedPage = int(req["page"].(float64))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items": [], "pagination": {"page": 2, "total_pages": 5}}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	err := Execute(context.Background(), []string{
		"dashboard", "orders", "--contact", "123", "--no-resolve", "--pg", "2",
	})
	if err != nil {
		t.Fatalf("--pg alias failed: %v", err)
	}

	if receivedPage != 2 {
		t.Errorf("Expected page 2 via --pg alias, got %d", receivedPage)
	}
}

func TestDashboardCommand_CompactWarningsPreserved(t *testing.T) {
	// When --compact is used with auto-resolve, _warnings should still appear
	chatwootHandler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/12345", jsonResponse(200, `{
			"id": 12345,
			"contact_id": 99999,
			"status": "open"
		}`))

	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {"membership_tier_name": "Gold"},
			"items": [{"number": "X", "order_total": 100}],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "12345",
			"-o", "json", "--compact",
		})
		if err != nil {
			t.Fatalf("compact with warnings failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, output)
	}

	if _, ok := result["_warnings"]; !ok {
		t.Error("_warnings should be preserved in compact mode")
	}
}

func TestCompactDashboardResult_MissingDateField(t *testing.T) {
	// When shopline_created_at is missing, date should not be set
	input := map[string]any{
		"items": []any{
			map[string]any{
				"number":         "ORD-001",
				"order_total":    100.0,
				"order_status":   "completed",
				"payment_status": "paid",
			},
		},
	}

	result := compactDashboardResult(input)
	order := result["items"].([]any)[0].(map[string]any)

	// Only the fields that exist in the source should appear
	if _, exists := order["dt"]; exists {
		t.Error("dt should not be set when shopline_created_at is missing")
	}
}

func TestDashboardCommand_Light(t *testing.T) {
	// --light should return compact output with at most 3 items
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {
				"customer_name": "Test User",
				"membership_tier_name": "Gold",
				"total_spend": 50000
			},
			"items": [
				{"number": "ORD-001", "shopline_created_at": "2026-01-10T10:00:00Z", "order_total": 1000, "order_status": "completed", "payment_status": "paid", "delivery_status": "delivered", "total_items_count": 2, "uuid": "aaa"},
				{"number": "ORD-002", "shopline_created_at": "2026-01-15T10:00:00Z", "order_total": 2000, "order_status": "completed", "payment_status": "paid", "delivery_status": "delivered", "total_items_count": 1, "uuid": "bbb"},
				{"number": "ORD-003", "shopline_created_at": "2026-01-20T10:00:00Z", "order_total": 3000, "order_status": "pending", "payment_status": "unpaid", "delivery_status": "unfulfilled", "total_items_count": 4, "uuid": "ccc"},
				{"number": "ORD-004", "shopline_created_at": "2026-02-01T10:00:00Z", "order_total": 4000, "order_status": "completed", "payment_status": "paid", "delivery_status": "delivered", "total_items_count": 1, "uuid": "ddd"},
				{"number": "ORD-005", "shopline_created_at": "2026-02-10T10:00:00Z", "order_total": 5000, "order_status": "completed", "payment_status": "paid", "delivery_status": "delivered", "total_items_count": 3, "uuid": "eee"}
			],
			"pagination": {"page": 1, "total_pages": 1, "total_records": 5}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve", "--light",
		})
		if err != nil {
			t.Fatalf("dashboard --light failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse light JSON: %v\noutput: %s", err, output)
	}

	// Should have tier from compact transform
	if result["tier"] != "Gold" {
		t.Errorf("tier = %v, want Gold", result["tier"])
	}
	if result["n"] != float64(5) {
		t.Errorf("n = %v, want 5", result["n"])
	}

	// Should have at most 3 items
	items, ok := result["it"].([]any)
	if !ok {
		t.Fatalf("expected it array, got %T", result["it"])
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items (capped), got %d", len(items))
	}

	// First item should be minimal light format.
	order := items[0].(map[string]any)
	if order["num"] != "ORD-001" {
		t.Errorf("first order num = %v, want ORD-001", order["num"])
	}
	if order["st"] != "d" {
		t.Errorf("first order st = %v, want d", order["st"])
	}
	third := items[2].(map[string]any)
	if third["st"] != "p" {
		t.Errorf("third order st = %v, want p", third["st"])
	}
	if _, exists := order["pay"]; exists {
		t.Error("light output should not contain pay")
	}
	if _, exists := order["dlv"]; exists {
		t.Error("light output should not contain dlv")
	}
	if _, exists := order["items"]; exists {
		t.Error("light output should not contain item count")
	}
}

func TestDashboardCommand_LightFewItems(t *testing.T) {
	// --light with fewer than 3 items should return all of them
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"customer_info": {"membership_tier_name": "Silver"},
			"items": [
				{"number": "ORD-001", "order_total": 1000, "order_status": "completed"}
			],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve", "--li",
		})
		if err != nil {
			t.Fatalf("dashboard --li failed: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse light JSON: %v\noutput: %s", err, output)
	}

	items := result["it"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	order := items[0].(map[string]any)
	if order["st"] != "d" {
		t.Errorf("order st = %v, want d", order["st"])
	}
}

func TestDashboardCommand_AgentModeSlicing(t *testing.T) {
	// The exact command from the bug report: cw dash ods --cv CONV_ID --jq '[.it[-3:] | .[] | {n: .number, ot}]'
	chatwootHandler := newRouteHandler()
	dashboardHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{
			"items": [
				{"number": "001", "order_total": 1000},
				{"number": "002", "order_total": 2000},
				{"number": "003", "order_total": 3000},
				{"number": "004", "order_total": 4000},
				{"number": "005", "order_total": 5000}
			],
			"pagination": {"page": 1, "total_pages": 1}
		}`))
	})

	setupDashboardTestEnv(t, chatwootHandler, dashboardHandler, "orders")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"dashboard", "orders", "--contact", "123", "--no-resolve",
			"-o", "agent", "--jq", `[.it[-3:] | .[] | {n: .number, ot}]`,
		})
		if err != nil {
			t.Fatalf("slicing --jq in agent mode failed: %v", err)
		}
	})

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse output JSON: %v\nOutput: %s", err, output)
	}

	if len(result) != 3 {
		t.Fatalf("Expected last 3 items, got %d. Output: %s", len(result), output)
	}

	// Verify the items are the last 3
	expected := []struct {
		number     string
		orderTotal float64
	}{
		{"003", 3000},
		{"004", 4000},
		{"005", 5000},
	}
	for i, exp := range expected {
		if n, ok := result[i]["n"].(string); !ok || n != exp.number {
			t.Errorf("result[%d].n = %v, want %q", i, result[i]["n"], exp.number)
		}
		if ot, ok := result[i]["order_total"].(float64); !ok || ot != exp.orderTotal {
			t.Errorf("result[%d].order_total = %v, want %v", i, result[i]["order_total"], exp.orderTotal)
		}
	}
}
