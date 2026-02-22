package cmd

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestHandledErrorExitCode(t *testing.T) {
	e := &handledError{err: errors.New("boom"), exitCode: 7}
	if got := e.ExitCode(); got != 7 {
		t.Fatalf("ExitCode() = %d, want 7", got)
	}
}

func TestFormatTimestampWithZoneAndDate(t *testing.T) {
	orig := timeLocation
	defer setTimeLocation(orig)

	loc := time.FixedZone("PST", -8*60*60)
	setTimeLocation(loc)
	input := time.Date(2026, 2, 14, 16, 30, 0, 0, time.UTC)

	withZone := formatTimestampWithZone(input)
	if !strings.Contains(withZone, "PST") {
		t.Fatalf("expected zone in formatted timestamp, got %q", withZone)
	}

	if got := formatDate(input); got == "" || strings.Contains(got, " ") {
		t.Fatalf("unexpected formatted date: %q", got)
	}
}
