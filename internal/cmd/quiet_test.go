package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestQuietFlagExists(t *testing.T) {
	// Verify quiet-related flags appear in the embedded help text
	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"--help"}); err != nil {
			t.Fatalf("Execute() with --help failed: %v", err)
		}
	})

	// The embedded help text documents -Q as the quiet shorthand
	if !strings.Contains(output, "-Q") {
		t.Error("-Q quiet shorthand not found in help output")
	}

	// Verify the quiet flag actually works (functional test)
	quietOutput := captureStdout(t, func() {
		_ = Execute(context.Background(), []string{"version", "--quiet"})
	})
	if quietOutput != "" {
		t.Errorf("--quiet should suppress text output, got %q", quietOutput)
	}
}

func TestIsQuiet(t *testing.T) {
	// Save original flags
	origFlags := flags

	// Reset flags
	flags = rootFlags{}

	if isQuiet(nil) {
		t.Error("should not be quiet by default")
	}

	flags.Quiet = true
	if !isQuiet(nil) {
		t.Error("should be quiet when flag is set")
	}

	// Restore original flags
	flags = origFlags
}

func TestPrintIfNotQuiet(t *testing.T) {
	// Save original flags
	origFlags := flags

	t.Run("prints when not quiet", func(t *testing.T) {
		flags = rootFlags{Quiet: false}
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		output := captureStdout(t, func() {
			printIfNotQuiet(cmd, "Hello %s\n", "world")
		})

		if output != "Hello world\n" {
			t.Errorf("expected 'Hello world\\n', got %q", output)
		}
	})

	t.Run("suppresses when quiet", func(t *testing.T) {
		flags = rootFlags{Quiet: true}
		cmd := &cobra.Command{}
		cmd.SetContext(context.Background())

		output := captureStdout(t, func() {
			printIfNotQuiet(cmd, "Hello %s\n", "world")
		})

		if output != "" {
			t.Errorf("expected empty output when quiet, got %q", output)
		}
	})

	// Restore original flags
	flags = origFlags
}

func TestFlagShorthands(t *testing.T) {
	// Verify -Q and -q shorthands work functionally
	// (the root help is now static embedded text, so we test actual behaviour)

	// -Q should suppress output (quiet)
	quietOutput := captureStdout(t, func() {
		_ = Execute(context.Background(), []string{"version", "-Q"})
	})
	if quietOutput != "" {
		t.Errorf("-Q should suppress text output, got %q", quietOutput)
	}

	// -q should work as --query (e.g., on schema list)
	queryOutput := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"schema", "list", "-q", ".items | length"}); err != nil {
			t.Fatalf("-q as --query failed: %v", err)
		}
	})
	queryOutput = strings.TrimSpace(queryOutput)
	if queryOutput == "" {
		t.Error("-q as --query returned empty output")
	}
}
