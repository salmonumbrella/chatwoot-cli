package cmd

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// DefaultConcurrency is the default number of concurrent workers
const DefaultConcurrency = 5

// BulkResult represents the outcome of a single bulk operation
type BulkResult struct {
	ID      int
	Success bool
	Error   error
	Data    any
}

// runBulkOperation executes operations concurrently with bounded parallelism
func runBulkOperation[T any](
	ctx context.Context,
	ids []int,
	concurrency int64,
	operation func(ctx context.Context, id int) (T, error),
) []BulkResult {
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}

	sem := semaphore.NewWeighted(concurrency)
	var mu sync.Mutex
	results := make([]BulkResult, 0, len(ids))

	g, ctx := errgroup.WithContext(ctx)

	for _, id := range ids {
		id := id // capture for goroutine

		g.Go(func() error {
			// Acquire semaphore slot
			if err := sem.Acquire(ctx, 1); err != nil {
				return nil // context cancelled, don't add to results
			}
			defer sem.Release(1)

			// Check context before executing
			if ctx.Err() != nil {
				return nil
			}

			// Execute the operation
			data, err := operation(ctx, id)

			// Thread-safe result collection
			mu.Lock()
			if err != nil {
				results = append(results, BulkResult{
					ID:      id,
					Success: false,
					Error:   err,
				})
			} else {
				results = append(results, BulkResult{
					ID:      id,
					Success: true,
					Data:    data,
				})
			}
			mu.Unlock()

			return nil // don't fail the group on individual errors
		})
	}

	// Wait for all goroutines
	_ = g.Wait()

	return results
}

// countResults returns success and failure counts from bulk results
func countResults(results []BulkResult) (success, failure int) {
	for _, r := range results {
		if r.Success {
			success++
		} else {
			failure++
		}
	}
	return
}
