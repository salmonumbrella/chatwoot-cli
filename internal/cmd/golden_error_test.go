package cmd

import (
	"context"
	"net/http"
	"testing"
)

func TestGoldenErrorValidationText(t *testing.T) {
	output := captureStderr(t, func() {
		err := Execute(context.Background(), []string{"--color", "never", "labels", "create"})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	assertGolden(t, "error_validation.txt", output)
}

func TestGoldenErrorAPIText(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels/404", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Request-Id", "req-404")
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStderr(t, func() {
		err := Execute(context.Background(), []string{"--color", "never", "labels", "get", "404"})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	assertGolden(t, "error_api_404.txt", output)
}

func TestGoldenErrorValidationJSON(t *testing.T) {
	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"--color", "never", "labels", "create", "-o", "json"})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	assertGolden(t, "error_validation.json", output)
}
