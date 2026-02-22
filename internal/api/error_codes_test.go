package api

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestErrorCodeFromStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       ErrorCode
	}{
		{"400 Bad Request", 400, ErrBadRequest},
		{"401 Unauthorized", 401, ErrUnauthorized},
		{"403 Forbidden", 403, ErrForbidden},
		{"404 Not Found", 404, ErrNotFound},
		{"409 Conflict", 409, ErrConflict},
		{"422 Validation", 422, ErrValidation},
		{"429 Rate Limited", 429, ErrRateLimited},
		{"500 Server Error", 500, ErrServerError},
		{"502 Bad Gateway", 502, ErrServerError},
		{"503 Service Unavailable", 503, ErrServerError},
		{"599 Unknown Server Error", 599, ErrServerError},
		{"200 OK (unknown)", 200, ErrUnknown},
		{"418 Teapot (unknown)", 418, ErrUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ErrorCodeFromStatus(tt.statusCode)
			if got != tt.want {
				t.Errorf("ErrorCodeFromStatus(%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestErrorCodeIsRetryable(t *testing.T) {
	retryableCodes := []ErrorCode{ErrRateLimited, ErrServerError, ErrTimeout, ErrCircuitOpen}
	nonRetryableCodes := []ErrorCode{ErrBadRequest, ErrUnauthorized, ErrForbidden, ErrNotFound, ErrConflict, ErrValidation, ErrUnknown}

	for _, code := range retryableCodes {
		t.Run(string(code)+"_retryable", func(t *testing.T) {
			if !code.IsRetryable() {
				t.Errorf("%v.IsRetryable() = false, want true", code)
			}
		})
	}

	for _, code := range nonRetryableCodes {
		t.Run(string(code)+"_not_retryable", func(t *testing.T) {
			if code.IsRetryable() {
				t.Errorf("%v.IsRetryable() = true, want false", code)
			}
		})
	}
}

func TestErrorCodeSuggestion(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrUnauthorized, "Run 'cw auth login' to authenticate"},
		{ErrForbidden, "Check your account permissions"},
		{ErrNotFound, "Verify the resource ID exists"},
		{ErrRateLimited, "Wait a moment and retry"},
		{ErrValidation, "Check the input values"},
		{ErrBadRequest, "Check the request format and parameters"},
		{ErrConflict, "The resource state may have changed; refresh and retry"},
		{ErrServerError, "The server encountered an error; try again later"},
		{ErrTimeout, "The request timed out; check network connectivity and retry"},
		{ErrCircuitOpen, "Too many recent failures; wait before retrying"},
		{ErrUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			got := tt.code.Suggestion()
			if got != tt.expected {
				t.Errorf("%v.Suggestion() = %q, want %q", tt.code, got, tt.expected)
			}
		})
	}
}

