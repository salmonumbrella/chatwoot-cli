package api

import (
	"errors"
	"testing"
	"time"
)

func TestAPIError_Error(t *testing.T) {
	err := &APIError{StatusCode: 404, Body: "Not found"}
	if err.Error() != "API error (status 404): Not found" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestRateLimitError(t *testing.T) {
	err := &RateLimitError{RetryAfter: 5 * time.Second}
	if !IsRateLimitError(err) {
		t.Error("IsRateLimitError should return true")
	}
}

func TestAuthError(t *testing.T) {
	err := &AuthError{Reason: "invalid token"}
	if !IsAuthError(err) {
		t.Error("IsAuthError should return true")
	}
}

func TestContextualError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := WrapError("GET", "/api/test", 500, inner)

	var ctxErr *ContextualError
	if !errors.As(err, &ctxErr) {
		t.Error("should be ContextualError")
	}
	if !errors.Is(err, inner) {
		t.Error("should unwrap to inner error")
	}
}

func TestRateLimitError_Error(t *testing.T) {
	err := &RateLimitError{RetryAfter: 30 * time.Second}
	expected := "rate limit exceeded, retry after 30s"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestAuthError_Error(t *testing.T) {
	err := &AuthError{Reason: "invalid credentials"}
	expected := "authentication error: invalid credentials"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestCircuitBreakerError_Error(t *testing.T) {
	err := &CircuitBreakerError{}
	expected := "circuit breaker is open, too many recent failures"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestIsCircuitBreakerError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"circuit breaker error", &CircuitBreakerError{}, true},
		{"rate limit error", &RateLimitError{RetryAfter: time.Second}, false},
		{"auth error", &AuthError{Reason: "test"}, false},
		{"nil error", nil, false},
		{"plain error", errors.New("some error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCircuitBreakerError(tt.err); got != tt.expected {
				t.Errorf("IsCircuitBreakerError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestContextualError_Error(t *testing.T) {
	inner := errors.New("connection refused")
	err := &ContextualError{
		Method:     "POST",
		URL:        "/api/v1/messages",
		StatusCode: 500,
		Err:        inner,
	}
	expected := "POST /api/v1/messages failed (status 500): connection refused"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"not_found code", &APIError{StatusCode: 404, Body: "not_found"}, true},
		{"contains not found", &APIError{StatusCode: 404, Body: "Resource not found"}, true},
		{"other error", &APIError{StatusCode: 500, Body: "server error"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.expected)
			}
		})
	}
}
