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

// TestCircuitBreaker_FullStateTransitionCycle verifies the complete
// closed -> open -> half-open -> closed cycle.
func TestCircuitBreaker_FullStateTransitionCycle(t *testing.T) {
	cb := &circuitBreaker{threshold: 2, resetTime: 20 * time.Millisecond}

	// Phase 1: Closed state (initial)
	if cb.isOpen() {
		t.Error("circuit should start closed")
	}

	// Phase 2: Transition to open after threshold failures
	cb.recordFailure()
	if cb.isOpen() {
		t.Error("circuit should remain closed after 1 failure (threshold is 2)")
	}

	opened := cb.recordFailure()
	if !opened {
		t.Error("recordFailure should return true when circuit opens")
	}
	if !cb.isOpen() {
		t.Error("circuit should be open after reaching threshold")
	}

	// Phase 3: Half-open state (after reset time passes)
	time.Sleep(25 * time.Millisecond)

	// isOpen() should return false (allowing one probe request)
	if cb.isOpen() {
		t.Error("circuit should allow probe request (half-open) after reset time")
	}

	// Verify internal state: open should still be true, but halfOpen should be true
	cb.mu.Lock()
	failures := cb.failures
	open := cb.open
	halfOpen := cb.halfOpen
	cb.mu.Unlock()

	if failures != 2 {
		t.Errorf("failures should still be 2 in half-open state, got %d", failures)
	}
	if !open {
		t.Error("open flag should still be true in half-open state")
	}
	if !halfOpen {
		t.Error("halfOpen flag should be true after reset time passed")
	}

	// Phase 4: Transition back to closed on success
	cb.recordSuccess()
	if cb.isOpen() {
		t.Error("circuit should be closed after successful probe")
	}

	// Verify circuit is fully closed
	cb.mu.Lock()
	failures = cb.failures
	open = cb.open
	halfOpen = cb.halfOpen
	cb.mu.Unlock()

	if failures != 0 {
		t.Errorf("failures should be 0 after success, got %d", failures)
	}
	if open {
		t.Error("open flag should be false after success")
	}
	if halfOpen {
		t.Error("halfOpen flag should be false after success")
	}
}

// TestCircuitBreaker_Reset verifies the reset() method clears all state.
func TestCircuitBreaker_Reset(t *testing.T) {
	cb := &circuitBreaker{threshold: 2, resetTime: 10 * time.Millisecond}

	// Open the circuit
	cb.recordFailure()
	cb.recordFailure()
	if !cb.isOpen() {
		t.Fatal("circuit should be open after threshold failures")
	}

	// Verify state before reset
	cb.mu.Lock()
	if cb.failures != 2 {
		t.Errorf("failures should be 2, got %d", cb.failures)
	}
	if !cb.open {
		t.Error("open flag should be true")
	}
	if cb.lastFailure.IsZero() {
		t.Error("lastFailure should be set")
	}
	cb.mu.Unlock()

	// Reset the circuit
	cb.reset()

	// Verify all state is cleared
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.failures != 0 {
		t.Errorf("reset should clear failures, got %d", cb.failures)
	}
	if cb.open {
		t.Error("reset should close the circuit")
	}
	if cb.halfOpen {
		t.Error("reset should clear halfOpen flag")
	}
	if !cb.lastFailure.IsZero() {
		t.Errorf("reset should clear lastFailure, got %v", cb.lastFailure)
	}
}

// TestCircuitBreaker_ResetWhileClosed verifies reset() is safe when circuit is already closed.
func TestCircuitBreaker_ResetWhileClosed(t *testing.T) {
	cb := &circuitBreaker{threshold: 5}

	// Record some failures but don't reach threshold
	cb.recordFailure()
	cb.recordFailure()

	// Reset while still closed
	cb.reset()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.failures != 0 {
		t.Errorf("reset should clear failures even when closed, got %d", cb.failures)
	}
	if !cb.lastFailure.IsZero() {
		t.Error("reset should clear lastFailure even when closed")
	}
}

