package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestContactsCreateCommand_StdinJSON(t *testing.T) {
	var receivedBody map[string]any

	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"payload": {
					"contact": {"id": 123, "name": "JSON Test", "email": "json@test.com", "phone_number": "+1234567890"}
				}
			}`))
		})

	setupTestEnvWithHandler(t, handler)

	// Create a pipe to mock stdin with JSON input
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	// Write JSON input to the pipe
	go func() {
		_, _ = w.Write([]byte(`{"name": "JSON Test", "email": "json@test.com", "phone_number": "+1234567890"}`))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "create", "--json"})
		if err != nil {
			t.Errorf("contacts create --json failed: %v", err)
		}
	})

	// Verify the API received the correct data
	if receivedBody["name"] != "JSON Test" {
		t.Errorf("expected name 'JSON Test', got %v", receivedBody["name"])
	}
	if receivedBody["email"] != "json@test.com" {
		t.Errorf("expected email 'json@test.com', got %v", receivedBody["email"])
	}
	if receivedBody["phone_number"] != "+1234567890" {
		t.Errorf("expected phone_number '+1234567890', got %v", receivedBody["phone_number"])
	}

	// Verify output contains the created contact
	if !strings.Contains(output, "JSON Test") {
		t.Errorf("output missing 'JSON Test': %s", output)
	}
}

func TestContactsCreateCommand_StdinJSON_FlagsOverride(t *testing.T) {
	var receivedBody map[string]any

	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"payload": {
					"contact": {"id": 123, "name": "Override Name", "email": "json@test.com", "phone_number": "+9999999999"}
				}
			}`))
		})

	setupTestEnvWithHandler(t, handler)

	// Create a pipe to mock stdin with JSON input
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	// Write JSON input with some values
	go func() {
		_, _ = w.Write([]byte(`{"name": "JSON Name", "email": "json@test.com", "phone_number": "+1234567890"}`))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		// Flags should override JSON values
		err := Execute(context.Background(), []string{
			"contacts", "create", "--json",
			"--name", "Override Name",
			"--phone", "+9999999999",
		})
		if err != nil {
			t.Errorf("contacts create --json with overrides failed: %v", err)
		}
	})

	// Verify flags took precedence over JSON
	if receivedBody["name"] != "Override Name" {
		t.Errorf("expected name 'Override Name' (from flag), got %v", receivedBody["name"])
	}
	// Email should come from JSON since no flag was provided
	if receivedBody["email"] != "json@test.com" {
		t.Errorf("expected email 'json@test.com' (from JSON), got %v", receivedBody["email"])
	}
	// Phone should be overridden by flag
	if receivedBody["phone_number"] != "+9999999999" {
		t.Errorf("expected phone_number '+9999999999' (from flag), got %v", receivedBody["phone_number"])
	}

	if !strings.Contains(output, "Override Name") {
		t.Errorf("output missing 'Override Name': %s", output)
	}
}

func TestContactsCreateCommand_StdinJSON_InvalidJSON(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	// Create a pipe to mock stdin with invalid JSON
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	// Write invalid JSON to the pipe
	go func() {
		_, _ = w.Write([]byte(`{invalid json}`))
		_ = w.Close()
	}()

	err = Execute(context.Background(), []string{"contacts", "create", "--json"})
	if err == nil {
		t.Error("expected error for invalid JSON input")
	}
}

func TestContactsCreateCommand_StdinJSON_EmptyStdin(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	// Create a pipe to mock empty stdin
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	// Close immediately to simulate empty stdin
	_ = w.Close()

	err = Execute(context.Background(), []string{"contacts", "create", "--json"})
	if err == nil {
		t.Error("expected error for empty stdin when --json is set")
	}
}

func TestContactsCreateCommand_StdinJSON_NameMissing(t *testing.T) {
	setupTestEnv(t, jsonResponse(200, `{}`))

	// Create a pipe to mock stdin with JSON missing name
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	// Write JSON without name field
	go func() {
		_, _ = w.Write([]byte(`{"email": "test@example.com"}`))
		_ = w.Close()
	}()

	err = Execute(context.Background(), []string{"contacts", "create", "--json"})
	if err == nil {
		t.Error("expected error when name is missing from JSON and not provided via flag")
	}
}

func TestContactsCreateCommand_StdinJSON_WithAdditionalFields(t *testing.T) {
	var receivedBody map[string]any

	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/contacts", func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"payload": {
					"contact": {"id": 123, "name": "Full Contact", "email": "full@test.com"}
				}
			}`))
		})

	setupTestEnvWithHandler(t, handler)

	// Create a pipe to mock stdin with additional fields
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	// Write JSON with additional fields like identifier and custom_attributes
	go func() {
		_, _ = w.Write([]byte(`{
			"name": "Full Contact",
			"email": "full@test.com",
			"identifier": "ext-123",
			"custom_attributes": {"plan": "enterprise", "region": "us-west"}
		}`))
		_ = w.Close()
	}()

	_ = captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "create", "--json"})
		if err != nil {
			t.Errorf("contacts create --json with additional fields failed: %v", err)
		}
	})

	// Verify all fields were passed through
	if receivedBody["identifier"] != "ext-123" {
		t.Errorf("expected identifier 'ext-123', got %v", receivedBody["identifier"])
	}

	customAttrs, ok := receivedBody["custom_attributes"].(map[string]any)
	if !ok {
		t.Errorf("expected custom_attributes to be a map, got %T", receivedBody["custom_attributes"])
	} else {
		if customAttrs["plan"] != "enterprise" {
			t.Errorf("expected custom_attributes.plan 'enterprise', got %v", customAttrs["plan"])
		}
		if customAttrs["region"] != "us-west" {
			t.Errorf("expected custom_attributes.region 'us-west', got %v", customAttrs["region"])
		}
	}
}
