// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool

import (
	"context"
	"errors"
	"iter"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestPool_Basic(t *testing.T) {
	p := New[int](Config{
		MinWorkers:  2,
		MaxWorkers:  5,
		IdleTimeout: 100 * time.Millisecond,
		QueueLimit:  10,
	})
	defer func() {
		_ = p.Close()
	}()

	ctx := context.Background()

	fut, err := p.Submit(ctx, func(ctx context.Context) (int, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := fut.Get(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

func TestPool_ScaleUp(t *testing.T) {
	p := New[string](Config{
		MinWorkers:  1,
		MaxWorkers:  3,
		IdleTimeout: 100 * time.Millisecond,
		QueueLimit:  10,
	})
	defer func() {
		_ = p.Close()
	}()

	ctx := context.Background()

	blockCh := make(chan struct{})
	defer close(blockCh)

	var wg sync.WaitGroup
	wg.Add(3)

	for range 3 {
		_, err := p.Submit(ctx, func(ctx context.Context) (string, error) {
			wg.Done()
			<-blockCh
			return "done", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Wait until all 3 tasks are running concurrently.
	wg.Wait()

	p.mu.Lock()
	workers := p.currentWorkers
	p.mu.Unlock()

	if workers != 3 {
		t.Errorf("expected pool to scale up to 3 workers, got %d", workers)
	}
}

func TestPool_ScaleDown(t *testing.T) {
	p := New[string](Config{
		MinWorkers:  1,
		MaxWorkers:  3,
		IdleTimeout: 10 * time.Millisecond,
		QueueLimit:  10,
	})
	defer func() {
		_ = p.Close()
	}()

	ctx := context.Background()
	blockCh := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(3)

	for range 3 {
		_, err := p.Submit(ctx, func(ctx context.Context) (string, error) {
			wg.Done()
			<-blockCh
			return "done", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	wg.Wait()
	close(blockCh) // Release all tasks to make workers idle.

	// Wait for idle timeout and worker scale down.
	time.Sleep(30 * time.Millisecond)

	p.mu.Lock()
	workers := p.currentWorkers
	p.mu.Unlock()

	if workers != 1 {
		t.Errorf("expected pool to scale down to 1 worker, got %d", workers)
	}
}

func TestPool_Closed(t *testing.T) {
	p := New[int](Config{
		MinWorkers: 1,
	})
	_ = p.Close()

	// Submitting to a closed pool must return ErrClosed.
	_, err := p.Submit(context.Background(), func(ctx context.Context) (int, error) {
		return 1, nil
	})
	if !errors.Is(err, ErrClosed) {
		t.Errorf("expected ErrClosed, got %v", err)
	}

	// Graceful idempotent Close.
	err = p.Close()
	if err != nil {
		t.Errorf("unexpected error on redundant Close(): %v", err)
	}
}

func TestPool_QueueFull(t *testing.T) {
	p := New[int](Config{
		MinWorkers: 1,
		MaxWorkers: 1,
		QueueLimit: 1,
	})
	defer func() {
		_ = p.Close()
	}()

	ctx := context.Background()
	blockCh := make(chan struct{})
	startCh := make(chan struct{})

	defer close(blockCh)

	// Submit a task that blocks the only available worker.
	_, err := p.Submit(ctx, func(ctx context.Context) (int, error) {
		close(startCh)
		<-blockCh
		return 1, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for the first task to start executing (popping it from the queue).
	<-startCh

	// Submit a task that enters and fills the queue (queue length = 1).
	_, err = p.Submit(ctx, func(ctx context.Context) (int, error) {
		return 2, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Submit a task that exceeds the queue limit (must return ErrQueueFull).
	_, err = p.Submit(ctx, func(ctx context.Context) (int, error) {
		return 3, nil
	})
	if !errors.Is(err, ErrQueueFull) {
		t.Errorf("expected ErrQueueFull, got %v", err)
	}
}

// TestPool_PanicSafety verifies that a panic inside a task is safely recovered
// and captured within its [Future], preventing the worker goroutine from crashing.
func TestPool_PanicSafety(t *testing.T) {
	p := New[int](Config{
		MinWorkers: 1,
	})
	defer func() {
		_ = p.Close()
	}()

	ctx := context.Background()

	fut, err := p.Submit(ctx, func(ctx context.Context) (int, error) {
		panic("something went wrong")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = fut.Get(ctx)
	if err == nil {
		t.Fatal("expected panic error, got nil")
	}

	expectedSub := "task panicked: something went wrong"
	if err.Error() != expectedSub {
		t.Errorf("expected %q, got %q", expectedSub, err.Error())
	}
}

func TestPool_FutureCancel(t *testing.T) {
	p := New[int](Config{
		MinWorkers: 1,
	})
	defer func() {
		_ = p.Close()
	}()

	ctx := context.Background()

	blockCh := make(chan struct{})
	defer close(blockCh)

	fut, err := p.Submit(ctx, func(ctx context.Context) (int, error) {
		<-blockCh
		return 1, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctxCancel, cancel := context.WithCancel(context.Background())
	cancel() // Pre-cancel the context.

	_, err = fut.Get(ctxCancel)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{}
	cfg.ResolveDefaults()

	if cfg.MinWorkers != 1 {
		t.Errorf("expected MinWorkers to be 1, got %d", cfg.MinWorkers)
	}

	if cfg.MaxWorkers != 1 {
		t.Errorf("expected MaxWorkers to be 1, got %d", cfg.MaxWorkers)
	}

	if cfg.IdleTimeout != 5*time.Second {
		t.Errorf("expected IdleTimeout to be 5s, got %v", cfg.IdleTimeout)
	}

	if cfg.QueueLimit != 100 {
		t.Errorf("expected QueueLimit to be 100, got %d", cfg.QueueLimit)
	}

	cfg2 := Config{
		MinWorkers: 5,
		MaxWorkers: 2, // MaxWorkers < MinWorkers
	}
	cfg2.ResolveDefaults()

	if cfg2.MaxWorkers != 5 {
		t.Errorf("expected MaxWorkers to be bumped to MinWorkers (5), got %d", cfg2.MaxWorkers)
	}
}

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
	p := New[int](Config{MinWorkers: 4, MaxWorkers: 4, QueueLimit: 50})
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
	p := New[int](Config{MinWorkers: 4, MaxWorkers: 4, QueueLimit: 100})
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
	p := New[int](Config{MinWorkers: 2, MaxWorkers: 2, QueueLimit: 10})
	defer p.Close()

	errExpected := errors.New("boom")
	failingSeq := func(yield func(func(context.Context) (int, error)) bool) {
		task := func(ctx context.Context) (int, error) {
			return 0, errExpected
		}
		yield(task)
	}

	var gotErr error
	for _, err := range ItemsSeq(ctx, p, failingSeq) {
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
	p := New[int](Config{MinWorkers: 4, MaxWorkers: 4, QueueLimit: 50})
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
