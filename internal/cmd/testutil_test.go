// Package cmd provides test utilities for the chatwoot CLI commands.
//
// # Test Infrastructure Overview
//
// This file provides utilities for testing CLI commands against mock HTTP servers.
// The main components are:
//
//   - routeHandler: A chainable HTTP handler for routing requests to mock responses
//   - setupTestEnv / setupTestEnvWithHandler: Environment setup with automatic cleanup
//   - captureStdout / captureStderr: Output capture utilities
//   - jsonResponse: Helper for creating JSON response handlers
//
// # Quick Start
//
// Here's a minimal example of testing a command:
//
//	func TestMyCommand(t *testing.T) {
//	    handler := newRouteHandler().
//	        On("GET", "/api/v1/accounts/1/resource", jsonResponse(200, `{"id": 1}`))
//
//	    setupTestEnvWithHandler(t, handler)
//
//	    output := captureStdout(t, func() {
//	        err := Execute(context.Background(), []string{"resource", "get", "1"})
//	        if err != nil {
//	            t.Fatalf("command failed: %v", err)
//	        }
//	    })
//
//	    // Assert on output...
//	}
//
// # Route Handler Pattern
//
// The routeHandler allows you to define mock responses for specific HTTP methods and paths.
// It uses a fluent/chainable API for readability:
//
//	handler := newRouteHandler().
//	    On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{"payload": [...]}`)).
//	    On("POST", "/api/v1/accounts/1/labels", jsonResponse(201, `{"id": 1}`)).
//	    On("DELETE", "/api/v1/accounts/1/labels/1", jsonResponse(200, `{}`))
//
// For more complex scenarios (e.g., inspecting request bodies), use a custom handler:
//
//	var receivedBody map[string]any
//	handler := newRouteHandler().
//	    On("POST", "/api/v1/accounts/1/labels", func(w http.ResponseWriter, r *http.Request) {
//	        _ = json.NewDecoder(r.Body).Decode(&receivedBody)
//	        w.Header().Set("Content-Type", "application/json")
//	        w.WriteHeader(http.StatusOK)
//	        _, _ = w.Write([]byte(`{"id": 1}`))
//	    })
//
// # Common Patterns
//
// Testing list commands with JSON output:
//
//	func TestResourceListJSON(t *testing.T) {
//	    handler := newRouteHandler().
//	        On("GET", "/api/v1/accounts/1/resources", jsonResponse(200, `{
//	            "payload": [{"id": 1, "name": "first"}, {"id": 2, "name": "second"}]
//	        }`))
//
//	    setupTestEnvWithHandler(t, handler)
//
//	    output := captureStdout(t, func() {
//	        if err := Execute(context.Background(), []string{"resources", "list", "-o", "json"}); err != nil {
//	            t.Fatalf("failed: %v", err)
//	        }
//	    })
//
//	    items := decodeItems(t, output)  // Returns []map[string]any from {"items": [...]}
//	    if len(items) != 2 {
//	        t.Errorf("expected 2 items, got %d", len(items))
//	    }
//	}
//
// Testing error responses:
//
//	func TestResourceNotFound(t *testing.T) {
//	    handler := newRouteHandler().
//	        On("GET", "/api/v1/accounts/1/resources/999", jsonResponse(404, `{"error": "Not found"}`))
//
//	    setupTestEnvWithHandler(t, handler)
//
//	    err := Execute(context.Background(), []string{"resources", "get", "999"})
//	    if err == nil {
//	        t.Error("expected error for not found")
//	    }
//	}
//
// Testing multiple endpoints in one test:
//
//	func TestResourceWorkflow(t *testing.T) {
//	    handler := newRouteHandler().
//	        On("GET", "/api/v1/accounts/1/resources", jsonResponse(200, `{"payload": [...]}`)).
//	        On("POST", "/api/v1/accounts/1/resources", jsonResponse(201, `{"id": 1}`))
//
//	    setupTestEnvWithHandler(t, handler)
//	    // Test multiple commands...
//	}
package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// captureStdout executes a function and captures its stdout output.
// Use this to capture and verify command output in tests.
//
// Example:
//
//	output := captureStdout(t, func() {
//	    err := Execute(context.Background(), []string{"labels", "list"})
//	    if err != nil {
//	        t.Fatalf("failed: %v", err)
//	    }
//	})
//	if !strings.Contains(output, "expected text") {
//	    t.Error("missing expected text")
//	}
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// captureStderr executes a function and captures its stderr output.
// Use this to capture error messages or "no results" messages.
//
// Example:
//
//	output := captureStderr(t, func() {
//	    _ = Execute(context.Background(), []string{"labels", "list"})
//	})
//	if !strings.Contains(output, "No labels found") {
//	    t.Error("expected empty list message")
//	}
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// testEnv holds the original environment variables and restores them on cleanup.
// It also provides access to the mock test server.
type testEnv struct {
	t         *testing.T
	server    *httptest.Server
	origURL   string
	origToken string
	origAcct  string
}