func TestStructuredErrorError(t *testing.T) {
	err := &StructuredError{
		Code:    ErrNotFound,
		Message: "resource not found",
	}

	expected := "[not_found] resource not found"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestStructuredErrorJSONSerialization(t *testing.T) {
	t.Run("full error with context", func(t *testing.T) {
		err := &StructuredError{
			Code:       ErrRateLimited,
			Message:    "rate limit exceeded",
			Retryable:  true,
			Suggestion: "Wait a moment and retry",
			Context: map[string]any{
				"retry_after": "30s",
				"limit":       100,
			},
		}

		data, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Fatalf("json.Marshal failed: %v", marshalErr)
		}

		var decoded StructuredError
		if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
			t.Fatalf("json.Unmarshal failed: %v", unmarshalErr)
		}

		if decoded.Code != err.Code {
			t.Errorf("Code = %v, want %v", decoded.Code, err.Code)
		}
		if decoded.Message != err.Message {
			t.Errorf("Message = %v, want %v", decoded.Message, err.Message)
		}
		if decoded.Retryable != err.Retryable {
			t.Errorf("Retryable = %v, want %v", decoded.Retryable, err.Retryable)
		}
		if decoded.Suggestion != err.Suggestion {
			t.Errorf("Suggestion = %v, want %v", decoded.Suggestion, err.Suggestion)
		}
		if decoded.Context["retry_after"] != "30s" {
			t.Errorf("Context[retry_after] = %v, want 30s", decoded.Context["retry_after"])
		}
	})

	t.Run("minimal error without optional fields", func(t *testing.T) {
		err := &StructuredError{
			Code:      ErrBadRequest,
			Message:   "invalid input",
			Retryable: false,
		}

		data, marshalErr := json.Marshal(err)
		if marshalErr != nil {
			t.Fatalf("json.Marshal failed: %v", marshalErr)
		}

		jsonStr := string(data)
		// Suggestion and Context should be omitted when empty
		if containsField(jsonStr, "suggestion") {
			t.Error("JSON should omit empty suggestion field")
		}
		if containsField(jsonStr, "context") {
			t.Error("JSON should omit empty context field")
		}
	})

	t.Run("JSON structure is correct", func(t *testing.T) {
		err := NewStructuredError(ErrUnauthorized, "auth failed")
		data, _ := json.Marshal(err)

		var raw map[string]any
		if unmarshalErr := json.Unmarshal(data, &raw); unmarshalErr != nil {
			t.Fatalf("json.Unmarshal failed: %v", unmarshalErr)
		}

		if raw["code"] != "unauthorized" {
			t.Errorf("code = %v, want unauthorized", raw["code"])
		}
		if raw["message"] != "auth failed" {
			t.Errorf("message = %v, want 'auth failed'", raw["message"])
		}
		if raw["retryable"] != false {
			t.Errorf("retryable = %v, want false", raw["retryable"])
		}
	})
}

func TestNewStructuredError(t *testing.T) {
	err := NewStructuredError(ErrServerError, "internal error")

	if err.Code != ErrServerError {
		t.Errorf("Code = %v, want %v", err.Code, ErrServerError)
	}
	if err.Message != "internal error" {
		t.Errorf("Message = %v, want 'internal error'", err.Message)
	}
	if !err.Retryable {
		t.Error("Retryable should be true for server errors")
	}
	if err.Suggestion == "" {
		t.Error("Suggestion should be populated for server errors")
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("priority", "critical", []string{"urgent", "high", "medium", "low", "none"})

	if err.Code != ErrValidation {
		t.Errorf("Code = %v, want %v", err.Code, ErrValidation)
	}
	if err.Retryable {
		t.Error("Retryable should be false for validation errors")
	}
	if len(err.AllowedValues) != 5 {
		t.Errorf("AllowedValues length = %d, want 5", len(err.AllowedValues))
	}
	if err.AllowedValues[0] != "urgent" {
		t.Errorf("AllowedValues[0] = %q, want 'urgent'", err.AllowedValues[0])
	}
	if err.Context["field"] != "priority" {
		t.Errorf("Context[field] = %v, want 'priority'", err.Context["field"])
	}
	if err.Context["got"] != "critical" {
		t.Errorf("Context[got] = %v, want 'critical'", err.Context["got"])
	}
	if err.Suggestion == "" {
		t.Error("Suggestion should be populated")
	}

	// Verify JSON serialization includes allowed_values
	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("json.Marshal failed: %v", marshalErr)
	}
	var raw map[string]any
	if unmarshalErr := json.Unmarshal(data, &raw); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal failed: %v", unmarshalErr)
	}
	if _, ok := raw["allowed_values"]; !ok {
		t.Error("JSON should contain allowed_values field")
	}
}

func TestStructuredErrorFromError_StructuredError(t *testing.T) {
	original := NewValidationError("status", "bogus", []string{"open", "resolved"})
	result := StructuredErrorFromError(original)
	if result != original {
		t.Error("StructuredErrorFromError should return the original StructuredError")
	}
}

func TestNewStructuredErrorWithContext(t *testing.T) {
	ctx := map[string]any{
		"resource_id":   123,
		"resource_type": "conversation",
	}
	err := NewStructuredErrorWithContext(ErrNotFound, "conversation not found", ctx)

	if err.Code != ErrNotFound {
		t.Errorf("Code = %v, want %v", err.Code, ErrNotFound)
	}
	if err.Context["resource_id"] != 123 {
		t.Errorf("Context[resource_id] = %v, want 123", err.Context["resource_id"])
	}
	if err.Context["resource_type"] != "conversation" {
		t.Errorf("Context[resource_type] = %v, want 'conversation'", err.Context["resource_type"])
	}
}

