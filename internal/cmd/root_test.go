package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/validation"
	"github.com/spf13/cobra"
)

func TestExecute_Help(t *testing.T) {
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

	// Should contain key sections from the embedded help.txt
	for _, want := range []string{
		"cw - CLI for Chatwoot",
		"Aliases (resource",
		"Aliases (shortcut",
		"Reading conversations:",
		"Exit codes:",
		"Environment:",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("Help output missing %q", want)
		}
	}
}

func TestExecute_SubcommandHelpUsesCobra(t *testing.T) {
	output := captureStdout(t, func() {
		_ = Execute(context.Background(), []string{"conversations", "--help"})
	})
	if !strings.Contains(output, "Available Commands") {
		t.Error("Subcommand --help should show Cobra 'Available Commands' section")
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
		{`unknown command "foo" for "cw"`, "foo"},
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
		{`unknown shorthand flag: 'a' in -a`, "-a"},
		{`no flag here`, ""},
	}
	for _, tt := range tests {
		got := extractFlag(tt.input)
		if got != tt.want {
			t.Errorf("extractFlag(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtensionExecCandidates(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{name: "view-images", want: []string{"view-images"}},
		{name: "vi", want: []string{"vi", "view-images"}},
		{name: "unknown", want: []string{"unknown"}},
		{name: "", want: nil},
		{name: "   ", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extensionExecCandidates(tt.name)
			if len(got) != len(tt.want) {
				t.Fatalf("extensionExecCandidates(%q) len=%d, want %d (%v)", tt.name, len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("extensionExecCandidates(%q)[%d]=%q, want %q", tt.name, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestExecute_SubcommandsExist(t *testing.T) {
	// Verify essential subcommands exist by checking help output
	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"--help"}); err != nil {
			t.Fatalf("Execute() with --help failed: %v", err)
		}
	})

	// Verify essential resource aliases appear in the embedded help text
	subcommands := []string{
		"conversations",
		"contacts",
		"inboxes",
		"messages",
		"agents",
		"teams",
		"labels",
		"campaigns",
		"reports",
		"search",
		"integrations",
		"mentions",
	}

	for _, name := range subcommands {
		if !strings.Contains(output, name) {
			t.Errorf("Missing subcommand in help: %s", name)
		}
	}
}

func TestExecute_GlobalFlags(t *testing.T) {
	// Verify key flags/options appear in the embedded help text
	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"--help"}); err != nil {
			t.Fatalf("Execute() with --help failed: %v", err)
		}
	})

	// Verify flags/sections documented in the embedded help.txt
	flagSnippets := []string{
		"--jq",
		"--fields",
		"--template",
		"--items-only",
		"--dry-run",
		"--yes",
		"--cj",
		"--help-json",
		"-o agent",
		"-o json",
		"-Q",
	}

	for _, snippet := range flagSnippets {
		if !strings.Contains(output, snippet) {
			t.Errorf("Missing flag/option in help: %s", snippet)
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

func TestExecute_HelpHidesLongAliasFlags(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"--help"}); err != nil {
			t.Fatalf("Execute() with --help failed: %v", err)
		}
	})

	hiddenAliases := []string{"out", "qr", "qf", "ro", "j"}
	for _, alias := range hiddenAliases {
		pattern := regexp.MustCompile(`(^|\s)--` + alias + `(\s|,|=|$)`)
		if pattern.MatchString(output) {
			t.Errorf("hidden alias --%s should not appear in help output", alias)
		}
	}
}

