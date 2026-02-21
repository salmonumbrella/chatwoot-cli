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

func TestCustomAttributesListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_attribute_definitions", jsonResponse(200, `[
			{"id": 1, "attribute_display_name": "Customer ID", "attribute_key": "customer_id", "attribute_model": "contact_attribute", "attribute_display_type": "text"},
			{"id": 2, "attribute_display_name": "Priority", "attribute_key": "priority", "attribute_model": "conversation_attribute", "attribute_display_type": "list"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-attributes", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-attributes list failed: %v", err)
	}

	if !strings.Contains(output, "Customer ID") {
		t.Errorf("output missing 'Customer ID': %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "KEY") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestCustomAttributesListCommand_Empty(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_attribute_definitions", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-attributes", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-attributes list failed: %v", err)
	}

	if !strings.Contains(output, "No custom attributes found") {
		t.Errorf("expected 'No custom attributes found' message, got: %s", output)
	}
}

func TestCustomAttributesListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_attribute_definitions", jsonResponse(200, `[
			{"id": 1, "attribute_display_name": "Customer ID", "attribute_key": "customer_id"}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-attributes", "list", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-attributes list failed: %v", err)
	}

	attrs := decodeItems(t, output)
	if len(attrs) != 1 {
		t.Errorf("expected 1 attribute, got %d", len(attrs))
	}
}

func TestCustomAttributesGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_attribute_definitions/123", jsonResponse(200, `{
			"id": 123, "attribute_display_name": "Test Attr", "attribute_key": "test_attr", "attribute_model": "contact_attribute", "attribute_display_type": "text"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-attributes", "get", "123"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-attributes get failed: %v", err)
	}

	if !strings.Contains(output, "Test Attr") {
		t.Errorf("output missing 'Test Attr': %s", output)
	}
	if !strings.Contains(output, "test_attr") {
		t.Errorf("output missing key 'test_attr': %s", output)
	}
}

func TestCustomAttributesGetCommand_AcceptsHashAndPrefixedIDs(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_attribute_definitions/123", jsonResponse(200, `{
			"id": 123, "attribute_display_name": "Test Attr", "attribute_key": "test_attr", "attribute_model": "contact_attribute", "attribute_display_type": "text"
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-attributes", "get", "#123"}); err != nil {
			t.Fatalf("custom-attributes get hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output, "Test Attr") {
		t.Errorf("output missing 'Test Attr': %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"custom-attributes", "get", "custom-attribute:123"}); err != nil {
			t.Fatalf("custom-attributes get prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "Test Attr") {
		t.Errorf("output missing 'Test Attr': %s", output2)
	}
}

func TestCustomAttributesGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_attribute_definitions/123", jsonResponse(200, `{
			"id": 123, "attribute_display_name": "Test Attr", "attribute_key": "test_attr"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-attributes", "get", "123", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-attributes get failed: %v", err)
	}

	var attr map[string]any
	if err := json.Unmarshal([]byte(output), &attr); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestCustomAttributesGetCommand_NotFound(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_attribute_definitions/999", jsonResponse(404, `{"error": "Not found"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"custom-attributes", "get", "999"})
	if err == nil {
		t.Error("expected error for not found custom attribute")
	}
}

func TestCustomAttributesGetCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"custom-attributes", "get", "abc"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestCustomAttributesCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/custom_attribute_definitions", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 456, "attribute_display_name": "Customer ID", "attribute_key": "customer_id", "attribute_model": "contact_attribute", "attribute_display_type": "text"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"custom-attributes", "create",
		"--name", "Customer ID",
		"--model", "contact",
		"--type", "text",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-attributes create failed: %v", err)
	}

	if !strings.Contains(output, "Created custom attribute 456") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestCustomAttributesCreateCommand_WithKey(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/custom_attribute_definitions", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 456, "attribute_display_name": "Customer ID", "attribute_key": "cust_id"}`))
		})

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{
		"custom-attributes", "create",
		"--name", "Customer ID",
		"--key", "cust_id",
		"--model", "contact",
		"--type", "text",
	})
	if err != nil {
		t.Errorf("custom-attributes create failed: %v", err)
	}

	if receivedBody["attribute_key"] != "cust_id" {
		t.Errorf("expected attribute_key 'cust_id', got %v", receivedBody["attribute_key"])
	}
}

func TestCustomAttributesCreateCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/custom_attribute_definitions", jsonResponse(200, `{
			"id": 456,
			"attribute_display_name": "Customer ID",
			"attribute_key": "customer_id"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"custom-attributes", "create",
		"--name", "Customer ID",
		"--model", "contact",
		"--type", "text",
		"-o", "json",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-attributes create failed: %v", err)
	}

	var attr map[string]any
	if err := json.Unmarshal([]byte(output), &attr); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}
}

func TestCustomAttributesCreateCommand_MissingName(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"custom-attributes", "create",
		"--model", "contact",
		"--type", "text",
	})
	if err == nil {
		t.Error("expected error when name is missing")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected '--name is required' error, got: %v", err)
	}
}

func TestCustomAttributesCreateCommand_MissingModel(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"custom-attributes", "create",
		"--name", "Test",
		"--type", "text",
	})
	if err == nil {
		t.Error("expected error when model is missing")
	}
	if !strings.Contains(err.Error(), "--model is required") {
		t.Errorf("expected '--model is required' error, got: %v", err)
	}
}

func TestCustomAttributesCreateCommand_MissingType(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"custom-attributes", "create",
		"--name", "Test",
		"--model", "contact",
	})
	if err == nil {
		t.Error("expected error when type is missing")
	}
	if !strings.Contains(err.Error(), "--type is required") {
		t.Errorf("expected '--type is required' error, got: %v", err)
	}
}

func TestCustomAttributesUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/custom_attribute_definitions/123", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 123, "attribute_display_name": "Updated Name"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"custom-attributes", "update", "123",
		"--name", "Updated Name",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-attributes update failed: %v", err)
	}

	if !strings.Contains(output, "Updated custom attribute 123") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestCustomAttributesUpdateCommand_MissingName(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"custom-attributes", "update", "123"})
	if err == nil {
		t.Error("expected error when name is missing")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected '--name is required' error, got: %v", err)
	}
}

func TestCustomAttributesUpdateCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"custom-attributes", "update", "abc", "--name", "test"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestCustomAttributesDeleteCommand(t *testing.T) {
	deleteRequested := false
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/custom_attribute_definitions/456", func(w http.ResponseWriter, r *http.Request) {
			deleteRequested = true
			w.WriteHeader(http.StatusOK)
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"custom-attributes", "delete", "456"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("custom-attributes delete failed: %v", err)
	}

	if !deleteRequested {
		t.Error("expected DELETE request to be made")
	}

	if !strings.Contains(output, "Deleted custom attribute 456") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestCustomAttributesDeleteCommand_InvalidID(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"custom-attributes", "delete", "abc"})
	if err == nil {
		t.Error("expected error for invalid ID")
	}
}

func TestCustomAttributesCommand_Alias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_attribute_definitions", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	// Test using 'attrs' alias
	err := Execute(context.Background(), []string{"attrs", "list"})
	if err != nil {
		t.Errorf("attrs alias should work: %v", err)
	}

	// Test using 'ca' alias
	err = Execute(context.Background(), []string{"ca", "list"})
	if err != nil {
		t.Errorf("ca alias should work: %v", err)
	}
}

func TestCustomAttributesCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/custom_attribute_definitions", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"custom-attributes", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}
