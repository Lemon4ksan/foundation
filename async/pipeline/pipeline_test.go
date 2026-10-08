// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pipeline

import (
	"context"
	"errors"
	"iter"
	"math/rand"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestPipeline_OrderPreservation(t *testing.T) {
	cfg := Config{
		Workers: 12,
	}
	p := New[int, int](cfg)

	inputs := make([]int, 1000)
	for i := range inputs {
		inputs[i] = i + 1
	}

	mapper := func(ctx context.Context, in int) (int, error) {
		// Random sleep to disrupt scheduling order
		time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
		return in * 10, nil
	}

	outputs, err := p.Transform(context.Background(), inputs, mapper)
	require.NoError(t, err)
	require.Len(t, outputs, 1000)

	for i := range inputs {
		assert.Equal(t, inputs[i]*10, outputs[i])
	}
}

func TestPipeline_FailFast(t *testing.T) {
	cfg := Config{
		Workers:  1,
		FailFast: true,
	}
	p := New[int, int](cfg)

	inputs := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	errDummy := errors.New("dummy error")

	var runCount atomic.Int32

	mapper := func(ctx context.Context, in int) (int, error) {
		runCount.Add(1)

		if in == 2 {
			return 0, errDummy
		}

		time.Sleep(10 * time.Millisecond)

		return in, nil
	}

	_, err := p.Transform(context.Background(), inputs, mapper)
	assert.ErrorIs(t, err, errDummy)

	// Due to FailFast with 1 worker, only 2 or 3 tasks should be processed before context cancel takes effect
	assert.LessOrEqual(t, runCount.Load(), int32(3))
}

func TestPipeline_NoFailFast(t *testing.T) {
	cfg := Config{
		Workers:  3,
		FailFast: false,
	}
	p := New[int, int](cfg)

	inputs := []int{1, 2, 3}
	errDummy := errors.New("dummy error")

	mapper := func(ctx context.Context, in int) (int, error) {
		if in == 2 || in == 3 {
			return 0, errDummy
		}

		return in, nil
	}

	_, err := p.Transform(context.Background(), inputs, mapper)
	assert.Error(t, err)
	// Since FailFast is false, it should run all and join errors
	assert.Contains(t, err.Error(), "dummy error")
}

func TestPipeline_RateLimiting(t *testing.T) {
	cfg := Config{
		Workers: 2,
		RPS:     50, // 50 requests per second, so 20ms per request
		Burst:   1,
	}
	p := New[int, int](cfg)

	inputs := []int{1, 2, 3}
	start := time.Now()

	_, err := p.Transform(context.Background(), inputs, func(ctx context.Context, in int) (int, error) {
		return in, nil
	})
	require.NoError(t, err)

	duration := time.Since(start)
	// 3 requests with limit 50/sec should take at least ~40ms
	assert.GreaterOrEqual(t, duration, 35*time.Millisecond)
}

func TestPipeline_ContextCancelled_NoLeaks(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	cfg := Config{
		Workers: 5,
	}
	p := New[int, int](cfg)

	inputs := make([]int, 100)
	for i := range inputs {
		inputs[i] = i
	}

	ctx, cancel := context.WithCancel(context.Background())

	mapper := func(ctx context.Context, in int) (int, error) {
		if in == 5 {
			cancel() // Cancel context mid-execution
		}

		time.Sleep(10 * time.Millisecond)

		return in, nil
	}

	_, err := p.Transform(ctx, inputs, mapper)
	assert.ErrorIs(t, err, context.Canceled)

	// Wait for goroutines to clean up
	time.Sleep(50 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	// Check that we didn't leak worker goroutines
	assert.LessOrEqual(t, finalGoroutines, initialGoroutines+2)
}

func TestPipeline_Stream(t *testing.T) {
	cfg := Config{
		Workers: 3,
	}
	p := New[int, int](cfg)

	inCh := make(chan int, 10)
	for i := 1; i <= 5; i++ {
		inCh <- i
	}

	close(inCh)

	outCh, errCh := p.Stream(context.Background(), inCh, func(ctx context.Context, in int) (int, error) {
		return in * 2, nil
	})

	var results []int
	for val := range outCh {
		results = append(results, val)
	}

	// We expect 5 results
	assert.Len(t, results, 5)
	assert.ElementsMatch(t, []int{2, 4, 6, 8, 10}, results)

	for err := range errCh {
		t.Fatalf("unexpected error from stream: %v", err)
	}
}

func TestPipeline_Stream_FailFast(t *testing.T) {
	cfg := Config{
		Workers:  3,
		FailFast: true,
	}
	p := New[int, int](cfg)

	inCh := make(chan int, 10)
	for i := 1; i <= 10; i++ {
		inCh <- i
	}

	close(inCh)

	errDummy := errors.New("stream error")
	outCh, errCh := p.Stream(context.Background(), inCh, func(ctx context.Context, in int) (int, error) {
		if in == 3 {
			return 0, errDummy
		}

		time.Sleep(10 * time.Millisecond)

		return in, nil
	})

	// Drain output
	for range outCh {
	}

	var encounteredErrs []error
	for err := range errCh {
		encounteredErrs = append(encounteredErrs, err)
	}

	assert.NotEmpty(t, encounteredErrs)
	assert.ErrorIs(t, encounteredErrs[0], errDummy)
}

func TestMapAndForEach(t *testing.T) {
	ctx := context.Background()
	cfg := Config{Workers: 4}

	// Test Map
	res, err := Map(ctx, cfg, []int{1, 2, 3}, func(ctx context.Context, in int) (string, error) {
		return "val", nil
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"val", "val", "val"}, res)

	// Test ForEach
	var sum atomic.Int64

	err = ForEach(ctx, cfg, []int{10, 20, 30}, func(ctx context.Context, in int) error {
		sum.Add(int64(in))
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, int64(60), sum.Load())
}

func TestPipeline_EmptyAndDefaults(t *testing.T) {
	// Empty inputs
	res, err := Map[int, int](
		context.Background(),
		Config{Workers: 3},
		nil,
		func(ctx context.Context, in int) (int, error) {
			return in, nil
		},
	)
	assert.NoError(t, err)
	assert.Nil(t, res)

	// Resolve defaults for Workers <= 0 and Burst <= 0
	p := New[int, int](Config{Workers: 0, Burst: 0})
	assert.Equal(t, 1, p.config.Workers)
	assert.Equal(t, 1, p.config.Burst)

	// Already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = p.Transform(ctx, []int{1}, func(c context.Context, in int) (int, error) {
		return in, nil
	})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestPipeline_RateLimiting_Cancelled(t *testing.T) {
	cfg := Config{
		Workers: 1,
		RPS:     1, // very slow to force wait block
		Burst:   1,
	}
	p := New[int, int](cfg)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel context asynchronously so rate limiter Wait will fail
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := p.Transform(ctx, []int{1, 2}, func(c context.Context, in int) (int, error) {
		return in, nil
	})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestPipeline_Stream_Cancelled(t *testing.T) {
	cfg := Config{
		Workers: 2,
	}
	p := New[int, int](cfg)

	inCh := make(chan int, 5)
	inCh <- 1

	inCh <- 2

	ctx, cancel := context.WithCancel(context.Background())

	outCh, errCh := p.Stream(ctx, inCh, func(c context.Context, in int) (int, error) {
		if in == 1 {
			cancel() // cancel stream context mid-stream
		}

		time.Sleep(10 * time.Millisecond)

		return in, nil
	})

	// Drain output
	for range outCh {
	}

	var streamErrs []error
	for err := range errCh {
		streamErrs = append(streamErrs, err)
	}

	assert.Empty(t, streamErrs)
	assert.Contains(t, []error{nil, context.Canceled}, ctx.Err())
}

func TestPipeline_Stream_RateLimiting(t *testing.T) {
	cfg := Config{
		Workers: 1,
		RPS:     100,
		Burst:   1,
	}
	p := New[int, int](cfg)

	inCh := make(chan int, 2)
	inCh <- 1

	inCh <- 2

	close(inCh)

	outCh, errCh := p.Stream(context.Background(), inCh, func(ctx context.Context, in int) (int, error) {
		return in, nil
	})

	var results []int
	for val := range outCh {
		results = append(results, val)
	}

	assert.ElementsMatch(t, []int{1, 2}, results)

	for err := range errCh {
		t.Fatalf("unexpected stream error: %v", err)
	}
}

func TestPipeline_Stream_RateLimiting_Cancelled(t *testing.T) {
	cfg := Config{
		Workers:  1,
		RPS:      1, // slow
		Burst:    1,
		FailFast: true,
	}
	p := New[int, int](cfg)

	inCh := make(chan int, 2)
	inCh <- 1

	inCh <- 2

	close(inCh)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	outCh, errCh := p.Stream(ctx, inCh, func(c context.Context, in int) (int, error) {
		return in, nil
	})

	for range outCh {
	}

	var streamErrs []error
	for err := range errCh {
		streamErrs = append(streamErrs, err)
	}

	assert.NotEmpty(t, streamErrs)
	assert.ErrorIs(t, streamErrs[0], context.Canceled)
}

func TestPipeline_Helpers_EdgeCases(t *testing.T) {
	ctx := context.Background()
	cfg := Config{Workers: 2}

	// Map nil check
	_, err := Map[int, int](ctx, cfg, []int{1}, nil)
	assert.Error(t, err)

	// Map empty slice
	mapped, err := Map(ctx, cfg, []int{}, func(_ context.Context, in int) (int, error) { return in, nil })
	assert.NoError(t, err)
	assert.Empty(t, mapped)

	// ForEach nil check
	err = ForEach[int](ctx, cfg, []int{1}, nil)
	assert.Error(t, err)

	// ForEach empty slice
	err = ForEach(ctx, cfg, []int{}, func(_ context.Context, in int) error { return nil })
	assert.NoError(t, err)

	// Transform nil mapper
	p := New[int, int](cfg)
	_, err = p.Transform(ctx, []int{1}, nil)
	assert.Error(t, err)

	// Stream nil mapper
	inCh := make(chan int)
	close(inCh)
	_, errCh := p.Stream(ctx, inCh, nil)
	err = <-errCh
	assert.Error(t, err)
}

func BenchmarkPipeline_Transform(b *testing.B) {
	cfg := Config{
		Workers: 8,
	}
	p := New[int, int](cfg)

	inputs := make([]int, 100)
	for i := range inputs {
		inputs[i] = i
	}

	mapper := func(ctx context.Context, in int) (int, error) {
		return in * 2, nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = p.Transform(context.Background(), inputs, mapper)
	}
}

func intSeq(count int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range count {
			if !yield(i) {
				return
			}
		}
	}
}

func TestEmpirical_Pipeline_TransformSeq_FullTraversal(t *testing.T) {
	ctx := context.Background()
	cfg := Config{Workers: 4}

	mapper := func(ctx context.Context, in int) (int, error) {
		return in * 2, nil
	}

	received := 0
	for out, err := range TransformSeq(ctx, cfg, intSeq(50), mapper) {
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

func TestEmpirical_Pipeline_TransformSeq_EarlyTerminationAndNoLeak(t *testing.T) {
	ctx := context.Background()
	cfg := Config{Workers: 8}

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
	for out, err := range TransformSeq(ctx, cfg, intSeq(200), mapper) {
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
	for out, err := range TransformSeq(ctx, cfg, intSeq(200), mapper) {
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

func TestEmpirical_Pipeline_TransformSeq_NilMapperAndCancelledCtx(t *testing.T) {
	ctx := context.Background()
	cfg := Config{Workers: 2}

	// Nil mapper
	var nilMapperYielded bool
	for _, err := range TransformSeq[int, int](ctx, cfg, intSeq(10), nil) {
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
	for range TransformSeq(cancelCtx, cfg, intSeq(10), dummyMapper) {
	}
}

func TestEmpirical_Pipeline_TransformSeq_PipelineMethod(t *testing.T) {
	ctx := context.Background()
	p := New[int, string](Config{Workers: 2})

	mapper := func(ctx context.Context, in int) (string, error) {
		if in == 42 {
			return "", errors.New("bad 42")
		}
		return "ok", nil
	}

	items := 0
	for out, err := range p.TransformSeq(ctx, intSeq(5), mapper) {
		_ = out
		_ = err
		items++
	}
	if items != 5 {
		t.Fatalf("expected 5 items, got %d", items)
	}
}

func BenchmarkTransformSeq(b *testing.B) {
	ctx := context.Background()
	cfg := Config{Workers: 4}
	mapper := func(ctx context.Context, in int) (int, error) {
		return in * 2, nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for out, err := range TransformSeq(ctx, cfg, intSeq(10), mapper) {
			_ = out
			_ = err
		}
	}
}
