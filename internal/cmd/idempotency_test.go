package cmd

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestNewIdempotencyKey(t *testing.T) {
	k := newIdempotencyKey()
	if !strings.HasPrefix(k, "cwcli_") {
		t.Fatalf("idempotency key prefix mismatch: %q", k)
	}
	payload := strings.TrimPrefix(k, "cwcli_")
	if len(payload) == 32 {
		if _, err := hex.DecodeString(payload); err != nil {
			t.Fatalf("expected hex payload, got %q (%v)", payload, err)
		}
	}
	if payload == "" {
		t.Fatal("expected non-empty idempotency suffix")
	}
}
