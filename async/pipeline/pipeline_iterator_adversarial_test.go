// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pipeline_test

import (
	"context"
	"errors"
	"iter"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/async/pipeline"
)

func intSeq(count int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range count {
			if !yield(i) {
				return
			}
		}
	}
}

func TestEmpirical_Pipeline_ProcessSeq_FullTraversal(t *testing.T) {
	ctx := context.Background()
	cfg := pipeline.Config{Workers: 4}

	mapper := func(ctx context.Context, in int) (int, error) {
		return in * 2, nil
	}

	received := 0
	for out, err := range pipeline.ProcessSeq(ctx, cfg, intSeq(50), mapper) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = out
		received++
	}

	if received != 50 {
		t.Fatalf("expected 50 received items, got %d", received)
	}
}

func TestEmpirical_Pipeline_ProcessSeq_EarlyTerminationAndNoLeak(t *testing.T) {
	ctx := context.Background()
	cfg := pipeline.Config{Workers: 8}

	var activeMappers atomic.Int32
	mapper := func(ctx context.Context, in int) (int, error) {
		activeMappers.Add(1)
		defer activeMappers.Add(-1)
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(5 * time.Millisecond):
			return in * 2, nil
		}
	}

	// 1. Break after 0 iterations (first item returns false)
	beforeG := runtime.NumGoroutine()
	count := 0
	for out, err := range pipeline.ProcessSeq(ctx, cfg, intSeq(200), mapper) {
		_ = out
		_ = err
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1 on immediate break, got %d", count)
	}

	// Give cancellation a moment to propagate
	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	afterG := runtime.NumGoroutine()
	if afterG > beforeG+2 {
		t.Fatalf("goroutines leaked on break after 0: before=%d, after=%d", beforeG, afterG)
	}

	// 2. Break after 5 items mid-iteration
	count = 0
	for out, err := range pipeline.ProcessSeq(ctx, cfg, intSeq(200), mapper) {
		_ = out
		_ = err
		count++
		if count == 5 {
			break
		}
	}
	if count != 5 {
		t.Fatalf("expected count 5, got %d", count)
	}

	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	afterG2 := runtime.NumGoroutine()
	if afterG2 > beforeG+2 {
		t.Fatalf("goroutines leaked on break mid-iteration: before=%d, after=%d", beforeG, afterG2)
	}
}

func TestEmpirical_Pipeline_ProcessSeq_NilMapperAndCancelledCtx(t *testing.T) {
	ctx := context.Background()
	cfg := pipeline.Config{Workers: 2}

	// Nil mapper
	var nilMapperYielded bool
	for _, err := range pipeline.ProcessSeq[int, int](ctx, cfg, intSeq(10), nil) {
		if err == nil {
			t.Fatal("expected error for nil mapper")
		}
		nilMapperYielded = true
		break
	}
	if !nilMapperYielded {
		t.Fatal("expected nil mapper to yield error")
	}

	// Pre-cancelled context
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	dummyMapper := func(ctx context.Context, in int) (int, error) {
		return in, nil
	}
	for range pipeline.ProcessSeq(cancelCtx, cfg, intSeq(10), dummyMapper) {
		// Cancelled context should finish or yield context error
	}
}

func TestEmpirical_Pipeline_ProcessSeq_PipelineMethod(t *testing.T) {
	ctx := context.Background()
	p := pipeline.New[int, string](pipeline.Config{Workers: 2})

	mapper := func(ctx context.Context, in int) (string, error) {
		if in == 42 {
			return "", errors.New("bad 42")
		}
		return "ok", nil
	}

	items := 0
	for out, err := range p.ProcessSeq(ctx, intSeq(5), mapper) {
		_ = out
		_ = err
		items++
	}
	if items != 5 {
		t.Fatalf("expected 5 items, got %d", items)
	}
}

func BenchmarkProcessSeq(b *testing.B) {
	ctx := context.Background()
	cfg := pipeline.Config{Workers: 4}
	mapper := func(ctx context.Context, in int) (int, error) {
		return in * 2, nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for out, err := range pipeline.ProcessSeq(ctx, cfg, intSeq(10), mapper) {
			_ = out
			_ = err
		}
	}
}
