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

func TestLabelsListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{
			"payload": [
				{"id": 1, "title": "urgent", "color": "#FF0000", "description": "High priority", "show_on_sidebar": true},
				{"id": 2, "title": "bug", "color": "#00FF00", "description": "Bug reports", "show_on_sidebar": false}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"labels", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("labels list failed: %v", err)
	}

	// Verify output contains expected labels
	if !strings.Contains(output, "urgent") {
		t.Errorf("output missing 'urgent': %s", output)
	}
	if !strings.Contains(output, "bug") {
		t.Errorf("output missing 'bug': %s", output)
	}
	if !strings.Contains(output, "#FF0000") {
		t.Errorf("output missing color '#FF0000': %s", output)
	}
	// Check headers
	if !strings.Contains(output, "ID") || !strings.Contains(output, "TITLE") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestLabelsListCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{"payload": []}`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	output := captureStderr(t, func() {
		err := Execute(context.Background(), []string{"labels", "list"})
		if err != nil {
			t.Errorf("labels list failed: %v", err)
		}
	})

	if !strings.Contains(output, "No labels found") {
		t.Errorf("expected 'No labels found' message, got: %s", output)
	}
}

func TestLabelsListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{
			"payload": [
				{"id": 1, "title": "urgent", "color": "#FF0000"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"labels", "list", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("labels list failed: %v", err)
	}

	labels := decodeItems(t, output)
	if len(labels) != 1 {
		t.Errorf("expected 1 label, got %d", len(labels))
	}
}

func TestLabelsGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels/123", jsonResponse(200, `{
			"id": 123,
			"title": "important",
			"color": "#0000FF",
			"description": "Important issues",
			"show_on_sidebar": true
		}`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"labels", "get", "123"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("labels get failed: %v", err)
	}

	// Verify output contains label details
	if !strings.Contains(output, "important") {
		t.Errorf("output missing 'important': %s", output)
	}
	if !strings.Contains(output, "#0000FF") {
		t.Errorf("output missing color: %s", output)
	}
	if !strings.Contains(output, "Important issues") {
		t.Errorf("output missing description: %s", output)
	}
}

func TestLabelsGetCommand_AcceptsHashAndPrefixedIDs(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels/123", jsonResponse(200, `{
			"id": 123,
			"title": "important",
			"color": "#0000FF",
			"description": "Important issues",
			"show_on_sidebar": true
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"labels", "get", "#123"}); err != nil {
			t.Fatalf("labels get hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output, "important") {
		t.Errorf("output missing title: %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"labels", "get", "label:123"}); err != nil {
			t.Fatalf("labels get prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "important") {
		t.Errorf("output missing title: %s", output2)
	}
}

func TestLabelsGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels/123", jsonResponse(200, `{
			"id": 123,
			"title": "important",
			"color": "#0000FF",
			"description": "Important issues",
			"show_on_sidebar": true
		}`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"labels", "get", "123", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("labels get failed: %v", err)
	}

	// Verify it's valid JSON
	var label map[string]any
	if err := json.Unmarshal([]byte(output), &label); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if label["title"] != "important" {
		t.Errorf("expected title 'important', got %v", label["title"])
	}
}

func TestLabelsGetCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"labels", "get", "abc"})
	if err == nil {
		t.Error("expected error for invalid label ID")
	}

	err = Execute(context.Background(), []string{"labels", "get", "-1"})
	if err == nil {
		t.Error("expected error for negative label ID")
	}
}

func TestLabelsGetCommand_MissingID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"labels", "get"})
	if err == nil {
		t.Error("expected error when label ID is missing")
	}
}

func TestLabelsCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/labels", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 3, "title": "new-label", "color": "#0000FF", "description": "A new label", "show_on_sidebar": true}`))
		})

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"labels", "create",
		"--title", "new-label",
		"--color", "#0000FF",
		"--description", "A new label",
		"--show-on-sidebar",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("labels create failed: %v", err)
	}

	// Verify output
	if !strings.Contains(output, "Created label 3") {
		t.Errorf("expected success message, got: %s", output)
	}

	// Verify request body
	if receivedBody["title"] != "new-label" {
		t.Errorf("expected title 'new-label', got %v", receivedBody["title"])
	}
	if receivedBody["color"] != "#0000FF" {
		t.Errorf("expected color '#0000FF', got %v", receivedBody["color"])
	}
	if receivedBody["description"] != "A new label" {
		t.Errorf("expected description 'A new label', got %v", receivedBody["description"])
	}
	if receivedBody["show_on_sidebar"] != true {
		t.Errorf("expected show_on_sidebar true, got %v", receivedBody["show_on_sidebar"])
	}
}

