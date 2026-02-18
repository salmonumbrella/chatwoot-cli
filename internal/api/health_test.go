package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectOK     bool
	}{
		{
			name:         "healthy server",
			statusCode:   http.StatusOK,
			responseBody: `{"status":"woot"}`,
			expectOK:     true,
		},
		{
			name:       "unhealthy server",
			statusCode: http.StatusServiceUnavailable,
			expectOK:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/health" {
					t.Errorf("expected /health path, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
				if tt.responseBody != "" {
					_, _ = w.Write([]byte(tt.responseBody))
				}
			}))
			defer server.Close()

			client := newTestClient(server.URL, "test-token", 123)
			ok, err := client.HealthCheck(context.Background())

			if tt.expectOK {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !ok {
					t.Error("expected healthy but got unhealthy")
				}
			} else {
				if ok {
					t.Error("expected unhealthy but got healthy")
				}
			}
		})
	}
}