// setupTestEnv creates a mock server with a simple handler and sets up the environment.
// Use this when you only need a single response handler for all requests.
// For routing multiple endpoints, use setupTestEnvWithHandler with a routeHandler instead.
//
// Example:
//
//	env := setupTestEnv(t, jsonResponse(200, `{"status": "ok"}`))
//	// env.server.URL contains the test server URL
func setupTestEnv(t *testing.T, handler http.HandlerFunc) *testEnv {
	t.Helper()

	server := httptest.NewServer(handler)

	env := &testEnv{
		t:         t,
		server:    server,
		origURL:   os.Getenv("CHATWOOT_BASE_URL"),
		origToken: os.Getenv("CHATWOOT_API_TOKEN"),
		origAcct:  os.Getenv("CHATWOOT_ACCOUNT_ID"),
	}

	_ = os.Setenv("CHATWOOT_BASE_URL", server.URL)
	_ = os.Setenv("CHATWOOT_API_TOKEN", "test-token")
	_ = os.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_OUTPUT", "text") // Ensure tests use text output by default

	t.Cleanup(func() {
		server.Close()
		_ = os.Setenv("CHATWOOT_BASE_URL", env.origURL)
		_ = os.Setenv("CHATWOOT_API_TOKEN", env.origToken)
		_ = os.Setenv("CHATWOOT_ACCOUNT_ID", env.origAcct)
	})

	return env
}

// setupTestEnvWithHandler creates a mock server with any http.Handler and sets up the environment.
// This is the preferred method for most tests as it works with routeHandler for multi-endpoint routing.
//
// The function automatically:
//   - Creates a test HTTP server
//   - Sets CHATWOOT_BASE_URL to point to the test server
//   - Sets CHATWOOT_API_TOKEN to "test-token"
//   - Sets CHATWOOT_ACCOUNT_ID to "1"
//   - Sets CHATWOOT_TESTING to skip URL validation
//   - Restores all original values on test cleanup
//
// Example with routeHandler:
//
//	handler := newRouteHandler().
//	    On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{"payload": []}`))
//	setupTestEnvWithHandler(t, handler)
//	// Now execute commands that will hit the mock server
func setupTestEnvWithHandler(t *testing.T, handler http.Handler) *testEnv {
	t.Helper()

	server := httptest.NewServer(handler)

	env := &testEnv{
		t:         t,
		server:    server,
		origURL:   os.Getenv("CHATWOOT_BASE_URL"),
		origToken: os.Getenv("CHATWOOT_API_TOKEN"),
		origAcct:  os.Getenv("CHATWOOT_ACCOUNT_ID"),
	}

	_ = os.Setenv("CHATWOOT_BASE_URL", server.URL)
	_ = os.Setenv("CHATWOOT_API_TOKEN", "test-token")
	_ = os.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")   // Skip URL validation for localhost
	t.Setenv("CHATWOOT_OUTPUT", "text") // Ensure tests use text output by default

	t.Cleanup(func() {
		server.Close()
		_ = os.Setenv("CHATWOOT_BASE_URL", env.origURL)
		_ = os.Setenv("CHATWOOT_API_TOKEN", env.origToken)
		_ = os.Setenv("CHATWOOT_ACCOUNT_ID", env.origAcct)
	})

	return env
}

// jsonResponse creates an http.HandlerFunc that returns a JSON response with the given status and body.
// This is the most common way to create mock responses.
//
// Example:
//
//	// Simple success response
//	jsonResponse(200, `{"id": 1, "name": "test"}`)
//
//	// List response with Chatwoot's typical "payload" wrapper
//	jsonResponse(200, `{"payload": [{"id": 1}, {"id": 2}]}`)
//
//	// Error response
//	jsonResponse(404, `{"error": "Not found"}`)
//
//	// Rate limit response with Retry-After header (use custom handler for headers)
func jsonResponse(statusCode int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}
}

// routeHandler is a test HTTP handler that routes requests based on method and path.
// It provides a fluent API for defining mock responses for different API endpoints.
//
// Routes are matched by exact "METHOD PATH" combination. If no route matches,
// it returns 404 Not Found.
//
// Example:
//
//	handler := newRouteHandler().
//	    On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{"payload": []}`)).
//	    On("POST", "/api/v1/accounts/1/labels", jsonResponse(201, `{"id": 1}`))
type routeHandler struct {
	routes map[string]http.HandlerFunc
}

