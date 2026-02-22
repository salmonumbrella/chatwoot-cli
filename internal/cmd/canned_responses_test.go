package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestCannedResponsesListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 1, "short_code": "greeting", "content": "Hello! How can I help you?"},
			{"id": 2, "short_code": "thanks", "content": "Thank you for contacting us!"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"canned-responses", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses list failed: %v", err)
	}

	if !strings.Contains(output, "greeting") {
		t.Errorf("output missing 'greeting': %s", output)
	}
	if !strings.Contains(output, "thanks") {
		t.Errorf("output missing 'thanks': %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "SHORT_CODE") || !strings.Contains(output, "CONTENT") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestCannedResponsesListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 1, "short_code": "greeting", "content": "Hello!"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"canned-responses", "list", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses list failed: %v", err)
	}

	responses := decodeItems(t, output)
	if len(responses) != 1 {
		t.Errorf("expected 1 response, got %d", len(responses))
	}
}

func TestCannedResponsesListCommand_LongContent(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 1, "short_code": "long", "content": "This is a very long message that should be truncated in the table output to keep things readable and manageable for the user"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"canned-responses", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses list failed: %v", err)
	}

	if !strings.Contains(output, "...") {
		t.Errorf("expected truncated content with '...', got: %s", output)
	}
}

func TestCannedResponsesGetCommand(t *testing.T) {
	// GetCannedResponse calls ListCannedResponses and filters
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 123, "short_code": "test", "content": "Test content"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"canned-responses", "get", "123"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses get failed: %v", err)
	}

	if !strings.Contains(output, "test") {
		t.Errorf("output missing 'test': %s", output)
	}
	if !strings.Contains(output, "Test content") {
		t.Errorf("output missing content: %s", output)
	}
}

func TestCannedResponsesGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 123, "short_code": "test", "content": "Test content"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"canned-responses", "get", "123", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses get failed: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if response["short_code"] != "test" {
		t.Errorf("expected short_code 'test', got %v", response["short_code"])
	}
}

func TestCannedResponsesGetCommand_NotFound(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"canned-responses", "get", "999"})
	if err == nil {
		t.Error("expected error for not found canned response")
	}
}

func TestCannedResponsesGetCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"canned-responses", "get", "abc"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestCannedResponsesGetCommand_MissingID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"canned-responses", "get"})
	if err == nil {
		t.Error("expected error when ID is missing")
	}
}

func TestCannedResponsesCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/canned_responses", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 456, "short_code": "new-code", "content": "New content"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"canned-responses", "create",
		"--short-code", "new-code",
		"--content", "New content",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses create failed: %v", err)
	}

	if !strings.Contains(output, "Created canned response 456") {
		t.Errorf("expected success message, got: %s", output)
	}

	// Request body is nested under canned_response
	cannedResponse, ok := receivedBody["canned_response"].(map[string]any)
	if !ok {
		t.Fatalf("expected canned_response object, got %T", receivedBody["canned_response"])
	}
	if cannedResponse["short_code"] != "new-code" {
		t.Errorf("expected short_code 'new-code', got %v", cannedResponse["short_code"])
	}
	if cannedResponse["content"] != "New content" {
		t.Errorf("expected content 'New content', got %v", cannedResponse["content"])
	}
}

func TestCannedResponsesCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `{
			"id": 456,
			"short_code": "new-code",
			"content": "New content"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"canned-responses", "create",
		"--short-code", "new-code",
		"--content", "New content",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses create failed: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestCannedResponsesCreateCommand_MissingShortCode(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"canned-responses", "create",
		"--content", "Some content",
	})
	if err == nil {
		t.Error("expected error when short-code is missing")
	}
}

func TestCannedResponsesCreateCommand_MissingContent(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"canned-responses", "create",
		"--short-code", "some-code",
	})
	if err == nil {
		t.Error("expected error when content is missing")
	}
}

func TestCannedResponsesUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	// Update first fetches the existing response
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v1/accounts/1/canned_responses" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`[{"id": 123, "short_code": "old-code", "content": "Old content"}]`))
			return
		}
		if r.Method == "PATCH" && r.URL.Path == "/api/v1/accounts/1/canned_responses/123" {
			callCount++
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id": 123, "short_code": "new-code", "content": "Old content"}`))
			return
		}
		http.NotFound(w, r)
	})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"canned-responses", "update", "123",
		"--short-code", "new-code",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses update failed: %v", err)
	}

	if !strings.Contains(output, "Updated canned response 123") {
		t.Errorf("expected success message, got: %s", output)
	}

	if callCount != 1 {
		t.Errorf("expected 1 PATCH call, got %d", callCount)
	}
}

