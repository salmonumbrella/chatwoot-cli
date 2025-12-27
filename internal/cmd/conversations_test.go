package cmd

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestPrintConversationsTable(t *testing.T) {
	tests := []struct {
		name           string
		conversations  []api.Conversation
		expectedOutput []string
	}{
		{
			name:          "empty conversations",
			conversations: []api.Conversation{},
			expectedOutput: []string{
				"ID",
				"INBOX",
				"STATUS",
				"PRIORITY",
				"UNREAD",
				"CREATED",
			},
		},
		{
			name: "single conversation with all fields",
			conversations: []api.Conversation{
				{
					ID:        123,
					DisplayID: intPtr(456),
					InboxID:   1,
					Status:    "open",
					Priority:  strPtr("high"),
					Unread:    3,
					CreatedAt: 1700000000,
				},
			},
			expectedOutput: []string{
				"456", // DisplayID used when present
				"1",   // InboxID
				"open",
				"high",
				"3",
				"2023-11-14", // Date portion of formatted time
			},
		},
		{
			name: "conversation without optional fields",
			conversations: []api.Conversation{
				{
					ID:        789,
					InboxID:   2,
					Status:    "resolved",
					Priority:  nil, // No priority
					Unread:    0,
					CreatedAt: 1700000000,
				},
			},
			expectedOutput: []string{
				"789", // ID used when DisplayID is nil
				"2",
				"resolved",
				"-", // Priority placeholder
				"0",
			},
		},
		{
			name: "multiple conversations",
			conversations: []api.Conversation{
				{
					ID:        1,
					DisplayID: intPtr(10),
					InboxID:   1,
					Status:    "open",
					Priority:  strPtr("urgent"),
					Unread:    5,
					CreatedAt: 1700000000,
				},
				{
					ID:        2,
					InboxID:   2,
					Status:    "pending",
					Priority:  nil,
					Unread:    0,
					CreatedAt: 1700001000,
				},
				{
					ID:        3,
					DisplayID: intPtr(30),
					InboxID:   1,
					Status:    "resolved",
					Priority:  strPtr("low"),
					Unread:    1,
					CreatedAt: 1700002000,
				},
			},
			expectedOutput: []string{
				"10", "1", "open", "urgent", "5",
				"2", "2", "pending", "-", "0",
				"30", "1", "resolved", "low", "1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printConversationsTable(tt.conversations)

			// Restore stdout
			_ = w.Close()
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			output := buf.String()

			// Verify expected strings are present
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, output)
				}
			}

			// Verify header is always present
			if !strings.Contains(output, "ID") ||
				!strings.Contains(output, "INBOX") ||
				!strings.Contains(output, "STATUS") {
				t.Errorf("Output missing expected headers. Got:\n%s", output)
			}
		})
	}
}

func TestPrintConversationsTable_Formatting(t *testing.T) {
	conversations := []api.Conversation{
		{
			ID:        1,
			DisplayID: intPtr(100),
			InboxID:   5,
			Status:    "open",
			Priority:  strPtr("high"),
			Unread:    10,
			CreatedAt: 1700000000,
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printConversationsTable(conversations)

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Verify tabular format (multiple spaces/tabs between columns)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 lines (header + data), got %d", len(lines))
	}

	// Check that header and data rows have similar structure
	headerFields := strings.Fields(lines[0])
	dataFields := strings.Fields(lines[1])

	if len(headerFields) != 6 {
		t.Errorf("Expected 6 header fields, got %d: %v", len(headerFields), headerFields)
	}

	if len(dataFields) < 6 {
		t.Errorf("Expected at least 6 data fields, got %d: %v", len(dataFields), dataFields)
	}
}

// Helper functions for test data
func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func TestParseSnoozedUntil(t *testing.T) {
	// Get current time for relative tests
	now := time.Now().Unix()
	futureTimestamp := now + 3600                         // 1 hour from now
	farFutureTimestamp := now + (11 * 365 * 24 * 60 * 60) // 11 years from now

	// Create a timestamp far enough in the future (2026)
	futureDate := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	futureTimestampStr := strconv.FormatInt(futureDate.Unix(), 10)

	tests := []struct {
		name        string
		input       string
		expectError bool
		validate    func(*testing.T, int64)
	}{
		{
			name:        "valid unix timestamp",
			input:       futureTimestampStr,
			expectError: false,
			validate: func(t *testing.T, result int64) {
				if result != futureDate.Unix() {
					t.Errorf("Expected %d, got %d", futureDate.Unix(), result)
				}
			},
		},
		{
			name:        "valid RFC3339",
			input:       "2026-12-31T23:59:59Z",
			expectError: false,
			validate: func(t *testing.T, result int64) {
				expected := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC).Unix()
				if result != expected {
					t.Errorf("Expected %d, got %d", expected, result)
				}
			},
		},
		{
			name:        "valid RFC3339 with timezone",
			input:       "2026-01-15T10:00:00-05:00",
			expectError: false,
			validate: func(t *testing.T, result int64) {
				parsed, _ := time.Parse(time.RFC3339, "2026-01-15T10:00:00-05:00")
				expected := parsed.Unix()
				if result != expected {
					t.Errorf("Expected %d, got %d", expected, result)
				}
			},
		},
		{
			name:        "negative timestamp",
			input:       "-1",
			expectError: true,
		},
		{
			name:        "zero timestamp",
			input:       "0",
			expectError: true,
		},
		{
			name:        "timestamp in the past",
			input:       "1000000000",
			expectError: true,
		},
		{
			name:        "RFC3339 in the past",
			input:       "2020-01-01T00:00:00Z",
			expectError: true,
		},
		{
			name:        "invalid format",
			input:       "not-a-date",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "malformed RFC3339",
			input:       "2025-13-45T99:99:99Z",
			expectError: true,
		},
		{
			name:        "timestamp too far in future",
			input:       strconv.FormatInt(farFutureTimestamp, 10),
			expectError: true,
		},
		{
			name:        "RFC3339 too far in future",
			input:       "2040-01-01T00:00:00Z",
			expectError: true,
		},
		{
			name:        "valid future timestamp (1 hour from now)",
			input:       strconv.FormatInt(futureTimestamp, 10),
			expectError: false,
			validate: func(t *testing.T, result int64) {
				// Allow small timing differences
				if result < futureTimestamp-5 || result > futureTimestamp+5 {
					t.Errorf("Expected around %d, got %d", futureTimestamp, result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSnoozedUntil(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
