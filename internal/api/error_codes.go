package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrorCode represents machine-readable error codes for agent error handling.
type ErrorCode string

const (
	// ErrBadRequest indicates a malformed request (HTTP 400).
	ErrBadRequest ErrorCode = "bad_request"
	// ErrUnauthorized indicates authentication is required or failed (HTTP 401).
	ErrUnauthorized ErrorCode = "unauthorized"
	// ErrForbidden indicates the user lacks permission (HTTP 403).
	ErrForbidden ErrorCode = "forbidden"
	// ErrNotFound indicates the requested resource does not exist (HTTP 404).
	ErrNotFound ErrorCode = "not_found"
	// ErrConflict indicates a conflict with current state (HTTP 409).
	ErrConflict ErrorCode = "conflict"
	// ErrValidation indicates input validation failed (HTTP 422).
	ErrValidation ErrorCode = "validation_failed"
	// ErrRateLimited indicates too many requests (HTTP 429).
	ErrRateLimited ErrorCode = "rate_limited"
	// ErrServerError indicates an internal server error (HTTP 5xx).
	ErrServerError ErrorCode = "server_error"
	// ErrTimeout indicates the request timed out.
	ErrTimeout ErrorCode = "timeout"
	// ErrCircuitOpen indicates the circuit breaker is open.
	ErrCircuitOpen ErrorCode = "circuit_open"
	// ErrUnknown indicates an unknown or unclassified error.
	ErrUnknown ErrorCode = "unknown"
)

// IsRetryable returns true if errors with this code may succeed on retry.
func (c ErrorCode) IsRetryable() bool {
	switch c {
	case ErrRateLimited, ErrServerError, ErrTimeout, ErrCircuitOpen:
		return true
	default:
		return false
	}
}

// Suggestion returns a human-readable suggestion for resolving this error.
func (c ErrorCode) Suggestion() string {
	switch c {
	case ErrUnauthorized:
		return "Run 'cw auth login' to authenticate"
	case ErrForbidden:
		return "Check your account permissions"
	case ErrNotFound:
		return "Verify the resource ID exists"
	case ErrRateLimited:
		return "Wait a moment and retry"
	case ErrValidation:
		return "Check the input values"
	case ErrBadRequest:
		return "Check the request format and parameters"
	case ErrConflict:
		return "The resource state may have changed; refresh and retry"
	case ErrServerError:
		return "The server encountered an error; try again later"
	case ErrTimeout:
		return "The request timed out; check network connectivity and retry"
	case ErrCircuitOpen:
		return "Too many recent failures; wait before retrying"
	default:
		return ""
	}
}

// ErrorCodeFromStatus maps an HTTP status code to an ErrorCode.
func ErrorCodeFromStatus(statusCode int) ErrorCode {
	switch statusCode {
	case 400:
		return ErrBadRequest
	case 401:
		return ErrUnauthorized
	case 403:
		return ErrForbidden
	case 404:
		return ErrNotFound
	case 409:
		return ErrConflict
	case 422:
		return ErrValidation
	case 429:
		return ErrRateLimited
	default:
		if statusCode >= 500 && statusCode < 600 {
			return ErrServerError
		}
		return ErrUnknown
	}
}

// StructuredError provides machine-readable error information for agents.
type StructuredError struct {
	Code          ErrorCode      `json:"code"`
	Message       string         `json:"message"`
	Retryable     bool           `json:"retryable"`
	Suggestion    string         `json:"suggestion,omitempty"`
	Context       map[string]any `json:"context,omitempty"`
	AllowedValues []string       `json:"allowed_values,omitempty"`
}

// Error implements the error interface.
func (e *StructuredError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// MarshalJSON implements custom JSON marshaling.
func (e *StructuredError) MarshalJSON() ([]byte, error) {
	type Alias StructuredError
	return json.Marshal((*Alias)(e))
}

// NewStructuredError creates a StructuredError from an ErrorCode and message.
func NewStructuredError(code ErrorCode, message string) *StructuredError {
	return &StructuredError{
		Code:       code,
		Message:    message,
		Retryable:  code.IsRetryable(),
		Suggestion: code.Suggestion(),
	}
}

// NewStructuredErrorWithContext creates a StructuredError with additional context.
func NewStructuredErrorWithContext(code ErrorCode, message string, ctx map[string]any) *StructuredError {
	err := NewStructuredError(code, message)
	err.Context = ctx
	return err
}

// NewValidationError creates a StructuredError for input validation failures,
// including the list of allowed values so agents can self-correct.
func NewValidationError(field string, got string, allowed []string) *StructuredError {
	return &StructuredError{
		Code:          ErrValidation,
		Message:       fmt.Sprintf("invalid %s %q: must be one of %s", field, got, strings.Join(allowed, ", ")),
		Retryable:     false,
		Suggestion:    fmt.Sprintf("Use one of: %s", strings.Join(allowed, ", ")),
		AllowedValues: allowed,
		Context:       map[string]any{"field": field, "got": got},
	}
}

// StructuredErrorFromAPIError converts an APIError to a StructuredError.
func StructuredErrorFromAPIError(apiErr *APIError) *StructuredError {
	code := ErrorCodeFromStatus(apiErr.StatusCode)
	ctx := map[string]any{
		"status_code": apiErr.StatusCode,
	}
	if apiErr.RequestID != "" {
		ctx["request_id"] = apiErr.RequestID
	}
	return &StructuredError{
		Code:       code,
		Message:    apiErr.Body,
		Retryable:  code.IsRetryable(),
		Suggestion: code.Suggestion(),
		Context:    ctx,
	}
}

// StructuredErrorFromError attempts to convert any error to a StructuredError.
// It handles StructuredError, APIError, RateLimitError, AuthError, CircuitBreakerError, and generic errors.
func StructuredErrorFromError(err error) *StructuredError {
	if err == nil {
		return nil
	}

	var se *StructuredError
	if errors.As(err, &se) {
		return se
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return StructuredErrorFromAPIError(apiErr)
	}

	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return &StructuredError{
			Code:       ErrRateLimited,
			Message:    rateLimitErr.Error(),
			Retryable:  true,
			Suggestion: ErrRateLimited.Suggestion(),
			Context: map[string]any{
				"retry_after": rateLimitErr.RetryAfter.String(),
			},
		}
	}

	var authErr *AuthError
	if errors.As(err, &authErr) {
		return &StructuredError{
			Code:       ErrUnauthorized,
			Message:    authErr.Error(),
			Retryable:  false,
			Suggestion: ErrUnauthorized.Suggestion(),
		}
	}

	var cbErr *CircuitBreakerError
	if errors.As(err, &cbErr) {
		return &StructuredError{
			Code:       ErrCircuitOpen,
			Message:    cbErr.Error(),
			Retryable:  true,
			Suggestion: ErrCircuitOpen.Suggestion(),
		}
	}

	// Generic error - classify as unknown
	return &StructuredError{
		Code:       ErrUnknown,
		Message:    err.Error(),
		Retryable:  false,
		Suggestion: "",
	}
}
