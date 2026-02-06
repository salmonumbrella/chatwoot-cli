package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestJSONErrorIncludesRequestID(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels/404", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Request-Id", "req-404")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStderr(t, func() {
		err := Execute(context.Background(), []string{"labels", "get", "404", "-o", "json"})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("failed to parse JSON error output: %v", err)
	}

	if payload["code"] != "not_found" {
		t.Fatalf("expected code not_found, got %v", payload["code"])
	}
	if payload["message"] != "not found" {
		t.Fatalf("expected message not found, got %v", payload["message"])
	}
	if payload["retryable"] != false {
		t.Fatalf("expected retryable false, got %v", payload["retryable"])
	}
	if payload["suggestion"] != "Verify the resource ID exists" {
		t.Fatalf("unexpected suggestion: %v", payload["suggestion"])
	}

	contextVal, ok := payload["context"].(map[string]any)
	if !ok {
		t.Fatalf("expected context map, got %T", payload["context"])
	}
	if contextVal["request_id"] != "req-404" {
		t.Fatalf("expected request_id req-404, got %v", contextVal["request_id"])
	}
	if contextVal["status_code"] != float64(404) {
		t.Fatalf("expected status_code 404, got %v", contextVal["status_code"])
	}
}
