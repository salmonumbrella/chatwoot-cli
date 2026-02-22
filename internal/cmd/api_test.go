package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAPICmdGetRequest(t *testing.T) {
	var receivedMethod, receivedPath string
	var receivedToken string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedToken = r.Header.Get("api_access_token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id": 123, "name": "Test Conversation"}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"api", "/conversations/123"})
		if err != nil {
			t.Errorf("api command failed: %v", err)
		}
	})

	if receivedMethod != "GET" {
		t.Errorf("expected GET method, got %s", receivedMethod)
	}
	if receivedPath != "/api/v1/accounts/1/conversations/123" {
		t.Errorf("expected path /api/v1/accounts/1/conversations/123, got %s", receivedPath)
	}
	if receivedToken != "test-token" {
		t.Errorf("expected token 'test-token', got %s", receivedToken)
	}
	if !strings.Contains(output, "Test Conversation") {
		t.Errorf("output missing expected content: %s", output)
	}
}

func TestAPICmdPostRequest(t *testing.T) {
	var receivedMethod string
	var receivedBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id": 456, "status": "created"}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/conversations",
			"-X", "POST",
			"-f", "status=open",
			"-f", "priority=high",
		})
		if err != nil {
			t.Errorf("api POST command failed: %v", err)
		}
	})

	if receivedMethod != "POST" {
		t.Errorf("expected POST method, got %s", receivedMethod)
	}
	if receivedBody["status"] != "open" {
		t.Errorf("expected status=open in body, got %v", receivedBody)
	}
	if receivedBody["priority"] != "high" {
		t.Errorf("expected priority=high in body, got %v", receivedBody)
	}
	if !strings.Contains(output, "created") {
		t.Errorf("output missing expected content: %s", output)
	}
}

func TestAPICmdRawField(t *testing.T) {
	var receivedBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	_ = captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/conversations/123",
			"-X", "PATCH",
			"-F", `labels=["bug", "urgent"]`,
			"-f", "status=resolved",
		})
		if err != nil {
			t.Errorf("api command with raw field failed: %v", err)
		}
	})

	// Raw field should be parsed as JSON array
	labels, ok := receivedBody["labels"].([]any)
	if !ok {
		t.Errorf("expected labels to be array, got %T: %v", receivedBody["labels"], receivedBody["labels"])
	} else if len(labels) != 2 || labels[0] != "bug" || labels[1] != "urgent" {
		t.Errorf("expected labels=[bug, urgent], got %v", labels)
	}
	if receivedBody["status"] != "resolved" {
		t.Errorf("expected status=resolved, got %v", receivedBody["status"])
	}
}

func TestAPICmdInputFromStdin(t *testing.T) {
	var receivedBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	// Create a temp file with JSON content to simulate stdin
	tmpfile, err := os.CreateTemp("", "input*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	jsonContent := `{"name": "Test Contact", "email": "test@example.com"}`
	if _, err := tmpfile.WriteString(jsonContent); err != nil {
		t.Fatal(err)
	}
	if _, err := tmpfile.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	_ = captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/contacts",
			"-X", "POST",
			"-i", tmpfile.Name(),
		})
		if err != nil {
			t.Errorf("api command with input file failed: %v", err)
		}
	})

	if receivedBody["name"] != "Test Contact" {
		t.Errorf("expected name='Test Contact', got %v", receivedBody["name"])
	}
	if receivedBody["email"] != "test@example.com" {
		t.Errorf("expected email='test@example.com', got %v", receivedBody["email"])
	}
}

func TestAPICmdJqFilter(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"payload": [{"id": 1, "name": "First"}, {"id": 2, "name": "Second"}]}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/contacts",
			"--output", "json",
			"--jq", ".payload[0].name",
		})
		if err != nil {
			t.Errorf("api command with jq filter failed: %v", err)
		}
	})

	output = strings.TrimSpace(output)
	if output != `"First"` {
		t.Errorf("expected jq output '\"First\"', got %q", output)
	}
}

func TestAPICmdSilentMode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id": 123}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/conversations/123",
			"--silent",
		})
		if err != nil {
			t.Errorf("api command with silent mode failed: %v", err)
		}
	})

	if output != "" {
		t.Errorf("expected no output in silent mode, got: %s", output)
	}
}

func TestAPICmdIncludeHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id": 123}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/conversations/123",
			"--include",
		})
		if err != nil {
			t.Errorf("api command with --include failed: %v", err)
		}
	})

	if !strings.Contains(output, "X-Custom-Header") {
		t.Errorf("output missing header 'X-Custom-Header': %s", output)
	}
	if !strings.Contains(output, "test-value") {
		t.Errorf("output missing header value 'test-value': %s", output)
	}
}

func TestAPICmdJSONOutputArray(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`[{"id": 1}, {"id": 2}]`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/conversations",
			"--output", "json",
		})
		if err != nil {
			t.Errorf("api command with JSON output failed: %v", err)
		}
	})

	var parsed any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if _, ok := parsed.([]any); !ok {
		t.Fatalf("expected JSON array output, got %T: %v", parsed, parsed)
	}
}

func TestAPICmdJSONOutputNonJSONBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("plain text"))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/health",
			"--output", "json",
		})
		if err != nil {
			t.Errorf("api command with JSON output failed: %v", err)
		}
	})

	var parsed string
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if parsed != "plain text" {
		t.Fatalf("expected JSON string output 'plain text', got %q", parsed)
	}
}

func TestAPICmdJSONIncludeHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id": 123}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/conversations/123",
			"--output", "json",
			"--include",
		})
		if err != nil {
			t.Errorf("api command with JSON include failed: %v", err)
		}
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}
	if parsed["status"] != float64(200) {
		t.Fatalf("expected status 200, got %v", parsed["status"])
	}
	headers, ok := parsed["headers"].(map[string]any)
	if !ok {
		t.Fatalf("expected headers object, got %T: %v", parsed["headers"], parsed["headers"])
	}
	if headers["X-Custom-Header"] == nil {
		t.Fatalf("expected X-Custom-Header in headers, got %v", headers)
	}
	body, ok := parsed["body"].(map[string]any)
	if !ok || body["id"] != float64(123) {
		t.Fatalf("expected body with id=123, got %T: %v", parsed["body"], parsed["body"])
	}
}

func TestAPICmdDeleteMethod(t *testing.T) {
	var receivedMethod string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.WriteHeader(204)
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	_ = captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/conversations/123",
			"-X", "DELETE",
		})
		if err != nil {
			t.Errorf("api DELETE command failed: %v", err)
		}
	})

	if receivedMethod != "DELETE" {
		t.Errorf("expected DELETE method, got %s", receivedMethod)
	}
}

func TestAPICmdPutMethod(t *testing.T) {
	var receivedMethod string
	var receivedBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"updated": true}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	_ = captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/conversations/123",
			"-X", "PUT",
			"-f", "status=resolved",
		})
		if err != nil {
			t.Errorf("api PUT command failed: %v", err)
		}
	})

	if receivedMethod != "PUT" {
		t.Errorf("expected PUT method, got %s", receivedMethod)
	}
	if receivedBody["status"] != "resolved" {
		t.Errorf("expected status=resolved, got %v", receivedBody["status"])
	}
}

func TestAPICmdMissingEndpoint(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{"api"})
	if err == nil {
		t.Error("expected error for missing endpoint argument")
	}
}

func TestAPICmdInvalidMethod(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"api", "/conversations",
		"-X", "INVALID",
	})
	if err == nil {
		t.Error("expected error for invalid HTTP method")
	}
}

func TestAPICmdInvalidFieldFormat(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"api", "/conversations",
		"-X", "POST",
		"-f", "invalid-no-equals",
	})
	if err == nil {
		t.Error("expected error for invalid field format")
	}
}

func TestAPICmdInvalidRawFieldJSON(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"api", "/conversations",
		"-X", "POST",
		"-F", "data=not-valid-json",
	})
	if err == nil {
		t.Error("expected error for invalid JSON in raw field")
	}
}

func TestAPICmdInvalidJqQuery(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id": 123}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{
		"api", "/conversations/123",
		"--output", "json",
		"--jq", ".invalid[",
	})
	if err == nil {
		t.Error("expected error for invalid jq query")
	}
}

func TestAPICmdAPIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"error": "Not Found"}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	err := Execute(context.Background(), []string{"api", "/conversations/99999"})
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestAPICmdPatchMethod(t *testing.T) {
	var receivedMethod string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	_ = captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/conversations/123",
			"-X", "PATCH",
			"-f", "status=resolved",
		})
		if err != nil {
			t.Errorf("api PATCH command failed: %v", err)
		}
	})

	if receivedMethod != "PATCH" {
		t.Errorf("expected PATCH method, got %s", receivedMethod)
	}
}

func TestAPICmdNumericField(t *testing.T) {
	var receivedBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	_ = captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/conversations/123",
			"-X", "PATCH",
			"-F", "assignee_id=42",
		})
		if err != nil {
			t.Errorf("api command with numeric field failed: %v", err)
		}
	})

	// Numeric raw fields should be parsed as numbers
	if receivedBody["assignee_id"] != float64(42) {
		t.Errorf("expected assignee_id=42 (number), got %T: %v", receivedBody["assignee_id"], receivedBody["assignee_id"])
	}
}

func TestAPICmdBooleanField(t *testing.T) {
	var receivedBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	setupTestEnv(t, handler)
	t.Setenv("CHATWOOT_TESTING", "1")

	_ = captureStdout(t, func() {
		err := Execute(context.Background(), []string{
			"api", "/conversations/123",
			"-X", "PATCH",
			"-F", "muted=true",
		})
		if err != nil {
			t.Errorf("api command with boolean field failed: %v", err)
		}
	})

	// Boolean raw fields should be parsed as booleans
	if receivedBody["muted"] != true {
		t.Errorf("expected muted=true (boolean), got %T: %v", receivedBody["muted"], receivedBody["muted"])
	}
}

// Test DoRaw method in the API client
func TestClientDoRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("api_access_token") != "test-token" {
			t.Errorf("expected token 'test-token', got %s", r.Header.Get("api_access_token"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Test-Header", "test-value")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"test": "data"}`))
	}))
	defer server.Close()

	t.Setenv("CHATWOOT_BASE_URL", server.URL)
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")
	t.Setenv("CHATWOOT_TESTING", "1")

	client, err := getClient()
	if err != nil {
		t.Fatalf("failed to get client: %v", err)
	}

	respBody, headers, statusCode, err := client.DoRaw(context.Background(), "GET", "/test-path", nil)
	if err != nil {
		t.Fatalf("DoRaw failed: %v", err)
	}

	if statusCode != 200 {
		t.Errorf("expected status 200, got %d", statusCode)
	}
	if !bytes.Contains(respBody, []byte("test")) {
		t.Errorf("expected response body to contain 'test', got %s", string(respBody))
	}
	if headers.Get("X-Test-Header") != "test-value" {
		t.Errorf("expected header X-Test-Header=test-value, got %s", headers.Get("X-Test-Header"))
	}
}
