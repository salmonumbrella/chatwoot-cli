package api

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Default retry configuration values
const (
	DefaultMaxRateLimitRetries     = 3
	DefaultMax5xxRetries           = 1
	DefaultRateLimitBaseDelay      = 1 * time.Second
	DefaultServerErrorRetryDelay   = 1 * time.Second
	DefaultCircuitBreakerThreshold = 5
	DefaultCircuitBreakerResetTime = 30 * time.Second
)

// Backwards compatibility aliases
const (
	MaxRateLimitRetries     = DefaultMaxRateLimitRetries
	Max5xxRetries           = DefaultMax5xxRetries
	RateLimitBaseDelay      = DefaultRateLimitBaseDelay
	ServerErrorRetryDelay   = DefaultServerErrorRetryDelay
	CircuitBreakerThreshold = DefaultCircuitBreakerThreshold
	CircuitBreakerResetTime = DefaultCircuitBreakerResetTime
)

// RetryConfig holds configuration for retry behavior and circuit breaker.
type RetryConfig struct {
	MaxRateLimitRetries     int
	Max5xxRetries           int
	RateLimitBaseDelay      time.Duration
	ServerErrorRetryDelay   time.Duration
	CircuitBreakerThreshold int
	CircuitBreakerResetTime time.Duration
}

// DefaultRetryConfig returns a RetryConfig populated from environment variables
// with fallback to default values.
//
// Environment variables:
//   - CHATWOOT_MAX_RATE_LIMIT_RETRIES: max retries for 429 errors (default: 3)
//   - CHATWOOT_MAX_5XX_RETRIES: max retries for 5xx errors (default: 1)
//   - CHATWOOT_RATE_LIMIT_DELAY: base delay for rate limit retries (default: "1s")
//   - CHATWOOT_SERVER_ERROR_DELAY: delay for server error retries (default: "1s")
//   - CHATWOOT_CIRCUIT_BREAKER_THRESHOLD: failures before circuit opens (default: 5)
//   - CHATWOOT_CIRCUIT_BREAKER_RESET_TIME: time before circuit resets (default: "30s")
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRateLimitRetries:     getEnvInt("CHATWOOT_MAX_RATE_LIMIT_RETRIES", DefaultMaxRateLimitRetries),
		Max5xxRetries:           getEnvInt("CHATWOOT_MAX_5XX_RETRIES", DefaultMax5xxRetries),
		RateLimitBaseDelay:      getEnvDuration("CHATWOOT_RATE_LIMIT_DELAY", DefaultRateLimitBaseDelay),
		ServerErrorRetryDelay:   getEnvDuration("CHATWOOT_SERVER_ERROR_DELAY", DefaultServerErrorRetryDelay),
		CircuitBreakerThreshold: getEnvInt("CHATWOOT_CIRCUIT_BREAKER_THRESHOLD", DefaultCircuitBreakerThreshold),
		CircuitBreakerResetTime: getEnvDuration("CHATWOOT_CIRCUIT_BREAKER_RESET_TIME", DefaultCircuitBreakerResetTime),
	}
}

// getEnvInt reads an integer from an environment variable with a default fallback.
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

// getEnvDuration reads a duration from an environment variable with a default fallback.
func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if parsed, err := time.ParseDuration(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

type circuitBreaker struct {
	mu          sync.Mutex
	failures    int
	lastFailure time.Time
	open        bool
	threshold   int
	resetTime   time.Duration
}

// sleepWithContext waits for the duration or returns early on context cancellation.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// retryAfterDuration parses Retry-After header values (seconds or HTTP date).
func retryAfterDuration(h http.Header) (time.Duration, bool) {
	value := strings.TrimSpace(h.Get("Retry-After"))
	if value == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(value); err == nil {
		if secs < 0 {
			secs = 0
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(value); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

// recordSuccess resets failures to 0 and sets open to false.
// Logs if the circuit was previously open.
func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.open = false
}

// recordFailure increments failures, sets lastFailure to now,
// and returns true if the circuit just opened.
func (cb *circuitBreaker) recordFailure() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	threshold := cb.threshold
	if threshold <= 0 {
		threshold = DefaultCircuitBreakerThreshold
	}
	if cb.failures >= threshold && !cb.open {
		cb.open = true
		return true
	}
	return false
}

// isOpen returns true if open AND not past reset time.
// Auto-closes if past reset time.
func (cb *circuitBreaker) isOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.open {
		return false
	}

	// Auto-close if past reset time
	resetTime := cb.resetTime
	if resetTime <= 0 {
		resetTime = DefaultCircuitBreakerResetTime
	}
	if time.Since(cb.lastFailure) >= resetTime {
		cb.open = false
		cb.failures = 0
		return false
	}

	return true
}

// reset clears all failure state and closes the circuit.
// This is useful when reusing a client across logical sessions.
func (cb *circuitBreaker) reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.open = false
	cb.lastFailure = time.Time{}
}