func TestExecute_GlobalLongAliasFlagsWork(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"status", "--j"}); err != nil {
			t.Fatalf("status --j failed: %v", err)
		}
	})
	var jsonPayload map[string]any
	if err := json.Unmarshal([]byte(output), &jsonPayload); err != nil {
		t.Fatalf("status --j did not produce JSON output: %v\noutput: %q", err, output)
	}

	output = captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"status", "--out", "json"}); err != nil {
			t.Fatalf("status --out json failed: %v", err)
		}
	})
	if err := json.Unmarshal([]byte(output), &jsonPayload); err != nil {
		t.Fatalf("status --out json did not produce JSON output: %v\noutput: %q", err, output)
	}

	output = captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"status", "--out", "json", "--qr", ".authenticated"}); err != nil {
			t.Fatalf("status --qr failed: %v", err)
		}
	})
	got := strings.TrimSpace(output)
	if got != "true" && got != "false" {
		t.Fatalf("status --qr expected boolean output, got %q", output)
	}
}

func TestExecute_QueryFileFlagsWork(t *testing.T) {
	queryFile := filepath.Join(t.TempDir(), "query.jq")
	if err := os.WriteFile(queryFile, []byte(".items | length"), 0o600); err != nil {
		t.Fatalf("failed to write query file: %v", err)
	}

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"schema", "list", "--query-file", queryFile}); err != nil {
			t.Fatalf("schema list --query-file failed: %v", err)
		}
	})

	if !regexp.MustCompile(`^\s*\d+\s*$`).MatchString(output) {
		t.Fatalf("schema list --query-file expected numeric output, got %q", output)
	}

	output = captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"schema", "list", "--qf", queryFile}); err != nil {
			t.Fatalf("schema list --qf failed: %v", err)
		}
	})
	if !regexp.MustCompile(`^\s*\d+\s*$`).MatchString(output) {
		t.Fatalf("schema list --qf expected numeric output, got %q", output)
	}
}

func TestExecute_QueryFileConflictsWithQuery(t *testing.T) {
	queryFile := filepath.Join(t.TempDir(), "query.jq")
	if err := os.WriteFile(queryFile, []byte(".items"), 0o600); err != nil {
		t.Fatalf("failed to write query file: %v", err)
	}

	err := Execute(context.Background(), []string{"schema", "list", "--query-file", queryFile, "--query", ".items"})
	if err == nil {
		t.Fatal("expected --query-file with --query to fail")
	}
	if !strings.Contains(err.Error(), "--query-file cannot be used with --query or --jq") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_QueryAliasesAreRewritten(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"schema", "list", "--output", "json", "--query", ".it | type"}); err != nil {
			t.Fatalf("schema list --query with aliases failed: %v", err)
		}
	})

	got := strings.TrimSpace(output)
	if got != `"array"` {
		t.Fatalf("expected alias-rewritten query result \"array\", got %q", got)
	}
}

func TestExecute_QueryAliases_DoNotRewriteQuotedBracketKeys(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"schema", "list", "--output", "json", "--query", `.["it"] | type`}); err != nil {
			t.Fatalf("schema list quoted bracket query failed: %v", err)
		}
	})

	got := strings.TrimSpace(output)
	if got != `"null"` {
		t.Fatalf("expected quoted key to remain literal (null), got %q", got)
	}
}

func TestExecute_QueryAliases_DoNotRewriteMixedCaseTokens(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"schema", "list", "--output", "json", "--query", ".It | type"}); err != nil {
			t.Fatalf("schema list mixed-case query failed: %v", err)
		}
	})

	got := strings.TrimSpace(output)
	if got != `"null"` {
		t.Fatalf("expected mixed-case token to remain unchanged (null), got %q", got)
	}
}

func TestExecute_FieldAliasesAreRewritten(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"schema", "list", "--output", "json", "--fields", "it"}); err != nil {
			t.Fatalf("schema list --fields alias failed: %v", err)
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v\noutput: %q", err, output)
	}
	items, ok := payload["items"].([]any)
	if !ok {
		t.Fatalf("expected items key in output, got: %v", payload)
	}
	if len(items) == 0 {
		t.Fatalf("expected non-empty items array in output, got: %v", payload)
	}
}

