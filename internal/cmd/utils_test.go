package cmd

import (
	"testing"
)

func TestParseIntList(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
	}{
		{"single id", "1", []int{1}, false},
		{"multiple ids", "1,2,3", []int{1, 2, 3}, false},
		{"with spaces", "1, 2, 3", []int{1, 2, 3}, false},
		{"empty parts", "1,,2", []int{1, 2}, false},
		{"trailing comma", "1,2,", []int{1, 2}, false},
		{"leading comma", ",1,2", []int{1, 2}, false},
		{"empty string", "", nil, true},
		{"only commas", ",,,", nil, true},
		{"negative number", "-1", nil, true},
		{"zero", "0", nil, true},
		{"non-numeric", "abc", nil, true},
		{"mixed valid invalid", "1,abc,2", nil, true},
		{"float", "1.5", nil, true},
		{"large number", "999999999", []int{999999999}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIntList(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIntList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("parseIntList() = %v, want %v", got, tt.want)
					return
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("parseIntList() = %v, want %v", got, tt.want)
						return
					}
				}
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid date", "2024-01-15", false},
		{"valid date start of year", "2024-01-01", false},
		{"valid date end of year", "2024-12-31", false},
		{"invalid format slashes", "2024/01/15", true},
		{"invalid format dots", "2024.01.15", true},
		{"invalid month", "2024-13-01", true},
		{"invalid day", "2024-01-32", true},
		{"invalid format short", "24-01-15", true},
		{"empty string", "", true},
		{"random string", "not-a-date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Errorf("parseDate(%q) returned empty string", tt.input)
			}
		})
	}
}
