// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pipeline

import (
	"context"
	"fmt"
	"math/rand/v2"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

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
				time.Sleep(time.Duration(rand.IntN(4)) * time.Millisecond)
				return in * 2, nil
			}

			breakAt := 2 + rand.IntN(8)
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
