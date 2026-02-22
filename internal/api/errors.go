package api

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// RateLimitError represents a rate limit exceeded error.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded, retry after %s", e.RetryAfter)
}

// AuthError represents an authentication or authorization error.
type AuthError struct {
	Reason string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication error: %s", e.Reason)
}

// CircuitBreakerError indicates the circuit breaker is open.
type CircuitBreakerError struct{}

func (e *CircuitBreakerError) Error() string {
	return "circuit breaker is open, too many recent failures"
}

// IsRateLimitError checks if the error is a rate limit error.
func IsRateLimitError(err error) bool {
	var e *RateLimitError
	return errors.As(err, &e)
}

// IsAuthError checks if the error is an authentication error.
func IsAuthError(err error) bool {
	var e *AuthError
	return errors.As(err, &e)
}

// IsCircuitBreakerError checks if the error is a circuit breaker error.
func IsCircuitBreakerError(err error) bool {
	var e *CircuitBreakerError
	return errors.As(err, &e)
}

// IsNotFoundError checks if the error indicates a resource was not found.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404 ||
			strings.Contains(strings.ToLower(apiErr.Body), "not found")
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

// ContextualError wraps an API error with request context
type ContextualError struct {
	Method     string
	URL        string
	StatusCode int
	Err        error
}

func (e *ContextualError) Error() string {
	return fmt.Sprintf("%s %s failed (status %d): %v", e.Method, e.URL, e.StatusCode, e.Err)
}

func (e *ContextualError) Unwrap() error {
	return e.Err
}

// WrapError adds request context to an API error
func WrapError(method, url string, statusCode int, err error) error {
	return &ContextualError{
		Method:     method,
		URL:        url,
		StatusCode: statusCode,
		Err:        err,
	}
}