func TestLabelsCreateCommand_MinimalOptions(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/labels", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 4, "title": "minimal", "show_on_sidebar": false}`))
		})

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"labels", "create", "--title", "minimal"})
	if err != nil {
		t.Errorf("labels create failed: %v", err)
	}

	// Verify only required fields are sent
	if receivedBody["title"] != "minimal" {
		t.Errorf("expected title 'minimal', got %v", receivedBody["title"])
	}
	// Color and description should not be in the body when not specified
	if _, ok := receivedBody["color"]; ok {
		t.Errorf("color should not be in body when not specified")
	}
	if _, ok := receivedBody["description"]; ok {
		t.Errorf("description should not be in body when not specified")
	}
}

func TestLabelsCreateCommand_MissingTitle(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"labels", "create"})
	if err == nil {
		t.Error("expected error when title is missing")
	}
	if !strings.Contains(err.Error(), "--title is required") {
		t.Errorf("expected '--title is required' error, got: %v", err)
	}
}

func TestLabelsCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/labels", jsonResponse(200, `{
			"id": 5,
			"title": "json-label",
			"color": "#FF00FF"
		}`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"labels", "create",
		"--title", "json-label",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("labels create failed: %v", err)
	}

	// Verify it's valid JSON
	var label map[string]any
	if err := json.Unmarshal([]byte(output), &label); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if label["title"] != "json-label" {
		t.Errorf("expected title 'json-label', got %v", label["title"])
	}
}

func TestLabelsUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/labels/10", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 10, "title": "updated-title", "color": "#00FFFF"}`))
		})

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"labels", "update", "10",
		"--title", "updated-title",
		"--color", "#00FFFF",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("labels update failed: %v", err)
	}

	// Verify output
	if !strings.Contains(output, "Updated label 10") {
		t.Errorf("expected success message, got: %s", output)
	}

	// Verify request body
	if receivedBody["title"] != "updated-title" {
		t.Errorf("expected title 'updated-title', got %v", receivedBody["title"])
	}
	if receivedBody["color"] != "#00FFFF" {
		t.Errorf("expected color '#00FFFF', got %v", receivedBody["color"])
	}
}

func TestLabelsUpdateCommand_PartialUpdate(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/labels/20", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 20, "title": "only-color-updated", "color": "#ABCDEF"}`))
		})

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"labels", "update", "20", "--color", "#ABCDEF"})
	if err != nil {
		t.Errorf("labels update failed: %v", err)
	}

	// Verify only color is in body
	if receivedBody["color"] != "#ABCDEF" {
		t.Errorf("expected color '#ABCDEF', got %v", receivedBody["color"])
	}
	if _, ok := receivedBody["title"]; ok {
		t.Errorf("title should not be in body when not specified")
	}
}

func TestLabelsUpdateCommand_ShowOnSidebar(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/labels/30", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 30, "title": "sidebar-test", "show_on_sidebar": true}`))
		})

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"labels", "update", "30", "--show-on-sidebar"})
	if err != nil {
		t.Errorf("labels update failed: %v", err)
	}

	// Verify show_on_sidebar is in body
	if receivedBody["show_on_sidebar"] != true {
		t.Errorf("expected show_on_sidebar true, got %v", receivedBody["show_on_sidebar"])
	}
}

func TestLabelsUpdateCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"labels", "update", "abc", "--title", "test"})
	if err == nil {
		t.Error("expected error for invalid label ID")
	}
}

func TestLabelsUpdateCommand_MissingID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"labels", "update", "--title", "test"})
	if err == nil {
		t.Error("expected error when label ID is missing")
	}
}

func TestLabelsDeleteCommand(t *testing.T) {
	deleteRequested := false
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/labels/50", func(w http.ResponseWriter, r *http.Request) {
			deleteRequested = true
			w.WriteHeader(http.StatusOK)
		})

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"labels", "delete", "50"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("labels delete failed: %v", err)
	}

	if !deleteRequested {
		t.Error("expected DELETE request to be made")
	}

	if !strings.Contains(output, "Deleted label 50") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestLabelsDeleteCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"labels", "delete", "abc"})
	if err == nil {
		t.Error("expected error for invalid label ID")
	}
}

func TestLabelsDeleteCommand_MissingID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"labels", "delete"})
	if err == nil {
		t.Error("expected error when label ID is missing")
	}
}

func TestLabelsCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"labels", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

func TestLabelsGetCommand_NotFound(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels/999", jsonResponse(404, `{"error": "Label not found"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"labels", "get", "999"})
	if err == nil {
		t.Error("expected error for not found label")
	}
}

func TestLabelsCommand_Alias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{"payload": []}`))

	setupTestEnvWithHandler(t, handler)

	// Test using 'label' alias instead of 'labels'
	err := Execute(context.Background(), []string{"label", "list"})
	if err != nil {
		t.Errorf("label alias should work: %v", err)
	}
}

func TestLabelsListCommand_LongDescription(t *testing.T) {
	// Test that long descriptions are truncated
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{
			"payload": [
				{"id": 1, "title": "test", "color": "#000000", "description": "This is a very long description that should be truncated in the table output to keep things readable"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"labels", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("labels list failed: %v", err)
	}

	// Description should be truncated with "..."
	if !strings.Contains(output, "...") {
		t.Errorf("expected truncated description with '...', got: %s", output)
	}
}
