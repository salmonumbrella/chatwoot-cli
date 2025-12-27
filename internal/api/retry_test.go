package api

import (
	"testing"
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