func TestCannedResponsesUpdateCommand_JSON(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v1/accounts/1/canned_responses" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`[{"id": 123, "short_code": "old-code", "content": "Old content"}]`))
			return
		}
		if r.Method == "PATCH" && r.URL.Path == "/api/v1/accounts/1/canned_responses/123" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id": 123, "short_code": "new-code", "content": "New content"}`))
			return
		}
		http.NotFound(w, r)
	})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"canned-responses", "update", "123",
		"--content", "New content",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses update failed: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestCannedResponsesUpdateCommand_NoFlags(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"canned-responses", "update", "123"})
	if err == nil {
		t.Error("expected error when no update flags provided")
	}
	if !strings.Contains(err.Error(), "at least one of --short-code or --content must be provided") {
		t.Errorf("expected 'at least one' error, got: %v", err)
	}
}

func TestCannedResponsesUpdateCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"canned-responses", "update", "abc", "--content", "test"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestCannedResponsesDeleteCommand(t *testing.T) {
	deleteRequested := false
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/canned_responses/456", func(w http.ResponseWriter, r *http.Request) {
			deleteRequested = true
			w.WriteHeader(http.StatusOK)
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"canned-responses", "delete", "456"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses delete failed: %v", err)
	}

	if !deleteRequested {
		t.Error("expected DELETE request to be made")
	}

	if !strings.Contains(output, "Deleted canned response 456") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestCannedResponsesDeleteCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"canned-responses", "delete", "abc"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestCannedResponsesDeleteCommand_MissingID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"canned-responses", "delete"})
	if err == nil {
		t.Error("expected error when ID is missing")
	}
}

func TestCannedResponsesCommand_Alias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	// Test using 'cr' alias
	err := Execute(context.Background(), []string{"cr", "list"})
	if err != nil {
		t.Errorf("cr alias should work: %v", err)
	}

	// Test using 'canned' alias
	err = Execute(context.Background(), []string{"canned", "list"})
	if err != nil {
		t.Errorf("canned alias should work: %v", err)
	}
}

func TestCannedResponsesCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"canned-responses", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestCannedResponsesSearchCommand_ByContent(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 1, "short_code": "greeting", "content": "Hello! How can I help you?"},
			{"id": 2, "short_code": "thanks", "content": "Thank you for contacting us!"},
			{"id": 3, "short_code": "bye", "content": "Goodbye, have a great day!"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"canned-responses", "search", "--query", "thank"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses search failed: %v", err)
	}

	// Should match "Thank you for contacting us!" (case-insensitive)
	if !strings.Contains(output, "thanks") {
		t.Errorf("expected 'thanks' in output, got: %s", output)
	}
	// Should NOT match other responses
	if strings.Contains(output, "greeting") {
		t.Errorf("should not contain 'greeting', got: %s", output)
	}
	if strings.Contains(output, "bye") {
		t.Errorf("should not contain 'bye', got: %s", output)
	}
}

func TestCannedResponsesSearchCommand_ByShortCode(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 1, "short_code": "greeting", "content": "Hello! How can I help you?"},
			{"id": 2, "short_code": "thanks", "content": "Thank you for contacting us!"},
			{"id": 3, "short_code": "bye", "content": "Goodbye, have a great day!"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"canned-responses", "search", "--query", "GREET"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses search failed: %v", err)
	}

	// Should match "greeting" short code (case-insensitive)
	if !strings.Contains(output, "greeting") {
		t.Errorf("expected 'greeting' in output, got: %s", output)
	}
	// Should NOT match other responses
	if strings.Contains(output, "thanks") {
		t.Errorf("should not contain 'thanks', got: %s", output)
	}
}

func TestCannedResponsesSearchCommand_NoMatches(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 1, "short_code": "greeting", "content": "Hello!"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"canned-responses", "search", "--query", "nonexistent"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses search failed: %v", err)
	}

	if !strings.Contains(output, "No canned responses found matching query") {
		t.Errorf("expected 'No canned responses found matching query' message, got: %s", output)
	}
}

func TestCannedResponsesSearchCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/canned_responses", jsonResponse(200, `[
			{"id": 1, "short_code": "greeting", "content": "Hello!"},
			{"id": 2, "short_code": "thanks", "content": "Thank you!"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"canned-responses", "search", "--query", "hello", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("canned-responses search failed: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if response["query"] != "hello" {
		t.Errorf("expected query 'hello', got %v", response["query"])
	}

	items, ok := response["items"].([]any)
	if !ok {
		t.Errorf("expected items array, got %T", response["items"])
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestCannedResponsesSearchCommand_MissingQuery(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"canned-responses", "search"})
	if err == nil {
		t.Error("expected error when query is missing")
	}
}
