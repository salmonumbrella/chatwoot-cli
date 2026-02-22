package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunBulkOperation_Success(t *testing.T) {
	ids := []int{1, 2, 3, 4, 5}
	var callCount atomic.Int32

	results := runBulkOperation(
		context.Background(),
		ids,
		5,
		false,
		nil,
		func(ctx context.Context, id int) (string, error) {
			callCount.Add(1)
			return "ok", nil
		},
	)

	if int(callCount.Load()) != 5 {
		t.Errorf("expected 5 calls, got %d", callCount.Load())
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	if successCount != 5 {
		t.Errorf("expected 5 successes, got %d", successCount)
	}
}

func TestRunBulkOperation_PartialFailure(t *testing.T) {
	ids := []int{1, 2, 3}

	results := runBulkOperation(
		context.Background(),
		ids,
		5,
		false,
		nil,
		func(ctx context.Context, id int) (string, error) {
			if id == 2 {
				return "", errors.New("failed")
			}
			return "ok", nil
		},
	)

	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}

	if successCount != 2 {
		t.Errorf("expected 2 successes, got %d", successCount)
	}
	if failCount != 1 {
		t.Errorf("expected 1 failure, got %d", failCount)
	}
}

func TestRunBulkOperation_Concurrency(t *testing.T) {
	ids := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	var maxConcurrent atomic.Int32
	var current atomic.Int32

	_ = runBulkOperation(
		context.Background(),
		ids,
		3, // limit to 3 concurrent
		false,
		nil,
		func(ctx context.Context, id int) (string, error) {
			cur := current.Add(1)
			// Track max concurrent
			for {
				max := maxConcurrent.Load()
				if cur <= max || maxConcurrent.CompareAndSwap(max, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			current.Add(-1)
			return "ok", nil
		},
	)

	if maxConcurrent.Load() > 3 {
		t.Errorf("max concurrent exceeded limit: got %d, want <= 3", maxConcurrent.Load())
	}
}

func TestCountResults(t *testing.T) {
	results := []BulkResult{
		{ID: 1, Success: true},
		{ID: 2, Success: false},
		{ID: 3, Success: true},
		{ID: 4, Success: true},
		{ID: 5, Success: false},
	}

	success, failure := countResults(results)
	if success != 3 {
		t.Errorf("expected 3 successes, got %d", success)
	}
	if failure != 2 {
		t.Errorf("expected 2 failures, got %d", failure)
	}
}

func TestRunBulkOperation_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ids := []int{1, 2, 3, 4, 5}
	var callCount atomic.Int32

	// Cancel after brief delay
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_ = runBulkOperation(
		ctx,
		ids,
		1, // sequential to make cancellation predictable
		false,
		nil,
		func(ctx context.Context, id int) (string, error) {
			callCount.Add(1)
			time.Sleep(50 * time.Millisecond)
			return "ok", nil
		},
	)

	// Should have processed fewer than all items due to cancellation
	if callCount.Load() >= 5 {
		t.Errorf("expected fewer than 5 calls due to cancellation, got %d", callCount.Load())
	}
}

func TestRunBulkOperationProgress(t *testing.T) {
	var buf bytes.Buffer
	_ = runBulkOperation(
		context.Background(),
		[]int{1, 2},
		1,
		true,
		&buf,
		func(ctx context.Context, id int) (string, error) {
			return "ok", nil
		},
	)
	if !strings.Contains(buf.String(), "Processed 2/2") {
		t.Fatalf("expected progress output, got %q", buf.String())
	}
}
