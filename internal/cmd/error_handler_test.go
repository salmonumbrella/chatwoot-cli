package cmd

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestHandleError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantContains []string
	}{
		{
			name:         "nil error",
			err:          nil,
			wantContains: []string{},
		},
		{
			name: "rate limit error",
			err:  &api.RateLimitError{RetryAfter: 5 * time.Second},
			wantContains: []string{
				"Rate limit exceeded",
				"Wait a few seconds",
			},
		},
		{
			name: "circuit breaker error",
			err:  &api.CircuitBreakerError{},
			wantContains: []string{
				"circuit breaker",
				"Wait 30 seconds",
			},
		},
		{
			name: "auth error",
			err:  &api.AuthError{Reason: "invalid token"},
			wantContains: []string{
				"Authentication failed",
				"cw auth login",
			},
		},
		{
			name: "404 API error",
			err:  &api.APIError{StatusCode: 404, Body: "not found"},
			wantContains: []string{
				"API error (HTTP 404)",
				"doesn't exist",
			},
		},
		{
			name: "401 API error",
			err:  &api.APIError{StatusCode: 401, Body: "unauthorized"},
			wantContains: []string{
				"API error (HTTP 401)",
				"token may be invalid",
			},
		},
		{
			name: "403 API error",
			err:  &api.APIError{StatusCode: 403, Body: "forbidden"},
			wantContains: []string{
				"API error (HTTP 403)",
				"permission",
			},
		},
		{
			name: "500 API error",
			err:  &api.APIError{StatusCode: 500, Body: "internal error"},
			wantContains: []string{
				"API error (HTTP 500)",
				"Server error",
			},
		},
		{
			name: "API error with request ID",
			err:  &api.APIError{StatusCode: 400, Body: "bad request", RequestID: "req-123"},
			wantContains: []string{
				"API error (HTTP 400)",
				"Request ID: req-123",
			},
		},
		{
			name: "connection refused",
			err:  errors.New("dial tcp: connection refused"),
			wantContains: []string{
				"Connection refused",
				"server is running",
			},
		},
		{
			name: "DNS error",
			err:  errors.New("no such host"),
			wantContains: []string{
				"DNS resolution failed",
				"URL spelling",
			},
		},
		{
			name: "certificate error",
			err:  errors.New("x509: certificate has expired"),
			wantContains: []string{
				"TLS certificate error",
				"certificate",
			},
		},
		{
			name: "generic error",
			err:  errors.New("something went wrong"),
			wantContains: []string{
				"Error:",
				"something went wrong",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HandleError(tt.err)

			if tt.err == nil {
				if result != "" {
					t.Errorf("HandleError(nil) = %q, want empty", result)
				}
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("HandleError() missing %q in output:\n%s", want, result)
				}
			}
		})
	}
}

func TestSuggestionsForStatusCode(t *testing.T) {
	tests := []struct {
		code     int
		body     string
		contains string
	}{
		{400, "field required", "required"},
		{401, "", "token"},
		{403, "", "permission"},
		{404, "", "doesn't exist"},
		{422, "", "Validation"},
		{429, "", "Too many"},
		{500, "", "Server error"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.code)), func(t *testing.T) {
			result := suggestionsForStatusCode(tt.code, tt.body)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("suggestionsForStatusCode(%d) missing %q", tt.code, tt.contains)
			}
		})
	}
}

// TestHandleError_RateLimitViaAPIError tests rate limit returned as API error (HTTP 429)
func TestHandleError_RateLimitViaAPIError(t *testing.T) {
	err := &api.APIError{
		StatusCode: 429,
		Body:       "rate limit exceeded",
	}

	result := HandleError(err)

	// API error with 429 should show API error message and rate limit suggestions
	if !strings.Contains(result, "API error (HTTP 429)") {
		t.Errorf("expected HTTP 429 in message, got: %s", result)
	}
	if !strings.Contains(result, "Too many requests") {
		t.Errorf("expected rate limit suggestion, got: %s", result)
	}
}

// TestHandleError_NotFoundViaAPIError tests 404 error handling
func TestHandleError_NotFoundViaAPIError(t *testing.T) {
	err := &api.APIError{
		StatusCode: 404,
		Body:       "resource not found",
	}

	result := HandleError(err)

	if !strings.Contains(result, "API error (HTTP 404)") {
		t.Errorf("expected HTTP 404 in message, got: %s", result)
	}
	if !strings.Contains(result, "doesn't exist") {
		t.Errorf("expected not found suggestion, got: %s", result)
	}
}

// TestHandleError_ServerErrors tests various server error codes (500, 502, 503, 504)
func TestHandleError_ServerErrors(t *testing.T) {
	serverCodes := []int{500, 502, 503, 504}

	for _, code := range serverCodes {
		t.Run(fmt.Sprintf("HTTP_%d", code), func(t *testing.T) {
			err := &api.APIError{
				StatusCode: code,
				Body:       "server error",
			}

			result := HandleError(err)

			if !strings.Contains(result, "API error") {
				t.Errorf("expected API error message for %d, got: %s", code, result)
			}
			if !strings.Contains(result, "Server error") {
				t.Errorf("expected server error suggestion for %d, got: %s", code, result)
			}
		})
	}
}

// TestHandleError_NetworkError tests generic network error handling
func TestHandleError_NetworkError(t *testing.T) {
	err := errors.New("dial tcp: connection refused")

	result := HandleError(err)

	if !strings.Contains(result, "Connection refused") {
		t.Errorf("expected connection refused message, got: %s", result)
	}
	if !strings.Contains(result, "server is running") {
		t.Errorf("expected helpful suggestion, got: %s", result)
	}
}

// TestHandleError_GenericError tests fallback error handling
func TestHandleError_GenericError(t *testing.T) {
	err := errors.New("something unexpected happened")

	result := HandleError(err)

	if !strings.Contains(result, "something unexpected happened") {
		t.Errorf("expected error message in output, got: %s", result)
	}
	if !strings.Contains(result, "Error:") {
		t.Errorf("expected Error: prefix, got: %s", result)
	}
}

// TestHandleError_BadRequestWithRequiredField tests 400 error with "required" in body
func TestHandleError_BadRequestWithRequiredField(t *testing.T) {
	err := &api.APIError{
		StatusCode: 400,
		Body:       "field 'name' is required",
	}

	result := HandleError(err)

	if !strings.Contains(result, "API error (HTTP 400)") {
		t.Errorf("expected HTTP 400 in message, got: %s", result)
	}
	if !strings.Contains(result, "required field may be missing") {
		t.Errorf("expected required field suggestion, got: %s", result)
	}
}

// TestHandleError_ValidationError tests 422 unprocessable entity
func TestHandleError_ValidationError(t *testing.T) {
	err := &api.APIError{
		StatusCode: 422,
		Body:       "email format invalid",
	}

	result := HandleError(err)

	if !strings.Contains(result, "API error (HTTP 422)") {
		t.Errorf("expected HTTP 422 in message, got: %s", result)
	}
	if !strings.Contains(result, "Validation failed") {
		t.Errorf("expected validation suggestion, got: %s", result)
	}
}

// TestSuggestionsForStatusCode_UnknownCode tests default suggestion for unknown status codes
func TestSuggestionsForStatusCode_UnknownCode(t *testing.T) {
	result := suggestionsForStatusCode(418, "") // I'm a teapot

	if !strings.Contains(result, "--debug") {
		t.Errorf("expected --debug suggestion for unknown code, got: %s", result)
	}
	if !strings.Contains(result, "API documentation") {
		t.Errorf("expected API documentation suggestion, got: %s", result)
	}
}
