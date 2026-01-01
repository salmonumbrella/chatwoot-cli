package cmd

import "testing"

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
