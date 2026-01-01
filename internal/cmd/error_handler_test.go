package cmd

import (
	"errors"
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
				"chatwoot auth login",
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
