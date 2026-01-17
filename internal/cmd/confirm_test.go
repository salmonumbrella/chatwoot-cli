package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/iocontext"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func TestConfirmAction_RequiresForceForJSON(t *testing.T) {
	cmd := &cobra.Command{}
	ctx := outfmt.WithMode(context.Background(), outfmt.JSON)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{
		In:     bytes.NewBufferString("y\n"),
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	})
	cmd.SetContext(ctx)

	_, err := confirmAction(cmd, confirmOptions{
		Prompt:              "Confirm? (y/N): ",
		Expected:            "y",
		RequireForceForJSON: true,
	})
	if err == nil {
		t.Fatal("expected error when JSON output without --force")
	}
}

func TestConfirmAction_AcceptsExpectedInput(t *testing.T) {
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	ctx := outfmt.WithMode(context.Background(), outfmt.Text)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{
		In:     bytes.NewBufferString("merge\n"),
		Out:    out,
		ErrOut: &bytes.Buffer{},
	})
	cmd.SetContext(ctx)

	ok, err := confirmAction(cmd, confirmOptions{
		Prompt:        "Type 'merge' to confirm: ",
		Expected:      "merge",
		CancelMessage: "Merge cancelled.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected confirmation to succeed")
	}
}

func TestRequireForceForJSON(t *testing.T) {
	cmd := &cobra.Command{}
	ctx := outfmt.WithMode(context.Background(), outfmt.JSON)
	cmd.SetContext(ctx)

	if err := requireForceForJSON(cmd, false); err == nil {
		t.Fatal("expected error when JSON output without --force")
	}
	if err := requireForceForJSON(cmd, true); err != nil {
		t.Fatalf("unexpected error when --force is set: %v", err)
	}
}
