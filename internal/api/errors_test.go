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
