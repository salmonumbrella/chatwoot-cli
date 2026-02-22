package cmd

import (
	"os"
	"path/filepath"
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
			got, err := ParseIntList(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIntList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("ParseIntList() = %v, want %v", got, tt.want)
					return
				}
				for i := range got {
					if got[i] != tt.want[i] {
						t.Errorf("ParseIntList() = %v, want %v", got, tt.want)
						return
					}
				}
			}
		})
	}
}

func TestParseConversationIDList(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
	}{
		{"single id", "1", []int{1}, false},
		{"hash ids", "#1,#2", []int{1, 2}, false},
		{"prefixed ids", "conv:1,conversation:2", []int{1, 2}, false},
		{
			"urls",
			"https://app.chatwoot.com/app/accounts/1/conversations/10,https://app.chatwoot.com/app/accounts/1/conversations/20",
			[]int{10, 20},
			false,
		},
		{"empty", "", nil, true},
		{"invalid", "abc", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConversationIDList(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseConversationIDList() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseConversationIDList() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("ParseConversationIDList() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestParseResourceIDListFlag(t *testing.T) {
	t.Run("csv and whitespace", func(t *testing.T) {
		got, err := ParseResourceIDListFlag(
			"1, #2 contact:3 https://app.chatwoot.com/app/accounts/1/contacts/4",
			"contact",
		)
		if err != nil {
			t.Fatalf("ParseResourceIDListFlag() error = %v", err)
		}
		want := []int{1, 2, 3, 4}
		if len(got) != len(want) {
			t.Fatalf("ParseResourceIDListFlag() = %v, want %v", got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("ParseResourceIDListFlag() = %v, want %v", got, want)
			}
		}
	})

	t.Run("json array", func(t *testing.T) {
		got, err := ParseResourceIDListFlag(`[1,"#2","contact:3"]`, "contact")
		if err != nil {
			t.Fatalf("ParseResourceIDListFlag() error = %v", err)
		}
		want := []int{1, 2, 3}
		if len(got) != len(want) {
			t.Fatalf("ParseResourceIDListFlag() = %v, want %v", got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("ParseResourceIDListFlag() = %v, want %v", got, want)
			}
		}
	})

	t.Run("from file", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "ids.txt")
		if err := os.WriteFile(p, []byte("1\n#2\ncontact:3\n"), 0o600); err != nil {
			t.Fatalf("failed to write temp ids file: %v", err)
		}

		got, err := ParseResourceIDListFlag("@"+p, "contact")
		if err != nil {
			t.Fatalf("ParseResourceIDListFlag() error = %v", err)
		}
		want := []int{1, 2, 3}
		if len(got) != len(want) {
			t.Fatalf("ParseResourceIDListFlag() = %v, want %v", got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("ParseResourceIDListFlag() = %v, want %v", got, want)
			}
		}
	})
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

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "hel"}, // maxLen <= 3 returns raw truncation
		{"empty string", "", 10, ""},
		{"unicode chars", "héllo wörld", 8, "héll..."}, // byte-based, not rune-based
		{"max length 0", "hello", 0, ""},
		{"negative maxLen", "hello", -5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestValidateArticleStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		wantErr bool
	}{
		{"status 0 (draft)", 0, false},
		{"status 1 (published)", 1, false},
		{"status 2 (archived)", 2, false},
		{"negative", -1, true},
		{"too high", 3, true},
		{"way too high", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArticleStatus(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateArticleStatus(%d) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int
		want  string
	}{
		{"zero bytes", 0, "-"},
		{"bytes", 500, "500 B"},
		{"kilobytes", 1024, "1.0 KB"},
		{"kilobytes with decimal", 1536, "1.5 KB"},
		{"megabytes", 1048576, "1.0 MB"},
		{"megabytes with decimal", 1572864, "1.5 MB"},
		{"gigabytes", 1073741824, "1.0 GB"},
		{"large gigabytes", 2147483647, "2.0 GB"}, // max int32 to stay within int range
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFileSize(tt.bytes)
			if got != tt.want {
				t.Errorf("formatFileSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestGenerateAttributeKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple lowercase", "name", "name"},
		{"with spaces", "First Name", "first_name"},
		{"with uppercase", "CompanyName", "companyname"},
		{"with mixed", "User Email Address", "user_email_address"},
		{"already snake case", "user_name", "user_name"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateAttributeKey(tt.input)
			if got != tt.want {
				t.Errorf("generateAttributeKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