func TestExecute_QueryAliases_MessageTypeAndSenderID(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"schema", "show", "message", "--output", "json", "--query", ".properties.mty.type"}); err != nil {
			t.Fatalf("schema show message --query mty failed: %v", err)
		}
	})
	if strings.TrimSpace(output) != `"string"` {
		t.Fatalf("expected mty alias to resolve to message_type schema type string, got %q", output)
	}

	output = captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"schema", "show", "message", "--output", "json", "--query", ".properties.sdi.type"}); err != nil {
			t.Fatalf("schema show message --query sdi failed: %v", err)
		}
	})
	if strings.TrimSpace(output) != `"integer"` {
		t.Fatalf("expected sdi alias to resolve to sender_id schema type integer, got %q", output)
	}
}

func TestExecute_QueryFunctionAliases(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"schema", "list", "--output", "json", "--query", `[.it[] | sl(.n | ts("con"; "i"))] | length`}); err != nil {
			t.Fatalf("schema list function aliases failed: %v", err)
		}
	})

	if !regexp.MustCompile(`^\s*\d+\s*$`).MatchString(output) {
		t.Fatalf("expected numeric output for function-alias query, got %q", output)
	}
}

func TestExecute_ItemsOnlyAndResultsOnlyFlags(t *testing.T) {
	itemsOutput := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"schema", "list", "--items-only"}); err != nil {
			t.Fatalf("schema list --items-only failed: %v", err)
		}
	})

	var items []map[string]any
	if err := json.Unmarshal([]byte(itemsOutput), &items); err != nil {
		t.Fatalf("--items-only expected JSON array, got error: %v\noutput: %q", err, itemsOutput)
	}
	if len(items) == 0 {
		t.Fatalf("--items-only returned empty array")
	}

	resultsOutput := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"schema", "list", "--results-only"}); err != nil {
			t.Fatalf("schema list --results-only failed: %v", err)
		}
	})

	var results []map[string]any
	if err := json.Unmarshal([]byte(resultsOutput), &results); err != nil {
		t.Fatalf("--results-only expected JSON array, got error: %v\noutput: %q", err, resultsOutput)
	}
	if len(results) == 0 {
		t.Fatalf("--results-only returned empty array")
	}
}

func TestExecute_ItemsOnlyConflictsWithOutputText(t *testing.T) {
	err := Execute(context.Background(), []string{"schema", "list", "--items-only", "--output", "text"})
	if err == nil {
		t.Fatal("expected --items-only with --output text to fail")
	}
	if !strings.Contains(err.Error(), "--items-only/--results-only") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_ItemsOnlyHiddenAliasesWork(t *testing.T) {
	for _, alias := range []string{"--io", "--ro"} {
		output := captureStdout(t, func() {
			if err := Execute(context.Background(), []string{"schema", "list", alias}); err != nil {
				t.Fatalf("schema list %s failed: %v", alias, err)
			}
		})

		var items []map[string]any
		if err := json.Unmarshal([]byte(output), &items); err != nil {
			t.Fatalf("%s expected JSON array, got error: %v\noutput: %q", alias, err, output)
		}
		if len(items) == 0 {
			t.Fatalf("%s returned empty array", alias)
		}
	}
}

func TestExecute_OutputNDJSONAccepted(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"status", "--output", "ndjson", "--query", ".authenticated"}); err != nil {
			t.Fatalf("status --output ndjson --query failed: %v", err)
		}
	})

	got := strings.TrimSpace(output)
	if got != "true" && got != "false" {
		t.Fatalf("expected boolean output from ndjson+query, got %q", output)
	}
}

