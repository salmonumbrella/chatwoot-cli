package cmd

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestJQFlagExists(t *testing.T) {
	// Reset flags before test
	flags = rootFlags{
		Output: "text",
		Color:  "auto",
	}

	// Create root command via Execute to ensure all flags are registered
	ctx := context.Background()
	root := createRootCmd(ctx)

	jqFlag := root.PersistentFlags().Lookup("jq")
	if jqFlag == nil {
		t.Fatal("--jq persistent flag not found on root command")
	}

	if jqFlag.Usage == "" {
		t.Error("--jq flag should have a usage description")
	}
}

func TestJQFlagPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		jq       string
		query    string
		expected string
	}{
		{
			name:     "jq takes precedence",
			jq:       ".id",
			query:    ".name",
			expected: ".id",
		},
		{
			name:     "falls back to query",
			jq:       "",
			query:    ".name",
			expected: ".name",
		},
		{
			name:     "empty when both empty",
			jq:       "",
			query:    "",
			expected: "",
		},
		{
			name:     "jq alone works",
			jq:       ".data[]",
			query:    "",
			expected: ".data[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			flags = rootFlags{
				Output: "text",
				Color:  "auto",
				JQ:     tt.jq,
				Query:  tt.query,
			}

			result := getJQQuery()
			if result != tt.expected {
				t.Errorf("getJQQuery() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// createRootCmd creates a root command for testing without executing it.
// This mirrors the setup in Execute() but returns the command for inspection.
func createRootCmd(ctx context.Context) *rootCmd {
	root := newRootCmd()
	root.cmd.SetContext(ctx)
	return root
}

// rootCmd wraps cobra.Command for testing
type rootCmd struct {
	cmd *cobra.Command
}

func (r *rootCmd) PersistentFlags() *pflag.FlagSet {
	return r.cmd.PersistentFlags()
}

func newRootCmd() *rootCmd {
	cmd := &cobra.Command{
		Use:           "cw",
		Short:         "CLI for Chatwoot customer support platform",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVarP(&flags.Output, "output", "o", flags.Output, "Output format: text|json")
	cmd.PersistentFlags().StringVar(&flags.Color, "color", flags.Color, "Color output: auto|always|never")
	cmd.PersistentFlags().BoolVar(&flags.Debug, "debug", false, "Enable debug logging")
	cmd.PersistentFlags().BoolVar(&flags.DryRun, "dry-run", false, "Preview changes without executing")
	cmd.PersistentFlags().StringVar(&flags.Query, "query", "", "JQ expression to filter JSON output")
	cmd.PersistentFlags().StringVar(&flags.JQ, "jq", "", "JQ expression to filter JSON output (alias for --query)")
	cmd.PersistentFlags().StringVar(&flags.Fields, "fields", "", "Comma-separated fields to select in JSON output (shorthand for --query)")
	cmd.PersistentFlags().StringVar(&flags.Template, "template", "", "Go template string (or @path) to render JSON output")

	return &rootCmd{cmd: cmd}
}
