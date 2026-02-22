package api

import (
	"net/http"
	"testing"
	"time"
)

func TestRetryAfterDurationSeconds(t *testing.T) {
	header := http.Header{}
	header.Set("Retry-After", "5")

	d, ok := retryAfterDuration(header)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if d != 5*time.Second {
		t.Fatalf("expected 5s, got %v", d)
	}
}

func TestRetryAfterDurationNegativeSeconds(t *testing.T) {
	header := http.Header{}
	header.Set("Retry-After", "-3")

	d, ok := retryAfterDuration(header)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if d != 0 {
		t.Fatalf("expected 0s, got %v", d)
	}
}

func TestRetryAfterDurationHTTPDate(t *testing.T) {
	header := http.Header{}
	future := time.Now().Add(2 * time.Second).UTC()
	header.Set("Retry-After", future.Format(http.TimeFormat))

	d, ok := retryAfterDuration(header)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if d <= 0 || d > 3*time.Second {
		t.Fatalf("expected duration within (0,3s], got %v", d)
	}
}

func TestRetryAfterDurationPastDate(t *testing.T) {
	header := http.Header{}
	past := time.Now().Add(-2 * time.Second).UTC()
	header.Set("Retry-After", past.Format(http.TimeFormat))

	d, ok := retryAfterDuration(header)
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if d != 0 {
		t.Fatalf("expected 0s for past date, got %v", d)
	}
}

func TestRetryAfterDurationInvalid(t *testing.T) {
	header := http.Header{}
	header.Set("Retry-After", "nope")

	_, ok := retryAfterDuration(header)
	if ok {
		t.Fatalf("expected ok=false")
	}
}
