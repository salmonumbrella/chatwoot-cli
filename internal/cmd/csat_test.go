package cmd

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRatingStars(t *testing.T) {
	tests := []struct {
		name     string
		rating   int
		expected string
	}{
		{
			name:     "rating 1",
			rating:   1,
			expected: "*----",
		},
		{
			name:     "rating 2",
			rating:   2,
			expected: "**---",
		},
		{
			name:     "rating 3",
			rating:   3,
			expected: "***--",
		},
		{
			name:     "rating 4",
			rating:   4,
			expected: "****-",
		},
		{
			name:     "rating 5",
			rating:   5,
			expected: "*****",
		},
		{
			name:     "rating 0",
			rating:   0,
			expected: "-----",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ratingStars(tt.rating)
			if result != tt.expected {
				t.Errorf("ratingStars(%d) = %q, want %q", tt.rating, result, tt.expected)
			}
		})
	}
}

func TestCSATListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/csat_survey_responses", jsonResponse(200, `[
			{"id": 1, "conversation_id": 100, "rating": 5, "feedback_message": "Great service!", "created_at": 1704067200},
			{"id": 2, "conversation_id": 101, "rating": 4, "feedback_message": "Good", "created_at": 1704153600}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"csat", "list"})
		if err != nil {
			t.Errorf("csat list failed: %v", err)
		}
	})

	if !strings.Contains(output, "Great service!") {
		t.Errorf("output missing feedback: %s", output)
	}
}

func TestCSATListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/csat_survey_responses", jsonResponse(200, `[
			{"id": 1, "rating": 5}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"csat", "list", "-o", "json"})
		if err != nil {
			t.Errorf("csat list --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"rating"`) {
		t.Errorf("JSON output missing rating: %s", output)
	}
}

func TestCSATListCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/csat_survey_responses", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"csat", "list"})
		if err != nil {
			t.Errorf("csat list failed: %v", err)
		}
	})

	if !strings.Contains(output, "No CSAT responses found") {
		t.Errorf("expected 'No CSAT responses found' message: %s", output)
	}
}

func TestCSATListCommand_WithFilters(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/csat_survey_responses", jsonResponse(200, `[
			{"id": 1, "rating": 4}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"csat", "list", "--from", "2024-01-01", "--to", "2024-01-31", "--rating", "4,5", "--inbox-id", "1"})
		if err != nil {
			t.Errorf("csat list with filters failed: %v", err)
		}
	})

	if output == "" {
		t.Error("expected output from csat list with filters")
	}
}

func TestCSATGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/csat_survey_responses", jsonResponse(200, `[
			{"id": 1, "conversation_id": 123, "rating": 5, "feedback_message": "Excellent!", "created_at": 1704067200}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"csat", "get", "123"})
		if err != nil {
			t.Errorf("csat get failed: %v", err)
		}
	})

	if !strings.Contains(output, "Rating:") || !strings.Contains(output, "5/5") {
		t.Errorf("output missing rating info: %s", output)
	}
}

func TestCSATGetCommand_NotFound(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/csat_survey_responses", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"csat", "get", "999"})
		if err != nil {
			t.Errorf("csat get failed: %v", err)
		}
	})

	if !strings.Contains(output, "No CSAT response for conversation") {
		t.Errorf("expected 'No CSAT response' message: %s", output)
	}
}

func TestCSATGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/csat_survey_responses", jsonResponse(200, `[
			{"id": 1, "conversation_id": 123, "rating": 5}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"csat", "get", "123", "-o", "json"})
		if err != nil {
			t.Errorf("csat get --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"rating"`) {
		t.Errorf("JSON output missing rating: %s", output)
	}
}

func TestCSATGetCommand_InvalidID(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `[]`))

	err := Execute(context.Background(), []string{"csat", "get", "invalid"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestCSATSummaryCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/csat_survey_responses", jsonResponse(200, `[
			{"id": 1, "rating": 5},
			{"id": 2, "rating": 4},
			{"id": 3, "rating": 5},
			{"id": 4, "rating": 3}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"csat", "summary"})
		if err != nil {
			t.Errorf("csat summary failed: %v", err)
		}
	})

	if !strings.Contains(output, "CSAT Summary") {
		t.Errorf("output missing summary header: %s", output)
	}
	if !strings.Contains(output, "Average Rating") {
		t.Errorf("output missing average rating: %s", output)
	}
}

func TestCSATSummaryCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/csat_survey_responses", jsonResponse(200, `[
			{"id": 1, "rating": 5},
			{"id": 2, "rating": 4}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"csat", "summary", "-o", "json"})
		if err != nil {
			t.Errorf("csat summary --json failed: %v", err)
		}
	})

	if !strings.Contains(output, `"total_responses"`) {
		t.Errorf("JSON output missing total_responses: %s", output)
	}
}

func TestCSATSummaryCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/csat_survey_responses", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"csat", "summary"})
		if err != nil {
			t.Errorf("csat summary failed: %v", err)
		}
	})

	if !strings.Contains(output, "No CSAT responses found") {
		t.Errorf("expected 'No CSAT responses found' message: %s", output)
	}
}

func TestCSATSummaryCommand_WithDateRange(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/csat_survey_responses", jsonResponse(200, `[
			{"id": 1, "rating": 5}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"csat", "summary", "--from", "2024-01-01", "--to", "2024-01-31"})
		if err != nil {
			t.Errorf("csat summary with date range failed: %v", err)
		}
	})

	if !strings.Contains(output, "2024-01-01") || !strings.Contains(output, "2024-01-31") {
		t.Errorf("output missing date range: %s", output)
	}
}

func TestNewCSATCmd(t *testing.T) {
	cmd := newCSATCmd()

	if cmd.Use != "csat" {
		t.Errorf("expected Use to be 'csat', got %q", cmd.Use)
	}

	// Check subcommands
	subs := []string{"list", "get", "summary"}
	for _, sub := range subs {
		found := false
		for _, c := range cmd.Commands() {
			if c.Use == sub || strings.HasPrefix(c.Use, sub+" ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected subcommand %q not found", sub)
		}
	}
}

func TestNormalizeCSATDate(t *testing.T) {
	// Fixed reference time: Wednesday, January 28, 2026 at 15:04:05 UTC
	now := time.Date(2026, 1, 28, 15, 4, 5, 0, time.UTC)

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "date string passthrough",
			input: "2026-01-27",
			want:  "2026-01-27",
		},
		{
			name:  "yesterday",
			input: "yesterday",
			want:  "2026-01-27",
		},
		{
			name:  "today",
			input: "today",
			want:  "2026-01-28",
		},
		{
			name:  "tomorrow",
			input: "tomorrow",
			want:  "2026-01-29",
		},
		{
			name:  "1d ago",
			input: "1d ago",
			want:  "2026-01-27",
		},
		{
			name:  "7d ago",
			input: "7d ago",
			want:  "2026-01-21",
		},
		{
			name:  "1w ago",
			input: "1w ago",
			want:  "2026-01-21",
		},
		{
			name:  "1mo ago",
			input: "1mo ago",
			want:  "2025-12-28",
		},
		{
			name:    "invalid date format",
			input:   "not-a-date",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid relative",
			input:   "abc ago",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeCSATDateWithNow(tt.input, now)
			if tt.wantErr {
				if err == nil {
					t.Errorf("normalizeCSATDateWithNow(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("normalizeCSATDateWithNow(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("normalizeCSATDateWithNow(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
