package validation

import (
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "empty name is allowed",
			input:     "",
			wantError: false,
		},
		{
			name:      "valid short name",
			input:     "John Doe",
			wantError: false,
		},
		{
			name:      "valid name at max length",
			input:     strings.Repeat("a", MaxNameLength),
			wantError: false,
		},
		{
			name:      "name exceeds max length by one",
			input:     strings.Repeat("a", MaxNameLength+1),
			wantError: true,
		},
		{
			name:      "name with unicode characters",
			input:     "José García-Pérez",
			wantError: false,
		},
		{
			name:      "name with emoji",
			input:     "John 👋 Doe",
			wantError: false,
		},
		{
			name:      "very long name",
			input:     strings.Repeat("a", 500),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateName() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "empty email is allowed",
			input:     "",
			wantError: false,
		},
		{
			name:      "valid short email",
			input:     "user@example.com",
			wantError: false,
		},
		{
			name:      "valid email at max length",
			input:     strings.Repeat("a", 64) + "@" + strings.Repeat("b", 250) + ".com",
			wantError: false,
		},
		{
			name:      "email exceeds max length",
			input:     strings.Repeat("a", 100) + "@" + strings.Repeat("b", 250) + ".com",
			wantError: true,
		},
		{
			name:      "email with subdomain",
			input:     "user@mail.example.com",
			wantError: false,
		},
		{
			name:      "email with plus addressing",
			input:     "user+tag@example.com",
			wantError: false,
		},
		{
			name:      "very long email",
			input:     strings.Repeat("a", 500) + "@example.com",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateEmail() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "empty phone is allowed",
			input:     "",
			wantError: false,
		},
		{
			name:      "valid US phone",
			input:     "+15551234567",
			wantError: false,
		},
		{
			name:      "valid phone at max length",
			input:     strings.Repeat("1", MaxPhoneLength),
			wantError: false,
		},
		{
			name:      "phone exceeds max length",
			input:     strings.Repeat("1", MaxPhoneLength+1),
			wantError: true,
		},
		{
			name:      "phone with spaces and dashes",
			input:     "+1 (555) 123-4567",
			wantError: false,
		},
		{
			name:      "very long phone",
			input:     strings.Repeat("1", 50),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhone(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePhone() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateMessageContent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "empty message is allowed (attachment-only)",
			input:     "",
			wantError: false,
		},
		{
			name:      "valid short message",
			input:     "Hello, world!",
			wantError: false,
		},
		{
			name:      "message at max length",
			input:     strings.Repeat("a", MaxMessageLength),
			wantError: false,
		},
		{
			name:      "message exceeds max length",
			input:     strings.Repeat("a", MaxMessageLength+1),
			wantError: true,
		},
		{
			name:      "message with unicode characters",
			input:     "Hello 世界! 🌍",
			wantError: false,
		},
		{
			name:      "message with newlines",
			input:     "Line 1\nLine 2\nLine 3",
			wantError: false,
		},
		{
			name:      "very long message",
			input:     strings.Repeat("a", 200000),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessageContent(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateMessageContent() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateJSONPayload(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "empty JSON is not allowed",
			input:     "",
			wantError: true,
		},
		{
			name:      "valid small JSON",
			input:     `{"key":"value"}`,
			wantError: false,
		},
		{
			name:      "valid JSON array",
			input:     `[{"attribute_key":"name","filter_operator":"contains","values":["test"]}]`,
			wantError: false,
		},
		{
			name:      "JSON at max size",
			input:     `{"data":"` + strings.Repeat("a", MaxJSONPayload-15) + `"}`,
			wantError: false,
		},
		{
			name:      "JSON exceeds max size",
			input:     `{"data":"` + strings.Repeat("a", MaxJSONPayload+100) + `"}`,
			wantError: true,
		},
		{
			name:      "JSON with unicode",
			input:     `{"message":"Hello 世界! 🌍"}`,
			wantError: false,
		},
		{
			name:      "very large JSON payload",
			input:     strings.Repeat(`{"key":"value"},`, 100000),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSONPayload(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateJSONPayload() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// Benchmark tests to ensure validation is fast
func BenchmarkValidateName(b *testing.B) {
	name := strings.Repeat("a", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateName(name)
	}
}

func BenchmarkValidateEmail(b *testing.B) {
	email := "user@" + strings.Repeat("a", 50) + ".com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateEmail(email)
	}
}

func BenchmarkValidatePhone(b *testing.B) {
	phone := "+1234567890"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePhone(phone)
	}
}

func BenchmarkValidateMessageContent(b *testing.B) {
	content := strings.Repeat("Hello world! ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateMessageContent(content)
	}
}

func BenchmarkValidateJSONPayload(b *testing.B) {
	payload := `{"key":"value","array":[1,2,3,4,5]}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateJSONPayload(payload)
	}
}

func TestParsePositiveInt(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		fieldName string
		want      int
		wantError bool
		errMsg    string
	}{
		{
			name:      "valid positive integer",
			input:     "123",
			fieldName: "contact ID",
			want:      123,
			wantError: false,
		},
		{
			name:      "valid single digit",
			input:     "1",
			fieldName: "ID",
			want:      1,
			wantError: false,
		},
		{
			name:      "max int32 value",
			input:     "2147483647",
			fieldName: "ID",
			want:      2147483647,
			wantError: false,
		},
		{
			name:      "zero is not allowed",
			input:     "0",
			fieldName: "contact ID",
			want:      0,
			wantError: true,
			errMsg:    "must be a positive integer",
		},
		{
			name:      "negative integer is not allowed",
			input:     "-1",
			fieldName: "contact ID",
			want:      0,
			wantError: true,
			errMsg:    "must be a positive integer",
		},
		{
			name:      "negative large integer",
			input:     "-999999",
			fieldName: "ID",
			want:      0,
			wantError: true,
			errMsg:    "must be a positive integer",
		},
		{
			name:      "exceeds int32 max",
			input:     "2147483648",
			fieldName: "ID",
			want:      0,
			wantError: true,
			errMsg:    "invalid ID",
		},
		{
			name:      "way too large number",
			input:     "99999999999999999999",
			fieldName: "conversation ID",
			want:      0,
			wantError: true,
			errMsg:    "invalid conversation ID",
		},
		{
			name:      "not a number",
			input:     "abc",
			fieldName: "message ID",
			want:      0,
			wantError: true,
			errMsg:    "invalid message ID",
		},
		{
			name:      "empty string",
			input:     "",
			fieldName: "ID",
			want:      0,
			wantError: true,
			errMsg:    "invalid ID",
		},
		{
			name:      "float number",
			input:     "123.45",
			fieldName: "ID",
			want:      0,
			wantError: true,
			errMsg:    "invalid ID",
		},
		{
			name:      "number with spaces",
			input:     " 123 ",
			fieldName: "ID",
			want:      123,
			wantError: false,
		},
		{
			name:      "number with leading hash",
			input:     "#123",
			fieldName: "ID",
			want:      123,
			wantError: false,
		},
		{
			name:      "hex number",
			input:     "0x123",
			fieldName: "ID",
			want:      0,
			wantError: true,
			errMsg:    "invalid ID",
		},
		{
			name:      "number with leading zero",
			input:     "0123",
			fieldName: "ID",
			want:      123,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePositiveInt(tt.input, tt.fieldName)
			if (err != nil) != tt.wantError {
				t.Errorf("ParsePositiveInt() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && got != tt.want {
				t.Errorf("ParsePositiveInt() = %v, want %v", got, tt.want)
			}
			if tt.wantError && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ParsePositiveInt() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func BenchmarkParsePositiveInt(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParsePositiveInt("123456", "ID")
	}
}

func TestValidateEmailFormat(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "empty email is allowed",
			input:     "",
			wantError: false,
		},
		{
			name:      "valid simple email",
			input:     "user@example.com",
			wantError: false,
		},
		{
			name:      "valid email with subdomain",
			input:     "user@mail.example.com",
			wantError: false,
		},
		{
			name:      "valid email with plus addressing",
			input:     "user+tag@example.com",
			wantError: false,
		},
		{
			name:      "valid email with dots in local part",
			input:     "first.last@example.com",
			wantError: false,
		},
		{
			name:      "valid email with numbers",
			input:     "user123@example456.com",
			wantError: false,
		},
		{
			name:      "missing @ symbol",
			input:     "userexample.com",
			wantError: true,
		},
		{
			name:      "missing domain",
			input:     "user@",
			wantError: true,
		},
		{
			name:      "missing local part",
			input:     "@example.com",
			wantError: true,
		},
		{
			name:      "multiple @ symbols",
			input:     "user@@example.com",
			wantError: true,
		},
		{
			name:      "invalid characters",
			input:     "user name@example.com",
			wantError: true,
		},
		{
			name:      "missing TLD",
			input:     "user@example",
			wantError: false, // mail.ParseAddress allows this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmailFormat(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateEmailFormat() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestValidatePhoneFormat(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "empty phone is allowed",
			input:     "",
			wantError: false,
		},
		{
			name:      "valid US phone with plus",
			input:     "+15551234567",
			wantError: false,
		},
		{
			name:      "valid phone with spaces",
			input:     "+1 555 123 4567",
			wantError: false,
		},
		{
			name:      "valid phone with dashes",
			input:     "+1-555-123-4567",
			wantError: false,
		},
		{
			name:      "valid phone with parentheses",
			input:     "+1 (555) 123-4567",
			wantError: false,
		},
		{
			name:      "valid phone without plus",
			input:     "15551234567",
			wantError: false,
		},
		{
			name:      "valid phone with mixed separators",
			input:     "+1 (555)-123 4567",
			wantError: false,
		},
		{
			name:      "valid international phone",
			input:     "+44 20 7946 0958",
			wantError: false,
		},
		{
			name:      "invalid character - letter",
			input:     "+1555123456a",
			wantError: true,
		},
		{
			name:      "invalid character - dot",
			input:     "+1.555.123.4567",
			wantError: true,
		},
		{
			name:      "invalid character - hash",
			input:     "+1555123#4567",
			wantError: true,
		},
		{
			name:      "plus in middle",
			input:     "1+5551234567",
			wantError: true,
		},
		{
			name:      "multiple plus signs",
			input:     "++15551234567",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhoneFormat(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidatePhoneFormat() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func BenchmarkValidateEmailFormat(b *testing.B) {
	email := "user@example.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateEmailFormat(email)
	}
}

func BenchmarkValidatePhoneFormat(b *testing.B) {
	phone := "+1 (555) 123-4567"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePhoneFormat(phone)
	}
}
