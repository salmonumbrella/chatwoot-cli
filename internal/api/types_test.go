package api

import (
	"testing"
	"time"
)

func TestCampaign_CreatedAtTime(t *testing.T) {
	tests := []struct {
		name      string
		createdAt int64
		expected  time.Time
	}{
		{
			name:      "valid timestamp",
			createdAt: 1700000000,
			expected:  time.Unix(1700000000, 0),
		},
		{
			name:      "zero timestamp",
			createdAt: 0,
			expected:  time.Unix(0, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Campaign{CreatedAt: tt.createdAt}
			result := c.CreatedAtTime()
			if !result.Equal(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCSATResponse_CreatedAtTime(t *testing.T) {
	tests := []struct {
		name      string
		createdAt float64
		expected  time.Time
	}{
		{
			name:      "valid timestamp",
			createdAt: 1700000000.0,
			expected:  time.Unix(1700000000, 0),
		},
		{
			name:      "zero timestamp",
			createdAt: 0.0,
			expected:  time.Unix(0, 0),
		},
		{
			name:      "fractional timestamp",
			createdAt: 1700000000.5,
			expected:  time.Unix(1700000000, 0), // Truncated to seconds
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CSATResponse{CreatedAt: tt.createdAt}
			result := c.CreatedAtTime()
			if !result.Equal(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
