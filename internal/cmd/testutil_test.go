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

// captureStdout executes a function and captures its stdout output
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

// captureStderr executes a function and captures its stderr output
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

// testEnv holds the original environment and restores it on cleanup
type testEnv struct {
	t         *testing.T
	server    *httptest.Server
	origURL   string
	origToken string
	origAcct  string
}

// setupTestEnv creates a mock server and sets up environment for testing
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

	t.Cleanup(func() {
		server.Close()
		_ = os.Setenv("CHATWOOT_BASE_URL", env.origURL)
		_ = os.Setenv("CHATWOOT_API_TOKEN", env.origToken)
		_ = os.Setenv("CHATWOOT_ACCOUNT_ID", env.origAcct)
	})

	return env
}

// setupTestEnvWithHandler creates a mock server with a routeHandler and sets up environment for testing
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
	t.Setenv("CHATWOOT_TESTING", "1") // Skip URL validation for localhost

	t.Cleanup(func() {
		server.Close()
		_ = os.Setenv("CHATWOOT_BASE_URL", env.origURL)
		_ = os.Setenv("CHATWOOT_API_TOKEN", env.origToken)
		_ = os.Setenv("CHATWOOT_ACCOUNT_ID", env.origAcct)
	})

	return env
}

// jsonResponse is a helper to create JSON response handlers
func jsonResponse(statusCode int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}
}

// routeHandler routes requests based on method and path
type routeHandler struct {
	routes map[string]http.HandlerFunc
}

func newRouteHandler() *routeHandler {
	return &routeHandler{routes: make(map[string]http.HandlerFunc)}
}

func (rh *routeHandler) On(method, path string, handler http.HandlerFunc) *routeHandler {
	rh.routes[method+" "+path] = handler
	return rh
}

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
