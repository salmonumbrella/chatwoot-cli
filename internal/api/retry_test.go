package api

import (
	"testing"
	"time"
)

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := &circuitBreaker{}
	cb.recordFailure()
	cb.recordFailure()
	cb.recordSuccess()

	if cb.failures != 0 {
		t.Error("recordSuccess should reset failures to 0")
	}
	if cb.isOpen() {
		t.Error("circuit should be closed after success")
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := &circuitBreaker{}

	for i := 0; i < CircuitBreakerThreshold; i++ {
		cb.recordFailure()
	}

	if !cb.isOpen() {
		t.Errorf("circuit should be open after %d failures", CircuitBreakerThreshold)
	}
}

func TestCircuitBreaker_CustomThreshold(t *testing.T) {
	cb := &circuitBreaker{threshold: 2}

	cb.recordFailure()
	if cb.isOpen() {
		t.Error("circuit should not be open after 1 failure with threshold 2")
	}

	cb.recordFailure()
	if !cb.isOpen() {
		t.Error("circuit should be open after 2 failures with threshold 2")
	}
}

func TestCircuitBreaker_CustomResetTime(t *testing.T) {
	cb := &circuitBreaker{threshold: 1, resetTime: 10 * time.Millisecond}

	cb.recordFailure()
	if !cb.isOpen() {
		t.Error("circuit should be open after 1 failure")
	}

	time.Sleep(15 * time.Millisecond)
	if cb.isOpen() {
		t.Error("circuit should auto-close after reset time")
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxRateLimitRetries != DefaultMaxRateLimitRetries {
		t.Errorf("MaxRateLimitRetries = %d, want %d", cfg.MaxRateLimitRetries, DefaultMaxRateLimitRetries)
	}
	if cfg.Max5xxRetries != DefaultMax5xxRetries {
		t.Errorf("Max5xxRetries = %d, want %d", cfg.Max5xxRetries, DefaultMax5xxRetries)
	}
	if cfg.RateLimitBaseDelay != DefaultRateLimitBaseDelay {
		t.Errorf("RateLimitBaseDelay = %v, want %v", cfg.RateLimitBaseDelay, DefaultRateLimitBaseDelay)
	}
	if cfg.ServerErrorRetryDelay != DefaultServerErrorRetryDelay {
		t.Errorf("ServerErrorRetryDelay = %v, want %v", cfg.ServerErrorRetryDelay, DefaultServerErrorRetryDelay)
	}
	if cfg.CircuitBreakerThreshold != DefaultCircuitBreakerThreshold {
		t.Errorf("CircuitBreakerThreshold = %d, want %d", cfg.CircuitBreakerThreshold, DefaultCircuitBreakerThreshold)
	}
	if cfg.CircuitBreakerResetTime != DefaultCircuitBreakerResetTime {
		t.Errorf("CircuitBreakerResetTime = %v, want %v", cfg.CircuitBreakerResetTime, DefaultCircuitBreakerResetTime)
	}
}

func TestDefaultRetryConfig_WithEnvVars(t *testing.T) {
	t.Setenv("CHATWOOT_MAX_RATE_LIMIT_RETRIES", "10")
	t.Setenv("CHATWOOT_MAX_5XX_RETRIES", "5")
	t.Setenv("CHATWOOT_RATE_LIMIT_DELAY", "2s")
	t.Setenv("CHATWOOT_SERVER_ERROR_DELAY", "500ms")
	t.Setenv("CHATWOOT_CIRCUIT_BREAKER_THRESHOLD", "3")
	t.Setenv("CHATWOOT_CIRCUIT_BREAKER_RESET_TIME", "1m")

	cfg := DefaultRetryConfig()

	if cfg.MaxRateLimitRetries != 10 {
		t.Errorf("MaxRateLimitRetries = %d, want 10", cfg.MaxRateLimitRetries)
	}
	if cfg.Max5xxRetries != 5 {
		t.Errorf("Max5xxRetries = %d, want 5", cfg.Max5xxRetries)
	}
	if cfg.RateLimitBaseDelay != 2*time.Second {
		t.Errorf("RateLimitBaseDelay = %v, want 2s", cfg.RateLimitBaseDelay)
	}
	if cfg.ServerErrorRetryDelay != 500*time.Millisecond {
		t.Errorf("ServerErrorRetryDelay = %v, want 500ms", cfg.ServerErrorRetryDelay)
	}
	if cfg.CircuitBreakerThreshold != 3 {
		t.Errorf("CircuitBreakerThreshold = %d, want 3", cfg.CircuitBreakerThreshold)
	}
	if cfg.CircuitBreakerResetTime != time.Minute {
		t.Errorf("CircuitBreakerResetTime = %v, want 1m", cfg.CircuitBreakerResetTime)
	}
}

func TestDefaultRetryConfig_InvalidEnvVars(t *testing.T) {
	t.Setenv("CHATWOOT_MAX_RATE_LIMIT_RETRIES", "not-a-number")
	t.Setenv("CHATWOOT_RATE_LIMIT_DELAY", "invalid-duration")

	cfg := DefaultRetryConfig()

	// Should fall back to defaults for invalid values
	if cfg.MaxRateLimitRetries != DefaultMaxRateLimitRetries {
		t.Errorf("MaxRateLimitRetries = %d, want %d (fallback)", cfg.MaxRateLimitRetries, DefaultMaxRateLimitRetries)
	}
	if cfg.RateLimitBaseDelay != DefaultRateLimitBaseDelay {
		t.Errorf("RateLimitBaseDelay = %v, want %v (fallback)", cfg.RateLimitBaseDelay, DefaultRateLimitBaseDelay)
	}
}
