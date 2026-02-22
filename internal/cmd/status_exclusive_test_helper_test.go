package cmd

import (
	"context"
	"strings"
	"testing"
)

func assertStatusFlagsMutuallyExclusive(t *testing.T, args []string) {
	t.Helper()

	err := Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for --resolve --pending, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected error containing 'mutually exclusive', got: %v", err)
	}
}
