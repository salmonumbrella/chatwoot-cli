package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestProfileGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/profile", jsonResponse(200, `{
			"id": 1,
			"name": "John Doe",
			"email": "john@example.com",
			"accounts": [
				{"id": 1, "name": "Test Account", "locale": "en"},
				{"id": 2, "name": "Other Account", "locale": "de"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"profile", "get"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("profile get failed: %v", err)
	}

	if !strings.Contains(output, "John Doe") {
		t.Errorf("output missing name: %s", output)
	}
	if !strings.Contains(output, "john@example.com") {
		t.Errorf("output missing email: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "EMAIL") {
		t.Errorf("output missing expected headers: %s", output)
	}
	// Check for available accounts section
	if !strings.Contains(output, "Available Accounts") {
		t.Errorf("output missing available accounts section: %s", output)
	}
	if !strings.Contains(output, "Test Account") {
		t.Errorf("output missing account name: %s", output)
	}
}

func TestProfileCommand_DefaultsToGet(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/profile", jsonResponse(200, `{
			"id": 1,
			"name": "John Doe",
			"email": "john@example.com",
			"accounts": []
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"profile"})
		if err != nil {
			t.Errorf("profile failed: %v", err)
		}
	})

	if !strings.Contains(output, "John Doe") {
		t.Errorf("output missing name: %s", output)
	}
	if !strings.Contains(output, "john@example.com") {
		t.Errorf("output missing email: %s", output)
	}
}

func TestProfileAlias_DefaultsToGet_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/profile", jsonResponse(200, `{
			"id": 7,
			"name": "Jane Doe",
			"email": "jane@example.com",
			"accounts": []
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"pr", "-j"})
		if err != nil {
			t.Errorf("pr -j failed: %v", err)
		}
	})

	var profile map[string]any
	if err := json.Unmarshal([]byte(output), &profile); err != nil {
		t.Fatalf("output is not valid JSON: %v, output: %s", err, output)
	}

	if profile["id"] != float64(7) {
		t.Errorf("expected id 7, got %v", profile["id"])
	}
	if profile["name"] != "Jane Doe" {
		t.Errorf("expected name 'Jane Doe', got %v", profile["name"])
	}
}

func TestProfileGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/profile", jsonResponse(200, `{
			"id": 1,
			"name": "John Doe",
			"email": "john@example.com",
			"accounts": [
				{"id": 1, "name": "Test Account", "locale": "en"}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"profile", "get", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("profile get failed: %v", err)
	}

	var profile map[string]any
	if err := json.Unmarshal([]byte(output), &profile); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if profile["name"] != "John Doe" {
		t.Errorf("expected name 'John Doe', got %v", profile["name"])
	}
	if profile["email"] != "john@example.com" {
		t.Errorf("expected email 'john@example.com', got %v", profile["email"])
	}
}

func TestProfileGetCommand_NoAccounts(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/profile", jsonResponse(200, `{
			"id": 1,
			"name": "John Doe",
			"email": "john@example.com",
			"accounts": []
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"profile", "get"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("profile get failed: %v", err)
	}

	// Should NOT show Available Accounts section when empty
	if strings.Contains(output, "Available Accounts") {
		t.Errorf("output should not show Available Accounts section when empty: %s", output)
	}
}

func TestProfileGetCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/profile", jsonResponse(401, `{"error": "Unauthorized"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"profile", "get"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}
