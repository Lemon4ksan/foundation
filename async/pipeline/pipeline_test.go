// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pipeline

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math/rand"
	"runtime"
	"sync"
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

	b.ReportAllocs()

	for b.Loop() {
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

	b.ReportAllocs()

	for b.Loop() {
		for out, err := range TransformSeq(ctx, cfg, intSeq(10), mapper) {
			_ = out
			_ = err
		}
	}
}

// TestAdversarial_Pipeline_InternalRateLimiter_Throughput verifies that the internal
// async/rate limiter correctly bounds throughput and operates safely under concurrent Transform calls.
func TestAdversarial_Pipeline_InternalRateLimiter_Throughput(t *testing.T) {
	const (
		rps   = 40.0
		burst = 2
	)

	p := New[int, int](Config{
		Workers: 8,
		RPS:     rps,
		Burst:   burst,
	})

	// Run two concurrent Transform calls on the SAME Pipeline instance
	const (
		itemsPerCall = 40
		numCallers   = 2
	)

	var wg sync.WaitGroup
	wg.Add(numCallers)

	start := time.Now()
	errorsList := make([]error, numCallers)
	resultsList := make([][]int, numCallers)

	for c := range numCallers {
		go func(callerIdx int) {
			defer wg.Done()
			inputs := make([]int, itemsPerCall)
			for i := range inputs {
				inputs[i] = callerIdx*1000 + i
			}

			resultsList[callerIdx], errorsList[callerIdx] = p.Transform(
				context.Background(),
				inputs,
				func(ctx context.Context, in int) (int, error) {
					return in * 2, nil
				},
			)
		}(c)
	}

	wg.Wait()
	elapsed := time.Since(start)

	for c := range numCallers {
		require.NoError(t, errorsList[c])
		require.Len(t, resultsList[c], itemsPerCall)
		for i := range itemsPerCall {
			expected := (c*1000 + i) * 2
			assert.Equal(t, expected, resultsList[c][i])
		}
	}

	// Total items processed across both callers = 80 items.
	// At 40 RPS, 80 items with burst 2 should take at least ~1.5s
	assert.GreaterOrEqual(t, elapsed, 1400*time.Millisecond)
}