func TestStructuredErrorFromAPIError(t *testing.T) {
	apiErr := &APIError{
		StatusCode: 404,
		Body:       "resource not found",
		RequestID:  "req-404",
	}

	structErr := StructuredErrorFromAPIError(apiErr)

	if structErr.Code != ErrNotFound {
		t.Errorf("Code = %v, want %v", structErr.Code, ErrNotFound)
	}
	if structErr.Message != "resource not found" {
		t.Errorf("Message = %v, want 'resource not found'", structErr.Message)
	}
	if structErr.Retryable {
		t.Error("Retryable should be false for not found errors")
	}
	if structErr.Context["status_code"] != 404 {
		t.Errorf("Context[status_code] = %v, want 404", structErr.Context["status_code"])
	}
	if structErr.Context["request_id"] != "req-404" {
		t.Errorf("Context[request_id] = %v, want req-404", structErr.Context["request_id"])
	}
}

func TestStructuredErrorFromError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := StructuredErrorFromError(nil)
		if result != nil {
			t.Errorf("StructuredErrorFromError(nil) = %v, want nil", result)
		}
	})

	t.Run("APIError", func(t *testing.T) {
		apiErr := &APIError{StatusCode: 403, Body: "access denied"}
		result := StructuredErrorFromError(apiErr)

		if result.Code != ErrForbidden {
			t.Errorf("Code = %v, want %v", result.Code, ErrForbidden)
		}
		if result.Message != "access denied" {
			t.Errorf("Message = %v, want 'access denied'", result.Message)
		}
	})

	t.Run("RateLimitError", func(t *testing.T) {
		rateLimitErr := &RateLimitError{RetryAfter: 30 * time.Second}
		result := StructuredErrorFromError(rateLimitErr)

		if result.Code != ErrRateLimited {
			t.Errorf("Code = %v, want %v", result.Code, ErrRateLimited)
		}
		if !result.Retryable {
			t.Error("Retryable should be true for rate limit errors")
		}
		if result.Context["retry_after"] != "30s" {
			t.Errorf("Context[retry_after] = %v, want '30s'", result.Context["retry_after"])
		}
	})

	t.Run("AuthError", func(t *testing.T) {
		authErr := &AuthError{Reason: "invalid token"}
		result := StructuredErrorFromError(authErr)

		if result.Code != ErrUnauthorized {
			t.Errorf("Code = %v, want %v", result.Code, ErrUnauthorized)
		}
		if result.Retryable {
			t.Error("Retryable should be false for auth errors")
		}
	})

	t.Run("CircuitBreakerError", func(t *testing.T) {
		cbErr := &CircuitBreakerError{}
		result := StructuredErrorFromError(cbErr)

		if result.Code != ErrCircuitOpen {
			t.Errorf("Code = %v, want %v", result.Code, ErrCircuitOpen)
		}
		if !result.Retryable {
			t.Error("Retryable should be true for circuit breaker errors")
		}
	})

	t.Run("generic error", func(t *testing.T) {
		genericErr := errors.New("something went wrong")
		result := StructuredErrorFromError(genericErr)

		if result.Code != ErrUnknown {
			t.Errorf("Code = %v, want %v", result.Code, ErrUnknown)
		}
		if result.Message != "something went wrong" {
			t.Errorf("Message = %v, want 'something went wrong'", result.Message)
		}
		if result.Retryable {
			t.Error("Retryable should be false for unknown errors")
		}
	})

	t.Run("wrapped APIError", func(t *testing.T) {
		apiErr := &APIError{StatusCode: 500, Body: "internal error"}
		wrappedErr := &ContextualError{
			Method:     "GET",
			URL:        "/api/test",
			StatusCode: 500,
			Err:        apiErr,
		}
		result := StructuredErrorFromError(wrappedErr)

		if result.Code != ErrServerError {
			t.Errorf("Code = %v, want %v", result.Code, ErrServerError)
		}
		if !result.Retryable {
			t.Error("Retryable should be true for server errors")
		}
	})
}

// Helper function to check if a JSON string contains a field
func containsField(jsonStr, field string) bool {
	var raw map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return false
	}
	_, exists := raw[field]
	return exists
}