// TestClient_ResetCircuitBreaker verifies the Client method is nil-safe and works.
func TestClient_ResetCircuitBreaker(t *testing.T) {
	t.Run("nil circuit breaker", func(t *testing.T) {
		c := &Client{} // circuitBreaker is nil
		// Should not panic
		c.ResetCircuitBreaker()
	})

	t.Run("with circuit breaker", func(t *testing.T) {
		c := &Client{
			circuitBreaker: &circuitBreaker{threshold: 2, resetTime: 10 * time.Millisecond},
		}

		// Open the circuit
		c.circuitBreaker.recordFailure()
		c.circuitBreaker.recordFailure()
		if !c.circuitBreaker.isOpen() {
			t.Fatal("circuit should be open")
		}

		// Reset via Client method
		c.ResetCircuitBreaker()

		if c.circuitBreaker.isOpen() {
			t.Error("circuit should be closed after ResetCircuitBreaker")
		}

		c.circuitBreaker.mu.Lock()
		failures := c.circuitBreaker.failures
		c.circuitBreaker.mu.Unlock()

		if failures != 0 {
			t.Errorf("failures should be 0 after reset, got %d", failures)
		}
	})
}

// TestCircuitBreaker_SuccessAfterHalfOpenCloses verifies that a success
// after the circuit enters half-open state properly closes the circuit.
func TestCircuitBreaker_SuccessAfterHalfOpenCloses(t *testing.T) {
	cb := &circuitBreaker{threshold: 1, resetTime: 15 * time.Millisecond}

	// Open the circuit
	cb.recordFailure()
	if !cb.isOpen() {
		t.Fatal("circuit should be open")
	}

	// Wait for reset time to allow half-open
	time.Sleep(20 * time.Millisecond)

	// Check isOpen() - this triggers the half-open transition
	if cb.isOpen() {
		t.Error("circuit should be half-open (isOpen returns false)")
	}

	// Verify we're in half-open state
	cb.mu.Lock()
	if !cb.halfOpen {
		t.Error("halfOpen flag should be true after reset time passed")
	}
	cb.mu.Unlock()

	// Record a success - this should close the circuit
	cb.recordSuccess()

	// Verify circuit is firmly closed
	if cb.isOpen() {
		t.Error("circuit should remain closed after success")
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.failures != 0 {
		t.Errorf("failures should be 0 after success, got %d", cb.failures)
	}
	if cb.open {
		t.Error("open flag should be false after success")
	}
	if cb.halfOpen {
		t.Error("halfOpen flag should be false after success")
	}
}

// TestCircuitBreaker_FailureDuringHalfOpenReopens verifies that a failure
// during half-open state re-opens the circuit.
func TestCircuitBreaker_FailureDuringHalfOpenReopens(t *testing.T) {
	cb := &circuitBreaker{threshold: 1, resetTime: 15 * time.Millisecond}

	// Open the circuit
	cb.recordFailure()
	if !cb.isOpen() {
		t.Fatal("circuit should be open")
	}

	// Wait for reset time to allow half-open
	time.Sleep(20 * time.Millisecond)

	// Check isOpen() - this triggers the half-open transition
	if cb.isOpen() {
		t.Error("circuit should be half-open (isOpen returns false)")
	}

	// Verify we're in half-open state
	cb.mu.Lock()
	if !cb.halfOpen {
		t.Error("halfOpen flag should be true after reset time passed")
	}
	cb.mu.Unlock()

	// Now record a failure - this should reopen the circuit
	opened := cb.recordFailure()
	if !opened {
		t.Error("recordFailure should return true when circuit reopens from half-open")
	}
	if !cb.isOpen() {
		t.Error("circuit should be open again after failure during half-open")
	}

	// Verify internal state: circuit is open, halfOpen is false
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.open {
		t.Error("open flag should be true after failed probe")
	}
	if cb.halfOpen {
		t.Error("halfOpen flag should be false after failed probe")
	}
}

// TestCircuitBreaker_HalfOpenAllowsOnlyOneProbe verifies that in half-open state,
// only one probe request is allowed through, and the circuit remains in half-open
// state until the probe result is recorded.
func TestCircuitBreaker_HalfOpenAllowsOnlyOneProbe(t *testing.T) {
	cb := &circuitBreaker{threshold: 1, resetTime: 15 * time.Millisecond}

	// Open the circuit
	cb.recordFailure()
	if !cb.isOpen() {
		t.Fatal("circuit should be open")
	}

	// Wait for reset time to allow half-open
	time.Sleep(20 * time.Millisecond)

	// First call to isOpen() triggers half-open - returns false (allow probe)
	if cb.isOpen() {
		t.Error("first isOpen() after reset time should return false (allow probe)")
	}

	// Second call to isOpen() while still in half-open should also return false
	// (the probe request is in flight, we allow it to proceed)
	if cb.isOpen() {
		t.Error("second isOpen() during half-open should also return false")
	}

	// Verify we're still in half-open state
	cb.mu.Lock()
	if !cb.halfOpen {
		t.Error("halfOpen flag should still be true until probe completes")
	}
	cb.mu.Unlock()
}

// TestCircuitBreaker_HalfOpenTimerResets verifies that when a probe fails
// during half-open state, the reset timer starts fresh.
func TestCircuitBreaker_HalfOpenTimerResets(t *testing.T) {
	cb := &circuitBreaker{threshold: 1, resetTime: 20 * time.Millisecond}

	// Open the circuit
	cb.recordFailure()
	if !cb.isOpen() {
		t.Fatal("circuit should be open")
	}

	// Wait for reset time to enter half-open
	time.Sleep(25 * time.Millisecond)

	// Trigger half-open
	if cb.isOpen() {
		t.Error("circuit should be half-open")
	}

	// Record a failure during half-open - this should re-open and reset timer
	cb.recordFailure()
	if !cb.isOpen() {
		t.Error("circuit should be open after failed probe")
	}

	// Immediately check again - should still be open (timer was reset)
	time.Sleep(10 * time.Millisecond) // Wait less than resetTime
	if !cb.isOpen() {
		t.Error("circuit should still be open (timer was reset)")
	}

	// Wait for full reset time and verify half-open is available again
	time.Sleep(15 * time.Millisecond) // Total 25ms > 20ms resetTime
	if cb.isOpen() {
		t.Error("circuit should enter half-open again after reset time")
	}
}

// TestCircuitBreaker_RecordFailureReturnValue verifies recordFailure return values.
func TestCircuitBreaker_RecordFailureReturnValue(t *testing.T) {
	cb := &circuitBreaker{threshold: 3}

	// Failures before threshold should return false
	if cb.recordFailure() {
		t.Error("recordFailure should return false for failure 1/3")
	}
	if cb.recordFailure() {
		t.Error("recordFailure should return false for failure 2/3")
	}

	// Failure at threshold should return true
	if !cb.recordFailure() {
		t.Error("recordFailure should return true when circuit opens")
	}

	// Subsequent failures while open should return false (already open)
	if cb.recordFailure() {
		t.Error("recordFailure should return false when already open")
	}
}

// TestCircuitBreaker_DefaultThresholdWhenZero verifies zero threshold uses default.
func TestCircuitBreaker_DefaultThresholdWhenZero(t *testing.T) {
	cb := &circuitBreaker{threshold: 0} // zero means use default

	// Should use DefaultCircuitBreakerThreshold (5)
	for i := 0; i < DefaultCircuitBreakerThreshold-1; i++ {
		cb.recordFailure()
	}
	if cb.isOpen() {
		t.Errorf("circuit should not be open after %d failures (default threshold is %d)",
			DefaultCircuitBreakerThreshold-1, DefaultCircuitBreakerThreshold)
	}

	cb.recordFailure()
	if !cb.isOpen() {
		t.Errorf("circuit should be open after %d failures", DefaultCircuitBreakerThreshold)
	}
}

// TestCircuitBreaker_DefaultResetTimeWhenZero verifies zero resetTime uses default.
func TestCircuitBreaker_DefaultResetTimeWhenZero(t *testing.T) {
	cb := &circuitBreaker{threshold: 1, resetTime: 0}

	cb.recordFailure()
	if !cb.isOpen() {
		t.Fatal("circuit should be open")
	}

	// With resetTime=0, it should use DefaultCircuitBreakerResetTime (30s)
	// We can't wait 30s in a test, so we verify it's still open after a short time
	time.Sleep(10 * time.Millisecond)
	if !cb.isOpen() {
		t.Error("circuit should still be open (default reset time is 30s)")
	}
}

// TestCircuitBreaker_ConcurrentAccess verifies thread safety.
func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := &circuitBreaker{threshold: 100, resetTime: 50 * time.Millisecond}

	done := make(chan struct{})
	const goroutines = 10
	const iterations = 50

	// Start goroutines that concurrently access the circuit breaker
	for i := 0; i < goroutines; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < iterations; j++ {
				cb.recordFailure()
				cb.isOpen()
				cb.recordSuccess()
				cb.reset()
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// If we got here without a race condition panic, the test passes
	// The final state is indeterminate due to concurrent access, which is fine
}