// TestAdversarial_Pipeline_Stream_RateLimiterAndBackpressure tests Stream backpressure
// and rate limiting under consumer delays.
func TestAdversarial_Pipeline_Stream_RateLimiterAndBackpressure(t *testing.T) {
	p := New[int, int](Config{
		Workers: 4,
		RPS:     100.0,
		Burst:   2,
	})

	const numItems = 60
	inCh := make(chan int, 5) // constrained input buffer

	// Producer
	go func() {
		defer close(inCh)
		for i := range numItems {
			inCh <- i
		}
	}()

	start := time.Now()
	outCh, errCh := p.Stream(context.Background(), inCh, func(ctx context.Context, in int) (int, error) {
		return in + 10, nil
	})

	received := make([]int, 0, numItems)
	for val := range outCh {
		received = append(received, val)
		// Consumer backpressure: slow reader
		if len(received)%10 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	for err := range errCh {
		t.Fatalf("unexpected stream error: %v", err)
	}

	elapsed := time.Since(start)
	assert.Len(t, received, numItems)
	// 60 items at 100 RPS should take at least ~500ms
	assert.GreaterOrEqual(t, elapsed, 450*time.Millisecond)
}

// TestAdversarial_Pipeline_FailFast_WorkerPanicStorm verifies panic resilience
// and zero deadlock when multiple worker tasks panic simultaneously.
func TestAdversarial_Pipeline_FailFast_WorkerPanicStorm(t *testing.T) {
	t.Run("FailFast True on Panics", func(t *testing.T) {
		p := New[int, int](Config{
			Workers:  8,
			FailFast: true,
		})

		inputs := make([]int, 100)
		for i := range inputs {
			inputs[i] = i
		}

		_, err := p.Transform(context.Background(), inputs, func(ctx context.Context, in int) (int, error) {
			if in >= 10 && in <= 20 {
				panic(fmt.Sprintf("deliberate worker explosion at %d", in))
			}
			time.Sleep(5 * time.Millisecond)
			return in, nil
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "pipeline: mapper panicked")
	})

	t.Run("FailFast False on Panics joins errors without hang", func(t *testing.T) {
		p := New[int, int](Config{
			Workers:  4,
			FailFast: false,
		})

		inputs := []int{1, 2, 3, 4, 5}
		outputs, err := p.Transform(context.Background(), inputs, func(ctx context.Context, in int) (int, error) {
			if in%2 == 0 {
				panic(fmt.Sprintf("even panic: %d", in))
			}
			return in * 10, nil
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "pipeline: mapper panicked")
		assert.Nil(t, outputs)
	})
}

// TestAdversarial_Pipeline_TransformSeq_ConcurrentStreams verifies TransformSeq
// under multiple concurrent iterators with random early breaks.
func TestAdversarial_Pipeline_TransformSeq_ConcurrentStreams(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	const numCallers = 15
	var wg sync.WaitGroup
	wg.Add(numCallers)

	for c := range numCallers {
		go func(callerIdx int) {
			defer wg.Done()
			cfg := Config{Workers: 4}

			mapper := func(ctx context.Context, in int) (int, error) {
				time.Sleep(time.Duration(rand.Intn(4)) * time.Millisecond)
				return in * 2, nil
			}

			breakAt := 2 + rand.Intn(8)
			count := 0
			for out, err := range TransformSeq(context.Background(), cfg, intSeq(50), mapper) {
				if err != nil {
					return
				}
				_ = out
				count++
				if count == breakAt {
					break
				}
			}
		}(c)
	}

	wg.Wait()

	// Allow goroutines to exit after cancellation propagation
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	finalGoroutines := runtime.NumGoroutine()
	assert.LessOrEqual(t, finalGoroutines, initialGoroutines+3)
}

// TestAdversarial_Pipeline_HighWorkerContention verifies index preservation and
// sync.Pool task recycling under heavy load.
func TestAdversarial_Pipeline_HighWorkerContention(t *testing.T) {
	p := New[int, int](Config{
		Workers: 32,
	})

	const numItems = 2000
	inputs := make([]int, numItems)
	for i := range inputs {
		inputs[i] = i
	}

	var counter atomic.Int64
	outputs, err := p.Transform(context.Background(), inputs, func(ctx context.Context, in int) (int, error) {
		counter.Add(1)
		return in ^ 0x5A5A, nil
	})

	require.NoError(t, err)
	assert.Equal(t, int64(numItems), counter.Load())
	require.Len(t, outputs, numItems)

	for i := range inputs {
		assert.Equal(t, i^0x5A5A, outputs[i])
	}
}

func TestRepro_Pipeline_EdgeCasesAndPanicCoverage(t *testing.T) {
	ctx := context.Background()

	// 1. Nil pipeline receiver in Transform
	var nilP *Pipeline[int, int]
	_, err := nilP.Transform(ctx, []int{1}, func(ctx context.Context, in int) (int, error) {
		return in, nil
	})
	assert.Error(t, err)

	// 2. Nil mapper error handling in Map, ForEach, TransformSeq, Stream
	_, err = Map[int, int](ctx, Config{}, []int{1}, nil)
	assert.Error(t, err)

	err = ForEach[int](ctx, Config{}, []int{1}, nil)
	assert.Error(t, err)

	for _, err := range TransformSeq[int, int](ctx, Config{}, nil, nil) {
		assert.Error(t, err)
	}

	p := New[int, int](Config{Workers: 1})
	_, errCh := p.Stream(ctx, nil, nil)
	err = <-errCh
	assert.Error(t, err)

	// 3. Panic recovery in package-level TransformSeq
	inputs := func(yield func(int) bool) {
		yield(1)
	}
	var panickedErr error
	for _, err := range TransformSeq(ctx, Config{Workers: 1}, inputs, func(ctx context.Context, in int) (int, error) {
		panic("transformseq-panic-test")
	}) {
		if err != nil {
			panickedErr = err
		}
	}
	assert.NotNil(t, panickedErr)

	// 4. Panic recovery in Stream with FailFast: true
	pFailFast := New[int, int](Config{Workers: 1, FailFast: true})
	inCh := make(chan int, 1)
	inCh <- 1
	close(inCh)
	sOut, sErr := pFailFast.Stream(ctx, inCh, func(ctx context.Context, in int) (int, error) {
		panic("stream-panic-test")
	})
	for range sOut {
	}
	var streamErrs []error
	for err := range sErr {
		streamErrs = append(streamErrs, err)
	}
	assert.NotEmpty(t, streamErrs)

	// 5. Rate limiter error in Stream
	ctxCancel, cancel := context.WithCancel(context.Background())
	pLimiter := New[int, int](Config{Workers: 1, RPS: 0.001, Burst: 1, FailFast: true})
	inCh2 := make(chan int, 2)
	inCh2 <- 1
	inCh2 <- 2
	close(inCh2)
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()
	sOut2, sErr2 := pLimiter.Stream(ctxCancel, inCh2, func(ctx context.Context, in int) (int, error) {
		return in, nil
	})
	for range sOut2 {
	}
	var limiterErrs []error
	for err := range sErr2 {
		limiterErrs = append(limiterErrs, err)
	}
	assert.NotEmpty(t, limiterErrs)

	// 6. Saturated errs channel in Stream to trigger non-blocking drop
	pDrop := New[int, int](Config{Workers: 1, FailFast: false})
	inCh3 := make(chan int, 5)
	for i := range 5 {
		inCh3 <- i
	}
	close(inCh3)
	sOut3, sErr3 := pDrop.Stream(ctx, inCh3, func(ctx context.Context, in int) (int, error) {
		return 0, errors.New("saturated-error")
	})
	for range sOut3 {
	}
	for range sErr3 {
	}
}

func TestTransformSeq_FailFastAndRateLimit(t *testing.T) {
	t.Run("FailFast", func(t *testing.T) {
		cfg := Config{
			Workers:  1,
			FailFast: true,
		}
		errDummy := errors.New("seq fail fast")
		var processed atomic.Int32

		seq := TransformSeq(context.Background(), cfg, intSeq(50), func(ctx context.Context, in int) (int, error) {
			processed.Add(1)
			if in == 2 {
				return 0, errDummy
			}
			time.Sleep(5 * time.Millisecond)
			return in * 2, nil
		})

		var errs []error
		for _, err := range seq {
			if err != nil {
				errs = append(errs, err)
			}
		}

		require.NotEmpty(t, errs)
		assert.ErrorIs(t, errs[0], errDummy)
		assert.LessOrEqual(t, processed.Load(), int32(5))
	})

	t.Run("RateLimiting", func(t *testing.T) {
		cfg := Config{
			Workers: 2,
			RPS:     50, // 20ms per request
			Burst:   1,
		}
		start := time.Now()
		count := 0
		for out, err := range TransformSeq(context.Background(), cfg, intSeq(3), func(ctx context.Context, in int) (int, error) {
			return in, nil
		}) {
			require.NoError(t, err)
			_ = out
			count++
		}
		elapsed := time.Since(start)
		assert.Equal(t, 3, count)
		assert.GreaterOrEqual(t, elapsed, 35*time.Millisecond)
	})
}
