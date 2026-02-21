package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/iocontext"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func TestIsJSON(t *testing.T) {
	tests := []struct {
		name     string
		mode     outfmt.Mode
		expected bool
	}{
		{"text output", outfmt.Text, false},
		{"json output", outfmt.JSON, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			ctx := outfmt.WithMode(context.Background(), tt.mode)
			cmd.SetContext(ctx)

			result := isJSON(cmd)
			if result != tt.expected {
				t.Errorf("isJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsJSON_NoContext(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	// Without mode set, should default to text (false)
	result := isJSON(cmd)
	if result != false {
		t.Errorf("isJSON() with no mode set = %v, want false", result)
	}
}

func TestPrintJSON(t *testing.T) {
	// printJSON writes to the command's configured output (defaults to os.Stdout).
	// These tests verify error handling and that the function completes successfully.
	tests := []struct {
		name    string
		data    any
		wantErr bool
	}{
		{
			name:    "simple struct",
			data:    struct{ Name string }{"test"},
			wantErr: false,
		},
		{
			name:    "slice",
			data:    []int{1, 2, 3},
			wantErr: false,
		},
		{
			name:    "nil",
			data:    nil,
			wantErr: false,
		},
		{
			name:    "map",
			data:    map[string]int{"count": 42},
			wantErr: false,
		},
		{
			name:    "nested struct",
			data:    struct{ Inner struct{ Value int } }{struct{ Value int }{42}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())

			err := printJSON(cmd, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("printJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetClientTimeoutOverride(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://example.com")
	t.Setenv("CHATWOOT_API_TOKEN", "token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	original := flags.Timeout
	flags.Timeout = 45 * time.Second
	defer func() { flags.Timeout = original }()

	client, err := getClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.HTTP.Timeout != 45*time.Second {
		t.Fatalf("expected timeout 45s, got %s", client.HTTP.Timeout)
	}
}

func TestPrintJSON_WithQuery(t *testing.T) {
	cmd := &cobra.Command{}
	ctx := outfmt.WithQuery(context.Background(), ".name")
	cmd.SetContext(ctx)

	data := map[string]string{"name": "test", "other": "value"}
	err := printJSON(cmd, data)
	if err != nil {
		t.Errorf("printJSON() with query error = %v", err)
	}
}

func TestPrintJSON_WithInvalidQuery(t *testing.T) {
	cmd := &cobra.Command{}
	ctx := outfmt.WithQuery(context.Background(), ".[invalid")
	cmd.SetContext(ctx)

	data := map[string]string{"name": "test"}
	err := printJSON(cmd, data)
	if err == nil {
		t.Error("printJSON() with invalid query should return error")
	}
}

func TestCmdContext(t *testing.T) {
	cmd := &cobra.Command{}
	ctx := context.Background()
	cmd.SetContext(ctx)

	result := cmdContext(cmd)
	if result == nil {
		t.Error("cmdContext() returned nil")
	}

	// Should return a valid context
	if result.Err() != nil {
		t.Errorf("cmdContext() returned cancelled context: %v", result.Err())
	}
}

func TestCmdContext_NilContext(t *testing.T) {
	cmd := &cobra.Command{}
	// Don't set context - cobra returns nil by default

	result := cmdContext(cmd)
	// When no context is set, cobra.Command.Context() returns nil
	if result != nil {
		t.Logf("cmdContext() returned non-nil context when none was set")
	}
}

func TestNewTabWriter(t *testing.T) {
	w := newTabWriter(io.Discard)
	if w == nil {
		t.Error("newTabWriter() returned nil")
	}
}

func TestValidatePriority(t *testing.T) {
	tests := []struct {
		name     string
		priority string
		want     string
		wantErr  bool
	}{
		{"urgent", "urgent", "urgent", false},
		{"high", "high", "high", false},
		{"medium", "medium", "medium", false},
		{"low", "low", "low", false},
		{"none", "none", "none", false},
		{"invalid", "invalid", "", true},
		{"empty", "", "", true},
		{"uppercase", "HIGH", "high", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validatePriority(tt.priority)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePriority(%q) error = %v, wantErr %v", tt.priority, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("validatePriority(%q) = %q, want %q", tt.priority, got, tt.want)
			}
		})
	}
}

func TestValidatePriority_StructuredError(t *testing.T) {
	_, err := validatePriority("critical")
	var se *api.StructuredError
	if !errors.As(err, &se) {
		t.Fatal("expected StructuredError")
	}
	if se.Code != api.ErrValidation {
		t.Errorf("got code %q, want validation_failed", se.Code)
	}
	if len(se.AllowedValues) != 5 {
		t.Errorf("expected 5 allowed values, got %d", len(se.AllowedValues))
	}
	if se.Context["field"] != "priority" {
		t.Errorf("expected field=priority, got %v", se.Context["field"])
	}
	if se.Context["got"] != "critical" {
		t.Errorf("expected got=critical, got %v", se.Context["got"])
	}
}

func TestValidateStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		want    string
		wantErr bool
	}{
		{"open", "open", "open", false},
		{"resolved", "resolved", "resolved", false},
		{"pending", "pending", "pending", false},
		{"snoozed", "snoozed", "snoozed", false},
		{"invalid", "closed", "", true},
		{"empty", "", "", true},
		{"uppercase", "OPEN", "open", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateStatus(tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateStatus(%q) error = %v, wantErr %v", tt.status, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("validateStatus(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestValidateStatus_StructuredError(t *testing.T) {
	_, err := validateStatus("closed")
	var se *api.StructuredError
	if !errors.As(err, &se) {
		t.Fatal("expected StructuredError")
	}
	if se.Code != api.ErrValidation {
		t.Errorf("got code %q, want validation_failed", se.Code)
	}
	if len(se.AllowedValues) != 4 {
		t.Errorf("expected 4 allowed values, got %d", len(se.AllowedValues))
	}
	if se.Context["field"] != "status" {
		t.Errorf("expected field=status, got %v", se.Context["field"])
	}
}

func TestValidateAssigneeType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty", "", "", false},
		{"me", "me", "me", false},
		{"assigned", "assigned", "assigned", false},
		{"unassigned", "unassigned", "unassigned", false},
		{"prefix m", "m", "me", false},
		{"prefix a", "a", "assigned", false},
		{"prefix u", "u", "unassigned", false},
		{"invalid", "bogus", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateAssigneeType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAssigneeType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("validateAssigneeType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{"valid lowercase", "my-slug", false},
		{"valid with numbers", "slug-123", false},
		{"valid all numbers", "123", false},
		{"empty", "", true},
		{"uppercase", "My-Slug", true},
		{"spaces", "my slug", true},
		{"underscores", "my_slug", true},
		{"special chars", "my@slug", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSlug(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSlug(%q) error = %v, wantErr %v", tt.slug, err, tt.wantErr)
			}
		})
	}
}

func TestParseSortOrder(t *testing.T) {
	tests := []struct {
		name      string
		sort      string
		order     string
		wantSort  string
		wantOrder string
		wantErr   bool
	}{
		{"empty", "", "", "", "", false},
		{"sort only", "name", "", "name", "", false},
		{"sort with asc", "name", "asc", "name", "asc", false},
		{"sort with desc", "name", "desc", "name", "desc", false},
		{"prefix desc", "-name", "", "name", "desc", false},
		{"alias sort", "la", "", "last_activity_at", "", false},
		{"alias prefix desc", "-la", "", "last_activity_at", "desc", false},
		{"mixed-case alias not rewritten", "La", "", "La", "", false},
		{"invalid order", "name", "invalid", "", "", true},
		{"prefix with order conflict", "-name", "asc", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sort, order, err := parseSortOrder(tt.sort, tt.order)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSortOrder(%q, %q) error = %v, wantErr %v", tt.sort, tt.order, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if sort != tt.wantSort {
					t.Errorf("parseSortOrder(%q, %q) sort = %q, want %q", tt.sort, tt.order, sort, tt.wantSort)
				}
				if order != tt.wantOrder {
					t.Errorf("parseSortOrder(%q, %q) order = %q, want %q", tt.sort, tt.order, order, tt.wantOrder)
				}
			}
		})
	}
}

func TestSplitCommaList(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []string
	}{
		{"empty", "", nil},
		{"single", "one", []string{"one"}},
		{"multiple", "one,two,three", []string{"one", "two", "three"}},
		{"with spaces", "one, two, three", []string{"one", "two", "three"}},
		{"trailing comma", "one,two,", []string{"one", "two"}},
		{"leading comma", ",one,two", []string{"one", "two"}},
		{"multiple commas", "one,,two", []string{"one", "two"}},
		{"only spaces", "  ,  ,  ", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitCommaList(tt.value)
			if len(result) != len(tt.expected) {
				t.Errorf("splitCommaList(%q) = %v, want %v", tt.value, result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitCommaList(%q)[%d] = %q, want %q", tt.value, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestIsInteractive_Forced(t *testing.T) {
	orig := flags.NoInput
	flags.NoInput = false
	t.Cleanup(func() { flags.NoInput = orig })

	t.Setenv("CHATWOOT_FORCE_INTERACTIVE", "true")
	if !isInteractive() {
		t.Fatal("expected isInteractive() to return true when forced via env")
	}
}

func TestIsInteractive_NoInputOverrides(t *testing.T) {
	orig := flags.NoInput
	flags.NoInput = true
	t.Cleanup(func() { flags.NoInput = orig })

	t.Setenv("CHATWOOT_FORCE_INTERACTIVE", "true")
	if isInteractive() {
		t.Fatal("expected isInteractive() to return false when --no-input is set")
	}
}

func TestRunE_CallsInnerFunction(t *testing.T) {
	called := false
	inner := func(cmd *cobra.Command, args []string) error {
		called = true
		return nil
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	wrapped := RunE(inner)
	err := wrapped(cmd, nil)

	if !called {
		t.Error("RunE wrapper did not call inner function")
	}
	if err != nil {
		t.Errorf("RunE wrapper returned error on success: %v", err)
	}
}

func TestRunE_WritesErrorToStderr(t *testing.T) {
	testErr := errors.New("test error message")
	inner := func(cmd *cobra.Command, args []string) error {
		return testErr
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	wrapped := RunE(inner)
	err := wrapped(cmd, nil)

	if err == nil {
		t.Error("RunE wrapper should return error when inner function errors")
	}

	stderrOutput := stderr.String()
	if stderrOutput == "" {
		t.Error("RunE wrapper should write error to stderr")
	}
	if !bytes.Contains([]byte(stderrOutput), []byte("test error message")) {
		t.Errorf("stderr output should contain error message, got: %s", stderrOutput)
	}
}

func TestRunE_WritesJSONErrorToStderr(t *testing.T) {
	apiErr := &api.APIError{StatusCode: 404, Body: "not found"}
	inner := func(cmd *cobra.Command, args []string) error {
		return apiErr
	}

	cmd := &cobra.Command{}

	var stdout, stderr bytes.Buffer
	ctx := outfmt.WithMode(context.Background(), outfmt.JSON)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{
		Out:    &stdout,
		ErrOut: &stderr,
		In:     nil,
	})
	cmd.SetContext(ctx)
	cmd.SetErr(&stderr)
	cmd.SetOut(&stdout)

	wrapped := RunE(inner)
	err := wrapped(cmd, nil)

	if err == nil {
		t.Fatal("RunE wrapper should return error when inner function errors")
	}

	if stdout.Len() != 0 {
		t.Errorf("JSON error should NOT appear on stdout, got: %s", stdout.String())
	}

	stderrOutput := stderr.String()
	if stderrOutput == "" {
		t.Fatal("JSON error should appear on stderr, but stderr is empty")
	}

	var parsed map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &parsed); err != nil {
		t.Fatalf("stderr should contain valid JSON, got: %s", stderrOutput)
	}
	if _, ok := parsed["code"]; !ok {
		t.Errorf("stderr JSON should contain 'code' field, got: %s", stderrOutput)
	}
	if _, ok := parsed["message"]; !ok {
		t.Errorf("stderr JSON should contain 'message' field, got: %s", stderrOutput)
	}
}

func TestRunE_ReturnsNilOnSuccess(t *testing.T) {
	inner := func(cmd *cobra.Command, args []string) error {
		return nil
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	wrapped := RunE(inner)
	err := wrapped(cmd, nil)
	if err != nil {
		t.Errorf("RunE wrapper should return nil on success, got: %v", err)
	}
}

func TestRunE_ReturnsSentinelErrorOnFailure(t *testing.T) {
	inner := func(cmd *cobra.Command, args []string) error {
		return errors.New("any error")
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	wrapped := RunE(inner)
	err := wrapped(cmd, nil)

	if err == nil {
		t.Error("RunE wrapper should return error when inner function errors")
	}
	if !errors.Is(err, errAlreadyHandled) {
		t.Errorf("RunE wrapper should return errAlreadyHandled sentinel, got: %v", err)
	}
}

func TestParseIDOrURL(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedResource string
		wantID           int
		wantErr          bool
	}{
		{"plain ID", "123", "", 123, false},
		{"plain ID with resource", "456", "conversation", 456, false},
		{"hash ID", "#123", "", 123, false},
		{"prefixed ID (conv)", "conv:123", "conversation", 123, false},
		{"prefixed ID (conversation)", "conversation:123", "conversation", 123, false},
		{"prefixed wrong resource", "contact:42", "conversation", 0, true},
		{"conversation URL", "https://app.chatwoot.com/app/accounts/1/conversations/789", "conversation", 789, false},
		{"contact URL", "https://app.chatwoot.com/app/accounts/1/contacts/42", "contact", 42, false},
		{"wrong resource type", "https://app.chatwoot.com/app/accounts/1/contacts/42", "conversation", 0, true},
		{"invalid number", "abc", "", 0, true},
		{"zero ID", "0", "", 0, true},
		{"negative ID", "-5", "", 0, true},
		{"URL without resource ID", "https://app.chatwoot.com/app/accounts/1/conversations", "", 0, true},
		{"inbox URL", "https://app.chatwoot.com/app/accounts/1/inboxes/5", "inbox", 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIDOrURL(tt.input, tt.expectedResource)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIDOrURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantID {
				t.Errorf("parseIDOrURL() = %v, want %v", got, tt.wantID)
			}
		})
	}
}

func TestResolveContactID_NumericID(t *testing.T) {
	// Plain numeric ID should return immediately without API call
	id, err := resolveContactID(context.Background(), nil, "123")
	if err != nil {
		t.Errorf("resolveContactID() unexpected error: %v", err)
	}
	if id != 123 {
		t.Errorf("resolveContactID() = %v, want 123", id)
	}
}

func TestResolveContactID_URL(t *testing.T) {
	// URL should be parsed without API call
	id, err := resolveContactID(context.Background(), nil, "https://app.chatwoot.com/app/accounts/1/contacts/456")
	if err != nil {
		t.Errorf("resolveContactID() unexpected error: %v", err)
	}
	if id != 456 {
		t.Errorf("resolveContactID() = %v, want 456", id)
	}
}

func TestResolveInboxID_NumericID(t *testing.T) {
	// Plain numeric ID should return immediately without API call
	id, err := resolveInboxID(context.Background(), nil, "5")
	if err != nil {
		t.Errorf("resolveInboxID() unexpected error: %v", err)
	}
	if id != 5 {
		t.Errorf("resolveInboxID() = %v, want 5", id)
	}
}

func TestNormalizeEnum(t *testing.T) {
	statuses := []string{"open", "resolved", "pending", "snoozed"}
	tests := []struct {
		input string
		want  string
		err   bool
	}{
		{"open", "open", false},
		{"o", "open", false},
		{"r", "resolved", false},
		{"res", "resolved", false},
		{"p", "pending", false},
		{"s", "snoozed", false},
		{"sn", "snoozed", false},
		{"x", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		got, err := normalizeEnum("status", tt.input, statuses)
		if tt.err && err == nil {
			t.Errorf("normalizeEnum(%q) expected error", tt.input)
		}
		if !tt.err && err != nil {
			t.Errorf("normalizeEnum(%q) unexpected error: %v", tt.input, err)
		}
		if !tt.err && got != tt.want {
			t.Errorf("normalizeEnum(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeEnumWithAll(t *testing.T) {
	statuses := []string{"open", "resolved", "pending", "snoozed", "all"}
	got, err := normalizeEnum("status", "a", statuses)
	if err != nil {
		t.Fatalf("normalizeEnum(%q) unexpected error: %v", "a", err)
	}
	if got != "all" {
		t.Errorf("normalizeEnum(%q) = %q, want %q", "a", got, "all")
	}
}

func TestNormalizeEnumPriority(t *testing.T) {
	priorities := []string{"urgent", "high", "medium", "low", "none"}
	tests := []struct {
		input, want string
	}{
		{"u", "urgent"},
		{"h", "high"},
		{"m", "medium"},
		{"l", "low"},
		{"n", "none"},
		{"urg", "urgent"},
	}
	for _, tt := range tests {
		got, err := normalizeEnum("priority", tt.input, priorities)
		if err != nil {
			t.Errorf("normalizeEnum(%q) unexpected error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("normalizeEnum(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeEnumAmbiguous(t *testing.T) {
	// "al" matches both "alpha" and "also"
	values := []string{"alpha", "also", "beta"}
	_, err := normalizeEnum("test", "al", values)
	if err == nil {
		t.Error("expected ambiguity error for 'al' matching alpha and also")
	}
}

func TestShortStatus(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"open", "o"},
		{"pending", "p"},
		{"resolved", "r"},
		{"snoozed", "s"},
		{"unknown", "unknown"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := shortStatus(tt.input); got != tt.want {
			t.Errorf("shortStatus(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestValidateExclusiveStatus(t *testing.T) {
	tests := []struct {
		name    string
		resolve bool
		pending bool
		snooze  string
		wantErr bool
	}{
		{"none set", false, false, "", false},
		{"resolve only", true, false, "", false},
		{"pending only", false, true, "", false},
		{"snooze only", false, false, "2h", false},
		{"resolve+pending", true, true, "", true},
		{"resolve+snooze", true, false, "2h", true},
		{"pending+snooze", false, true, "2h", true},
		{"all three", true, true, "2h", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExclusiveStatus(tt.resolve, tt.pending, tt.snooze)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateExclusiveStatus(%v, %v, %q) error = %v, wantErr %v",
					tt.resolve, tt.pending, tt.snooze, err, tt.wantErr)
			}
		})
	}
}