func TestExecute_JSONConflictsWithOutAlias(t *testing.T) {
	err := Execute(context.Background(), []string{"status", "--json", "--out", "text"})
	if err == nil {
		t.Fatal("expected conflict error when using --json with --out text")
	}
	if !strings.Contains(err.Error(), "--json conflicts with --output text") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandAliases(t *testing.T) {
	tests := []struct {
		alias   string
		wantCmd string
	}{
		{"customers", "contacts"},
		{"conv", "conversations"},
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
		{"dh", "dashboard"},
		{"v", "version"},
	}

	// Build a root command with all subcommands
	ctx := context.Background()
	// We need to construct the root command the same way Execute does.
	// Use Execute internals by resolving against a fresh root.
	root := &cobra.Command{Use: "cw"}
	root.AddCommand(newSearchCmd())
	root.AddCommand(newAssignCmd())
	root.AddCommand(newReplyCmd())
	root.AddCommand(newSnoozeCmd())
	root.AddCommand(newHandoffCmd())
	root.AddCommand(newDashboardCmd())
	root.AddCommand(newContactsCmd())
	root.AddCommand(newConversationsCmd())
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

func TestExecute_AllowPrivateDoesNotLeakAcrossRuns(t *testing.T) {
	t.Setenv("CHATWOOT_ALLOW_PRIVATE", "0")
	validation.SetAllowPrivate(false)
	t.Cleanup(func() { validation.SetAllowPrivate(false) })

	if validation.AllowPrivateEnabled() {
		t.Fatalf("expected allow-private to start disabled")
	}

	if err := Execute(context.Background(), []string{"version", "--allow-private"}); err != nil {
		t.Fatalf("first execute failed: %v", err)
	}
	if !validation.AllowPrivateEnabled() {
		t.Fatalf("expected allow-private to be enabled after --allow-private")
	}

	if err := Execute(context.Background(), []string{"version"}); err != nil {
		t.Fatalf("second execute failed: %v", err)
	}
	if validation.AllowPrivateEnabled() {
		t.Fatalf("expected allow-private to reset for execute without flag")
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

func TestLoadOpenClawEnv_SetsUnsetVars(t *testing.T) {
	// Create a temporary directory to act as $HOME
	tmpHome := t.TempDir()
	openclawDir := filepath.Join(tmpHome, ".openclaw")
	if err := os.MkdirAll(openclawDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	envContent := "CHATWOOT_BASE_URL=https://openclaw.example.com\nCHATWOOT_API_TOKEN=oc-token\n"
	if err := os.WriteFile(filepath.Join(openclawDir, ".env"), []byte(envContent), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	// Point HOME so loadOpenClawEnv finds the file
	t.Setenv("HOME", tmpHome)

	// Ensure vars are truly unset before loading. t.Setenv registers cleanup
	// so the original value is restored after the test.
	t.Setenv("CHATWOOT_BASE_URL", "")
	t.Setenv("CHATWOOT_API_TOKEN", "")
	_ = os.Unsetenv("CHATWOOT_BASE_URL")
	_ = os.Unsetenv("CHATWOOT_API_TOKEN")

	loadOpenClawEnv()

	if got := os.Getenv("CHATWOOT_BASE_URL"); got != "https://openclaw.example.com" {
		t.Fatalf("CHATWOOT_BASE_URL = %q, want %q", got, "https://openclaw.example.com")
	}
	if got := os.Getenv("CHATWOOT_API_TOKEN"); got != "oc-token" {
		t.Fatalf("CHATWOOT_API_TOKEN = %q, want %q", got, "oc-token")
	}
}

func TestLoadOpenClawEnv_DoesNotOverwriteExisting(t *testing.T) {
	tmpHome := t.TempDir()
	openclawDir := filepath.Join(tmpHome, ".openclaw")
	if err := os.MkdirAll(openclawDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	envContent := "CHATWOOT_BASE_URL=https://openclaw.example.com\n"
	if err := os.WriteFile(filepath.Join(openclawDir, ".env"), []byte(envContent), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("HOME", tmpHome)
	t.Setenv("CHATWOOT_BASE_URL", "https://already-set.example.com")

	loadOpenClawEnv()

	if got := os.Getenv("CHATWOOT_BASE_URL"); got != "https://already-set.example.com" {
		t.Fatalf("CHATWOOT_BASE_URL = %q, want %q (should not be overwritten)", got, "https://already-set.example.com")
	}
}

func TestLoadOpenClawEnv_NoFileIsNoop(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Should not panic or error when ~/.openclaw/.env doesn't exist
	loadOpenClawEnv()
}
