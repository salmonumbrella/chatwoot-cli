package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	client := newTestClient("https://example.com", "test-token", 42)

	if client.BaseURL != "https://example.com" {
		t.Errorf("Expected BaseURL https://example.com, got %s", client.BaseURL)
	}
	if client.APIToken != "test-token" {
		t.Errorf("Expected APIToken test-token, got %s", client.APIToken)
	}
	if client.AccountID != 42 {
		t.Errorf("Expected AccountID 42, got %d", client.AccountID)
	}
	if client.HTTP == nil {
		t.Error("Expected HTTP client to be initialized")
	}
}

func TestAccountPath(t *testing.T) {
	client := newTestClient("https://example.com", "token", 123)

	tests := []struct {
		path     string
		expected string
	}{
		{"/contacts", "https://example.com/api/v1/accounts/123/contacts"},
		{"/conversations/1", "https://example.com/api/v1/accounts/123/conversations/1"},
		{"conversations/1", "https://example.com/api/v1/accounts/123/conversations/1"},
		{"", "https://example.com/api/v1/accounts/123"},
	}

	for _, tt := range tests {
		result := client.accountPath(tt.path)
		if result != tt.expected {
			t.Errorf("accountPath(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful GET",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "name": "test"}`,
			expectError:  false,
		},
		{
			name:         "not found",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
			expectError:  true,
		},
		{
			name:         "server error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "internal error"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				if r.Header.Get("api_access_token") != "test-token" {
					t.Error("Missing or wrong api_access_token header")
				}
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			var result map[string]any
			err := client.Get(context.Background(), "/test", &result)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestGet_RequestIDCaptured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-123")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	var result map[string]any
	err := client.Get(context.Background(), "/test", &result)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.RequestID != "req-123" {
		t.Fatalf("expected RequestID req-123, got %q", apiErr.RequestID)
	}
}

func TestPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "value" {
			t.Errorf("Expected body key=value, got %v", body)
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	var result map[string]int
	err := client.Post(context.Background(), "/test", map[string]string{"key": "value"}, &result)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result["id"] != 1 {
		t.Errorf("Expected id=1, got %v", result)
	}
}

func TestPost_IdempotencyKeyHeader(t *testing.T) {
	var gotKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	client.IdempotencyKey = "idem-123"

	var result map[string]any
	if err := client.Post(context.Background(), "/test", map[string]string{"key": "value"}, &result); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if gotKey != "idem-123" {
		t.Fatalf("Expected Idempotency-Key header to be set, got %q", gotKey)
	}
}

func TestGet_RetriesOn429(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	client.RetryConfig.MaxRateLimitRetries = 1
	client.RetryConfig.RateLimitBaseDelay = 0

	var result map[string]any
	if err := client.Get(context.Background(), "/test", &result); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestPost_RetriesOn429WithIdempotencyKey(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	client.IdempotencyKey = "idem-1"
	client.RetryConfig.MaxRateLimitRetries = 1
	client.RetryConfig.RateLimitBaseDelay = 0

	var result map[string]any
	if err := client.Post(context.Background(), "/test", map[string]string{"key": "value"}, &result); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestPost_NoRetryOn429WithoutIdempotencyKey(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	client.RetryConfig.MaxRateLimitRetries = 1
	client.RetryConfig.RateLimitBaseDelay = 0

	var result map[string]any
	err := client.Post(context.Background(), "/test", map[string]string{"key": "value"}, &result)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !IsRateLimitError(err) {
		t.Fatalf("expected RateLimitError, got %T", err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestGet_RetriesOn5xx(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	client.RetryConfig.Max5xxRetries = 1
	client.RetryConfig.ServerErrorRetryDelay = 0

	var result map[string]any
	if err := client.Get(context.Background(), "/test", &result); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	client.RetryConfig.Max5xxRetries = 0
	client.RetryConfig.ServerErrorRetryDelay = 0
	if client.circuitBreaker != nil {
		client.circuitBreaker.threshold = 1
		client.circuitBreaker.resetTime = time.Hour
	}

	var result map[string]any
	if err := client.Get(context.Background(), "/test", &result); err == nil {
		t.Fatalf("expected error, got nil")
	}

	if err := client.Get(context.Background(), "/test", &result); err == nil {
		t.Fatalf("expected error, got nil")
	} else if !IsCircuitBreakerError(err) {
		t.Fatalf("expected CircuitBreakerError, got %T", err)
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected 1 call before circuit opened, got %d", calls)
	}
}

// TestCircuitBreakerHalfOpenProbeSuccess verifies that a successful probe
// request during half-open state closes the circuit.
func TestCircuitBreakerHalfOpenProbeSuccess(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		if call == 1 {
			// First call fails to open the circuit
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Subsequent calls succeed (probe request)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	client.RetryConfig.Max5xxRetries = 0
	client.RetryConfig.ServerErrorRetryDelay = 0
	if client.circuitBreaker != nil {
		client.circuitBreaker.threshold = 1
		client.circuitBreaker.resetTime = 15 * time.Millisecond
	}

	var result map[string]any

	// First request fails and opens the circuit
	if err := client.Get(context.Background(), "/test", &result); err == nil {
		t.Fatalf("expected error on first request, got nil")
	}

	// Second request should be blocked by circuit breaker
	if err := client.Get(context.Background(), "/test", &result); !IsCircuitBreakerError(err) {
		t.Fatalf("expected CircuitBreakerError, got %v", err)
	}

	// Wait for reset time to allow half-open
	time.Sleep(20 * time.Millisecond)

	// Third request should succeed (probe) and close the circuit
	if err := client.Get(context.Background(), "/test", &result); err != nil {
		t.Fatalf("expected probe request to succeed, got %v", err)
	}

	// Fourth request should also succeed (circuit is now closed)
	if err := client.Get(context.Background(), "/test", &result); err != nil {
		t.Fatalf("expected request to succeed after circuit closed, got %v", err)
	}

	// Verify call count: 1 (fail) + 1 (probe success) + 1 (normal) = 3
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("expected 3 server calls, got %d", calls)
	}
}

// TestCircuitBreakerHalfOpenProbeFails verifies that a failed probe
// request during half-open state re-opens the circuit.
func TestCircuitBreakerHalfOpenProbeFails(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		// All calls fail
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	client.RetryConfig.Max5xxRetries = 0
	client.RetryConfig.ServerErrorRetryDelay = 0
	if client.circuitBreaker != nil {
		client.circuitBreaker.threshold = 1
		client.circuitBreaker.resetTime = 15 * time.Millisecond
	}

	var result map[string]any

	// First request fails and opens the circuit
	if err := client.Get(context.Background(), "/test", &result); err == nil {
		t.Fatalf("expected error on first request, got nil")
	}

	// Wait for reset time to allow half-open
	time.Sleep(20 * time.Millisecond)

	// Second request is the probe - it fails (API error, not circuit breaker)
	err := client.Get(context.Background(), "/test", &result)
	if err == nil {
		t.Fatalf("expected probe request to fail, got nil")
	}
	if IsCircuitBreakerError(err) {
		t.Fatalf("probe request should hit server, not be blocked by circuit breaker")
	}

	// Third request should be blocked (circuit is open again)
	if err := client.Get(context.Background(), "/test", &result); !IsCircuitBreakerError(err) {
		t.Fatalf("expected CircuitBreakerError after failed probe, got %v", err)
	}

	// Verify call count: 1 (initial fail) + 1 (probe fail) = 2
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 server calls, got %d", calls)
	}
}

func TestPostMultipartNoRetryOn429(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)

		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("api_access_token") != "test-token" {
			t.Error("Missing or wrong api_access_token header")
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
			t.Errorf("Expected multipart content-type, got %s", r.Header.Get("Content-Type"))
		}

		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.PostMultipart(context.Background(), "/test", map[string]string{"key": "value"}, map[string][]byte{"file.txt": []byte("hello")}, nil)
	if err == nil {
		t.Fatalf("Expected rate limit error, got nil")
	}
	if !IsRateLimitError(err) {
		t.Fatalf("Expected RateLimitError, got %T", err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("Expected 1 attempt, got %d", calls)
	}
}

func TestEnsureBaseURLValidatedCachesSuccess(t *testing.T) {
	original := validateChatwootURL
	var calls int
	validateChatwootURL = func(rawURL string) error {
		calls++
		return nil
	}
	defer func() { validateChatwootURL = original }()

	client := newTestClient("https://example.com", "token", 1)
	client.skipURLValidation = false

	if err := client.ensureBaseURLValidated(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := client.ensureBaseURLValidated(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected validation to run once, got %d calls", calls)
	}
}

func TestAPIError(t *testing.T) {
	err := &APIError{StatusCode: 404, Body: "not found"}
	expected := "API error (status 404): not found"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

func TestPut(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful PUT",
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1, "updated": true}`,
			expectError:  false,
		},
		{
			name:         "not found",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
			expectError:  true,
		},
		{
			name:         "server error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `{"error": "internal error"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("Expected PUT, got %s", r.Method)
				}
				if r.Header.Get("api_access_token") != "test-token" {
					t.Error("Missing or wrong api_access_token header")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Verify body
				var body map[string]string
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["key"] != "value" {
					t.Errorf("Expected body key=value, got %v", body)
				}

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			var result map[string]any
			err := client.Put(context.Background(), "/test", map[string]string{"key": "value"}, &result)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestPatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["key"] != "value" {
			t.Errorf("Expected body key=value, got %v", body)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1, "patched": true}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	var result map[string]any
	err := client.Patch(context.Background(), "/test", map[string]string{"key": "value"}, &result)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result["patched"] != true {
		t.Errorf("Expected patched=true, got %v", result)
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		expectError bool
	}{
		{
			name:        "successful DELETE",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "not found",
			statusCode:  http.StatusNotFound,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE, got %s", r.Method)
				}
				w.WriteHeader(tt.statusCode)
				if tt.statusCode >= 400 {
					_, _ = w.Write([]byte(`{"error": "error"}`))
				}
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			err := client.Delete(context.Background(), "/test")

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestDeleteWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}

		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["reason"] != "test" {
			t.Errorf("Expected body reason=test, got %v", body)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	err := client.DeleteWithBody(context.Background(), "/test", map[string]string{"reason": "test"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetRaw(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectError  bool
	}{
		{
			name:         "successful GET raw",
			statusCode:   http.StatusOK,
			responseBody: `{"raw": "data", "unparsed": true}`,
			expectError:  false,
		},
		{
			name:         "not found",
			statusCode:   http.StatusNotFound,
			responseBody: `{"error": "not found"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)
			result, err := client.GetRaw(context.Background(), "/test")

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && string(result) != tt.responseBody {
				t.Errorf("Expected body %s, got %s", tt.responseBody, string(result))
			}
		})
	}
}

func TestPlatformPath(t *testing.T) {
	client := newTestClient("https://example.com", "token", 123)

	tests := []struct {
		path     string
		expected string
	}{
		{"/users", "https://example.com/platform/api/v1/users"},
		{"/accounts", "https://example.com/platform/api/v1/accounts"},
		{"", "https://example.com/platform/api/v1"},
	}

	for _, tt := range tests {
		result := client.platformPath(tt.path)
		if result != tt.expected {
			t.Errorf("platformPath(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestPublicPath(t *testing.T) {
	client := newTestClient("https://example.com", "token", 123)

	tests := []struct {
		path     string
		expected string
	}{
		{"/inboxes/abc123/contacts", "https://example.com/public/api/v1/inboxes/abc123/contacts"},
		{"/inboxes/abc123/messages", "https://example.com/public/api/v1/inboxes/abc123/messages"},
		{"", "https://example.com/public/api/v1"},
	}

	for _, tt := range tests {
		result := client.publicPath(tt.path)
		if result != tt.expected {
			t.Errorf("publicPath(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestSanitizeErrorBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "JSON with error field",
			input:    `{"error": "Invalid token"}`,
			expected: "Invalid token",
		},
		{
			name:     "JSON with message field",
			input:    `{"message": "Resource not found"}`,
			expected: "Resource not found",
		},
		{
			name:     "JSON with both fields prefers error",
			input:    `{"error": "Primary error", "message": "Secondary message"}`,
			expected: "Primary error",
		},
		{
			name:     "Invalid JSON",
			input:    `not json at all`,
			expected: "API request failed (response body redacted for security)",
		},
		{
			name:     "JSON without error or message",
			input:    `{"status": "failed", "code": 500}`,
			expected: "API request failed (response body redacted for security)",
		},
		{
			name:     "Empty string",
			input:    ``,
			expected: "API request failed (response body redacted for security)",
		},
		// Validation error test cases
		{
			name:  "Validation errors with string values",
			input: `{"errors": {"email": "is invalid"}}`,
			expected: `Validation errors:
  email: is invalid`,
		},
		{
			name:  "Validation errors with array values",
			input: `{"errors": {"email": ["is invalid", "can't be blank"]}}`,
			expected: `Validation errors:
  email: can't be blank
  email: is invalid`,
		},
		{
			name:  "Message with validation errors",
			input: `{"message": "Validation failed", "errors": {"email": "is invalid", "name": "is required"}}`,
			expected: `Validation failed
Validation errors:
  email: is invalid
  name: is required`,
		},
		{
			name:  "Error with validation errors",
			input: `{"error": "Invalid request", "errors": {"field": "bad value"}}`,
			expected: `Invalid request
Validation errors:
  field: bad value`,
		},
		{
			name:  "Multiple fields with array errors",
			input: `{"errors": {"email": ["is invalid"], "name": ["is required", "is too short"]}}`,
			expected: `Validation errors:
  email: is invalid
  name: is required
  name: is too short`,
		},
		{
			name:     "Empty errors object",
			input:    `{"errors": {}}`,
			expected: "API request failed (response body redacted for security)",
		},
		{
			name:     "Errors field is not an object",
			input:    `{"errors": "some string"}`,
			expected: "API request failed (response body redacted for security)",
		},
		{
			name:     "Message only with empty errors",
			input:    `{"message": "Something went wrong", "errors": {}}`,
			expected: "Something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeErrorBody(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeErrorBody(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDoWithNilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no body for GET request
		body, _ := io.ReadAll(r.Body)
		if len(body) != 0 {
			t.Errorf("Expected empty body, got %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	var result map[string]int
	err := client.Get(context.Background(), "/test", &result)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDoWithNilResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": 1}`))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	// Pass nil result - should not error
	err := client.Post(context.Background(), "/test", map[string]string{"key": "value"}, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDoWithEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		// No body
	}))
	defer server.Close()

	client := newTestClient(server.URL, "test-token", 1)
	var result map[string]any
	err := client.Post(context.Background(), "/test", map[string]string{"key": "value"}, &result)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestPostMultipart(t *testing.T) {
	tests := []struct {
		name         string
		files        map[string][]byte
		fields       map[string]string
		statusCode   int
		responseBody string
		expectError  bool
		validateFunc func(*testing.T, *http.Request)
	}{
		{
			name: "single file upload",
			files: map[string][]byte{
				"test.png": []byte("fake image data"),
			},
			fields: map[string]string{
				"content":      "Hello",
				"message_type": "outgoing",
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 1}`,
			expectError:  false,
			validateFunc: func(t *testing.T, r *http.Request) {
				contentType := r.Header.Get("Content-Type")
				if !strings.HasPrefix(contentType, "multipart/form-data") {
					t.Errorf("Expected multipart/form-data, got %s", contentType)
				}
				if err := r.ParseMultipartForm(10 << 20); err != nil {
					t.Fatalf("Failed to parse multipart form: %v", err)
				}
				if r.FormValue("content") != "Hello" {
					t.Errorf("Expected content 'Hello', got %s", r.FormValue("content"))
				}
				files := r.MultipartForm.File["attachments[]"]
				if len(files) != 1 {
					t.Errorf("Expected 1 file, got %d", len(files))
				}
			},
		},
		{
			name: "multiple files upload",
			files: map[string][]byte{
				"doc1.pdf": []byte("pdf content"),
				"img1.png": []byte("png content"),
			},
			fields: map[string]string{
				"content": "See attached",
			},
			statusCode:   http.StatusOK,
			responseBody: `{"id": 2}`,
			expectError:  false,
			validateFunc: func(t *testing.T, r *http.Request) {
				if err := r.ParseMultipartForm(10 << 20); err != nil {
					t.Fatalf("Failed to parse multipart form: %v", err)
				}
				files := r.MultipartForm.File["attachments[]"]
				if len(files) != 2 {
					t.Errorf("Expected 2 files, got %d", len(files))
				}
			},
		},
		{
			name:         "API error response",
			files:        map[string][]byte{"test.txt": []byte("data")},
			fields:       map[string]string{},
			statusCode:   http.StatusBadRequest,
			responseBody: `{"error": "Invalid file type"}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRequest *http.Request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Clone request body for validation
				body, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(bytes.NewBuffer(body))
				capturedRequest = r.Clone(context.Background())
				capturedRequest.Body = io.NopCloser(bytes.NewBuffer(body))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 1)

			var result map[string]any
			err := client.PostMultipart(context.Background(), "/test", tt.fields, tt.files, &result)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.validateFunc != nil && capturedRequest != nil {
				tt.validateFunc(t, capturedRequest)
			}
		})
	}
}

func TestSetRetryConfig_UpdatesClientAndCircuitBreaker(t *testing.T) {
	client := newTestClient("https://example.com", "token", 1)
	cfg := RetryConfig{
		MaxRateLimitRetries:     3,
		Max5xxRetries:           4,
		RateLimitBaseDelay:      2 * time.Second,
		ServerErrorRetryDelay:   3 * time.Second,
		CircuitBreakerThreshold: 11,
		CircuitBreakerResetTime: 30 * time.Second,
	}

	client.SetRetryConfig(cfg)

	if client.RetryConfig.CircuitBreakerThreshold != 11 {
		t.Fatalf("RetryConfig threshold = %d, want 11", client.RetryConfig.CircuitBreakerThreshold)
	}
	if client.circuitBreaker == nil {
		t.Fatal("expected circuitBreaker to be initialized")
	}
	if client.circuitBreaker.threshold != 11 {
		t.Fatalf("circuitBreaker threshold = %d, want 11", client.circuitBreaker.threshold)
	}
	if client.circuitBreaker.resetTime != 30*time.Second {
		t.Fatalf("circuitBreaker resetTime = %s, want 30s", client.circuitBreaker.resetTime)
	}
}

func TestSetRetryConfig_WhenCircuitBreakerIsNil(t *testing.T) {
	client := newTestClient("https://example.com", "token", 1)
	client.circuitBreaker = nil

	cfg := client.RetryConfig
	cfg.CircuitBreakerThreshold = 9
	client.SetRetryConfig(cfg)

	if client.RetryConfig.CircuitBreakerThreshold != 9 {
		t.Fatalf("RetryConfig threshold = %d, want 9", client.RetryConfig.CircuitBreakerThreshold)
	}
}

func TestDoRaw(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		statusCode  int
		body        string
		expectError bool
	}{
		{
			name:        "success",
			method:      http.MethodPost,
			path:        "conversations",
			statusCode:  http.StatusCreated,
			body:        `{"ok":true}`,
			expectError: false,
		},
		{
			name:        "api error",
			method:      http.MethodGet,
			path:        "/conversations/404",
			statusCode:  http.StatusNotFound,
			body:        `{"error":"not found"}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := newTestClient(server.URL, "token", 1)
			respBody, headers, statusCode, err := client.DoRaw(context.Background(), tt.method, tt.path, map[string]any{"x": 1})

			if tt.expectError && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if statusCode != tt.statusCode {
				t.Fatalf("status = %d, want %d", statusCode, tt.statusCode)
			}
			if string(respBody) != tt.body {
				t.Fatalf("body = %q, want %q", string(respBody), tt.body)
			}
			if headers == nil {
				t.Fatal("expected non-nil headers")
			}
		})
	}
}

func TestDoRaw_InvalidBodyMarshaling(t *testing.T) {
	client := newTestClient("https://example.com", "token", 1)
	_, _, _, err := client.DoRaw(context.Background(), http.MethodPost, "/test", map[string]any{"bad": make(chan int)})
	if err == nil {
		t.Fatal("expected marshaling error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to marshal request body") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostMultipart_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer server.Close()

	client := newTestClient(server.URL, "token", 1)
	var result map[string]any
	err := client.PostMultipart(context.Background(), "/test", map[string]string{"a": "b"}, map[string][]byte{"f.txt": []byte("x")}, &result)
	if err == nil {
		t.Fatal("expected JSON decode error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected API response format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestIDFromHeader(t *testing.T) {
	if got := requestIDFromHeader(nil); got != "" {
		t.Fatalf("requestIDFromHeader(nil) = %q, want empty", got)
	}

	h1 := http.Header{}
	h1.Set("X-Request-Id", "req-lower")
	if got := requestIDFromHeader(h1); got != "req-lower" {
		t.Fatalf("requestIDFromHeader(X-Request-Id) = %q", got)
	}

	h2 := http.Header{}
	h2.Set("X-Request-ID", "req-upper")
	if got := requestIDFromHeader(h2); got != "req-upper" {
		t.Fatalf("requestIDFromHeader(X-Request-ID) = %q", got)
	}

	h3 := http.Header{}
	if got := requestIDFromHeader(h3); got != "" {
		t.Fatalf("requestIDFromHeader(empty) = %q, want empty", got)
	}
}

func TestPlatformPathAndPublicPath_WithoutLeadingSlash(t *testing.T) {
	client := newTestClient("https://example.com", "token", 1)
	if got := client.platformPath("users"); got != "https://example.com/platform/api/v1/users" {
		t.Fatalf("platformPath without slash = %q", got)
	}
	if got := client.publicPath("inboxes/abc123/messages"); got != "https://example.com/public/api/v1/inboxes/abc123/messages" {
		t.Fatalf("publicPath without slash = %q", got)
	}
}
