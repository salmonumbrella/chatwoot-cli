package validation

import (
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Input length limits to prevent resource exhaustion
const (
	MaxNameLength    = 255
	MaxEmailLength   = 320     // RFC 5321: 64 chars (local) + 1 (@) + 255 (domain) = 320
	MaxPhoneLength   = 20      // International E.164 format
	MaxMessageLength = 100000  // 100KB for message content
	MaxJSONPayload   = 1048576 // 1MB for JSON payloads
	MaxURLLength     = 2048    // Standard browser URL limit
)

// ValidateName validates a contact name length
func ValidateName(name string) error {
	if name == "" {
		return nil // Empty names are allowed (field is optional in some contexts)
	}

	length := utf8.RuneCountInString(name)
	if length > MaxNameLength {
		return fmt.Errorf("name exceeds maximum length of %d characters (got %d)", MaxNameLength, length)
	}

	return nil
}

// ValidateEmail validates an email address length
// Note: This only validates length. Format validation is handled separately (Task 12).
func ValidateEmail(email string) error {
	if email == "" {
		return nil // Empty emails are allowed (field is optional in some contexts)
	}

	length := utf8.RuneCountInString(email)
	if length > MaxEmailLength {
		return fmt.Errorf("email exceeds maximum length of %d characters (got %d)", MaxEmailLength, length)
	}

	return nil
}

// ValidatePhone validates a phone number length
func ValidatePhone(phone string) error {
	if phone == "" {
		return nil // Empty phone numbers are allowed (field is optional in some contexts)
	}

	length := utf8.RuneCountInString(phone)
	if length > MaxPhoneLength {
		return fmt.Errorf("phone number exceeds maximum length of %d characters (got %d)", MaxPhoneLength, length)
	}

	return nil
}

// ValidateMessageContent validates message content length
// Note: Empty content is allowed (e.g., attachment-only messages).
// Callers should check if content is required before calling this function.
func ValidateMessageContent(content string) error {
	if content == "" {
		return nil // Empty content is allowed for attachment-only messages
	}

	// Use byte length for message content as it's transmitted as UTF-8
	length := len(content)
	if length > MaxMessageLength {
		return fmt.Errorf("message content exceeds maximum size of %d bytes (got %d)", MaxMessageLength, length)
	}

	return nil
}

// ValidateJSONPayload validates JSON payload size
func ValidateJSONPayload(payload string) error {
	if payload == "" {
		return fmt.Errorf("JSON payload cannot be empty")
	}

	// Use byte length for JSON payloads as they're transmitted as UTF-8
	length := len(payload)
	if length > MaxJSONPayload {
		return fmt.Errorf("JSON payload exceeds maximum size of %d bytes (got %d)", MaxJSONPayload, length)
	}

	return nil
}

// ValidateEmailFormat validates the format of an email address.
// Returns nil for empty emails (optional field).
func ValidateEmailFormat(email string) error {
	if email == "" {
		return nil // Optional field
	}
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email format: %w", err)
	}
	return nil
}

// ValidatePhoneFormat validates phone number format (basic validation).
// Returns nil for empty phones (optional field).
// Allows digits, spaces, dashes, parentheses, and leading +.
func ValidatePhoneFormat(phone string) error {
	if phone == "" {
		return nil
	}
	// Basic validation: must contain at least some digits
	// and only allowed characters
	// Pattern: optional +, then digits with optional separators
	for i, r := range phone {
		if r == '+' && i == 0 {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == ' ' || r == '-' || r == '(' || r == ')' {
			continue
		}
		return fmt.Errorf("invalid phone format: contains invalid character '%c'", r)
	}
	return nil
}

// ParsePositiveInt parses a string as a positive integer ID.
// Returns error if the value is not a positive integer or exceeds int32 range.
func ParsePositiveInt(s string, fieldName string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	id64, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", fieldName, err)
	}
	if id64 <= 0 {
		return 0, fmt.Errorf("invalid %s: must be a positive integer", fieldName)
	}
	return int(id64), nil
}
