package cmd

import (
	"context"
	"testing"
)

func TestHandoffCommandExists(t *testing.T) {
	err := Execute(context.Background(), []string{"handoff", "--help"})
	if err != nil {
		t.Fatalf("handoff --help failed: %v", err)
	}
}

func TestHandoffRequiresAssignment(t *testing.T) {
	err := Execute(context.Background(), []string{"handoff", "123", "--reason", "test"})
	if err == nil {
		t.Fatal("expected error when no --agent or --team specified")
	}
}
