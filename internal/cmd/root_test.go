package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestExecute_Help(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := Execute(ctx, []string{"--help"})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("Execute() with --help failed: %v", err)
	}

	if len(output) == 0 {
		t.Error("Help output is empty")
	}

	// Should contain key sections
	if !strings.Contains(output, "Available Commands") {
		t.Error("Help output missing 'Available Commands'")
	}

	if !strings.Contains(output, "chatwoot") {
		t.Error("Help output missing 'chatwoot' command name")
	}
}

func TestExecute_Version(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := Execute(ctx, []string{"version"})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err != nil {
		t.Errorf("Execute() with 'version' failed: %v", err)
	}
}

func TestExecute_QuietSuppressesTextOutput(t *testing.T) {
	output := captureStdout(t, func() {
		_ = Execute(context.Background(), []string{"version", "--quiet"})
	})
	if output != "" {
		t.Fatalf("expected no stdout with --quiet, got %q", output)
	}
}

func TestExecute_InvalidCommand(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ctx := context.Background()
	err := Execute(ctx, []string{"nonexistent-command"})

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err == nil {
		t.Error("Execute() with invalid command should return error")
	}
}

func TestExecute_UnknownCommand_DidYouMean(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ctx := context.Background()
	_ = Execute(ctx, []string{"conversatins"})

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Did you mean") {
		t.Errorf("expected 'Did you mean' suggestion in stderr, got: %s", output)
	}
	if !strings.Contains(output, "conversations") {
		t.Errorf("expected 'conversations' suggestion in stderr, got: %s", output)
	}
}

