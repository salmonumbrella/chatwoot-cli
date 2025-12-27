package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestAPIError(t *testing.T) {
	err := &APIError{StatusCode: 404, Body: "not found"}
	expected := "API error (status 404): not found"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
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
