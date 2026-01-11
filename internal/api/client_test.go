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