func TestExtractQuoted(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`unknown command "foo" for "chatwoot"`, "foo"},
		{`no quotes here`, ""},
		{`only "one quote`, ""},
		{`"hello"`, "hello"},
	}
	for _, tt := range tests {
		got := extractQuoted(tt.input)
		if got != tt.want {
			t.Errorf("extractQuoted(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractFlag(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`unknown flag: --staus`, "--staus"},
		{`flag provided but not defined: --pririty`, "--pririty"},
		{`no flag here`, ""},
	}
	for _, tt := range tests {
		got := extractFlag(tt.input)
		if got != tt.want {
			t.Errorf("extractFlag(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExecute_SubcommandsExist(t *testing.T) {
	// Verify essential subcommands exist by checking help output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := Execute(ctx, []string{"--help"})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Execute() with --help failed: %v", err)
	}

	// Verify essential subcommands exist in help output
	subcommands := []string{
		"auth",
		"conversations",
		"contacts",
		"inboxes",
		"messages",
		"agents",
		"teams",
		"labels",
		"webhooks",
		"version",
		"config",
		"campaigns",
		"reports",
	}

	for _, name := range subcommands {
		if !strings.Contains(output, name) {
			t.Errorf("Missing subcommand in help: %s", name)
		}
	}
}

func TestExecute_GlobalFlags(t *testing.T) {
	// Verify global flags exist by checking help output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := Execute(ctx, []string{"--help"})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Execute() with --help failed: %v", err)
	}

	// Verify global flags exist in help output
	flags := []string{
		"--output",
		"--json",
		"--debug",
		"--color",
		"--dry-run",
		"--allow-private",
		"--query",
		"--fields",
		"--no-input",
		"--template",
		"--utc",
		"--time-zone",
		"--max-rate-limit-retries",
		"--max-5xx-retries",
		"--rate-limit-delay",
		"--server-error-delay",
		"--circuit-breaker-threshold",
		"--circuit-breaker-reset-time",
	}

	for _, flagName := range flags {
		if !strings.Contains(output, flagName) {
			t.Errorf("Missing global flag in help: %s", flagName)
		}
	}
}

func TestExecute_OutputFlagShorthand(t *testing.T) {
	// Verify -o shorthand for --output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := Execute(ctx, []string{"--help"})

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Execute() with --help failed: %v", err)
	}

	// The help output should show "-o, --output"
	if !strings.Contains(output, "-o") {
		t.Error("Missing -o shorthand for --output flag")
	}
}

func TestCommandAliases(t *testing.T) {
	tests := []struct {
		alias   string
		wantCmd string
	}{
		{"find", "search"},
		{"s", "search"},
		{"reassign", "assign"},
		{"respond", "reply"},
		{"r", "reply"},
		{"pause", "snooze"},
		{"defer", "snooze"},
		{"escalate", "handoff"},
		{"transfer", "handoff"},
		{"dash", "dashboard"},
		{"db", "dashboard"},
		{"v", "version"},
	}

	// Build a root command with all subcommands
	ctx := context.Background()
	// We need to construct the root command the same way Execute does.
	// Use Execute internals by resolving against a fresh root.
	root := &cobra.Command{Use: "chatwoot"}
	root.AddCommand(newSearchCmd())
	root.AddCommand(newAssignCmd())
	root.AddCommand(newReplyCmd())
	root.AddCommand(newSnoozeCmd())
	root.AddCommand(newHandoffCmd())
	root.AddCommand(newDashboardCmd())
	root.AddCommand(newVersionCmd())
	root.SetContext(ctx)

	for _, tt := range tests {
		t.Run(tt.alias+"->"+tt.wantCmd, func(t *testing.T) {
			cmd, _, err := root.Find([]string{tt.alias})
			if err != nil {
				t.Fatalf("Find(%q) error: %v", tt.alias, err)
			}
			if cmd.Name() != tt.wantCmd {
				t.Errorf("Find(%q) resolved to %q, want %q", tt.alias, cmd.Name(), tt.wantCmd)
			}
		})
	}
}

func TestDefaultResolveNamesFromEnv(t *testing.T) {
	t.Setenv("CHATWOOT_RESOLVE_NAMES", "1")
	if !defaultResolveNames() {
		t.Fatalf("expected resolve-names default true when CHATWOOT_RESOLVE_NAMES=1")
	}

	t.Setenv("CHATWOOT_RESOLVE_NAMES", "0")
	if defaultResolveNames() {
		t.Fatalf("expected resolve-names default false when CHATWOOT_RESOLVE_NAMES=0")
	}
}

func TestExecute_InvalidOutputFormat(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ctx := context.Background()
	err := Execute(ctx, []string{"--output", "invalid", "version"})

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err == nil {
		t.Error("Execute() with invalid output format should return error")
	}
}

func TestExecute_TimeZoneConflict(t *testing.T) {
	ctx := context.Background()
	err := Execute(ctx, []string{"--utc", "--time-zone", "UTC", "version"})
	if err == nil {
		t.Error("expected error when --utc and --time-zone are both set")
	}
}

func TestExecute_InvalidTimeZone(t *testing.T) {
	ctx := context.Background()
	err := Execute(ctx, []string{"--time-zone", "Not/AZone", "version"})
	if err == nil {
		t.Error("expected error for invalid --time-zone")
	}
}

func TestParseFields(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{"single field", "id", []string{"id"}, false},
		{"multiple fields", "id,name,email", []string{"id", "name", "email"}, false},
		{"with spaces", "id, name, email", []string{"id", "name", "email"}, false},
		{"nested field", "contact.id", []string{"contact.id"}, false},
		{"empty", "", nil, true},
		{"only commas", ",,,", nil, true},
		{"only spaces", "  ,  ,  ", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFields(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFields(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("parseFields(%q) = %v, want %v", tt.input, got, tt.want)
					return
				}
				for i, v := range got {
					if v != tt.want[i] {
						t.Errorf("parseFields(%q)[%d] = %q, want %q", tt.input, i, v, tt.want[i])
					}
				}
			}
		})
	}
}

