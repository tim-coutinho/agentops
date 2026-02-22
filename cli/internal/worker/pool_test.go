package worker

import (
	"fmt"
	"runtime"
	"sort"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewPoolDefaultConcurrency(t *testing.T) {
	p := NewPool[string](0)
	if p.concurrency != runtime.NumCPU() {
		t.Errorf("expected concurrency %d, got %d", runtime.NumCPU(), p.concurrency)
	}

	p2 := NewPool[string](-1)
	if p2.concurrency != runtime.NumCPU() {
		t.Errorf("expected concurrency %d for -1, got %d", runtime.NumCPU(), p2.concurrency)
	}
}

func TestNewPoolExplicitConcurrency(t *testing.T) {
	p := NewPool[string](4)
	if p.concurrency != 4 {
		t.Errorf("expected concurrency 4, got %d", p.concurrency)
	}
}

func TestProcessEmpty(t *testing.T) {
	p := NewPool[string](2)
	results := p.Process(nil, func(s string) (string, error) {
		return s, nil
	})
	if results != nil {
		t.Errorf("expected nil results for empty input, got %v", results)
	}
}

func TestProcessPreservesOrder(t *testing.T) {
	p := NewPool[string](4)
	items := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	results := p.Process(items, func(s string) (string, error) {
		return "processed-" + s, nil
	})

	if len(results) != len(items) {
		t.Fatalf("expected %d results, got %d", len(items), len(results))
	}

	for i, r := range results {
		if r.Err != nil {
			t.Errorf("result[%d] unexpected error: %v", i, r.Err)
		}
		expected := "processed-" + items[i]
		if r.Value != expected {
			t.Errorf("result[%d] = %q, expected %q", i, r.Value, expected)
		}
		if r.Index != i {
			t.Errorf("result[%d].Index = %d, expected %d", i, r.Index, i)
		}
	}
}

func TestProcessCapturesErrors(t *testing.T) {
	p := NewPool[int](2)
	items := []string{"ok", "fail", "ok", "fail"}

	results := p.Process(items, func(s string) (int, error) {
		if s == "fail" {
			return 0, fmt.Errorf("failed on %s", s)
		}
		return 1, nil
	})

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// Check successes
	if results[0].Err != nil || results[0].Value != 1 {
		t.Errorf("result[0] should succeed, got err=%v val=%d", results[0].Err, results[0].Value)
	}
	if results[2].Err != nil || results[2].Value != 1 {
		t.Errorf("result[2] should succeed, got err=%v val=%d", results[2].Err, results[2].Value)
	}

	// Check failures
	if results[1].Err == nil {
		t.Error("result[1] should have error")
	}
	if results[3].Err == nil {
		t.Error("result[3] should have error")
	}
}

func TestProcessConcurrency(t *testing.T) {
	// Verify multiple workers are actually running concurrently
	p := NewPool[int](4)

	var maxConcurrent int64
	var current int64
	items := make([]string, 20)
	for i := range items {
		items[i] = fmt.Sprintf("item-%d", i)
	}

	results := p.Process(items, func(s string) (int, error) {
		c := atomic.AddInt64(&current, 1)
		// Track peak concurrency
		for {
			old := atomic.LoadInt64(&maxConcurrent)
			if c <= old || atomic.CompareAndSwapInt64(&maxConcurrent, old, c) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond) // Simulate I/O
		atomic.AddInt64(&current, -1)
		return 1, nil
	})

	if len(results) != 20 {
		t.Fatalf("expected 20 results, got %d", len(results))
	}

	peak := atomic.LoadInt64(&maxConcurrent)
	if peak < 2 {
		t.Errorf("expected concurrent execution (peak=%d), got sequential", peak)
	}
}

func TestProcessSingleItem(t *testing.T) {
	p := NewPool[string](4)
	results := p.Process([]string{"only"}, func(s string) (string, error) {
		return "done-" + s, nil
	})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Value != "done-only" {
		t.Errorf("expected done-only, got %s", results[0].Value)
	}
}

func TestProcessMoreWorkersThanItems(t *testing.T) {
	p := NewPool[string](100)
	items := []string{"a", "b"}

	results := p.Process(items, func(s string) (string, error) {
		return s + "!", nil
	})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Value != "a!" || results[1].Value != "b!" {
		t.Errorf("unexpected values: %v, %v", results[0].Value, results[1].Value)
	}
}

func TestProcessResultsAreSortable(t *testing.T) {
	p := NewPool[string](4)
	items := []string{"c", "a", "b"}

	results := p.Process(items, func(s string) (string, error) {
		return s, nil
	})

	// Results should already be in order by index
	for i, r := range results {
		if r.Index != i {
			t.Errorf("result[%d].Index = %d", i, r.Index)
		}
	}

	// But we can also sort by value if needed
	sort.Slice(results, func(i, j int) bool {
		return results[i].Value < results[j].Value
	})
	if results[0].Value != "a" || results[1].Value != "b" || results[2].Value != "c" {
		t.Error("sorting by value failed")
	}
}

// --- Benchmarks ---

func BenchmarkPoolProcess(b *testing.B) {
	items := make([]string, 100)
	for i := range items {
		items[i] = fmt.Sprintf("item-%d", i)
	}
	b.ResetTimer()
	for range b.N {
		p := NewPool[string](4)
		_ = p.Process(items, func(s string) (string, error) {
			return s + "-done", nil
		})
	}
}