// newRouteHandler creates a new routeHandler for defining mock API responses.
// Always use this with setupTestEnvWithHandler.
//
// Example:
//
//	handler := newRouteHandler().
//	    On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{"payload": []}`))
//	setupTestEnvWithHandler(t, handler)
func newRouteHandler() *routeHandler {
	return &routeHandler{routes: make(map[string]http.HandlerFunc)}
}

// On registers a handler for the given HTTP method and path.
// Returns the routeHandler to allow method chaining.
//
// Parameters:
//   - method: HTTP method (GET, POST, PATCH, PUT, DELETE)
//   - path: The full path including /api/v1/accounts/{id}/... prefix
//   - handler: An http.HandlerFunc (use jsonResponse() for simple cases)
//
// Example with jsonResponse:
//
//	handler := newRouteHandler().
//	    On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{"payload": []}`))
//
// Example with custom handler to inspect request body:
//
//	var receivedBody map[string]any
//	handler := newRouteHandler().
//	    On("POST", "/api/v1/accounts/1/labels", func(w http.ResponseWriter, r *http.Request) {
//	        _ = json.NewDecoder(r.Body).Decode(&receivedBody)
//	        w.Header().Set("Content-Type", "application/json")
//	        w.WriteHeader(http.StatusOK)
//	        _, _ = w.Write([]byte(`{"id": 1}`))
//	    })
//
// Example with custom headers (e.g., rate limiting):
//
//	handler := newRouteHandler().
//	    On("GET", "/api/v1/accounts/1/labels", func(w http.ResponseWriter, r *http.Request) {
//	        w.Header().Set("Content-Type", "application/json")
//	        w.Header().Set("Retry-After", "60")
//	        w.WriteHeader(429)
//	        _, _ = w.Write([]byte(`{"error": "Rate limited"}`))
//	    })
func (rh *routeHandler) On(method, path string, handler http.HandlerFunc) *routeHandler {
	rh.routes[method+" "+path] = handler
	return rh
}

// ServeHTTP implements http.Handler. It looks up the handler for the request's
// method and path combination. Returns 404 if no matching route is found.
func (rh *routeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Method + " " + r.URL.Path
	if handler, ok := rh.routes[key]; ok {
		handler(w, r)
		return
	}
	http.NotFound(w, r)
}

// TestTestInfrastructure validates that the test infrastructure works correctly
func TestTestInfrastructure(t *testing.T) {
	t.Run("setupTestEnv sets environment variables", func(t *testing.T) {
		env := setupTestEnv(t, jsonResponse(200, `{"status": "ok"}`))

		if os.Getenv("CHATWOOT_BASE_URL") != env.server.URL {
			t.Error("CHATWOOT_BASE_URL not set correctly")
		}
		if os.Getenv("CHATWOOT_API_TOKEN") != "test-token" {
			t.Error("CHATWOOT_API_TOKEN not set correctly")
		}
		if os.Getenv("CHATWOOT_ACCOUNT_ID") != "1" {
			t.Error("CHATWOOT_ACCOUNT_ID not set correctly")
		}
	})

	t.Run("routeHandler routes requests correctly", func(t *testing.T) {
		handler := newRouteHandler().
			On("GET", "/api/v1/test", jsonResponse(200, `{"method": "get"}`)).
			On("POST", "/api/v1/test", jsonResponse(201, `{"method": "post"}`))

		env := setupTestEnvWithHandler(t, handler)

		// Test GET request
		resp, err := http.Get(env.server.URL + "/api/v1/test")
		if err != nil {
			t.Fatalf("GET request failed: %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		// Test POST request
		resp, err = http.Post(env.server.URL+"/api/v1/test", "application/json", nil)
		if err != nil {
			t.Fatalf("POST request failed: %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != 201 {
			t.Errorf("expected status 201, got %d", resp.StatusCode)
		}

		// Test 404 for unknown route
		resp, err = http.Get(env.server.URL + "/api/v1/unknown")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != 404 {
			t.Errorf("expected status 404 for unknown route, got %d", resp.StatusCode)
		}
	})
}

// decodeItems parses JSON output from list commands and returns the items array.
// CLI list commands with -o json output use the format: {"items": [...], "meta": {...}}
// This helper extracts just the items array for easy assertion.
//
// Example:
//
//	output := captureStdout(t, func() {
//	    _ = Execute(context.Background(), []string{"labels", "list", "-o", "json"})
//	})
//	items := decodeItems(t, output)
//	if len(items) != 2 {
//	    t.Errorf("expected 2 items, got %d", len(items))
//	}
//	if items[0]["title"] != "expected" {
//	    t.Error("first item has wrong title")
//	}
func decodeItems(t *testing.T, output string) []map[string]any {
	t.Helper()
	var wrapper struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal([]byte(output), &wrapper); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}
	return wrapper.Items
}
