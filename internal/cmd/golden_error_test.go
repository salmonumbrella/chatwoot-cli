package cmd

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/iocontext"
	"github.com/chatwoot/chatwoot-cli/internal/outfmt"
	"github.com/spf13/cobra"
)

func runJSONError(t *testing.T, err error) string {
	t.Helper()

	var errOut bytes.Buffer
	cmd := &cobra.Command{
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			return err
		}),
	}

	ctx := outfmt.WithMode(context.Background(), outfmt.JSON)
	ctx = iocontext.WithIO(ctx, &iocontext.IO{Out: ioDiscard{}, ErrOut: &errOut, In: nil})
	cmd.SetContext(ctx)

	_ = cmd.RunE(cmd, []string{})
	return errOut.String()
}

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
	output := captureStderr(t, func() {
		err := Execute(context.Background(), []string{"--color", "never", "labels", "create", "-o", "json"})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	assertGolden(t, "error_validation.json", output)
}

func TestGoldenErrorRateLimitText(t *testing.T) {
	t.Setenv("CHATWOOT_MAX_RATE_LIMIT_RETRIES", "0")
	t.Setenv("CHATWOOT_RATE_LIMIT_DELAY", "0s")

	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStderr(t, func() {
		err := Execute(context.Background(), []string{"--color", "never", "labels", "list"})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})

	assertGolden(t, "error_rate_limit.txt", output)
}

func TestGoldenErrorCircuitBreakerText(t *testing.T) {
	output := HandleError(&api.CircuitBreakerError{})
	assertGolden(t, "error_circuit_breaker.txt", output)
}

func TestGoldenErrorRateLimitJSON(t *testing.T) {
	output := runJSONError(t, &api.RateLimitError{RetryAfter: 2 * time.Second})
	assertGolden(t, "error_rate_limit.json", output)
}

func TestGoldenErrorCircuitBreakerJSON(t *testing.T) {
	output := runJSONError(t, &api.CircuitBreakerError{})
	assertGolden(t, "error_circuit_breaker.json", output)
}
