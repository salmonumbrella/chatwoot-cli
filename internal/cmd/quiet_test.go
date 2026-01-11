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

func TestQuietFlagExists(t *testing.T) {
	// Verify --quiet flag exists by checking help output
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

	// The help output should show "--quiet"
	if !strings.Contains(output, "--quiet") {
		t.Error("--quiet persistent flag not found in help output")
	}

	// The help output should show "-q" shorthand
	if !strings.Contains(output, "-q") {
		t.Error("-q shorthand not found in help output")
	}

	// The help output should show "--silent"
	if !strings.Contains(output, "--silent") {
		t.Error("--silent persistent flag not found in help output")
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