func TestBuildFieldsQuery(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		want   string
	}{
		{
			"single field",
			[]string{"id"},
			`if type=="array" then map({"id": .["id"]}) else {"id": .["id"]} end`,
		},
		{
			"multiple fields",
			[]string{"id", "name"},
			`if type=="array" then map({"id": .["id"], "name": .["name"]}) else {"id": .["id"], "name": .["name"]} end`,
		},
		{
			"nested field",
			[]string{"contact.id"},
			`if type=="array" then map({"contact.id": .["contact"]["id"]}) else {"contact.id": .["contact"]["id"]} end`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFieldsQuery(tt.fields)
			if got != tt.want {
				t.Errorf("buildFieldsQuery(%v) = %q, want %q", tt.fields, got, tt.want)
			}
		})
	}
}

func TestParseFieldsWithPresets(t *testing.T) {
	cmd := &cobra.Command{}
	registerFieldPresets(cmd, map[string][]string{
		"minimal": {"id", "name"},
	})

	fields, err := parseFieldsWithPresets(cmd, "minimal")
	if err != nil {
		t.Fatalf("parseFieldsWithPresets returned error: %v", err)
	}
	if len(fields) != 2 || fields[0] != "id" || fields[1] != "name" {
		t.Fatalf("unexpected fields: %v", fields)
	}

	fields, err = parseFieldsWithPresets(cmd, "id,email")
	if err != nil {
		t.Fatalf("parseFieldsWithPresets returned error: %v", err)
	}
	if len(fields) != 2 || fields[0] != "id" || fields[1] != "email" {
		t.Fatalf("unexpected fields: %v", fields)
	}
}

func TestJqKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"simple", "id", `"id"`},
		{"with dot", "contact.id", `"contact.id"`},
		{"with quote", `foo"bar`, `"foo\"bar"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jqKey(tt.key)
			if got != tt.want {
				t.Errorf("jqKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestJqPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"simple", "id", `.["id"]`},
		{"nested", "contact.id", `.["contact"]["id"]`},
		{"deeply nested", "a.b.c", `.["a"]["b"]["c"]`},
		{"empty", "", "."},
		{"with quote", `foo"bar`, `.["foo\"bar"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jqPath(tt.path)
			if got != tt.want {
				t.Errorf("jqPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestLoadTemplate(t *testing.T) {
	// Test inline template
	t.Run("inline template", func(t *testing.T) {
		got, err := loadTemplate("{{.Name}}")
		if err != nil {
			t.Errorf("loadTemplate() error = %v", err)
			return
		}
		if got != "{{.Name}}" {
			t.Errorf("loadTemplate() = %q, want %q", got, "{{.Name}}")
		}
	})

	// Test file template with non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		_, err := loadTemplate("@/nonexistent/file.tmpl")
		if err == nil {
			t.Error("loadTemplate() with non-existent file should return error")
		}
	})
}

func TestExecute_FieldsAndQueryConflict(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ctx := context.Background()
	err := Execute(ctx, []string{"--fields", "id", "--query", ".id", "version"})

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if err == nil {
		t.Error("Execute() with both --fields and --query should return error")
	}

	if !strings.Contains(err.Error(), "cannot be used together") {
		t.Errorf("Expected conflict error message, got: %v", err)
	}
}

func TestParseFields_FromStdin(t *testing.T) {
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	_, _ = w.WriteString("id\ncontact.id\n")
	_ = w.Close()

	fields, err := parseFields("@-")
	if err != nil {
		t.Fatalf("parseFields(@-) error: %v", err)
	}
	if len(fields) != 2 || fields[0] != "id" || fields[1] != "contact.id" {
		t.Fatalf("unexpected fields: %v", fields)
	}
}

func TestParseFields_JSONArray(t *testing.T) {
	fields, err := parseFields(`["id","contact.id"]`)
	if err != nil {
		t.Fatalf("parseFields(JSON) error: %v", err)
	}
	if len(fields) != 2 || fields[0] != "id" || fields[1] != "contact.id" {
		t.Fatalf("unexpected fields: %v", fields)
	}
}
