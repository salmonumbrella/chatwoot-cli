package api

import (
	"sync"
	"time"
)

const (
	MaxRateLimitRetries     = 3
	Max5xxRetries           = 1
	RateLimitBaseDelay      = 1 * time.Second
	ServerErrorRetryDelay   = 1 * time.Second
	CircuitBreakerThreshold = 5
	CircuitBreakerResetTime = 30 * time.Second
)

type circuitBreaker struct {
	mu          sync.Mutex
	failures    int
	lastFailure time.Time
	open        bool
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

	if cb.failures >= CircuitBreakerThreshold && !cb.open {
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
	if time.Since(cb.lastFailure) >= CircuitBreakerResetTime {
		cb.open = false
		cb.failures = 0
		return false
	}

	return true
}
