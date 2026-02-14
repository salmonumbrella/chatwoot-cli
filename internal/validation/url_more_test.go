package validation

import "testing"

func TestAllowPrivateEnabled(t *testing.T) {
	original := AllowPrivateEnabled()
	SetAllowPrivate(false)
	if AllowPrivateEnabled() {
		t.Fatal("expected AllowPrivateEnabled false")
	}
	SetAllowPrivate(true)
	if !AllowPrivateEnabled() {
		t.Fatal("expected AllowPrivateEnabled true")
	}
	SetAllowPrivate(original)
}
