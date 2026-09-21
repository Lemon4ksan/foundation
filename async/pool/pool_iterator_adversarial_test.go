// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool_test

import (
	"context"
	"errors"
	"iter"
	"runtime"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/async/pool"
)

func taskSeq(count int) iter.Seq[func(context.Context) (int, error)] {
	return func(yield func(func(context.Context) (int, error)) bool) {
		for i := range count {
			val := i
			task := func(ctx context.Context) (int, error) {
				return val * 10, nil
			}
			if !yield(task) {
				return
			}
		}
	}
}

func slowTaskSeq(count int, d time.Duration) iter.Seq[func(context.Context) (int, error)] {
	return func(yield func(func(context.Context) (int, error)) bool) {
		for i := range count {
			val := i
			task := func(ctx context.Context) (int, error) {
				select {
				case <-ctx.Done():
					return 0, ctx.Err()
				case <-time.After(d):
					return val * 10, nil
				}
			}
			if !yield(task) {
				return
			}
		}
	}
}

func TestEmpirical_Pool_ItemsSeq_FullTraversal(t *testing.T) {
	ctx := context.Background()
	p := pool.New[int](pool.Config{MinWorkers: 4, MaxWorkers: 4, QueueLimit: 50})
	defer p.Close()

	results := make(map[int]bool)
	for val, err := range p.ItemsSeq(ctx, taskSeq(20)) {
		if err != nil {
			t.Fatalf("unexpected task error: %v", err)
		}
		results[val] = true
	}

	if len(results) != 20 {
		t.Fatalf("expected 20 distinct results, got %d", len(results))
	}
}

func TestEmpirical_Pool_ItemsSeq_EarlyTerminationAndNoLeak(t *testing.T) {
	ctx := context.Background()
	p := pool.New[int](pool.Config{MinWorkers: 4, MaxWorkers: 4, QueueLimit: 100})
	defer p.Close()

	beforeG := runtime.NumGoroutine()

	// 1. Break after 0 iterations (first item returns false)
	count := 0
	for val, err := range p.ItemsSeq(ctx, slowTaskSeq(50, 10*time.Millisecond)) {
		_ = val
		_ = err
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1 on immediate break, got %d", count)
	}

	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	afterG1 := runtime.NumGoroutine()
	if afterG1 > beforeG+2 {
		t.Fatalf("goroutine leak on immediate break: before=%d, after=%d", beforeG, afterG1)
	}

	// 2. Break after 5 items mid-iteration
	count = 0
	for val, err := range p.ItemsSeq(ctx, slowTaskSeq(50, 5*time.Millisecond)) {
		_ = val
		_ = err
		count++
		if count == 5 {
			break
		}
	}
	if count != 5 {
		t.Fatalf("expected count 5 on mid break, got %d", count)
	}

	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	afterG2 := runtime.NumGoroutine()
	if afterG2 > beforeG+2 {
		t.Fatalf("goroutine leak on mid break: before=%d, after=%d", beforeG, afterG2)
	}
}

func TestEmpirical_Pool_ItemsSeq_ErrorPropagation(t *testing.T) {
	ctx := context.Background()
	p := pool.New[int](pool.Config{MinWorkers: 2, MaxWorkers: 2, QueueLimit: 10})
	defer p.Close()

	errExpected := errors.New("boom")
	failingSeq := func(yield func(func(context.Context) (int, error)) bool) {
		task := func(ctx context.Context) (int, error) {
			return 0, errExpected
		}
		yield(task)
	}

	var gotErr error
	for _, err := range pool.ItemsSeq(ctx, p, failingSeq) {
		if err != nil {
			gotErr = err
			break
		}
	}

	if !errors.Is(gotErr, errExpected) {
		t.Fatalf("expected error %v, got %v", errExpected, gotErr)
	}
}

func BenchmarkItemsSeq(b *testing.B) {
	ctx := context.Background()
	p := pool.New[int](pool.Config{MinWorkers: 4, MaxWorkers: 4, QueueLimit: 50})
	defer p.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for val, err := range p.ItemsSeq(ctx, taskSeq(10)) {
			_ = val
			_ = err
		}
	}
}
