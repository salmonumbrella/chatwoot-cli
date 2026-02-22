package resolve_test

import (
	"strings"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/resolve"
)

func TestAmbiguousErrorString(t *testing.T) {
	err := &resolve.AmbiguousError{
		Query: "support",
		Matches: []resolve.Match{
			{ID: 1, Name: "Support US"},
			{ID: 2, Name: "Support EU"},
		},
	}

	msg := err.Error()
	if !strings.Contains(msg, `ambiguous match for "support"`) {
		t.Fatalf("missing query in error message: %q", msg)
	}
	if !strings.Contains(msg, "1: Support US") || !strings.Contains(msg, "2: Support EU") {
		t.Fatalf("missing candidates in error message: %q", msg)
	}
}
