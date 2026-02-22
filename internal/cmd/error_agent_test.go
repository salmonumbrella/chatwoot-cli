package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestAgentErrorIsWrapped(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels/404", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Request-Id", "req-404")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStderr(t, func() {
		err := Execute(context.Background(), []string{"labels", "get", "404", "-o", "agent"})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if payload["kind"] != "labels.get" {
		t.Fatalf("expected kind labels.get, got %#v", payload["kind"])
	}
	errObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %#v", payload["error"])
	}
	if errObj["code"] != "not_found" {
		t.Fatalf("expected code not_found, got %#v", errObj["code"])
	}
}
