package cli

import (
	"testing"
	"time"
)

func TestParseRelativeTime(t *testing.T) {
	now := time.Date(2026, 1, 28, 15, 4, 5, 0, time.UTC) // Wednesday

	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{
			name:  "hours ago",
			input: "2h ago",
			want:  now.Add(-2 * time.Hour),
		},
		{
			name:  "days ago",
			input: "1d ago",
			want:  now.Add(-24 * time.Hour),
		},
		{
			name:  "weeks ago",
			input: "2w ago",
			want:  now.Add(-14 * 24 * time.Hour),
		},
		{
			name:  "months ago",
			input: "1mo ago",
			want:  now.AddDate(0, -1, 0),
		},
		{
			name:  "yesterday",
			input: "yesterday",
			want:  time.Date(2026, 1, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "today",
			input: "today",
			want:  time.Date(2026, 1, 28, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "tomorrow",
			input: "tomorrow",
			want:  time.Date(2026, 1, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "weekday",
			input: "monday",
			want:  time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "next weekday",
			input: "next friday",
			want:  time.Date(2026, 1, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "future minutes",
			input: "30m",
			want:  now.Add(30 * time.Minute),
		},
		{
			name:  "future hours",
			input: "2h",
			want:  now.Add(2 * time.Hour),
		},
		{
			name:  "date only",
			input: "2026-01-27",
			want:  time.Date(2026, 1, 27, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "rfc3339",
			input: "2026-01-27T10:00:00Z",
			want:  time.Date(2026, 1, 27, 10, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRelativeTime(tt.input, now)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Fatalf("expected %s, got %s", tt.want.Format(time.RFC3339Nano), got.Format(time.RFC3339Nano))
			}
		})
	}
}

func TestParseRelativeTime_Invalid(t *testing.T) {
	_, err := ParseRelativeTime("not-a-date", time.Now())
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestStartOfDay(t *testing.T) {
	sample := time.Date(2026, 1, 28, 15, 4, 5, 0, time.UTC)
	start := startOfDay(sample)

	if !start.Equal(time.Date(2026, 1, 28, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected start of day: %s", start.Format(time.RFC3339Nano))
	}
}
