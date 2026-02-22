package cmd

import (
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/config"
)

func TestClientFactory_RetryOverrides(t *testing.T) {
	flags = rootFlags{
		Output:  "text",
		Color:   "auto",
		Timeout: api.DefaultTimeout,
	}

	flags.MaxRateLimitRetries = 7
	flags.MaxRateLimitRetriesSet = true
	flags.CircuitBreakerThreshold = 9
	flags.CircuitBreakerThresholdSet = true

	factory := newClientFactory()
	client := factory.newClient(config.ClientConfig{
		BaseURL:   "https://example.com",
		Token:     "token",
		AccountID: 1,
	})

	if client.RetryConfig.MaxRateLimitRetries != 7 {
		t.Fatalf("expected MaxRateLimitRetries=7, got %d", client.RetryConfig.MaxRateLimitRetries)
	}
	if client.RetryConfig.CircuitBreakerThreshold != 9 {
		t.Fatalf("expected CircuitBreakerThreshold=9, got %d", client.RetryConfig.CircuitBreakerThreshold)
	}
}
