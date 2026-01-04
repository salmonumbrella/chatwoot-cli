package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
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
		"--debug",
		"--color",
		"--dry-run",
		"--query",
		"--fields",
		"--template",
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
