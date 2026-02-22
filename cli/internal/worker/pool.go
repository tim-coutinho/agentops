// Package worker provides a generic concurrent worker pool for fan-out/fan-in
// file processing. Used by forge, search, and inject commands to parallelize
// file I/O across available CPUs.
package worker

import (
	"runtime"
	"sync"
)

// Result pairs a processed value with its original index to preserve ordering.
type Result[T any] struct {
	Index int
	Value T
	Err   error
}

// Pool fans out work items to a fixed number of goroutine workers
// and collects results preserving the original input order.
type Pool[T any] struct {
	concurrency int
}

// NewPool creates a worker pool with the given concurrency.
// If concurrency <= 0, defaults to runtime.NumCPU().
func NewPool[T any](concurrency int) *Pool[T] {
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}
	return &Pool[T]{concurrency: concurrency}
}

// Process distributes items across workers, applies fn to each, and returns
// results in the same order as the input slice. Errors from individual items
// are captured per-result rather than aborting the whole batch.
func (p *Pool[T]) Process(items []string, fn func(string) (T, error)) []Result[T] {
	if len(items) == 0 {
		return nil
	}

	// Cap concurrency to number of items
	workers := p.concurrency
	if workers > len(items) {
		workers = len(items)
	}

	type job struct {
		index int
		item  string
	}

	jobs := make(chan job, len(items))
	results := make([]Result[T], len(items))
	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				val, err := fn(j.item)
				results[j.index] = Result[T]{
					Index: j.index,
					Value: val,
					Err:   err,
				}
			}
		}()
	}

	// Send jobs
	for i, item := range items {
		jobs <- job{index: i, item: item}
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()

	return results
}
