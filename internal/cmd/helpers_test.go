package cmd

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

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
	// printJSON writes to os.Stdout directly, not to cmd's output buffer.
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
	w := newTabWriter()
	if w == nil {
		t.Error("newTabWriter() returned nil")
	}
}

func TestValidatePriority(t *testing.T) {
	tests := []struct {
		name     string
		priority string
		wantErr  bool
	}{
		{"urgent", "urgent", false},
		{"high", "high", false},
		{"medium", "medium", false},
		{"low", "low", false},
		{"none", "none", false},
		{"invalid", "invalid", true},
		{"empty", "", true},
		{"uppercase", "HIGH", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePriority(tt.priority)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePriority(%q) error = %v, wantErr %v", tt.priority, err, tt.wantErr)
			}
		})
	}
}

func TestValidateStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		wantErr bool
	}{
		{"open", "open", false},
		{"resolved", "resolved", false},
		{"pending", "pending", false},
		{"snoozed", "snoozed", false},
		{"invalid", "closed", true},
		{"empty", "", true},
		{"uppercase", "OPEN", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStatus(tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateStatus(%q) error = %v, wantErr %v", tt.status, err, tt.wantErr)
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

func TestIsInteractive(t *testing.T) {
	// isInteractive checks os.Stdin, which in test environment is typically not a terminal
	// Just verify it returns a boolean without panicking
	result := isInteractive()
	// In test environment, stdin is usually not a terminal
	if result {
		t.Log("isInteractive() returned true (running in terminal)")
	} else {
		t.Log("isInteractive() returned false (not a terminal)")
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
